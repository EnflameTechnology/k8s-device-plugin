#!/bin/bash
#
# Copyright (c) 2024, ENFLAME INC.  All rights reserved.
#

# Color codes for output
RED='\033[31m'
GREEN='\033[32m'
YELLOW='\033[33m'
BLUE='\033[34m'
NC='\033[0m' # No Color

function log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" >&2
}

function log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" >&2
}

function log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" >&2
}

function log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

function check_helm() {
    if command -v helm >/dev/null 2>&1; then
        local helm_version=$(helm version --short 2>/dev/null)
        log_info "Helm detected: $helm_version"
        return 0
    else
        log_warning "Helm not found, will use kubectl with YAML files"
        return 1
    fi
}

function check_kubectl() {
    if ! command -v kubectl >/dev/null 2>&1; then
        log_error "kubectl is required but not found in PATH"
        exit 1
    fi
}

function get_yaml_by_mode() {
    local yaml_file=""
    case $1 in
        "basic")
            yaml_file="static/enflame-device-plugin.yaml"
            ;;
        "psa")
            yaml_file="static/enflame-device-plugin-pcie-switch-affinity.yaml"
            ;;
        "cm")
            yaml_file="static/enflame-device-plugin-compat-with-cpumanager.yaml"
            ;;
        "cdi")
            yaml_file="static/enflame-device-plugin-cdi.yaml"
            ;;
        *)
            log_error "Invalid mode: '$1'"
            exit 1
            ;;
    esac
    echo "$yaml_file"
}
