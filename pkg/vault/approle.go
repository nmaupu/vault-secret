package vault

import (
	"crypto/tls"
	"fmt"
	"net/http"

	vapi "github.com/hashicorp/vault/api"
)

var _ AuthProvider = (*AppRoleProvider)(nil)

// AppRoleProvider is a provider to connect to vault using AppRole
type AppRoleProvider struct {
	AppRoleName, RoleID, SecretID string
}

// NewAppRoleProvider creates a pointer to a AppRoleProvider struct
func NewAppRoleProvider(appRoleName, roleID, secretID string) *AppRoleProvider {
	return &AppRoleProvider{
		AppRoleName: appRoleName,
		RoleID:      roleID,
		SecretID:    secretID,
	}
}

// Login authenticates to the configured vault server
func (a AppRoleProvider) Login(c *Config) (*vapi.Client, error) {
	log.Info("Authenticating using AppRole auth method")
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

	data := map[string]interface{}{
		"role_id":   a.RoleID,
		"secret_id": a.SecretID,
	}
	s, err := vclient.Logical().Write(fmt.Sprintf("auth/%s/login", a.AppRoleName), data)
	if err != nil {
		return nil, err
	}

	vclient.SetToken(s.Auth.ClientToken)
	return vclient, nil
}
