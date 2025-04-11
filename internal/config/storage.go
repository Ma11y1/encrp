package config

import (
	"sync/atomic"
)

type StorageConfig struct {
	path          atomic.Value
	pathSeparator string
	saltLength    int
	rootName      string
}

func newStorageConfig() *StorageConfig {
	cfg := &StorageConfig{}
	cfg.path.Store("db")
	cfg.pathSeparator = "/"
	cfg.saltLength = 512
	cfg.rootName = "root"
	return cfg
}

func (c *StorageConfig) Path() string {
	return c.path.Load().(string)
}

func (c *StorageConfig) SetPath(p string) {
	c.path.Store(p)
}

func (c *StorageConfig) PathSeparator() string {
	return c.pathSeparator
}

func (c *StorageConfig) SaltLength() int {
	return c.saltLength
}

func (c *StorageConfig) RootName() string {
	return c.rootName
}
