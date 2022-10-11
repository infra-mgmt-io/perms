#!/bin/bash
set -e

make generate                                                                   
make manifests
make docker-build docker-push IMG="docker.io/kamleitermichael/infra-mgmt-io-perms:v$1"
make deploy IMG="docker.io/kamleitermichael/infra-mgmt-io-perms:v$1"