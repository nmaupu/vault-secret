package vault

const (
	// KvVersionAuto detects vault kv version automatically
	KvVersionAuto int = iota
	// KvVersion1 sets the vault kv version to 1
	KvVersion1
	// KvVersion2 sets the vault kv version to 2
	KvVersion2
)

// Client is an interface to read data from vault
type Client interface {
	Read(engine int, kvPath string, secretPath string) (map[string]interface{}, error)
}
