package storage

import (
	"encoding/json"
	"encrp/internal/errors"
	"sync"
	"sync/atomic"
	"time"
)

type Node struct {
	mtx         sync.RWMutex
	name        atomic.Value // string
	tsCreate    int64
	tsModify    atomic.Int64
	description atomic.Value // string
	tags        []string
	data        *Data
	parent      *Node
	children    *Children
}

type storageNode struct {
	Name        string    `json:"name"`
	TsCreate    int64     `json:"ts_create"`
	TsModify    int64     `json:"ts_modify"`
	Description string    `json:"desc"`
	Tags        []string  `json:"tags"`
	Data        *Data     `json:"data"`
	Children    *Children `json:"children"`
}

func NewStorageNode(name string) *Node {
	now := time.Now().Unix()
	node := &Node{
		tsCreate: now,
		data:     NewStorageData(),
		children: NewStorageChildren(),
		tags:     make([]string, 0),
	}
	node.name.Store(name)
	node.tsModify.Store(now)
	node.description.Store("")
	node.children.parent, node.data.parent = node, node
	return node
}

func (n *Node) Name() string        { return n.name.Load().(string) }
func (n *Node) TsCreate() int64     { return n.tsCreate }
func (n *Node) TsModify() int64     { return n.tsModify.Load() }
func (n *Node) Description() string { return n.description.Load().(string) }
func (n *Node) Data() *Data         { return n.data }
func (n *Node) Parent() *Node       { return n.parent }
func (n *Node) Children() *Children { return n.children }

func (n *Node) SetName(name string) error {
	if name == "" {
		return errors.New("Storage.Node.SetName()", "node name cannot be empty")
	}
	if parent := n.Parent(); parent != nil {
		children := parent.Children()
		if children.Has(name) {
			return errors.New("Storage.Node.SetName()", "The node's parent already contains a child with name "+name)
		} else {
			err := children.rename(n.Name(), name)
			if err != nil {
				return errors.Wrap(err, "Storage.Node.SetName()", "Failed to rename child "+name)
			}
			n.name.Store(name)
			n.updateTsModify()
		}
	} else {
		n.name.Store(name)
		n.updateTsModify()
	}
	return nil
}

func (n *Node) SetDescription(description string) {
	n.description.Store(description)
	n.updateTsModify()
}

func (n *Node) SetTags(tags ...string) {
	if len(tags) == 0 {
		return
	}
	n.mtx.Lock()
	defer n.mtx.Unlock()

	existing := make(map[string]struct{}, len(n.tags))
	for _, t := range n.tags {
		existing[t] = struct{}{}
	}

	isUpdated := false
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if _, exists := existing[tag]; !exists {
			n.tags = append(n.tags, tag)
			existing[tag] = struct{}{}
			isUpdated = true
		}
	}

	if isUpdated {
		n.updateTsModify()
	}
}

func (n *Node) RemoveTags(tags ...string) {
	if len(tags) == 0 {
		return
	}
	n.mtx.Lock()
	defer n.mtx.Unlock()

	isUpdated := false
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		for i := 0; i < len(n.tags); i++ {
			if n.tags[i] == tag {
				n.tags[i], n.tags[len(n.tags)-1] = n.tags[len(n.tags)-1], n.tags[i]
				n.tags = n.tags[:len(n.tags)-1]
				isUpdated = true
				break
			}
		}
	}

	if isUpdated {
		n.updateTsModify()
	}
}

func (n *Node) HasTag(tag string) bool {
	if tag == "" {
		return false
	}
	n.mtx.RLock()
	defer n.mtx.RUnlock()

	for _, t := range n.tags {
		if tag == t {
			return true
		}
	}

	return false
}

func (n *Node) RemoveParent() {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	if n.parent != nil {
		n.parent.Children().Remove(n.Name())
	}
}

func (n *Node) WalkToParent(fn func(*Node)) {
	for node := n; node != nil; node = node.parent {
		node.mtx.RLock()
		fn(node)
		node.mtx.RUnlock()
	}
}

func (n *Node) Wipe() {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	n.description.Store("")
	n.tsModify.Store(time.Now().Unix())

	if n.data != nil {
		n.data.Wipe(false)
	}

	if n.children != nil {
		n.children.Wipe(false)
	}

	if n.parent != nil {
		// in this method the method will be called updateTsModify()
		n.parent.Children().Remove(n.Name())
	}
}

func (n *Node) updateTsModify() {
	now := time.Now().Unix()
	for node := n; node != nil; node = node.parent {
		node.tsModify.Store(now)
	}
}

func (n *Node) UnmarshalJSON(data []byte) error {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	tempStorage := &storageNode{}
	if err := json.Unmarshal(data, tempStorage); err != nil {
		return errors.Wrap(err, "Node.UnmarshalJSON()", "Failed to unmarshal storage node")
	}

	n.name.Store(tempStorage.Name)
	n.tsCreate = tempStorage.TsCreate
	n.tsModify.Store(tempStorage.TsModify)
	n.description.Store(tempStorage.Description)
	n.tags = tempStorage.Tags
	n.data = tempStorage.Data
	n.children = tempStorage.Children
	n.data.parent, n.children.parent = n, n

	for _, child := range n.children.children {
		if child != nil {
			child.parent = n
		}
	}

	return nil
}

func (n *Node) MarshalJSON() ([]byte, error) {
	n.mtx.RLock()
	defer n.mtx.RUnlock()

	temp := &storageNode{
		Name:        n.Name(),
		TsCreate:    n.tsCreate,
		TsModify:    n.TsModify(),
		Description: n.Description(),
		Tags:        n.tags,
		Data:        n.data,
		Children:    n.children,
	}

	data, err := json.Marshal(temp)
	if err != nil {
		return nil, errors.Wrap(err, "Node.MarshalJSON()", "Failed marshal storage node")
	}

	return data, nil
}
