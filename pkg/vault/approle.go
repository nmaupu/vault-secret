package vault

import (
	"crypto/tls"
	"fmt"
	vapi "github.com/hashicorp/vault/api"
	"net/http"
)

var (
	_ VaultAuthProvider = AppRoleProvider{}
)

type AppRoleProvider struct {
	AppRoleName, RoleID, SecretID string
}

func NewAppRoleProvider(appRoleName, roleID, secretID string) *AppRoleProvider {
	return &AppRoleProvider{
		AppRoleName: appRoleName,
		RoleID:      roleID,
		SecretID:    secretID,
	}
}

func (a AppRoleProvider) Login(c *VaultConfig) (*vapi.Client, error) {
	config := vapi.DefaultConfig()
	config.Address = c.Address
	config.HttpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: c.Insecure},
	}

	vclient, err := vapi.NewClient(config)
	if err != nil {
		return nil, err
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
