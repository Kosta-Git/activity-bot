.DEFAULT_GOAL := help
help:
ifeq ($(OS),Windows_NT)
	powershell -NoLogo -NoProfile -Command "& .\scripts\help.ps1 $(abspath $(lastword $(MAKEFILE_LIST)))"
else
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-27s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
endif

.PHONY: install-abigen
install-abigen: ## Installs abigen.
	go install github.com/ethereum/go-ethereum/cmd/abigen@latest

.PHONY: build-abi
build-abi: ## Builds all ABI json into Go structs.
ifeq ($(OS),Windows_NT)
	powershell.exe -ExecutionPolicy Bypass -File ./scripts/abi-builder.ps1
else
	./scripts/abi-builder.sh
endif

.PHONY: build
build: ## Builds the project.
	go build -o ./bin/ ./...

.PHONY: docker
docker: ## Builds the docker image.
	docker build -t activity-bot .