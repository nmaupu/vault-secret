package vault

import (
	"path"
	"strings"

	vapi "github.com/hashicorp/vault/api"
)

const (
	VaultKVWarning = "Invalid path for a versioned K/V secrets engine."
)

// Read reads a path from vault taking account KV version 1 and 2 if specified or automatically if not
// to automatically discover the kvVersion of the backend, pass kvVersion = 0
func Read(vc *vapi.Client, kvPath string, secretPath string, kvVersion int) (map[string]interface{}, error) {
	switch kvVersion {
	case 1:
		sec, err := read(vc, path.Join(kvPath, secretPath))
		if err != nil {
			return nil, err
		}
		return sec.Data, nil
	case 2:
		sec, err := read(vc, path.Join(kvPath, "data", secretPath))
		if err != nil {
			return nil, err
		}
		return sec.Data["data"].(map[string]interface{}), nil
	}

	// Defaulting to auto detecting KV version
	return readWithAutoKvVersion(vc, kvPath, secretPath)
}

func readWithAutoKvVersion(vc *vapi.Client, kvPath string, secretPath string) (map[string]interface{}, error) {
	p := path.Join(kvPath, secretPath)

	pathV1 := path.Join(kvPath, secretPath)
	pathV2 := path.Join(kvPath, "data", secretPath)

	var data map[string]interface{}

	// Trying V1 type URL
	// Might fail (err!=nil with a 403) if policy is for a v2 backend (including /data in the path)
	sec, err := read(vc, pathV1)
	if err != nil {
		switch err.(type) {
		case *WrongVersionError:
			// Need a V2 KV type read
			sec, err := read(vc, pathV2)
			if err != nil {
				return nil, err
			}

			if sec != nil && sec.Data != nil && sec.Data["data"] != nil {
				// Get the inner data object (v2 KV)
				data = sec.Data["data"].(map[string]interface{})
			}
		default:
			return nil, err
		}
	} else if sec != nil {
		// Get the raw data object (v1 KV)
		data = sec.Data
	} else {
		return nil, &PathNotFound{p}
	}

	return data, nil
}

func read(vc *vapi.Client, p string) (*vapi.Secret, error) {
	//reqLogger := log.WithValues()

	sec, err := vc.Logical().Read(p)
	//reqLogger.Info(fmt.Sprintf("Reading from vault path=%s, sec=%+v", p, sec))

	if err != nil {
		// An unknown error occurred
		return nil, err
	} else if err == nil && sec != nil && contains(sec.Warnings, VaultKVWarning) >= 0 {
		// Calling with a v1 path but needs v2 path
		idx := contains(sec.Warnings, VaultKVWarning)
		return nil, &WrongVersionError{sec.Warnings[idx]}
	} else if err == nil && sec == nil {
		return nil, &PathNotFound{p}
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
