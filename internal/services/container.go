package services

type Container struct {
	Keys    *KeysService
	Crypt   CryptService
	Storage StorageService
}
