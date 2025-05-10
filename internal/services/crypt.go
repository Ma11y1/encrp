package services

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encrp/internal/config"
	"encrp/internal/errors"
	"golang.org/x/crypto/pbkdf2"
	"hash"
	"io"
	"math/big"
)

type CryptService interface {
	Encrypt(key []byte, in io.Reader, out io.Writer) error
	Decrypt(key []byte, in io.Reader, out io.Writer) error
}

type CryptAESGCMService struct {
	config            *config.Config
	services          *Container
	version           string
	headerMagicPrefix []byte
	saltSize          int
	keyDerivationIter int
	keyHashFactory    func() hash.Hash
	keyLength         int
	minBlockSize      int64
	maxBlockSize      int64
}

func NewCryptAESGCMService(cfg *config.Config, services *Container) *CryptAESGCMService {
	return &CryptAESGCMService{
		config:            cfg,
		services:          services,
		version:           "1.1",
		saltSize:          32,
		headerMagicPrefix: []byte("EncData__"),
		keyDerivationIter: 100_000,
		keyHashFactory:    sha256.New,
		keyLength:         32,
		minBlockSize:      cfg.Crypt.MinBlockSize(),
		maxBlockSize:      cfg.Crypt.MaxBlockSize(),
	}
}

func (s *CryptAESGCMService) deriveKey(passphrase, salt []byte) []byte {
	return pbkdf2.Key(passphrase, salt, s.keyDerivationIter, s.keyLength, s.keyHashFactory)
}

// writeHeader header structure: [magic prefix][version][salt]
func (s *CryptAESGCMService) writeHeader(out io.Writer, salt []byte) error {
	lenHeaderMagicPrefix := len(s.headerMagicPrefix)
	lenVersion := len(s.version)
	header := make([]byte, lenHeaderMagicPrefix+lenVersion+s.saltSize)
	copy(header, s.headerMagicPrefix)
	copy(header[lenHeaderMagicPrefix:], s.version)
	copy(header[lenHeaderMagicPrefix+lenVersion:], salt[:s.saltSize])
	_, err := out.Write(header)
	return err
}

func (s *CryptAESGCMService) readHeader(in io.Reader) ([]byte, error) {
	lenHeaderMagicPrefix := len(s.headerMagicPrefix)
	lenVersion := len(s.version)
	header := make([]byte, lenHeaderMagicPrefix+lenVersion+s.saltSize)

	if _, err := io.ReadFull(in, header); err != nil {
		return nil, errors.Wrap(err, "CryptService.readHeader()", "failed to read header")
	}

	if !bytes.HasPrefix(header[:lenHeaderMagicPrefix], s.headerMagicPrefix) {
		return nil, errors.New("CryptService.readHeader()", "invalid header")
	}

	if string(header[lenHeaderMagicPrefix:lenHeaderMagicPrefix+lenVersion]) != s.version {
		return nil, errors.New("CryptService.readHeader()", "unsupported version")
	}

	return header, nil
}

func (s *CryptAESGCMService) generateBlockSize() (int, error) {
	diff := s.maxBlockSize - s.minBlockSize + 1
	nBig, err := rand.Int(rand.Reader, big.NewInt(diff))
	if err != nil {
		return 0, errors.Wrap(err, "CryptService.generateBlockSize()", "failed to generate random block size")
	}
	return int(nBig.Int64() + s.minBlockSize), nil
}

func (s *CryptAESGCMService) wipeBytes(b []byte) {
	if b == nil {
		return
	}
	for i := 0; i < len(b); i++ {
		b[i] = 0
	}
}

