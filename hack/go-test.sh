#!/bin/sh

if [[ -z ${CI} ]]; then
    ./hack/go-vet.sh
    ./hack/go-fmt.sh
    ./hack/catalog-source.sh
fi
GOCACHE=off go test ./...