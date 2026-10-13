#!/bin/bash
#
# Copyright 2024 Enflame. All Rights Reserved.
#

set -eu -o pipefail

export LOGPATH="$PWD/test.log"

# It is recommended to run this script before submitting the code to ensure that all test cases pass
# Run all test case of k8s-device-plugin as root
# go test -v ./... -gcflags=all=-l -cover

[[ "$EUID" -ne 0 ]] && { echo -e "\033[31mError: Test cases must be run as root\033[0m"; exit 1; }

cd k8s-device-plugin/
go test -v enflame/k8s-device-plugin/internal/plugin -gcflags=all=-l -cover
# go test -v ./... -gcflags=all=-l -coverprofile=coverage.out
# go tool cover -func=coverage.out
cd -
