package v1beta1

import (
	"errors"
	nmvault "github.com/nmaupu/vault-secret/pkg/vault"
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

// Configuration part of a vault-secret object
// +k8s:openapi-gen=true
type VaultSecretSpecConfig struct {
	Addr     string                    `json:"addr,required"`
	Insecure bool                      `json:"insecure,omitempty"`
	Auth     VaultSecretSpecConfigAuth `json:"auth,required"`
}

// Mean of authentication for Vault
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

// Define secrets to create from Vault
// +k8s:openapi-gen=true
type VaultSecretSpecSecret struct {
	// Key name in the secret to create
	SecretKey string `json:"secretKey,required"`
	// Path of the vault secret
	Path string `json:"path,required"`
	// Field to retrieve from the path
	Field string `json:"field,required"`
}

// Status field regarding last custom resource process
// +k8s:openapi-gen=true
type VaultSecretStatus struct {
	Entries []VaultSecretStatusEntry `json:"entries,omitempty"`
}

// Entry for the status field
// +k8s:openapi-gen=true
type VaultSecretStatusEntry struct {
	Secret    VaultSecretSpecSecret `json:"secret,required"`
	Status    bool                  `json:"status,required"`
	Message   string                `json:"message,omitempty"`
	RootError string                `json:"rootError,omitempty"`
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

// Get VaultAuthProvider implem from custom resource object
func (cr *VaultSecret) GetVaultAuthProvider() (nmvault.VaultAuthProvider, error) {
	// Checking order:
	//   - Token
	//   - AppRole
	//   - Kubernetes Auth Method
	if cr.Spec.Config.Auth.Token != "" {
		return nmvault.NewTokenProvider(cr.Spec.Config.Auth.Token), nil
	} else if cr.Spec.Config.Auth.AppRole.RoleID != "" {
		appRoleName := "approle" // Default approle name value
		if cr.Spec.Config.Auth.AppRole.Name != "" {
			appRoleName = cr.Spec.Config.Auth.AppRole.Name
		}
		return nmvault.NewAppRoleProvider(
			appRoleName,
			cr.Spec.Config.Auth.AppRole.RoleID,
			cr.Spec.Config.Auth.AppRole.SecretID,
		), nil
	} else if cr.Spec.Config.Auth.Kubernetes.Role != "" {
		return nmvault.NewKubernetesProvider(
			cr.Spec.Config.Auth.Kubernetes.Role,
			cr.Spec.Config.Auth.Kubernetes.Cluster,
		), nil
	}

	return nil, errors.New("Cannot find a way to authenticate, please choose between Token, AppRole or Kubernetes")
}
