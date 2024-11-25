.PHONY:	test imports
SHELL := /bin/bash

VERSION?=0.0.1-local

IMAGE = quay.io/fortnox/renovator

build:
	CGO_ENABLED=0 GOOS=linux go build

docker: build
	docker build --pull --rm -t $(IMAGE):$(VERSION) .

push: docker
	docker push $(IMAGE):$(VERSION)

test: imports
	go test -v ./...

imports: SHELL:=/bin/bash
imports:
	go install golang.org/x/tools/cmd/goimports@latest
	ASD=$$(goimports -l . 2>&1); test -z "$$ASD" || (echo "Code is not formatted correctly according to goimports!  $$ASD" && exit 1)


docker-compose-up:
	docker compose up

docker-compose-down:
	docker compose down

docker-compose-build: build
	docker compose build --no-cache

docker-compose: docker-compose-down docker-compose-build docker-compose-up
