#!/bin/sh
#
# Helper script to start KinD
#
# Also adds a docker-registry and an ingress to aid local development
#
# See https://kind.sigs.k8s.io/docs/user/quick-start/ 
#
set -o errexit

[ "$TRACE" ] && set -x

VERBOSE=1
[ "$TRACE" ] && VERBOSE=3


KIND_K8S_IMAGE=${KIND_K8S_IMAGE:-"kindest/node:v1.20.2@sha256:8f7ea6e7642c0da54f04a7ee10431549c0257315b3a634f6ef2fecaaedb19bab"}
KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME:-"vs"}
KIND_WAIT=${KIND_WAIT:-"120s"}
KIND_API_SERVER_ADDRESS=${KIND_API_SERVER_ADDRESS:-"0.0.0.0"}
KIND_API_SERVER_PORT=${KIND_API_SERVER_PORT:-6443}

create() {
  cat <<EOF | kind create -v ${VERBOSE}  cluster --name="${KIND_CLUSTER_NAME}" --image="${KIND_K8S_IMAGE}" --wait="${KIND_WAIT}" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  apiServerAddress: ${KIND_API_SERVER_ADDRESS}
  apiServerPort: ${KIND_API_SERVER_PORT}

nodes:
- role: control-plane
- role: worker
EOF
}

create_vault_secret() {
  kustomize build vault | kubectl apply -f -
  kustomize build vault_secret | kubectl apply -f -
  kustomize build test_workload | kubectl apply -f -
}

## Delete the cluster
delete() {
  kind delete cluster --name "${KIND_CLUSTER_NAME}"
}

## Display usage
usage()
{
    echo "usage: $0 [create|delete]"
}

## Argument parsing
if [ "$#" = "0" ]; then
  usage
  exit 1
fi
    
while [ "$1" != "" ]; do
    case $1 in
        create )                create
                                create_vault_secret
                                ;;
        delete )                delete
                                ;;
        -h | --help )           usage
                                exit
                                ;;
        * )                     usage
                                exit 1
    esac
    shift
done
