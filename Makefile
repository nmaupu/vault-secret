default: all
APP_VERSION ?= latest
IMAGE_NAME = nmaupu/vault-secret:${APP_VERSION}

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
	dep ensure -v

.PHONY: dep-update
dep-update:
	dep ensure -update -v

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

.PHONY: test
test:
	go test -v ./...

release:
	mkdir -p release/

