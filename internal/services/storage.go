package services

import (
	"encrp/internal/config"
	"encrp/internal/errors"
	"encrp/internal/storage"
	"runtime"
	"strings"
)

type StorageService interface {
	LoadStorage(*storage.Storage) error
	GetStorage() *storage.Storage
	WipeStorage()

	GetNode(path string) (*storage.Node, error)
	PutNode(path string, node *storage.Node) error
	DeleteNode(path string) error
	WipeNode(path string) error
	HasNode(path string) bool
	WalkNodes(fn func(node *storage.Node)) error

	GetData(path string) (*storage.Data, error)
	PutData(path string, data *storage.Data) error
	DeleteData(path string) error
	HasData(path string) bool

	GetValue(path, key string) (string, error)
	SetValue(path, key, value string) error
	DeleteValue(path, key string) error
}

type FullStorageService struct {
	config        *config.Config
	services      *Container
	storage       *storage.Storage
	pathSeparator string
	rootNode      *storage.Node
	sepRune       rune
}

func NewStorageService(cfg *config.Config, services *Container) *FullStorageService {
	st := storage.NewStorage()
	sep := cfg.Storage.PathSeparator()
	return &FullStorageService{
		config:        cfg,
		services:      services,
		storage:       st,
		pathSeparator: sep,
		rootNode:      st.Data(),
		sepRune:       []rune(sep)[0],
	}
}

func (s *FullStorageService) LoadStorage(storage *storage.Storage) error {
	if storage == nil || storage.Data() == nil {
		return errors.New("FullStorageService.LoadStorage()", "storage is nil")
	}
	s.storage = storage
	s.rootNode = storage.Data()
	return nil
}

func (s *FullStorageService) GetStorage() *storage.Storage {
	return s.storage
}

func (s *FullStorageService) WipeStorage() {
	s.storage.Wipe()
	s.rootNode = s.storage.Data()
	runtime.GC()
}

func (s *FullStorageService) GetNode(path string) (*storage.Node, error) {
	if path == "" || path == s.pathSeparator {
		return s.rootNode, nil
	}

	names := strings.SplitN(strings.Trim(path, s.pathSeparator), s.pathSeparator, -1)
	if len(names) == 0 {
		return nil, errors.New("FullStorageService.GetNode()", "invalid path")
	}

	node := s.rootNode
	for _, name := range names {
		if child := node.Children().Get(name); child != nil {
			node = child
		} else {
			return nil, errors.Newf("FullStorageService.GetNode()", "child with name %s not found", name)
		}
	}
	return node, nil
}

func (s *FullStorageService) PutNode(path string, node *storage.Node) error {
	if node == nil {
		return errors.New("FullStorageService.PutNode()", "node is nil")
	}

	currentNode := s.rootNode

	if path != "" && path != s.pathSeparator {
		names := strings.SplitN(strings.Trim(path, s.pathSeparator), s.pathSeparator, -1)
		if len(names) == 0 {
			return errors.New("FullStorageService.PutNode()", "invalid path")
		}

		for _, name := range names {
			child := currentNode.Children().Get(name)
			if child == nil {
				child = storage.NewStorageNode(name)
				currentNode.Children().Add(child)
			}
			currentNode = child
		}
	}

	currentNode.Children().Add(node)
	return nil
}

func (s *FullStorageService) DeleteNode(path string) error {
	if path == "" || path == s.pathSeparator {
		return errors.New("FullStorageService.DeleteNode()", "invalid path or attempt to delete root")
	}

	names := strings.SplitN(strings.Trim(path, s.pathSeparator), s.pathSeparator, -1)
	if len(names) == 0 {
		return errors.New("FullStorageService.DeleteNode()", "invalid path")
	}

	node := s.rootNode
	lastIdx := len(names) - 1
	for _, name := range names[:lastIdx] {
		if child := node.Children().Get(name); child != nil {
			node = child
		} else {
			return errors.Newf("FullStorageService.DeleteNode()", "child with name %s not found", name)
		}
	}

	lastName := names[lastIdx]
	node.Children().Remove(lastName)
	return nil
}

func (s *FullStorageService) WipeNode(path string) error {
	if path == "" || path == s.pathSeparator {
		return errors.New("FullStorageService.WipeNode()", "invalid path")
	}

	names := strings.SplitN(strings.Trim(path, s.pathSeparator), s.pathSeparator, -1)
	if len(names) == 0 {
		return errors.New("FullStorageService.WipeNode()", "invalid path")
	}

	node := s.rootNode
	lastIdx := len(names) - 1
	for _, name := range names[:lastIdx] {
		if child := node.Children().Get(name); child != nil {
			node = child
		} else {
			return errors.Newf("FullStorageService.WipeNode()", "child with name %s not found", name)
		}
	}

	lastName := names[lastIdx]
	targetNode := node.Children().Get(lastName)
	if targetNode != nil {
		targetNode.Wipe()
	}
	return nil
}

