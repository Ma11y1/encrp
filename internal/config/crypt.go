package config

type CryptConfig struct {
	minBlockSize int64
	maxBlockSize int64
}

func newCryptConfig() *CryptConfig {
	return &CryptConfig{
		minBlockSize: 512,
		maxBlockSize: 64 * 1024,
	}
}

func (c *CryptConfig) MinBlockSize() int64 {
	return c.minBlockSize
}

func (c *CryptConfig) MaxBlockSize() int64 {
	return c.maxBlockSize
}
