package vault

import (
	"crypto/tls"
	"fmt"
	vapi "github.com/hashicorp/vault/api"
	"io/ioutil"
	"net/http"
)

const (
	KubernetesTokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

var (
	_ VaultAuthProvider = KubernetesProvider{}
)

type KubernetesProvider struct {
	Role    string
	Cluster string
}

func NewKubernetesProvider(role, cluster string) *KubernetesProvider {
	return &KubernetesProvider{
		Role:    role,
		Cluster: cluster,
	}
}

func (k KubernetesProvider) Login(c *VaultConfig) (*vapi.Client, error) {
	config := vapi.DefaultConfig()
	config.Address = c.Address
	config.HttpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: c.Insecure},
	}

	vclient, err := vapi.NewClient(config)
	if err != nil {
		return nil, err
	}

	jwtData, err := ioutil.ReadFile(KubernetesTokenFile)
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"role": k.Role,
		"jwt":  string(jwtData),
	}
	s, err := vclient.Logical().Write(fmt.Sprintf("auth/%s/login", k.Cluster), data)
	if err != nil {
		return nil, err
	}

	vclient.SetToken(s.Auth.ClientToken)
	return vclient, nil
}
