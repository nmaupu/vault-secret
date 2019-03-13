default: all
IMAGE_NAME = nmaupu/vault-secret:latest

.PHONY: all
all:
	$(MAKE) dep
	$(MAKE) build
	$(MAKE) push

.PHONY: clean
clean:
	rm -rf vendor/
	rm -f pkg/apis/maupu/v1beta1/zz_*

.PHONY: dep
dep:
	dep ensure

.PHONY: build
build:
	operator-sdk generate k8s
	operator-sdk build $(IMAGE_NAME)

.PHONY: openapi
openapi:
	operator-sdk generate openapi

.PHONY: push
push:
	docker push $(IMAGE_NAME)
