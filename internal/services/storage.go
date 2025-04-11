package services

import (
	"bytes"
	"encoding/json"
	"encrp/internal/config"
	"encrp/internal/errors"
	"encrp/internal/storage"
	"io"
	"os"
	"runtime"
	"strings"
)

type StorageService interface {
	LoadStorage(path string) error
	SaveStorage(path string) error
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

type FileStorageService struct {
	config        *config.Config
	services      *Container
	storage       *storage.Storage
	pathSeparator string
	rootNode      *storage.Node
	sepRune       rune
}

func NewFileStorageService(cfg *config.Config, services *Container) *FileStorageService {
	sep := cfg.Storage.PathSeparator()
	return &FileStorageService{
		config:        cfg,
		services:      services,
		pathSeparator: sep,
		sepRune:       []rune(sep)[0],
	}
}

func (s *FileStorageService) LoadStorage(path string) error {
	if path == "" {
		return errors.New("FileStorageService.LoadStorage()", "Path is empty")
	}

	file, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "FileStorageService.LoadStorage()", "Failed to open file "+path)
	}
	defer file.Close()

	salt := make([]byte, s.config.Storage.SaltLength())
	n, err := file.Read(salt)
	if err != nil && err != io.EOF {
		return errors.Wrap(err, "FileStorageService.LoadStorage()", "Failed to read salt "+path)
	}
	if err == io.EOF {
		s.storage = storage.NewStorage()
		s.rootNode = s.storage.Data()
		return nil
	}
	if n != s.config.Storage.SaltLength() {
		return errors.Newf("FileStorageService.LoadStorage()", "Failed to read salt %s. There is less %d salt than there should be %d", path, n, s.config.Storage.SaltLength())
	}

	key, err := s.services.Keys.GenerateArgon2IDKey(
		[]byte(s.config.General.Password()),
		salt,
		s.config.Keys.Argon2TimeCost(),
		s.config.Keys.Argon2MemoryCost(),
		s.config.Keys.Argon2KeyLength(),
		s.config.Keys.Argon2Threads(),
	)
	if err != nil {
		return errors.Wrap(err, "FileStorageService.LoadStorage()", "Failed to generate key")
	}

	buffer := &bytes.Buffer{}
	err = s.services.Crypt.DecryptPipe(key, file, buffer)
	if err != nil {
		return errors.Wrap(err, "FileStorageService.LoadStorage()", "Failed to decrypt storage")
	}

	if buffer.Len() == 0 {
		return errors.Wrap(err, "FileStorageService.LoadStorage()", "Failed to load storage "+path)
	}

	s.storage = storage.NewStorage()
	if err = json.NewDecoder(buffer).Decode(s.storage); err != io.EOF && err != nil { // If we receive an end-of-file error, then most likely it is empty and therefore we continue to execute with an empty storage so as not to cause errors
		return errors.Wrap(err, "FileStorageService.LoadStorage()", "Failed to decode file "+path)
	}
	s.storage.UpdateTsOpen()
	s.rootNode = s.storage.Data()
	return nil
}

func (s *FileStorageService) SaveStorage(path string) error {
	if path == "" {
		return errors.New("FileStorageService.SaveStorage()", "Path is empty")
	}

	if s.storage == nil {
		return errors.New("FileStorageService.SaveStorage()", "Storage not exist")
	}
	s.storage.LastDevice().SetFromConfig(s.config.Device)
	s.storage.UpdateTsSave()

	jsonData, err := s.storage.ToJSON()
	if err != nil {
		return errors.Wrap(err, "FileStorageService.SaveStorage()", "Failed to encode file "+path)
	}

	salt, err := s.services.Keys.GenerateSalt(s.config.Storage.SaltLength())
	if err != nil {
		return errors.Wrap(err, "FileStorageService.SaveStorage()", "Failed to generate salt")
	}

	key, err := s.services.Keys.GenerateArgon2IDKey(
		[]byte(s.config.General.Password()),
		salt,
		s.config.Keys.Argon2TimeCost(),
		s.config.Keys.Argon2MemoryCost(),
		s.config.Keys.Argon2KeyLength(),
		s.config.Keys.Argon2Threads(),
	)
	if err != nil {
		return errors.Wrap(err, "FileStorageService.SaveStorage()", "Failed to generate key")
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "FileStorageService.SaveStorage()", "Failed to open file "+path)
	}
	defer file.Close()

	_, err = file.Write(salt)
	if err != nil {
		return errors.Wrap(err, "FileStorageService.SaveStorage()", "Failed to write salt in file "+path)
	}

	err = s.services.Crypt.EncryptPipe(key, bytes.NewReader(jsonData), file)
	if err != nil {
		return errors.Wrap(err, "FileStorageService.SaveStorage()", "Failed to encrypt file")
	}

	return nil
}

