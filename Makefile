SHELL := /usr/bin/env bash

GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)
GOBUILD = go build -o bin/$(BINARY_BASENAME)-$(GOOS)-$(GOARCH) .

BINARY_BASENAME=tracer

DOCKER_REPO ?= datawire/tracer
TAG ?= latest

.PHONY: all build build.image image.push clean fmt run test.fast

all: clean fmt test.fast build

build: fmt
	$(GOBUILD)
	ln -sf $(BINARY_BASENAME)-$(GOOS)-$(GOARCH) bin/$(BINARY_BASENAME)

run: build
	bin/tracer

build.image:
	docker buildx create --use --name tracer-builder
	docker buildx build --platform linux/amd64,linux/arm64 -t $(DOCKER_REPO):$(TAG) . --push
	docker buildx rm tracer-builder

clean:
	rm -rf bin

fmt:
	go fmt ./...

test.fast:
	go test -v ./...
