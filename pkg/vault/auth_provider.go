package vault

import (
	vapi "github.com/hashicorp/vault/api"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	log = logf.Log.WithName("vault-auth-provider")
)

type VaultConfig struct {
	Address   string
	Namespace string
	Insecure  bool
}

func NewVaultConfig(address string) *VaultConfig {
	return &VaultConfig{
		Address:   address,
		Namespace: "",
		Insecure:  false,
	}
}

type VaultAuthProvider interface {
	Login(*VaultConfig) (*vapi.Client, error)
}
