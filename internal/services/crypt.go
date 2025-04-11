package services

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encrp/internal/config"
	"encrp/internal/errors"
	"fmt"
	"io"
	"math/big"
)

type CryptService interface {
	Encrypt(key, data []byte) ([]byte, error)
	EncryptPipe(key []byte, in io.Reader, out io.Writer) error
	Decrypt(key, data []byte) ([]byte, error)
	DecryptPipe(key []byte, in io.Reader, out io.Writer) error
}

type CryptAESGCMService struct {
	config       *config.Config
	services     *Container
	minBlockSize int64 // recommended 512
	maxBlockSize int64 // recommended 64 * 1024
}

func NewCryptAESGCMService(cfg *config.Config, services *Container) *CryptAESGCMService {
	return &CryptAESGCMService{config: cfg, services: services, minBlockSize: cfg.Crypt.MinBlockSize(), maxBlockSize: cfg.Crypt.MaxBlockSize()}
}

func (s *CryptAESGCMService) generateBlockSize() (int, error) {
	bigIntBlockSize, err := rand.Int(rand.Reader, big.NewInt(s.maxBlockSize-s.minBlockSize+1))
	if err != nil {
		return 0, fmt.Errorf("failed to generate block size: %w", err)
	}
	return int(bigIntBlockSize.Int64() + s.minBlockSize), nil
}

// Encrypt AES-GCM encrypts data from slice data
//
//	Adds a new 12-byte size initialization vector to each block
//	Adds a sha256 hash to the end of the block to check plaintext integrity
//	Decrypted only by the DecryptPipe() function
//	[iv1...(12 byte)][block size...(2 byte)][ciphertext+checksum-sha256(from plaintext, 32 byte)...(block size)][iv2...(12 byte)][block size...(2 byte)][ciphertext+checksum-sha256(from plaintext, 32 byte)...(block size)]...
func (s *CryptAESGCMService) Encrypt(key, data []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, errors.Newf("CryptAESGCMService.Encrypt()", "invalid key length %d bytes, it must be 16, 24, or 32 bytes", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, "CryptAESGCMService.Encrypt()", "failed to create cipher")
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(err, "CryptAESGCMService.Encrypt()", "failed to create GCM")
	}

	dataLen := len(data)
	hasher := sha256.New()
	iv := make([]byte, aesGCM.NonceSize())
	encryptedData := make([]byte, 0, dataLen)

	for position := 0; position < dataLen; {
		if _, err = rand.Read(iv); err != nil {
			return nil, errors.Wrap(err, "CryptAESGCMService.Encrypt()", "failed to generate IV")
		}

		blockSize, err := s.generateBlockSize()
		if err != nil {
			return nil, errors.Wrap(err, "CryptAESGCMService.Encrypt()", "")
		}

		if position+blockSize > dataLen {
			blockSize = dataLen - position
		}

		blockData := data[position : position+blockSize]

		if _, err = hasher.Write(blockData); err != nil {
			return nil, errors.Wrap(err, "CryptAESGCMService.Encrypt()", "failed to compute block hash of checksum")
		}
		checksum := hasher.Sum(nil)

		ciphertext := aesGCM.Seal(nil, iv, append(blockData, checksum...), nil)

		length := uint16(len(ciphertext))
		encryptedData = append(encryptedData, iv...)
		encryptedData = append(encryptedData, byte(length>>8), byte(length&0xFF))
		encryptedData = append(encryptedData, ciphertext...)

		position += blockSize

		hasher.Reset()
	}

	return encryptedData, nil
}

