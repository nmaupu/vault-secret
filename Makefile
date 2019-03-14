default: all
CIRCLE_TAG ?= latest
IMAGE_NAME = nmaupu/vault-secret:$(CIRCLE_TAG)

.PHONY: all
all:
	$(MAKE) dep
	$(MAKE) build
	$(MAKE) push

.PHONY: clean
clean:
	rm -rf vendor/
	rm -f pkg/apis/maupu/v1beta1/zz_*
	rm -rf release/

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

.PHONY: CI-release
CI-release-prepare:
	mkdir -p release/bin release
	cp -a deploy/crds/maupu_v1beta1_vaultsecret_crd.yaml release
	cp -a *.yaml release
	sed -i -e "s/latest/$(CIRCLE_TAG)/g" version/version.go

.PHONY: version
version:
	@grep Version version/version.go | sed -e 's/^.*Version = "\(.*\)"$$/\1/g'
