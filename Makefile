GO          ?= go
GOOS        ?= $(shell $(GO) env GOOS)
GOARCH      ?= $(shell $(GO) env GOARCH)
PACKAGENAME := $(shell $(GO) list -m -f '{{.Path}}')
GOBUILD     ?= CGO_ENABLED=0 $(GO) build
GO_FILES    := $(shell find . -type f -name '*.go' -not -path './vendor/*')

EXECUTABLE  := gadget
ARTIFACT    := dist/$(GOOS)-$(GOARCH)/$(EXECUTABLE)

DB_PASS      ?= gadget
DB_ROOT_PASS ?= gadget

.PHONY: all
all: clean verify lint test build

###############
##@ Development

# This is to allow make to detect when other targets should be rerun (source changes)
$(GO_FILES):
	@ls -l "$@"

.PHONY: build
build: $(ARTIFACT) ## Build binary
$(ARTIFACT): $(GO_FILES)
	@$(MAKE) --no-print-directory log-build
	@$(GOBUILD) -o $@

.PHONY: verify
verify:   ## Verify 'vendor' dependencies
	@ $(MAKE) --no-print-directory log-$@
	$(GO) mod verify

.PHONY: container
container: ## Build container using docker
	@$(MAKE) --no-print-directory log-$@
	@docker build -t gadget:local .

.PHONY: fmt
fmt: ## Check the project follows idiomatic formatting
	@$(MAKE) --no-print-directory	log-$@
	@golangci-lint fmt --diff

.PHONY: lint
lint: fmt ## Lint the project
	@$(MAKE) --no-print-directory log-$@
	@golangci-lint run

.PHONY: test
test: $(GO_FILES) ## Execute tests
	@$(MAKE) --no-print-directory log-$@
	$(GO) test -coverprofile=coverage.out -covermode=atomic -v ./...

.PHONY: clean
clean: ## Clean the workspace including modcache and dist/
	@$(MAKE) --no-print-directory log-$@
	@$(GO) clean -modcache
	@rm -rf dist/* coverage.out

.PHONY: tools
tools: ## Install tools needed for development
	@$(MAKE) --no-print-directory log-$@
	@$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.10.1

###############
##@ Database
.PHONY: start-db
start-db: ## Start maria db - export DB_ROOT_PASS and DB_PASS to set credentials
	@$(MAKE) --no-print-directory log-$@
	@docker run --name gadget-mariadb \
		-v ${HOME}/.gadget/db:/var/lib/mysql \
		-e MARIADB_ROOT_PASSWORD="${DB_ROOT_PASS}" \
		-e MARIADB_DATABASE=gadget_dev \
		-e MARIADB_USER=gadget \
		-e MARIADB_PASSWORD="${DB_PASS}" \
		-p 3306:3306 \
		-d mariadb:10.5

.PHONY: stop-db
stop-db: ## Stop maria db
	@$(MAKE) --no-print-directory log-$@
	@docker stop gadget-mariadb
	@docker rm gadget-mariadb

###########################################################################
## Self-Documenting Makefile Help and logging                            ##
## https://github.com/terraform-docs/terraform-docs/blob/master/Makefile ##
## https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html    ##
###########################################################################

########
##@ Help

.PHONY: help
help:   ## Display this help
	@awk \
		-v "col=\033[36m" -v "nocol=\033[0m" \
		' \
			BEGIN { \
				FS = ":.*##" ; \
				printf "Usage:\n  make %s<target>%s\n", col, nocol \
			} \
			/^[a-zA-Z_-]+:.*?##/ { \
				printf "  %s%-12s%s %s\n", col, $$1, nocol, $$2 \
			} \
			/^##@/ { \
				printf "\n%s%s%s\n", nocol, substr($$0, 5), nocol \
			} \
		' $(MAKEFILE_LIST)

log-%:
	@grep -h -E '^$*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk \
			'BEGIN { \
				FS = ":.*?## " \
			}; \
			{ \
				printf "\033[36m==> %s\033[0m\n", $$2 \
			}'
