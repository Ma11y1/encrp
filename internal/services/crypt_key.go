package services

import (
	"crypto/rand"
	"encrp/internal/config"
	"encrp/internal/errors"
	"encrp/internal/utils"
	"golang.org/x/crypto/argon2"
	"io"
)

type CryptKeysService struct {
	config   *config.Config
	services *Container
}

func NewKeyService(cfg *config.Config, services *Container) *CryptKeysService {
	return &CryptKeysService{
		config:   cfg,
		services: services,
	}
}

func (s *CryptKeysService) GetSalt(data []byte, from, to int) ([]byte, error) {
	res, err := utils.CutData(data, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "CryptKeysService.GenerateSalt()", "")
	}
	return res, nil
}

func (s *CryptKeysService) GetSaltReader(r io.Reader, from, to int) ([]byte, error) {
	res, err := utils.CutDataReader(r, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "CryptKeysService.GetSaltReader()", "")
	}
	return res, nil
}

func (s *CryptKeysService) GenerateSalt(size int) ([]byte, error) {
	salt := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, errors.Wrap(err, "CryptKeysService.GenerateSalt()", "failed to generate salt")
	}
	return salt, nil
}

func (s *CryptKeysService) GenerateArgon2IDKey(password []byte, salt []byte, timeCost, memoryCost, keyLength uint32, threads uint8) ([]byte, error) {
	if len(password) == 0 {
		return nil, errors.Wrap(nil, "CryptKeysService.GenerateArgon2IDKey()", "password cannot be empty")
	}
	return argon2.IDKey(password, salt, timeCost, memoryCost, threads, keyLength), nil
}

func (s *CryptKeysService) GenerateArgon2IKey(password []byte, salt []byte, timeCost, memoryCost, keyLength uint32, threads uint8) ([]byte, error) {
	if len(password) == 0 {
		return nil, errors.Wrap(nil, "CryptKeysService.GenerateArgon2IKey()", "password cannot be empty")
	}
	return argon2.Key(password, salt, timeCost, memoryCost, threads, keyLength), nil
}
