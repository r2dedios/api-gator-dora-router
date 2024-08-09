#
## Exate APIGator Dora Router Makefile
################################################################################

# Global vars
CONTAINER_ENGINE ?= $(shell which podman >/dev/null 2>&1 && echo podman || echo docker)
K8S_CLI ?= $(shell which oc >/dev/null 2>&1 && echo oc || echo kubectl)
REGISTRY ?= quay.io
PROJECT_NAME ?= apigator_dora_router
REGISTRY_REPO ?= avillega
IMAGE_NAME ?= $(REGISTRY)/$(REGISTRY_REPO)/$(PROJECT_NAME)
IMAGE_TAG := $(shell git rev-parse --short=7 HEAD)
#
# Help message
define HELP_MSG
\033[1;37mMakefile Rules\033[0m:
	\033[1;36mstart:\033[0m               \033[0;37m Starts the Router on local with INFO log level
	\033[1;36mstart-debug:\033[0m         \033[0;37m Starts the Router on local with DEBUG log level
	\033[1;36mdoc:\033[0m                 \033[0;37m Generates Go Documentation
	\033[1;36mbuild-image:\033[0m         \033[0;37m Builds the Container image for the APIGatorDoraRouter
	\033[1;36mpush:\033[0m                \033[0;37m Pushes the Container image to the image registry defined on this Makefile
	\033[1;36mhelp:\033[0m                \033[0;37m Displays this message
	\033[0m
endef
export HELP_MSG



start-debug:
	APIGATOR_DORA_ROUTER_LOG_LEVEL="DEBUG" go run cmd/router.go
start:
	APIGATOR_DORA_ROUTER_LOG_LEVEL="INFO" go run cmd/router.go

docs:
	go doc -C internal/apigator/ -all -u
	go doc -C internal/config/ -all -u
	go doc -C internal/logger/ -all -u

build-image:
	$(CONTAINER_ENGINE) build \
		-f manifests/container_image/Containerfile \
		--tag $(IMAGE_NAME):latest \
		.
	$(CONTAINER_ENGINE) build \
		-f manifests/container_image/Containerfile \
		--tag $(IMAGE_NAME):$(IMAGE_TAG) \
		.

push:
	$(CONTAINER_ENGINE) push $(IMAGE_NAME):latest
	$(CONTAINER_ENGINE) push $(IMAGE_NAME):$(IMAGE_TAG)

# Set the default target to "help"
.DEFAULT_GOAL := help

# Help
help:
	@echo -e "$$HELP_MSG"
