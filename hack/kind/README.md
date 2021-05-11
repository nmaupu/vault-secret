## Sping up a test Kind Cluster 

### Requirements

* kind 0.10.0+
* kustomize 


### Steps

* ./kind.sh create
* kubectl get pod --all-namespaces -w 
* wait for `secret` named `secret` in `test` namespace

