/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vault

import "fmt"

// KVWarning is the warning returned by the vault API when the K/V path is invalid (wrong version)
const KVWarning = "Invalid path for a versioned K/V secrets engine."

// WrongVersionError represents an error raised when the KV version is not correct
type WrongVersionError struct {
	Message string
}

// Error
func (e *WrongVersionError) Error() string {
	return e.Message
}

// PathNotFound represents an error when a path is not found in vault
type PathNotFound struct {
	Path string
}

// Error
func (e *PathNotFound) Error() string {
	return fmt.Sprintf("Path %s not found", e.Path)
}