func (s *CryptAESGCMService) Encrypt(passphrase []byte, in io.Reader, out io.Writer) error {
	salt := make([]byte, s.saltSize)
	if _, err := rand.Read(salt); err != nil {
		return errors.Wrap(err, "CryptService.Encrypt()", "failed to read salt")
	}

	bufOut := bufio.NewWriter(out)
	defer bufOut.Flush()

	if err := s.writeHeader(bufOut, salt); err != nil {
		return errors.Wrap(err, "CryptService.Encrypt()", "failed to write header")
	}

	key := s.deriveKey(passphrase, salt)
	defer s.wipeBytes(key)
	s.wipeBytes(passphrase)
	s.wipeBytes(salt)

	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return errors.Wrap(err, "CryptService.Encrypt()", "failed to create block cipher")
	}

	aesGCM, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return errors.Wrap(err, "CryptService.Encrypt()", "failed to create GCM")
	}

	sequence := uint64(0) // AAD
	buff := make([]byte, s.maxBlockSize)
	defer s.wipeBytes(buff)
	aad := make([]byte, 8)
	hasher := hmac.New(sha256.New, key)

	for {
		blockSize, err := s.generateBlockSize()
		if err != nil {
			return errors.Wrap(err, "CryptService.Encrypt()", "block size error")
		}

		n, err := io.ReadFull(in, buff[:blockSize])
		if err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF) && n == 0 {
			break
		}
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
			return errors.Wrap(err, "CryptService.Encrypt()", "read block data error")
		}
		data := buff[:n]

		iv := make([]byte, aesGCM.NonceSize())
		if _, err := rand.Read(iv); err != nil {
			return errors.Wrap(err, "CryptService.Encrypt()", "generate IV error")
		}

		binary.BigEndian.PutUint64(aad, sequence)

		hasher.Reset()
		hasher.Write(key)
		hasher.Write(aad)
		hasher.Write(iv)
		hasher.Write(data)
		checksum := hasher.Sum(nil)

		payload := make([]byte, len(data)+len(checksum))
		copy(payload, data)
		copy(payload[len(data):], checksum)
		s.wipeBytes(checksum)

		ciphertext := aesGCM.Seal(nil, iv, payload, aad)

		// header struct: [AAD: sequence (uint64 8 byte)][IV: nonce size (12 byte)][block size (uint32 4 byte)]
		header := make([]byte, 8+len(iv)+4)
		binary.BigEndian.PutUint64(header[0:8], sequence)
		copy(header[8:], iv)
		binary.BigEndian.PutUint32(header[8+len(iv):], uint32(len(ciphertext)))

		if _, err := bufOut.Write(header); err != nil {
			return errors.Wrap(err, "CryptService.Encrypt()", "fail write header")
		}
		if _, err := bufOut.Write(ciphertext); err != nil {
			return errors.Wrap(err, "CryptService.Encrypt()", "failed write ciphertext")
		}

		sequence++
	}
	return nil
}

func (s *CryptAESGCMService) Decrypt(passphrase []byte, in io.Reader, out io.Writer) error {
	header, err := s.readHeader(in)
	if err != nil {
		return errors.Wrap(err, "CryptService.Decrypt()", "failed to read header")
	}

	salt := header[len(s.headerMagicPrefix)+len(s.version):][:s.saltSize]
	key := s.deriveKey(passphrase, salt)
	defer s.wipeBytes(key)
	s.wipeBytes(passphrase)

	block, err := aes.NewCipher(key)
	if err != nil {
		return errors.Wrap(err, "CryptService.Decrypt()", "failed to create block cipher")
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return errors.Wrap(err, "CryptService.Decrypt()", "failed to create GCM")
	}

	bufIn := bufio.NewReader(in)
	bufOut := bufio.NewWriter(out)
	defer bufOut.Flush()

	// header struct: [AAD: sequence (uint64 8 byte)][IV: nonce size (12 byte)][block size (uint32 4 byte)]
	blockHeader := make([]byte, 8+aesGCM.NonceSize()+4)
	nonceSize := aesGCM.NonceSize()
	sequence := uint64(0)
	aad := make([]byte, 8)
	hasher := hmac.New(sha256.New, key)

	for {
		_, err = io.ReadFull(bufIn, blockHeader)
		if err == io.EOF {
			break
		} else if err != nil {
			return errors.Wrap(err, "CryptService.Decrypt()", "failed to read block header")
		}

		headerSequence := binary.BigEndian.Uint64(blockHeader[0:8])
		iv := blockHeader[8 : 8+nonceSize]
		length := binary.BigEndian.Uint32(blockHeader[8+nonceSize:])

		if headerSequence != sequence {
			return errors.Newf("CryptService.Decrypt()", "block sequence mismatch: got %d, expected %d", headerSequence, sequence)
		}
		sequence++

		ciphertext := make([]byte, length)
		if _, err = io.ReadFull(bufIn, ciphertext); err != nil {
			return errors.Wrap(err, "CryptService.Decrypt()", "incomplete ciphertext for block")
		}

		binary.BigEndian.PutUint64(aad, headerSequence)

		payload, err := aesGCM.Open(nil, iv, ciphertext, aad)
		if err != nil {
			return errors.Wrapf(err, "CryptService.Decrypt()", "decryption/auth failed for block %d", headerSequence)
		}

		// split data and checksum
		if len(payload) < sha256.Size {
			return errors.Newf("CryptService.Decrypt()", "payload too short in block %d", headerSequence)
		}

		data := payload[:len(payload)-sha256.Size]
		checksum := payload[len(payload)-sha256.Size:]

		hasher.Reset()
		hasher.Write(key)
		hasher.Write(aad)
		hasher.Write(iv)
		hasher.Write(data)
		expected := hasher.Sum(nil)

		if !hmac.Equal(checksum, expected) {
			return errors.Newf("CryptService.Decrypt()", "checksum mismatch in block %d", headerSequence)
		}

		if _, err = bufOut.Write(data); err != nil {
			return errors.Wrap(err, "CryptService.Decrypt()", "failed to write plaintext")
		}
	}

	return nil
}
