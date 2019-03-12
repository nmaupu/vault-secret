package vault

import (
	vapi "github.com/hashicorp/vault/api"
)

type VaultConfig struct {
	Address  string
	Insecure bool
}

func NewVaultConfig(address string) *VaultConfig {
	return &VaultConfig{
		Address:  address,
		Insecure: false,
	}
}

type VaultAuthProvider interface {
	Login(*VaultConfig) (*vapi.Client, error)
}
