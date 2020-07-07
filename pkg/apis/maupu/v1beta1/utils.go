package v1beta1

import (
	"errors"

	nmvault "github.com/nmaupu/vault-secret/pkg/vault"
)

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

// BySecretKey allows sorting an array of VaultSecretSpecSecret by SecretKey
type BySecretKey []VaultSecretSpecSecret

// Len returns the len of a BySecretKey object
func (a BySecretKey) Len() int { return len(a) }

// Swap swaps two elements of a BySecretKey object
func (a BySecretKey) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less checks if a given SecretKey object is lexicographically inferior to another SecretKey object
func (a BySecretKey) Less(i, j int) bool { return a[i].SecretKey < a[j].SecretKey }
