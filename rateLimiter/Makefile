AWSCLI				:= aws
SAMCLI				:= sam

PACKAGE				:= github.com/jumacloud/service-identity-scim
APP_NAME			:= rate-limiter
COMPONENT			:= rate-limiter

GOLANGCI_VERSION	:= 1.43.0
GO_DOCKER_VERSION	:= 1.16
BINDIR				?= $(shell pwd)/bin
GIT_HASH			?= $(shell git rev-parse --short HEAD)

BUILD_OVERRIDES = \
	-X "$(PACKAGE)/internal/app.Name=$(APP_NAME)" \
	-X "$(PACKAGE)/internal/app.BuildDate=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')" \
	-X "$(PACKAGE)/internal/app.Commit=$(GIT_HASH)" \
# the -w -s flags make the binary a bit smaller and
# trimpath shortens build paths in stack traces
LDFLAGS := -ldflags='-w -s $(BUILD_OVERRIDES)' -trimpath
# https://tip.golang.org/cmd/go/#hdr-Module_configuration_for_non_public_modules

####################
# Package commands #
####################
clean:
	rm -rf $(CURDIR)/dist
.PHONY: clean

lint: $(BINDIR)/golangci-lint generate
	@echo "--- lint all the things"
	@$(BINDIR)/golangci-lint run
.PHONY: lint

validate:
	@echo "--- validate all the things..."
	@cfn-lint --ignore-checks W2001 -- sam/api.sam.yaml
.PHONY: validate

bundle:
	@echo "--- package binary into zipped handler..."
	@cd $(CURDIR)/dist && zip -r -q ../handler.zip .
.PHONY: bundle

###################
#  Test commands  #
###################
test-swagger:
	docker run --rm -it -v $(CURDIR):/var/task stoplight/spectral:4.2 lint -r /var/task/.spectral.yaml -s api-servers -F warn --verbose /var/task/openapi/onboarding.yaml
.PHONY: test-swagger

test: generate
	@echo "--- test all the things"
	@mkdir -p coverage
	@go test -coverprofile=coverage/coverage.txt -covermode count ./...
	@go tool cover -func coverage/coverage.txt | grep total | awk '{print $3}'
.PHONY: test


###################
# Build commands  #
###################
$(BINDIR)/golangci-lint: $(BINDIR)/golangci-lint-${GOLANGCI_VERSION}
	@ln -sf golangci-lint-${GOLANGCI_VERSION} bin/golangci-lint

$(BINDIR)/golangci-lint-${GOLANGCI_VERSION}:
	@curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | bash -s -- -b ./bin v${GOLANGCI_VERSION}
	@mv $(BINDIR)/golangci-lint $@

generate:
	@echo "--- generate all the things"
	@go generate ./...
.PHONY: generate

build: generate
	@echo "--- build all the things"
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a $(LDFLAGS) -o dist/$(COMPONENT) ./cmd/$(COMPONENT)

###################
# Build commands  #
###################
build-run: clean generate validate lint build test-swagger test