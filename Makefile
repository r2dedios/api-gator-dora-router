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



.PHONY: start-dev
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

