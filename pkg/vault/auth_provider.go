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
