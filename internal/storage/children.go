package storage

import (
	"encoding/json"
	"encrp/internal/errors"
	"sync"
)

type Children struct {
	mtx      sync.RWMutex
	parent   *Node
	children map[string]*Node
	cNames   []string
	cNodes   []*Node
}

type storageChildren struct {
	Children map[string]*Node `json:"children"`
}

func NewStorageChildren() *Children {
	return &Children{children: make(map[string]*Node), cNames: make([]string, 0), cNodes: make([]*Node, 0)}
}

func (s *Children) Get(name string) *Node {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.children[name]
}

func (s *Children) Add(child *Node) {
	if child == nil || child.parent != nil {
		return
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	if child == s.parent {
		return
	}

	if existingChild, ok := s.children[child.Name()]; ok {
		existingChild.parent = nil
	}

	child.parent = s.parent
	s.children[child.Name()] = child
	s.cNames = s.cNames[:0]
	s.cNodes = s.cNodes[:0]

	if s.parent != nil {
		s.parent.updateTsModify()
	}
}

func (s *Children) Remove(name string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if child, ok := s.children[name]; ok {
		child.parent = nil
		delete(s.children, name)
		s.cNames = s.cNames[:0]
		s.cNodes = s.cNodes[:0]
		if s.parent != nil {
			s.parent.updateTsModify()
		}
	}
}

func (s *Children) rename(name, targetName string) error {
	if name == "" || targetName == "" || name == targetName {
		return errors.New("Storage.Children.rename()", "invalid argument names")
	}
	s.mtx.Lock()
	defer s.mtx.Unlock()
	_, nameOk := s.children[name]
	_, targetNameOk := s.children[targetName]

	if nameOk && !targetNameOk {
		s.children[targetName] = s.children[name]
		delete(s.children, name)
	} else {
		return errors.New("Storage.Children.rename()", "The node's parent already contains a child "+targetName)
	}
	return nil
}

func (s *Children) Has(name string) bool {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	_, ok := s.children[name]
	return ok
}

func (s *Children) Names() []string {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	if len(s.cNames) != 0 {
		return s.cNames
	}

	for name := range s.children {
		s.cNames = append(s.cNames, name)
	}
	return s.cNames
}

func (s *Children) Values() []*Node {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	if len(s.cNodes) != 0 {
		return s.cNodes
	}

	for _, child := range s.children {
		s.cNodes = append(s.cNodes, child)
	}
	return s.cNodes
}

func (s *Children) Count() int {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return len(s.children)
}

func (s *Children) Clear() {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for name, child := range s.children {
		child.parent = nil
		delete(s.children, name)
	}
	s.cNames = s.cNames[:0]
	s.cNodes = s.cNodes[:0]

	if s.parent != nil {
		s.parent.updateTsModify()
	}
}

func (s *Children) Wipe(isUpdateTs bool) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	nodes := make([]*Node, 0, len(s.children))
	for _, child := range s.children {
		if child != nil {
			nodes = append(nodes, child)
		}
	}

	if s.parent != nil && s.parent.data != nil {
		s.parent.data.Wipe(false)
	}
	s.children = make(map[string]*Node)
	s.cNames = make([]string, 0)
	s.cNodes = make([]*Node, 0)

	for len(nodes) > 0 {
		node := nodes[len(nodes)-1]
		nodes = nodes[:len(nodes)-1]

		node.mtx.Lock()
		node.description.Store("")
		node.tsModify.Store(0)
		if node.data != nil {
			node.data.Wipe(false)
		}
		if node.children != nil {
			for _, child := range node.children.children {
				if child != nil {
					nodes = append(nodes, child)
				}
			}
			node.children.children = make(map[string]*Node)
			node.children.cNames = make([]string, 0)
			node.children.cNodes = make([]*Node, 0)
		}
		node.parent = nil
		node.mtx.Unlock()
	}

	if isUpdateTs && s.parent != nil {
		s.parent.updateTsModify()
	}
}

func (s *Children) UnmarshalJSON(data []byte) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	temp := &storageChildren{}
	if err := json.Unmarshal(data, temp); err != nil {
		return errors.Wrap(err, "Children.UnmarshalJSON()", "Failed to unmarshal storage children")
	}

	s.children = temp.Children
	return nil
}

func (s *Children) MarshalJSON() ([]byte, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	temp := &storageChildren{Children: s.children}
	data, err := json.Marshal(temp)
	if err != nil {
		return nil, errors.Wrap(err, "Children.MarshalJSON()", "Failed to marshal storage children")
	}
	return data, nil
}
