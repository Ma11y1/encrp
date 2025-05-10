package config

import (
	"encrp/internal/utils"
)

type General struct {
	version  string
	password *utils.SecurePassword
}

func newGeneral() *General {
	cfg := &General{version: "1.2"}
	password, err := utils.NewSecurePassword("")
	if err != nil {
		panic(err)
	}
	cfg.password = password
	return cfg
}

func (h *General) Passphrase() string {
	return h.password.Get()
}

func (h *General) PassphraseBytes() []byte {
	return h.password.GetBytes()
}

func (h *General) WipePassphrase() {
	h.password.Wipe()
}

func (h *General) SetPassphrase(pwd string) {
	h.password.Set(pwd)
}

func (h *General) Version() string {
	return h.version
}
