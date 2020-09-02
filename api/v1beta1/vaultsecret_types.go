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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VaultSecretSpec defines the desired state of VaultSecret
// +k8s:openapi-gen=true
type VaultSecretSpec struct {
	Config VaultSecretSpecConfig `json:"config,required"`
	// +listType=set
	Secrets           []VaultSecretSpecSecret `json:"secrets,required"`
	SecretName        string                  `json:"secretName,omitempty"`
	SecretType        corev1.SecretType       `json:"secretType,omitempty"`
	SecretLabels      map[string]string       `json:"secretLabels,omitempty"`
	SecretAnnotations map[string]string       `json:"secretAnnotations,omitempty"`
	SyncPeriod        metav1.Duration         `json:"syncPeriod,omitempty"`
}

// VaultSecretSpecConfig Configuration part of a vault-secret object
type VaultSecretSpecConfig struct {
	Addr      string                    `json:"addr,required"`
	Namespace string                    `json:"namespace,omitempty"`
	Insecure  bool                      `json:"insecure,omitempty"`
	Auth      VaultSecretSpecConfigAuth `json:"auth,required"`
}

// VaultSecretSpecConfigAuth Mean of authentication for Vault
type VaultSecretSpecConfigAuth struct {
	Token      string             `json:"token,omitempty"`
	Kubernetes KubernetesAuthType `json:"kubernetes,omitempty"`
	AppRole    AppRoleAuthType    `json:"approle,omitempty"`
}

// KubernetesAuthType Kubernetes authentication type
type KubernetesAuthType struct {
	Role    string `json:"role,required"`
	Cluster string `json:"cluster,required"`
	// ServiceAccount to use for authentication, using "default" if not provided
	ServiceAccount string `json:"serviceAccount,omitempty"`
}

// AppRoleAuthType AppRole authentication type
type AppRoleAuthType struct {
	Name     string `json:"name,omitempty"`
	RoleID   string `json:"roleId,required"`
	SecretID string `json:"secretId,required"`
}

// VaultSecretSpecSecret Defines secrets to create from Vault
type VaultSecretSpecSecret struct {
	// Key name in the secret to create
	SecretKey string `json:"secretKey,required"`
	// Path of the key-value storage
	KvPath string `json:"kvPath,required"`
	// Path of the vault secret
	Path string `json:"path,required"`
	// Field to retrieve from the path
	Field string `json:"field,required"`
	// KvVersion is the version of the KV backend, if unspecified, try to automatically determine it
	KvVersion int `json:"kvVersion,omitempty"`
}

// VaultSecretStatus Status field regarding last custom resource process
// +k8s:openapi-gen=true
type VaultSecretStatus struct {
	// +listType=set
	Entries []VaultSecretStatusEntry `json:"entries,omitempty"`
}

// VaultSecretStatusEntry Entry for the status field
type VaultSecretStatusEntry struct {
	Secret    VaultSecretSpecSecret `json:"secret,required"`
	Status    bool                  `json:"status,required"`
	Message   string                `json:"message,omitempty"`
	RootError string                `json:"rootError,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VaultSecret is the Schema for the vaultsecrets API
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=vaultsecrets,scope=Namespaced
type VaultSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VaultSecretSpec   `json:"spec,omitempty"`
	Status VaultSecretStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// VaultSecretList contains a list of VaultSecret
type VaultSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VaultSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VaultSecret{}, &VaultSecretList{})
}
