package services

import (
	"bytes"
	"encoding/json"
	"encrp/internal/config"
	"encrp/internal/errors"
	"encrp/internal/storage"
	"io"
	"os"
)

type StorageProvider interface {
	LoadStorage() error
	SaveStorage() error
	SaveToFile() error
}

type FileStorageProviderService struct {
	config   *config.Config
	services *Container
}

func NewFileStorageProviderService(cfg *config.Config, services *Container) *FileStorageProviderService {
	return &FileStorageProviderService{config: cfg, services: services}
}

func (s *FileStorageProviderService) LoadStorage() error {
	path := s.config.Storage.Path()
	if path == "" {
		return errors.New("StorageProviderService.LoadStorage()", "Path is empty")
	}

	file, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "StorageProviderService.LoadStorage()", "Failed to open file "+path)
	}
	defer file.Close()

	salt := make([]byte, s.config.Storage.SaltLength())
	n, err := file.Read(salt)
	if err != nil && err != io.EOF {
		return errors.Wrap(err, "StorageProviderService.LoadStorage()", "Failed to read salt "+path)
	}
	if err == io.EOF {
		err = s.services.Storage.LoadStorage(storage.NewStorage())
		if err != nil {
			return errors.Wrap(err, "StorageProviderService.LoadStorage()", "Failed to load storage "+path)
		}
		return nil
	}
	if n != s.config.Storage.SaltLength() {
		return errors.Newf("StorageProviderService.LoadStorage()", "Failed to read salt %s. There is less %d salt than there should be %d", path, n, s.config.Storage.SaltLength())
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
		return errors.Wrap(err, "StorageProviderService.LoadStorage()", "Failed to generate key")
	}

	buffer := &bytes.Buffer{}
	err = s.services.Crypt.DecryptPipe(key, file, buffer)
	if err != nil {
		return errors.Wrap(err, "StorageProviderService.LoadStorage()", "Failed to decrypt storage")
	}

	if buffer.Len() == 0 {
		err = s.services.Storage.LoadStorage(storage.NewStorage())
		if err != nil {
			return errors.Wrap(err, "StorageProviderService.LoadStorage()", "Failed to load storage "+path)
		}
		return nil
	}

	strg := storage.NewStorage()
	if err = json.NewDecoder(buffer).Decode(strg); err != io.EOF && err != nil { // If we receive an end-of-file error, then most likely it is empty and therefore we continue to execute with an empty storage so as not to cause errors
		return errors.Wrap(err, "StorageProviderService.LoadStorage()", "Failed to decode file "+path)
	}
	strg.UpdateTsOpen()

	err = s.services.Storage.LoadStorage(strg)
	if err != nil {
		return errors.Wrap(err, "StorageProviderService.LoadStorage()", "Failed to load storage "+path)
	}

	return nil
}

func (s *FileStorageProviderService) SaveStorage() error {
	path := s.config.Storage.Path()
	if path == "" {
		return errors.New("StorageProviderService.SaveStorage()", "Path is empty")
	}

	strg := s.services.Storage.GetStorage()
	if strg == nil {
		return errors.New("StorageProviderService.SaveStorage()", "Storage not exist")
	}
	strg.LastDevice().SetFromConfig(s.config.Device)
	strg.UpdateTsSave()

	jsonData, err := strg.ToJSON()
	if err != nil {
		return errors.Wrap(err, "StorageProviderService.SaveStorage()", "Failed to encode file "+path)
	}

	salt, err := s.services.Keys.GenerateSalt(s.config.Storage.SaltLength())
	if err != nil {
		return errors.Wrap(err, "StorageProviderService.SaveStorage()", "Failed to generate salt")
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
		return errors.Wrap(err, "StorageProviderService.SaveStorage()", "Failed to generate key")
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "StorageProviderService.SaveStorage()", "Failed to open file "+path)
	}
	defer file.Close()

	_, err = file.Write(salt)
	if err != nil {
		return errors.Wrap(err, "StorageProviderService.SaveStorage()", "Failed to write salt in file "+path)
	}

	err = s.services.Crypt.EncryptPipe(key, bytes.NewReader(jsonData), file)
	if err != nil {
		return errors.Wrap(err, "StorageProviderService.SaveStorage()", "Failed to encrypt file")
	}

	return nil
}

func (s *FileStorageProviderService) SaveToFile() error {
	return s.SaveStorage()
}
