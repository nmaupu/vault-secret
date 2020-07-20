package k8sutils

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetTokenFromSA gets the token associated to the first secret located in a k8s' service account
func GetTokenFromSA(cli client.Client, ns, saName string) (string, error) {
	if cli == nil {
		return "", fmt.Errorf("Cannot get token from service account, k8s client is nil")
	}

	// Getting SA
	saClient := &corev1.ServiceAccount{}
	err := cli.Get(context.TODO(), types.NamespacedName{Name: saName, Namespace: ns}, saClient)
	if err != nil && errors.IsNotFound(err) {
		return "", fmt.Errorf("Unable to retrieve service account, err=%v", err)
	}

	if len(saClient.Secrets) == 0 {
		return "", fmt.Errorf("No secret associated with the service account %s/%s", ns, saName)
	}

	// TODO See how to handle this slice of Secrets instead of taking the first one
	saSecret := saClient.Secrets[0]
	secret := &corev1.Secret{}
	err = cli.Get(context.TODO(), types.NamespacedName{Name: saSecret.Name, Namespace: ns}, secret)
	if err != nil {
		return "", fmt.Errorf("Unable to retrieve the secret from the service account, err=%v", err)
	}

	// Finally, set the token
	return string(secret.Data["token"]), nil
}
