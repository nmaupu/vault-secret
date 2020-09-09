[![CircleCI](https://circleci.com/gh/nmaupu/vault-secret/tree/master.svg?style=shield)](https://circleci.com/gh/nmaupu/vault-secret/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/nmaupu/vault-secret)](https://goreportcard.com/report/github.com/nmaupu/vault-secret)

# Kubernetes Secrets from Hashicorp Vault

**Problem:** My secret are stored in Vault, how can I inject them into Kubernetes secret ?

**Solution:** Use vault-secret custom resource to specify Vault server, path and keys and the operator will retrieve all the needed information from vault and push them into a Kubernetes secret resource ready to be used in the cluster.

# Note on upgrading to 1.0.1 onward

From version `1.0.1`, k8s auth method switches from using the local *service account* configured on the operator side to using the one from the client's namespace defined in the *custom resource*.
This is improving security but as a result, you will probably have to check your vault configuration is in adequation with this change.

# Installation

## Kubernetes version requirements

This operator is supported from **Kubernetes `1.10`**.

If using *Kubernetes 1.10* version, the feature gate `CustomResourceSubresources` must be enabled for the Custom Resource status field to get updated!
This feature is enabled by default starting from *Kubernetes 1.11*.

## Operator

Get the latest release from https://github.com/nmaupu/vault-secret/releases

Deploy the Custom Resource Definition and the operator:
```
$ kubectl apply -f config/crd/bases/maupu.org_vaultsecrets.yaml
$ kubectl apply -f config/doc-samples/operator.yaml
$ kubectl apply -f config/doc-samples/role.yaml
$ kubectl apply -f config/doc-samples/role_binding.yaml
$ kubectl apply -f config/doc-samples/service_account.yaml
```

### Configuration

#### Env vars

The *vault-secret operator* can be configured to watch a unique namespace, a set of namespaces or can also be cluster wide. In that case, modify RBAC role and role binding to be cluster scoped.
The following environment variables are available to configure the operator:
- `WATCH_NAMESPACE`: namespace to watch for new CR. If not defined, use `WATCH_MULTINAMESPACES` or configure a cluster wide operator.
- `WATCH_MULTINAMESPACES`: comma separated list of namespaces to watch for new CR, if not defined, the operator will be cluster scoped except if `WATCH_NAMESPACE` is set.
- `OPERATOR_NAME`: name of the operator.

#### Label filtering

One can use the command line flag `--filter-label` to filter which vaultsecret custom resource to process by the operator.
This flag can be used multiple times.

Example usage:

```
--filter-label=mylabel=myvalue
```

## Custom resource

Here is an example (`config/doc-samples/maupu.org_v1beta1_vaultsecrets_cr.yaml`) :
```
apiVersion: maupu.org/v1beta1
kind: VaultSecret
metadata:
  name: example-vaultsecret
  namespace: nma
spec:
  secretName: vault-secret-test
  secretLabels:
    foo: bar
  secretAnnotations:
    foo: bar
  secrets:
    - secretKey: username
      kvPath: secrets/kv
      path: test
      field: username
    - secretKey: password
      kvPath: secrets/kv
      path: test
      field: password
  syncPeriod: 1h
  config:
    addr: https://vault.example.com
    auth:
      kubernetes:
        role: myrole
        cluster: kubernetes
```

A corresponding secret would be created in the same namespace as the *VaultSecret* custom resource.
This secret would contain two keys filled with vault content:
- `username`
- `password`

---

It's possible to add annotations and labels to the generated secret with `secretAnnotations` and `secretLabels`.

Here is another example for "dockerconfig" secrets:
```
apiVersion: maupu.org/v1beta1
kind: VaultSecret
metadata:
  name: dockerconfig-example
  namespace: nma
spec:
  secretName: dockerconfig-test
  secretType: kubernetes.io/dockerconfigjson
  secrets:
    - secretKey: .dockerconfigjson
      kvPath: secrets/dockerconfig
      field: dockerconfigjson
      path: /
  config:
    addr: https://vault.example.com
    auth:
      kubernetes:
        role: myrole
        cluster: kubernetes
```

It's possible to set the secret type in the spec with `secretType`, if it isn't specified the default value is `Opaque`.

---

Secret are resynced periodically (after a maximum of 10h) but it's possible to reduce this delay with the `syncPeriod` option (`syncPeriod: 1h`).

---

If your Vault is using *TLS* but if its certificates are not signed by a *known authority*, one can use the config option `insecure` to skip tls verification.

Do not use `TLS_SKIP_VERIFY` env variable when starting the operator, **it's not** being taken into account.

Here is an example:
```
apiVersion: maupu.org/v1beta1
kind: VaultSecret
metadata:
  name: example-vaultsecret-insecure
spec:
  secretName: vault-secret-test
  secrets:
    - secretKey: foo
      kvPath: secret
      path: foo/bar
      field: value
  config:
    insecure: true
    addr: https://localhost
    auth:
      ...
```

## Vault configuration

To authenticate, the operator uses the `config` section of the Custom Resource Definition. The following options are supported:
- AppRole Auth Method (https://www.vaultproject.io/docs/auth/approle.html)
- Vault Kubernetes Auth Method (https://www.vaultproject.io/docs/auth/kubernetes.html)
- Directly using a token

The prefered way is to use *Vault Kubernetes Auth Method* because the other authentication methods require to push a *secret* into the custom resource (e.g. `token` or `role_id/secret_id`).

### Kubernetes Auth Method usage

```
  config:
    addr: https://vault.example.com
    auth:
      kubernetes:
        role: myrole
        cluster: kubernetes
```

The section `kubernetes` takes two arguments:
  - `role`: role associated with the *service account* configured.
  - `cluster`: name used in the url when configuring auth on vault side.

### Token

```
  config:
    addr: https://vault.example.com
    auth:
      token: <mytoken>
```

### AppRole

```
  config:
    addr: https://vault.example.com
    auth:
      approle:
        roleId: <myroleid>
        secretId: <mysecretid>
```

If several configuration options are specified, there are used in the following order:
- Token
- AppRole
- Kubernetes Auth Method

# Development

## Prerequisites

- Operator SDK installation (https://github.com/operator-framework/operator-sdk)
- Go Dep (https://golang.github.io/dep/docs/installation.html)

## Building

To build, simply use *make*:
```
make docker-build
IMG=local/vault-secret:test make docker-build
```

This task will:
- build the binary (using docker)
- create a docker image

You can then push it to any docker repository or use it locally.
