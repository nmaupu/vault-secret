package vault

import (
	vapi "github.com/hashicorp/vault/api"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	log = logf.Log.WithName("vault-auth-provider")
)

// Config is a struct to configure a vault connection
type Config struct {
	Address   string
	Namespace string
	Insecure  bool
}

// NewConfig creates a pointer to a VaultConfig struct
func NewConfig(address string) *Config {
	return &Config{
		Address:   address,
		Namespace: "",
		Insecure:  false,
	}
}

// AuthProvider is an interface to abstract vault methods' connection
type AuthProvider interface {
	Login(*Config) (*vapi.Client, error)
}
