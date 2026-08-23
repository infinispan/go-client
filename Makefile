MODULE := $(shell go list -m)
GOBIN := $(shell go env GOPATH)/bin
DOC_PACKAGES := $(MODULE)/hotrod
DOC_OUT := _site
DOC_ZIP := _site.zip
DOCS_OUT := _docs
DOCS_ZIP := docs.zip

.PHONY: help build vet lint vulncheck test test-integration check doc doc-zip docs-zip clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Compile all packages
	go build ./...

vet: ## Run go vet
	go vet ./...

lint: $(GOBIN)/golangci-lint ## Run golangci-lint
	$(GOBIN)/golangci-lint run ./...

vulncheck: $(GOBIN)/govulncheck ## Run govulncheck for known vulnerabilities
	$(GOBIN)/govulncheck ./...

test: ## Run unit tests (no server required)
	go test -short ./...

test-integration: ## Run integration tests (requires Docker; set INFINISPAN_SERVER_IMAGE to override)
	go test -timeout 300s ./test/...

check: build vet lint vulncheck test ## Run build, vet, lint, vulncheck, and test

doc: $(GOBIN)/doc2go ## Generate HTML API docs to _site/
	rm -rf $(DOC_OUT)
	$(GOBIN)/doc2go -out $(DOC_OUT) -home $(MODULE) $(DOC_PACKAGES)

doc-zip: doc ## Generate API docs and package as _site.zip
	cd $(DOC_OUT) && zip -qr ../$(DOC_ZIP) .

docs-zip: doc ## Render user guide + API docs and package as docs.zip
	rm -rf $(DOCS_OUT)
	mkdir -p $(DOCS_OUT)/api
	asciidoctor -o $(DOCS_OUT)/go_client.html documentation/infinispan-go-client.adoc
	cp -r $(DOC_OUT)/* $(DOCS_OUT)/api/
	cd $(DOCS_OUT) && zip -qr ../$(DOCS_ZIP) .

clean: ## Remove generated artifacts
	rm -rf $(DOC_OUT) $(DOC_ZIP) $(DOCS_OUT) $(DOCS_ZIP)

$(GOBIN)/doc2go:
	go install go.abhg.dev/doc2go@latest

$(GOBIN)/golangci-lint:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

$(GOBIN)/govulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@latest
