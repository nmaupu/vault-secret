#!/bin/sh

set -eu

main() {
  vault auth enable -path=kubernetes/local kubernetes

  vault policy write admin - <<EOF
path "*" {
  capabilities = ["read", "list", "create", "update", "delete"]
}
EOF

  vault secrets enable -version=2 -path=kubernetes kv

  vault write auth/kubernetes/local/config kubernetes_host="https://kubernetes.default"

  vault write auth/kubernetes/local/role/admin \
    bound_service_account_names="*" \
    bound_service_account_namespaces="*" \
    policies=admin \
    ttl=1h
}

write_secret() {
  path=$1
  key=$2
  value=$3

  vault kv put "$path" "$key"="$value"
}

echo "Initializing vault..."
vault status || (echo "Vault not ready yet ; exiting..." ; exit 1)
main

write_secret kubernetes/secret1/secret foo bar
write_secret kubernetes/secret2/secret another value

echo "Vault initialized successfully..."
