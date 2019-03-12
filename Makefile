default: all
IMAGE_NAME = nmaupu/vault-secret:latest

.PHONY: all
all:
	$(MAKE) dep
	$(MAKE) build
	$(MAKE) push

.PHONY: dep
dep:
	dep ensure -update

.PHONY: build
build:
	operator-sdk generate k8s
	operator-sdk build $(IMAGE_NAME)

.PHONY: push
push:
	docker push $(IMAGE_NAME)
