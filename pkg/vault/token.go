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
