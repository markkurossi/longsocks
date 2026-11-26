VERSION ?= 0.1.0
COMMIT := $(shell git rev-parse HEAD)

.PHONY: package

package:
	rm -rf dist
	mkdir -p dist

	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -trimpath \
		-ldflags "-s -w -X main.version=$(VERSION) -X main.gitCommit=$(COMMIT)" \
		-o dist/longsocksd ./cmd/longsocksd

	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -trimpath \
		-ldflags "-s -w -X main.version=$(VERSION) -X main.gitCommit=$(COMMIT)" \
		-o dist/longsocks ./cmd/longsocks

	VERSION=$(VERSION) nfpm pkg \
		--packager deb \
		--config packaging/nfpm.yaml \
		--target dist/
