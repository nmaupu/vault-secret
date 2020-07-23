/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
