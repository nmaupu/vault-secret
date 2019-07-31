package vault

import (
	"crypto/tls"
	vapi "github.com/hashicorp/vault/api"
	"net/http"
)

var (
	_ VaultAuthProvider = TokenProvider{}
)

type TokenProvider struct {
	Token string
}

func NewTokenProvider(token string) *TokenProvider {
	return &TokenProvider{
		Token: token,
	}
}

func (t TokenProvider) Login(c *VaultConfig) (*vapi.Client, error) {
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