// EncryptPipe AES-GCM encrypts data from an input reader to an output writer
//
//	Adds a new 12-byte size initialization vector to each block
//	Adds a sha256 hash to the end of the block to check plaintext integrity
//	Decrypted only by the DecryptPipe() function
//	[iv1...(12 byte)][block size...(2 byte)][ciphertext+checksum-sha256(from plaintext, 32 byte)...(block size)][iv2...(12 byte)][block size...(2 byte)][ciphertext+checksum-sha256(from plaintext, 32 byte)...(block size)]...
func (s *CryptAESGCMService) EncryptPipe(key []byte, in io.Reader, out io.Writer) error {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return errors.Newf("CryptAESGCMService.EncryptPipe()", "invalid key length %d bytes, it must be 16, 24, or 32 bytes", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return errors.Wrap(err, "CryptAESGCMService.EncryptPipe()", "failed to create cipher")
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return errors.Wrap(err, "CryptAESGCMService.EncryptPipe()", "failed to create GCM")
	}

	hasher := sha256.New()
	iv := make([]byte, aesGCM.NonceSize())

	for {
		if _, err = rand.Read(iv); err != nil {
			return errors.Wrap(err, "CryptAESGCMService.EncryptPipe()", "failed to generate IV")
		}

		blockSize, err := s.generateBlockSize()
		if err != nil {
			return errors.Wrap(err, "CryptAESGCMService.EncryptPipe()", "")
		}

		buffer := make([]byte, blockSize+sha256.Size)

		n, err := in.Read(buffer[:sha256.Size])
		if err == io.EOF {
			break
		}

		if err != nil {
			return errors.Wrap(err, "CryptAESGCMService.EncryptPipe()", "read error")
		}

		blockData := buffer[:n]

		if _, err = hasher.Write(blockData); err != nil {
			return errors.Wrap(err, "CryptAESGCMService.EncryptPipe()", "failed to compute block hash")
		}

		checksum := hasher.Sum(nil)

		ciphertext := aesGCM.Seal(nil, iv, append(blockData, checksum...), nil)

		// write iv
		if _, err = out.Write(iv); err != nil {
			return errors.Wrap(err, "CryptAESGCMService.EncryptPipe()", "failed to write IV")
		}

		// write block size
		size := uint16(len(ciphertext))
		if _, err = out.Write([]byte{byte(size >> 8), byte(size & 0xFF)}); err != nil {
			return errors.Wrap(err, "CryptAESGCMService.EncryptPipe()", "failed to write block size")
		}

		// write data
		if _, err = out.Write(ciphertext); err != nil {
			return errors.Wrap(err, "CryptAESGCMService.EncryptPipe()", "failed to write ciphertext")
		}

		hasher.Reset()
	}

	return nil
}

// Decrypt decrypts data usage AES-GCM and return it
//
//	CryptKey length only 16, 24, 32
//
//	Expects each encrypted block to start with a 12-byte IV.
//	[iv1...(12 byte)][block size...(2 byte)][ciphertext+checksum-sha256(from plaintext, 32 byte)...(block size)][iv2...(12 byte)][block size...(2 byte)][ciphertext+checksum-sha256(from plaintext, 32 byte)...(block size)]...
func (s *CryptAESGCMService) Decrypt(key, data []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, errors.Newf("CryptAESGCMService.Decrypt()", "invalid key length %d bytes, it must be 16, 24, or 32 bytes", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, "CryptAESGCMService.Decrypt()", "failed to create cipher")
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(err, "CryptAESGCMService.Decrypt()", "failed to create GCM")
	}

	ivSize := aesGCM.NonceSize()
	if len(data) < ivSize+sha256.Size+aesGCM.Overhead() {
		return nil, errors.New("CryptAESGCMService.Decrypt()", "data is too short to contain IV, checksum and aes overhead")
	}

	hasher := sha256.New()
	decryptedData := make([]byte, 0, len(data))

	for position := 0; position < len(data); {
		if position+ivSize > len(data) {
			return nil, errors.Newf("CryptAESGCMService.Decrypt()", "incomplete IV at position %d", position)
		}

		iv := data[position : position+ivSize]
		position += ivSize

		if position+2 > len(data) {
			return nil, errors.Newf("CryptAESGCMService.Decrypt()", "incomplete length field at position %d", position)
		}

		blockSize := int(data[position])<<8 | int(data[position+1])
		position += 2

		if position+blockSize > len(data) {
			return nil, errors.Newf("CryptAESGCMService.Decrypt()", "incomplete block at position %d", position)
		}

		plaintextWithChecksum, err := aesGCM.Open(nil, iv, data[position:position+blockSize], nil)
		if err != nil {
			return nil, errors.Wrapf(err, "CryptAESGCMService.Decrypt()", "failed to decrypt block at position %d", position)
		}

		position += blockSize

		if len(plaintextWithChecksum) < sha256.Size {
			return nil, errors.Newf("CryptAESGCMService.Decrypt()", "decrypted block too short at position %d", position)
		}

		plaintext := plaintextWithChecksum[:len(plaintextWithChecksum)-sha256.Size]

		if _, err = hasher.Write(plaintext); err != nil {
			return nil, errors.Wrap(err, "CryptAESGCMService.Decrypt()", "failed to compute checksum")
		}

		// computed checksum vs expected checksum
		if !bytes.Equal(hasher.Sum(nil), plaintextWithChecksum[len(plaintextWithChecksum)-sha256.Size:]) {
			return nil, errors.Newf("CryptAESGCMService.Decrypt()", "checksum mismatch in block at position %d", position)
		}

		hasher.Reset()

		decryptedData = append(decryptedData, plaintext...)
	}

	return decryptedData, nil
}

