package config

import (
	"encrp/internal/utils"
)

type General struct {
	version  string
	password *utils.SecurePassword
}

func newGeneral() *General {
	cfg := &General{version: "1.1"}
	password, err := utils.NewSecurePassword("")
	if err != nil {
		panic(err)
	}
	cfg.password = password
	return cfg
}

func (h *General) Password() string {
	return h.password.Get()
}

func (h *General) PasswordBytes() []byte {
	return h.password.GetBytes()
}

func (h *General) WipePassword() {
	h.password.Wipe()
}

func (h *General) SetPassword(pwd string) {
	h.password.Set(pwd)
}

func (h *General) Version() string {
	return h.version
}