func (s *FullStorageService) HasNode(path string) bool {
	if path == "" || path == s.pathSeparator {
		return true
	}

	names := strings.SplitN(strings.Trim(path, s.pathSeparator), s.pathSeparator, -1)
	if len(names) == 0 {
		return false
	}

	node := s.rootNode
	for _, name := range names {
		if child := node.Children().Get(name); child != nil {
			node = child
		} else {
			return false
		}
	}
	return true
}

func (s *FullStorageService) WalkNodes(fn func(*storage.Node)) error {
	if fn == nil {
		return errors.New("FullStorageService.WalkNodes()", "callback func is nil")
	}
	if s.rootNode == nil {
		return nil
	}

	currentLevel := []*storage.Node{s.rootNode}

	for len(currentLevel) > 0 {
		nextLevel := make([]*storage.Node, 0, len(currentLevel)) // Оценка емкости

		for _, node := range currentLevel {
			if node != nil {
				fn(node)
				for _, child := range node.Children().Values() {
					if child != nil {
						nextLevel = append(nextLevel, child)
					}
				}
			}
		}

		currentLevel = nextLevel
	}

	return nil
}

func (s *FullStorageService) GetData(path string) (*storage.Data, error) {
	if node, err := s.GetNode(path); err == nil {
		return node.Data(), nil
	} else {
		return nil, errors.Wrap(err, "FullStorageService.GetData()", "")
	}
}

func (s *FullStorageService) PutData(path string, data *storage.Data) error {
	if path == "" || path == s.pathSeparator || data == nil {
		return errors.New("FullStorageService.PutData()", "invalid path or nil data")
	}

	names := strings.SplitN(strings.Trim(path, s.pathSeparator), s.pathSeparator, -1)
	if len(names) == 0 {
		return errors.New("FullStorageService.PutData()", "invalid path")
	}

	node := s.rootNode
	for _, name := range names {
		child := node.Children().Get(name)
		if child == nil {
			child = storage.NewStorageNode(name)
			node.Children().Add(child)
		}
		node = child
	}

	currentData := node.Data()
	currentData.Clear()
	dataMap := data.Keys()
	for _, key := range dataMap {
		currentData.Set(key, data.Get(key))
	}
	return nil
}

func (s *FullStorageService) DeleteData(path string) error {
	if path == "" || path == s.pathSeparator {
		return errors.New("FullStorageService.DeleteData()", "invalid path")
	}

	if node, err := s.GetNode(path); err == nil {
		node.Data().Clear()
		return nil
	} else {
		return errors.Wrap(err, "FullStorageService.DeleteData()", "node not found")
	}
}

func (s *FullStorageService) HasData(path string) bool {
	if data, err := s.GetData(path); err == nil && data != nil {
		return len(data.Keys()) > 0
	}
	return false
}

func (s *FullStorageService) GetValue(path, key string) (string, error) {
	if path == "" || path == s.pathSeparator || key == "" {
		return "", errors.New("FullStorageService.GetValue()", "invalid path or key")
	}

	node, err := s.GetNode(path)
	if err != nil {
		return "", errors.Wrap(err, "FullStorageService.GetValue()", "node not found")
	}

	value := node.Data().Get(key)
	if value == "" && !node.Data().Has(key) {
		return "", errors.Newf("FullStorageService.GetValue()", "key %s not found", key)
	}
	return value, nil
}

func (s *FullStorageService) SetValue(path, key, value string) error {
	if path == "" || path == s.pathSeparator || key == "" {
		return errors.New("FullStorageService.SetValue()", "invalid path or key")
	}

	names := strings.SplitN(strings.Trim(path, s.pathSeparator), s.pathSeparator, -1)
	if len(names) == 0 {
		return errors.New("FullStorageService.SetValue()", "invalid path")
	}

	node := s.rootNode
	for _, name := range names {
		if child := node.Children().Get(name); child != nil {
			node = child
		} else {
			child = storage.NewStorageNode(name)
			node.Children().Add(child)
			node = child
		}
	}

	node.Data().Set(key, value)
	return nil
}

func (s *FullStorageService) DeleteValue(path, key string) error {
	if path == "" || path == s.pathSeparator || key == "" {
		return errors.New("FullStorageService.DeleteValue()", "invalid path or key")
	}

	node, err := s.GetNode(path)
	if err != nil {
		return errors.Wrap(err, "FullStorageService.DeleteValue()", "node not found")
	}

	if !node.Data().Has(key) {
		return errors.Newf("FullStorageService.DeleteValue()", "key %s not found", key)
	}
	node.Data().Remove(key)
	return nil
}
