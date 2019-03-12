package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VaultSecretSpec defines the desired state of VaultSecret
// +k8s:openapi-gen=true
type VaultSecretSpec struct {
	Config          VaultSecretSpecConfig   `json:"config,required"`
	Secrets         []VaultSecretSpecSecret `json:"secrets,required"`
	SecretName      string                  `json:"secretName,omitempty"`
	TargetNamespace string                  `json:"targetNamespace,omitempty"`
}

type VaultSecretSpecConfig struct {
	Addr     string                    `json:"addr,required"`
	Insecure bool                      `json:"insecure,omitempty"`
	Auth     VaultSecretSpecConfigAuth `json:"auth,required"`
}

type VaultSecretSpecConfigAuth struct {
	Token      string `json:"token, omitempty"`
	Kubernetes struct {
		Role    string `json:"role,required"`
		Cluster string `json:"cluster,required"`
	} `json:"kubernetes,omitempty"`
	AppRole struct {
		Name     string `json:"name,omitemty"`
		RoleID   string `json:"role_id,required"`
		SecretID string `json:"secret_id,required"`
	} `json:"approle,omitempty"`
}

type VaultSecretSpecSecret struct {
	// Key name in the secret to create
	SecretKey string `json:"secretKey,required"`
	// Path of the vault secret
	Path string `json:"path,required"`
	// Field to retrieve from the path
	Field string `json:"field,required"`
}

// +k8s:openapi-gen=true
type VaultSecretStatus struct {
	Configured bool `json:"configured"`
	UpToDate   bool `json:"upToDate"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VaultSecret is the Schema for the vaultsecrets API
// +k8s:openapi-gen=true
type VaultSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VaultSecretSpec   `json:"spec,omitempty"`
	Status VaultSecretStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VaultSecretList contains a list of VaultSecret
type VaultSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VaultSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VaultSecret{}, &VaultSecretList{})
}
