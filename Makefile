default: all
CIRCLE_TAG ?= latest
IMAGE_NAME = nmaupu/vault-secret:$(CIRCLE_TAG)

.PHONY: all
all:
	$(MAKE) build
	$(MAKE) push

.PHONY: clean
clean:
	rm -rf vendor/
	rm -f pkg/apis/maupu/v1beta1/zz_*
	rm -rf release/

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
	mkdir -p release/manifests/crds
	cp -a deploy/*.yaml release/manifests
	sed -i -e "s/\(nmaupu.vault-secret\):latest$$/\1:$(CIRCLE_TAG)/g" release/manifests/operator.yaml
	cp -a deploy/crds/maupu_v1beta1_vaultsecret_crd.yaml release/manifests/crds
	cp -a deploy/crds/maupu_v1beta1_vaultsecret_cr.yaml release/manifests/crds/vault-secret-cr-example.yaml
	tar cfz release/vault-secret-manifests-$(CIRCLE_TAG).tar.gz -C release manifests
	rm -rf release/manifests/
	sed -i -e "s/latest/$(CIRCLE_TAG)/g" version/version.go

test-manifest:
	mkdir -p release/manifests
	cp deploy/operator.yaml release/manifests
	sed -i -e "s/\(nmaupu.vault-secret\):latest$$/\1:$(CIRCLE_TAG)/g" release/manifests/operator.yaml

.PHONY: CI-process-release
CI-process-release:
	cp ./build/_output/bin/vault-secret release/vault-secret-$(CIRCLE_TAG)-linux-amd64
	@echo "Version to be released: $(CIRCLE_TAG)"
	ghr -t $(GITHUB_TOKEN) \
		-u $(CIRCLE_PROJECT_USERNAME) \
		-r $(CIRCLE_PROJECT_REPONAME) \
		-c $(CIRCLE_SHA1) \
		-n "Release v$(CIRCLE_TAG)" \
		-delete \
		$(CIRCLE_TAG) release/

.PHONY: version
version:
	@grep Version version/version.go | sed -e 's/^.*Version = "\(.*\)"$$/\1/g'