func (s *FileStorageService) GetStorage() *storage.Storage {
	return s.storage
}

func (s *FileStorageService) WipeStorage() {
	if s.storage == nil {
		return
	}
	s.storage.Wipe()
	s.rootNode = s.storage.Data()
	runtime.GC()
}

func (s *FileStorageService) GetNode(path string) (*storage.Node, error) {
	if s.storage == nil {
		return nil, errors.New("FileStorageService.GetNode()", "Storage not exist")
	}

	if path == "" || path == s.pathSeparator {
		return s.rootNode, nil
	}

	names := strings.SplitN(strings.Trim(path, s.pathSeparator), s.pathSeparator, -1)
	if len(names) == 0 {
		return nil, errors.New("FileStorageService.GetNode()", "invalid path")
	}

	node := s.rootNode
	for _, name := range names {
		if child := node.Children().Get(name); child != nil {
			node = child
		} else {
			return nil, errors.Newf("FileStorageService.GetNode()", "child with name %s not found", name)
		}
	}
	return node, nil
}

func (s *FileStorageService) PutNode(path string, node *storage.Node) error {
	if s.storage == nil {
		return errors.New("FileStorageService.PutNode()", "Storage not exist")
	}

	if node == nil {
		return errors.New("FileStorageService.PutNode()", "node is nil")
	}

	currentNode := s.rootNode

	if path != "" && path != s.pathSeparator {
		names := strings.SplitN(strings.Trim(path, s.pathSeparator), s.pathSeparator, -1)
		if len(names) == 0 {
			return errors.New("FileStorageService.PutNode()", "invalid path")
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

func (s *FileStorageService) DeleteNode(path string) error {
	if s.storage == nil {
		return errors.New("FileStorageService.DeleteNode()", "Storage not exist")
	}

	if path == "" || path == s.pathSeparator {
		return errors.New("FileStorageService.DeleteNode()", "invalid path or attempt to delete root")
	}

	names := strings.SplitN(strings.Trim(path, s.pathSeparator), s.pathSeparator, -1)
	if len(names) == 0 {
		return errors.New("FileStorageService.DeleteNode()", "invalid path")
	}

	node := s.rootNode
	lastIdx := len(names) - 1
	for _, name := range names[:lastIdx] {
		if child := node.Children().Get(name); child != nil {
			node = child
		} else {
			return errors.Newf("FileStorageService.DeleteNode()", "child with name %s not found", name)
		}
	}

	lastName := names[lastIdx]
	node.Children().Remove(lastName)
	return nil
}

func (s *FileStorageService) WipeNode(path string) error {
	if s.storage == nil {
		return errors.New("FileStorageService.WipeNode()", "Storage not exist")
	}

	if path == "" || path == s.pathSeparator {
		return errors.New("FileStorageService.WipeNode()", "invalid path")
	}

	names := strings.SplitN(strings.Trim(path, s.pathSeparator), s.pathSeparator, -1)
	if len(names) == 0 {
		return errors.New("FileStorageService.WipeNode()", "invalid path")
	}

	node := s.rootNode
	lastIdx := len(names) - 1
	for _, name := range names[:lastIdx] {
		if child := node.Children().Get(name); child != nil {
			node = child
		} else {
			return errors.Newf("FileStorageService.WipeNode()", "child with name %s not found", name)
		}
	}

	lastName := names[lastIdx]
	targetNode := node.Children().Get(lastName)
	if targetNode != nil {
		targetNode.Wipe()
	}
	return nil
}

func (s *FileStorageService) HasNode(path string) bool {
	if s.storage == nil {
		return false
	}

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

func (s *FileStorageService) WalkNodes(fn func(*storage.Node)) error {
	if s.storage == nil {
		return errors.New("FileStorageService.WalkNodes()", "Storage not exist")
	}

	if fn == nil {
		return errors.New("FileStorageService.WalkNodes()", "callback func is nil")
	}
	if s.rootNode == nil {
		return nil
	}

	currentLevel := []*storage.Node{s.rootNode}

	for len(currentLevel) > 0 {
		nextLevel := make([]*storage.Node, 0, len(currentLevel))

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

func (s *FileStorageService) GetData(path string) (*storage.Data, error) {
	if node, err := s.GetNode(path); err == nil {
		return node.Data(), nil
	} else {
		return nil, errors.Wrap(err, "FileStorageService.GetData()", "")
	}
}

func (s *FileStorageService) PutData(path string, data *storage.Data) error {
	if s.storage == nil {
		return errors.New("FileStorageService.PutData()", "Storage not exist")
	}

	if path == "" || path == s.pathSeparator || data == nil {
		return errors.New("FileStorageService.PutData()", "invalid path or nil data")
	}

	names := strings.SplitN(strings.Trim(path, s.pathSeparator), s.pathSeparator, -1)
	if len(names) == 0 {
		return errors.New("FileStorageService.PutData()", "invalid path")
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

func (s *FileStorageService) DeleteData(path string) error {
	if s.storage == nil {
		return errors.New("FileStorageService.DeleteData()", "Storage not exist")
	}

	if path == "" || path == s.pathSeparator {
		return errors.New("FileStorageService.DeleteData()", "invalid path")
	}

	if node, err := s.GetNode(path); err == nil {
		node.Data().Clear()
		return nil
	} else {
		return errors.Wrap(err, "FileStorageService.DeleteData()", "node not found")
	}
}

func (s *FileStorageService) HasData(path string) bool {
	if data, err := s.GetData(path); err == nil && data != nil {
		return len(data.Keys()) > 0
	}
	return false
}

func (s *FileStorageService) GetValue(path, key string) (string, error) {
	if s.storage == nil {
		return "", errors.New("FileStorageService.GetValue()", "Storage not exist")
	}

	if path == "" || path == s.pathSeparator || key == "" {
		return "", errors.New("FileStorageService.GetValue()", "invalid path or key")
	}

	node, err := s.GetNode(path)
	if err != nil {
		return "", errors.Wrap(err, "FileStorageService.GetValue()", "node not found")
	}

	value := node.Data().Get(key)
	if value == "" && !node.Data().Has(key) {
		return "", errors.Newf("FileStorageService.GetValue()", "key %s not found", key)
	}
	return value, nil
}

func (s *FileStorageService) SetValue(path, key, value string) error {
	if s.storage == nil {
		return errors.New("FileStorageService.SetValue()", "Storage not exist")
	}

	if path == "" || path == s.pathSeparator || key == "" {
		return errors.New("FileStorageService.SetValue()", "invalid path or key")
	}

	names := strings.SplitN(strings.Trim(path, s.pathSeparator), s.pathSeparator, -1)
	if len(names) == 0 {
		return errors.New("FileStorageService.SetValue()", "invalid path")
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

func (s *FileStorageService) DeleteValue(path, key string) error {
	if s.storage == nil {
		return errors.New("FileStorageService.DeleteValue()", "Storage not exist")
	}

	if path == "" || path == s.pathSeparator || key == "" {
		return errors.New("FileStorageService.DeleteValue()", "invalid path or key")
	}

	node, err := s.GetNode(path)
	if err != nil {
		return errors.Wrap(err, "FileStorageService.DeleteValue()", "node not found")
	}

	if !node.Data().Has(key) {
		return errors.Newf("FileStorageService.DeleteValue()", "key %s not found", key)
	}
	node.Data().Remove(key)
	return nil
}
