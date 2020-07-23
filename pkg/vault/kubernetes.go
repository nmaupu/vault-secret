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

var (
	_ AuthProvider = KubernetesProvider{}
)

// KubernetesProvider is a provider to authenticate using the Vault Kubernetes Auth Method plugin
// https://www.vaultproject.io/docs/auth/kubernetes
type KubernetesProvider struct {
	// Role to use for the authentication
	Role string
	// Cluster is the path to use to call the login URL
	Cluster string
	// JWT token to use for the authentication
	jwt string
}

// NewKubernetesProvider creates a new KubernetesProvider object
func NewKubernetesProvider(role, cluster, jwt string) *KubernetesProvider {
	return &KubernetesProvider{
		Role:    role,
		Cluster: cluster,
		jwt:     jwt,
	}
}

// SetJWT set the jwt token to use for authentication
func (k *KubernetesProvider) SetJWT(jwt string) {
	k.jwt = jwt
}

// Login - godoc
func (k KubernetesProvider) Login(c *Config) (*vapi.Client, error) {
	reqLogger := log.WithValues("func", "KubernetesProvider.Login")
	reqLogger.Info("Authenticating using Kubernetes auth method")

	if k.jwt == "" {
		return nil, fmt.Errorf("Token is empty, please provide a valid jwt token")
	}

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
		"role": k.Role,
		"jwt":  k.jwt,
	}
	s, err := vclient.Logical().Write(fmt.Sprintf("auth/%s/login", k.Cluster), data)
	if err != nil {
		return nil, err
	}

	vclient.SetToken(s.Auth.ClientToken)
	return vclient, nil
}
