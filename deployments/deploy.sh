#!/bin/bash
#
# Copyright (c) 2024, ENFLAME INC.  All rights reserved.
#
# Intelligent deployment script that uses Helm if available, otherwise falls back to YAML files as static deployment

set -euo pipefail

MODE="basic"
STATIC=false
HELM_RELEASE_NAME="enflame-device-plugin"
NAMESPACE="kube-system"
IMAGE_TAG="v2.0.0"
IMAGE_PULL_POLICY="IfNotPresent"

source ./common.sh

function map_mode_to_variant() {
    case $MODE in
        "basic")
            echo "basic"
            ;;
        "psa")
            echo "pcie-switch-affinity"
            ;;
        "cm")
            echo "compat-with-cpumanager"
            ;;
        "cdi")
            echo "cdi"
            ;;
        *)
            log_error "Invalid mode: '$MODE'"
            exit 1
            ;;
    esac
}

function deploy_with_helm() {
    local variant=$(map_mode_to_variant)
    log_info "Deploying with Helm using variant: $variant"
    
    # Build helm command
    local helm_cmd="helm install $HELM_RELEASE_NAME helm/enflame-device-plugin --namespace $NAMESPACE"
    helm_cmd="$helm_cmd --set variant=$variant --set namespace=$NAMESPACE"
    helm_cmd="$helm_cmd --set image.tag=$IMAGE_TAG --set image.pullPolicy=$IMAGE_PULL_POLICY"
    log_info "Executing command: $helm_cmd"

    # Execute helm command
    if eval $helm_cmd; then
        log_success "Helm deployment completed successfully"
        log_info "You can check the deployment with:"
        log_info "  helm status $HELM_RELEASE_NAME -n $NAMESPACE"
    else
        log_error "Helm deployment failed"
        exit 1
    fi
}

function deploy_with_static() {
    log_info "Deploying as static deployment with kubectl using YAML files"
    local yaml_file=$(get_yaml_by_mode $MODE)
    log_info "Applying YAML file: $yaml_file"
    
    if kubectl create -f static/rbac.yaml && kubectl create -f "$yaml_file"; then
        log_success "Static deployment completed successfully"
        log_info "You can check the deployment with:"
        log_info "  kubectl get pods -n $NAMESPACE -l name=enflame-device-plugin-ds"
    else
        log_error "Static deployment failed. Please check static/rbac.yaml and $yaml_file"
        exit 1
    fi
}

function main() {
    log_info "Starting Enflame Device Plugin deployment using $MODE mode..."
    
    # Check required tools
    check_kubectl
    
    # Choose deployment method
    if [[ "$STATIC" == "true" ]]; then
        log_info "Forced to use static deployment method"
        deploy_with_static
    elif check_helm; then
        log_info "Using Helm deployment method"
        deploy_with_helm
    else
        log_info "Using static deployment method"
        deploy_with_static
    fi
    
    if [[ $? -eq 0 ]]; then
        log_success "Deployment completed successfully!"
        log_info "Next you can:"
        log_info "  Create a pod that request for enflame.com/gcu to apply for a GCU device"
    else
        log_error "Deployment failed!"
        exit 1
    fi
}

function usage() {
  cat <<EOF
Usage: ./deploy.sh [OPTIONS]
Description: Intelligent deployment script for Enflame Device Plugin.
             Automatically detects Helm and uses it if available, otherwise falls back to YAML files.

OPTIONS:
    --mode <mode>           Deploy mode, choose from: ['basic'(default), 'psa', 'cm', 'cdi']
                           basic: Basic deployment (maps to 'basic' variant in Helm)
                           psa:    PCIe Switch Affinity (maps to 'pcie-switch-affinity' variant)
                           cm:     CPU Manager compatibility (maps to 'compat-with-cpumanager' variant)
                           cdi:    Container Device Interface (maps to 'cdi' variant)
    
    --static               Force using static deployment method
    --helm-release <name>  Helm release name (default: enflame-device-plugin)
    --image-tag <tag>      Override container image tag (Helm only)
    --image-pull-policy <policy>  Override image pull policy (Helm only)
    --namespace <ns>       Target namespace (default: kube-system)
    
    --help, -h             Show this help message

EXAMPLES:
    # Basic deployment (auto-detect Helm)
    ./deploy.sh
    
    # CDI variant deployment
    ./deploy.sh --mode cdi
    
    # static deployment even if Helm is available
    ./deploy.sh --mode psa --static
    
    # Helm deployment with custom image tag
    ./deploy.sh --mode cdi --image-tag v2.1.0
    
    # Helm deployment with custom release name
    ./deploy.sh --mode cm --helm-release my-device-plugin

DEPLOYMENT METHODS:
    1. Helm Chart (Preferred): Uses helm/enflame-device-plugin chart
       - Unified configuration with variant selection
       - Easy upgrades and rollbacks
       - Advanced configuration options
    
    2. YAML Files (Fallback): Uses individual YAML files in static/ directory
       - Works without Helm installation
       - Direct kubectl apply method
       - Legacy compatibility

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --mode)
            MODE="$2"
            shift 2
            ;;
        --static)
            STATIC=true
            shift
            ;;
        --helm-release)
            HELM_RELEASE_NAME="$2"
            shift 2
            ;;
        --image-tag)
            IMAGE_TAG="$2"
            shift 2
            ;;
        --image-pull-policy)
            IMAGE_PULL_POLICY="$2"
            shift 2
            ;;
        --namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Validate mode
case $MODE in
    "basic"|"psa"|"cm"|"cdi")
        ;;
    *)
        log_error "Invalid mode: '$MODE', select from: basic, psa, cm, cdi"
        exit 1
        ;;
esac

main "$@"
