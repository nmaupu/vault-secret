default: all
CIRCLE_TAG ?= latest
DOCKER_ID ?= nmaupu
IMAGE_NAME = $(DOCKER_ID)/vault-secret:$(CIRCLE_TAG)

.PHONY: all
all: build push

.PHONY: deps
deps:
	go mod tidy

.PHONY: clean
clean:
	rm -rf build/_output build/_test
	rm -rf release/

.PHONY: build
build: deps
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
	cp -a deploy/crds/maupu.org_vaultsecrets_crd.yaml release/manifests/crds
	cp -a deploy/crds/maupu.org_v1beta1_vaultsecrets_cr.yaml release/manifests/crds/vault-secret-cr-example.yaml
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
