package config

import (
	"runtime"
	"sync/atomic"
)

type KeyConfig struct {
	argon2KeyLength  atomic.Uint32 // 32 = 256 bit || 24 = 192 bit || 16 = 128 bit
	argon2TimeCost   atomic.Uint32 // count cycles
	argon2MemoryCost atomic.Uint32 // 1 = 1KB
	argon2Threads    atomic.Uint32
}

func newKeyConfig() *KeyConfig {
	cfg := &KeyConfig{}
	cfg.argon2KeyLength.Store(32)
	cfg.argon2TimeCost.Store(5)
	cfg.argon2MemoryCost.Store(512 * 1024)
	cfg.argon2Threads.Store(uint32(runtime.NumCPU()))
	return cfg
}

func (cfg *KeyConfig) Argon2KeyLength() uint32 {
	return cfg.argon2KeyLength.Load()
}

func (cfg *KeyConfig) Argon2TimeCost() uint32 {
	return cfg.argon2TimeCost.Load()
}

func (cfg *KeyConfig) Argon2MemoryCost() uint32 {
	return cfg.argon2MemoryCost.Load()
}

func (cfg *KeyConfig) Argon2Threads() uint8 {
	return uint8(cfg.argon2Threads.Load())
}
