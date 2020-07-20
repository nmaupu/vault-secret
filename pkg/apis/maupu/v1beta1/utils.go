package v1beta1

import (
	"errors"

	"github.com/nmaupu/vault-secret/pkg/k8sutils"
	nmvault "github.com/nmaupu/vault-secret/pkg/vault"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BySecretKey allows sorting an array of VaultSecretSpecSecret by SecretKey
type BySecretKey []VaultSecretSpecSecret

// Len returns the len of a BySecretKey object
func (a BySecretKey) Len() int { return len(a) }

// Swap swaps two elements of a BySecretKey object
func (a BySecretKey) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less checks if a given SecretKey object is lexicographically inferior to another SecretKey object
func (a BySecretKey) Less(i, j int) bool { return a[i].SecretKey < a[j].SecretKey }

// GetVaultAuthProvider implem from custom resource object
func (cr *VaultSecret) GetVaultAuthProvider(c client.Client) (nmvault.VaultAuthProvider, error) {
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
		// Retrieving token from the serviceAccount configured
		saName := cr.Spec.Config.Auth.Kubernetes.ServiceAccount
		if saName == "" {
			saName = "default"
		}

		tok, err := k8sutils.GetTokenFromSA(c, cr.Namespace, saName)
		if err != nil {
			return nil, err
		}

		return nmvault.NewKubernetesProvider(
			cr.Spec.Config.Auth.Kubernetes.Role,
			cr.Spec.Config.Auth.Kubernetes.Cluster,
			tok,
		), nil
	}

	return nil, errors.New("Cannot find a way to authenticate, please choose between Token, AppRole or Kubernetes")
}
