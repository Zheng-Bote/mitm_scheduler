#!/bin/sh

MITM_VERSION=$(git describe --tags)

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=${MITM_SERVER_VERSION}" -o ./bin/mitm-server ./cmd/scheduler
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=${MITM_SERVER_VERSION}" -o ./bin/create-admin ./cmd/create-admin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=${MITM_SERVER_VERSION}" -o ./bin/encrypt-config ./cmd/encrypt-config
