package vault

import (
	"crypto/tls"
	"net/http"

	vapi "github.com/hashicorp/vault/api"
)

var _ AuthProvider = (*TokenProvider)(nil)

// TokenProvider connects to vaut using a bare token
type TokenProvider struct {
	Token string
}

// NewTokenProvider creates a pointer to a TokenProvider
func NewTokenProvider(token string) *TokenProvider {
	return &TokenProvider{
		Token: token,
	}
}

// Login - godoc
func (t TokenProvider) Login(c *Config) (*vapi.Client, error) {
	log.Info("Authenticating using Token auth method")
	config := vapi.DefaultConfig()
	config.Address = c.Address
	config.HttpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: c.Insecure},
	}

	vclient, err := vapi.NewClient(config)
	if err != nil {
		return nil, err
	}

	vaultNamespace := c.Namespace
	if vaultNamespace != "" {
		vclient.SetNamespace(vaultNamespace)
	}

	vclient.SetToken(t.Token)
	return vclient, nil
}
