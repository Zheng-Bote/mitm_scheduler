#!/bin/sh

openssl req -x509 -newkey rsa:4096 -nodes -keyout ./data/server_prd.key -out ./data/server_prd.crt -days 3650 \
      -subj "/CN=localhost" \
      -addext "subjectAltName = DNS:coritydf-coritydatafeed-app-prod-3cvd.scp.eu-central-1.aws.cloud.bmw"

openssl req -x509 -newkey rsa:4096 -nodes -keyout ./data/server_int.key -out ./data/server_int.crt -days 3650 \
      -subj "/CN=localhost" \
      -addext "subjectAltName = DNS:coritydf-coritydatafeed-app-int-3cvi.scp.eu-central-1.aws.cloud.bmw"

MITM_VERSION=$(git describe --tags)

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=${MITM_SERVER_VERSION}" -o ./bin/mitm-server ./cmd/scheduler
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=${MITM_SERVER_VERSION}" -o ./bin/create-admin ./cmd/create-admin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=${MITM_SERVER_VERSION}" -o ./bin/encrypt-config ./cmd/encrypt-config
