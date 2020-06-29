package vault

import (
	"errors"
	"fmt"
	"path"
	"strings"

	vapi "github.com/hashicorp/vault/api"
)

const (
	VaultKVWarning = "Invalid path for a versioned K/V secrets engine."
)

// Read a path from vault taking account KV version 1 and 2 automatically
func Read(vc *vapi.Client, kvPath string, secretPath string) (map[string]interface{}, error) {
	p := path.Join(kvPath, secretPath)

	pathV1 := path.Join(kvPath, secretPath)
	pathV2 := path.Join(kvPath, "data", secretPath)

	var data map[string]interface{}

	// Trying V1 type URL
	// Might fail (err!=nil with a 403) if policy is for a v2 backend (including data in the path)
	sec, err := read(vc, pathV1)
	if err != nil || (sec != nil && contains(sec.Warnings, VaultKVWarning)) {
		// Need a V2 KV type read
		sec, err := read(vc, pathV2)
		if err != nil {
			return nil, err
		}

		if sec != nil && sec.Data != nil && sec.Data["data"] != nil {
			data = sec.Data["data"].(map[string]interface{})
		}
	} else if sec != nil {
		data = sec.Data
	} else {
		data = nil
	}

	if data == nil {
		return nil, errors.New(fmt.Sprintf("No secret found for path=%v", p))
	} else {
		return data, nil
	}
}

func read(vc *vapi.Client, p string) (*vapi.Secret, error) {
	sec, err := vc.Logical().Read(p)
	if err != nil {
		// An unknown error occurred
		return nil, err
	} else if err == nil && sec == nil {
		return nil, errors.New(fmt.Sprintf("Secret path %s not found", p))
	} else {
		return sec, nil
	}
}

// Check wether s contains str or not
func contains(s []string, str string) bool {
	for _, v := range s {
		if strings.Contains(v, str) {
			return true
		}
	}

	return false
}
