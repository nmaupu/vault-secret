package vault

import (
	"errors"
	"fmt"
	vapi "github.com/hashicorp/vault/api"
	"path"
	"strings"
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
	sec, err := read(vc, pathV1)
	if err != nil {
		return nil, err
	}

	if contains(sec.Warnings, VaultKVWarning) {
		// Need a V2 KV type read
		sec, err := read(vc, pathV2)
		if err != nil {
			return nil, err
		}

		if sec != nil && sec.Data != nil && sec.Data["data"] != nil {
			data = sec.Data["data"].(map[string]interface{})
		}
	} else {
		data = sec.Data
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
		// An unknown error occured
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