// DecryptPipe decrypts data from an input reader to an output writer usage AES-GCM
//
//	CryptKey length only 16, 24, 32
//
//	 Expects each encrypted block to start with a 12-byte IV.
//	[iv1...(12 byte)][block size...(2 byte)][ciphertext+checksum-sha256(from plaintext, 32 byte)...(block size)][iv2...(12 byte)][block size...(2 byte)][ciphertext+checksum-sha256(from plaintext, 32 byte)...(block size)]...
func (s *CryptAESGCMService) DecryptPipe(key []byte, in io.Reader, out io.Writer) error {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return errors.Newf("CryptAESGCMService.DecryptPipe()", "invalid key length %d bytes, it must be 16, 24, or 32 bytes", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return errors.Wrap(err, "CryptAESGCMService.DecryptPipe()", "failed to create cipher")
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return errors.Wrap(err, "CryptAESGCMService.DecryptPipe()", "failed to create GCM")
	}

	if aesGCM.NonceSize() == 0 {
		return errors.Newf("CryptAESGCMService.DecryptPipe()", "invalid nonce size: %d", aesGCM.NonceSize())
	}

	hasher := sha256.New()
	iv := make([]byte, aesGCM.NonceSize())
	blockSizeBuffer := make([]byte, 2)

	for {
		if _, err = io.ReadFull(in, iv); err == io.EOF {
			break
		} else if err != nil {
			return errors.Wrap(err, "CryptAESGCMService.DecryptPipe()", "failed to read IV")
		}

		if _, err = io.ReadFull(in, blockSizeBuffer); err == io.EOF {
			break
		} else if err != nil {
			return errors.Wrap(err, "CryptAESGCMService.DecryptPipe()", "failed to read block size")
		}

		blockSize := binary.BigEndian.Uint16(blockSizeBuffer) // max block length 16 byte

		ciphertext := make([]byte, blockSize)
		if _, err = io.ReadFull(in, ciphertext); err != nil {
			return errors.Wrap(err, "CryptAESGCMService.DecryptPipe()", "failed to read ciphertext")
		}

		plaintext, err := aesGCM.Open(nil, iv, ciphertext, nil)
		if err != nil {
			return errors.Wrap(err, "CryptAESGCMService.DecryptPipe()", "failed to decrypt data")
		}

		dataLen := len(plaintext) - sha256.Size
		if dataLen < 0 {
			return errors.New("CryptAESGCMService.DecryptPipe()", "data too short for checksum")
		}

		data := plaintext[:dataLen]

		if _, err = hasher.Write(data); err != nil {
			return errors.Wrap(err, "CryptAESGCMService.DecryptPipe()", "failed to compute checksum")
		}

		// checksum vs computed checksum
		if !bytes.Equal(plaintext[dataLen:], hasher.Sum(nil)) {
			return errors.New("CryptAESGCMService.DecryptPipe()", "checksum mismatch: data integrity compromised")
		}

		hasher.Reset()

		if _, err = out.Write(data); err != nil {
			return errors.Wrap(err, "CryptAESGCMService.DecryptPipe()", "failed to write data")
		}
	}

	return nil
}
