package vault

import (
	"fmt"
	"path"
	"strings"

	vapi "github.com/hashicorp/vault/api"
)

var _ Client = (*SimpleClient)(nil)

// SimpleClient is a simplistic client to connect to vault
type SimpleClient struct {
	client *vapi.Client
}

// NewSimpleClient creates a pointer to a SimpleClient struct
func NewSimpleClient(client *vapi.Client) *SimpleClient {
	return &SimpleClient{
		client: client,
	}
}

// Read implem for SimpleClient struct
func (c *SimpleClient) Read(kvVersion int, kvPath string, secretPath string) (map[string]interface{}, error) {
	switch kvVersion {
	case KvVersion1:
		sec, err := c.read(path.Join(kvPath, secretPath))
		if err != nil {
			return nil, err
		}
		return sec.Data, nil
	case KvVersion2:
		sec, err := c.read(path.Join(kvPath, "data", secretPath))
		if err != nil {
			return nil, err
		}
		return sec.Data["data"].(map[string]interface{}), nil
	case KvVersionAuto:
		_, version, err := kvPreflightVersionRequest(c.client, kvPath)
		if err != nil {
			return nil, err
		}
		return c.Read(version, kvPath, secretPath)
	default:
		return nil, fmt.Errorf("unknown version %d", kvVersion)
	}
}

func (c *SimpleClient) read(path string) (*vapi.Secret, error) {
	sec, err := c.client.Logical().Read(path)

	if err != nil {
		// An unknown error occurred
		return nil, err
	} else if err == nil && sec != nil && contains(sec.Warnings, KVWarning) >= 0 {
		// Calling with a v1 path but needs v2 path
		idx := contains(sec.Warnings, KVWarning)
		return nil, &WrongVersionError{sec.Warnings[idx]}
	} else if err == nil && sec == nil {
		return nil, &PathNotFound{path}
	} else {
		return sec, nil
	}
}

// Check wether s contains str or not
func contains(s []string, str string) int {
	for k, v := range s {
		if strings.Contains(v, str) {
			return k
		}
	}
	return -1
}
