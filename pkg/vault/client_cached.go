package vault

import (
	"fmt"

	vapi "github.com/hashicorp/vault/api"
)

var _ Client = (*CachedClient)(nil)

// CachedClient represents a vault client which caches results from vault for later use
type CachedClient struct {
	SimpleClient
	cache map[string](map[string]interface{})
}

// NewCachedClient creates a pointer to a CachedClient struct
func NewCachedClient(client *vapi.Client) *CachedClient {
	return &CachedClient{
		SimpleClient: SimpleClient{
			client: client,
		},
		cache: make(map[string](map[string]interface{})),
	}
}

// Read implem for CachedClient struct
func (c *CachedClient) Read(kvVersion int, kvPath string, secretPath string) (map[string]interface{}, error) {
	reqLogger := log.WithValues("func", "CachedClient.Read")

	var err error
	var secret map[string]interface{}

	cacheKey := fmt.Sprintf("%s/%s", kvPath, secretPath)
	if cachedSecret, found := c.cache[cacheKey]; found {
		reqLogger.Info("Retreiving vault value from cache", "kvPath", kvPath, "path", secretPath)
		secret = cachedSecret
		err = nil
	} else {
		secret, err = c.SimpleClient.Read(kvVersion, kvPath, secretPath)
		if err == nil && secret != nil { // only cache value if there is no error and a sec returned
			reqLogger.Info("Caching vault value", "kvPath", kvPath, "path", secretPath)
			c.cache[cacheKey] = secret
		}
	}
	return secret, err
}

// Clear clears the existing cache
func (c *CachedClient) Clear() {
	c.cache = make(map[string](map[string]interface{}))
}
