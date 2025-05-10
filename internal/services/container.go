package services

type Container struct {
	Keys    *CryptKeysService
	Crypt   CryptService
	Storage StorageService
}
