package config

type Config struct {
	General *General
	Crypt   *CryptConfig
	Keys    *KeyConfig
	Storage *StorageConfig
	Device  *Device
}

func NewConfig() *Config {
	return &Config{
		General: newGeneral(),
		Crypt:   newCryptConfig(),
		Keys:    newKeyConfig(),
		Storage: newStorageConfig(),
		Device:  newDevice(),
	}
}
