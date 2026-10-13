#!/bin/bash
#
# Copyright (c) 2024, ENFLAME INC.  All rights reserved.
#
# Intelligent deletion script that detects deployment method and removes accordingly

set -euo pipefail

MODE="basic"
STATIC=false
HELM_RELEASE_NAME="enflame-device-plugin"
NAMESPACE="kube-system"

source ./common.sh

function check_helm_release_exists() {
    if helm list -n "$NAMESPACE" --short | grep -q "^${HELM_RELEASE_NAME}$"; then
        return 0
    else
        return 1
    fi
}

function detect_deployment_method() {
    if [[ "$STATIC" == "true" ]]; then
        log_info "Forced to use static deletion method"
        echo "static"
        return
    fi
    
    if check_helm && check_helm_release_exists; then
        log_info "Helm release '$HELM_RELEASE_NAME' found in namespace '$NAMESPACE'"
        echo "helm"
    else
        log_info "No Helm release found, checking for YAML-deployed resources as static deployment"
        # Check if there are any device plugin pods that might be from STATIC deployment
        if kubectl get pods -n "$NAMESPACE" -l name=enflame-device-plugin-ds --no-headers 2>/dev/null | grep -q .; then
            log_info "Found YAML-deployed resources as static deployment"
            echo "static"
        else
            log_error "No resources found to delete"
            exit 1
        fi
    fi
}

function delete_with_helm() {
    log_info "Deleting Helm release: $HELM_RELEASE_NAME"
    
    helm uninstall "$HELM_RELEASE_NAME" -n "$NAMESPACE"
    if [[ $? -ne 0 ]]; then
        log_error "Failed to delete Helm release"
        exit 1
    fi
    log_success "Helm release deleted successfully"
}

function delete_with_static() {
    log_info "Deleting as static deployment with kubectl using YAML files"
    local yaml_file=$(get_yaml_by_mode $MODE)
    log_info "Deleting resources from YAML file: $yaml_file"
    
    if ! (kubectl delete -f static/rbac.yaml && kubectl delete -f "$yaml_file" --ignore-not-found=true); then
        log_error "Failed to delete YAML resources"
        exit 1
    fi
    log_success "YAML resources deleted successfully"
    log_info "You can check the deletion with:"
    log_info "  kubectl get pods -n $NAMESPACE -l name=enflame-device-plugin-ds"
}

function main() {
    log_info "Starting Enflame Device Plugin deletion in namespace $NAMESPACE..."
    
    local deployment_method=$(detect_deployment_method)
    log_info "Detected deployment method: $deployment_method"

    case $deployment_method in
        "helm")
            delete_with_helm
            ;;
        "static")
            delete_with_static
            ;;
    esac
}

function usage() {
  cat <<EOF
Usage: ./delete.sh [OPTIONS]
Description: Intelligent deletion script for Enflame Device Plugin.
             Automatically detects deployment method (Helm or Static) and removes accordingly.

OPTIONS:
    --mode <mode>          Delete mode for static method, choose from: ['basic'(default), 'psa', 'cm', 'cdi']
                           basic: Basic deployment (deletes enflame-device-plugin.yaml)
                           psa:    PCIe Switch Affinity (deletes pcie-switch-affinity variant)
                           cm:     CPU Manager compatibility (deletes cpu manager variant)
                           cdi:    Container Device Interface (deletes cdi variant)
    --static               Force using static deletion method even if Helm release exists
    --helm-release <name>  Helm release name to delete (default: enflame-device-plugin)
    --namespace <ns>       Target namespace (default: kube-system)
    --help, -h             Show this help message

EXAMPLES:
    # Delete helm release or static deployment use mode 'basic'(default)
    ./delete.sh
    
    # Delete specific YAML variant
    ./delete.sh --mode cdi
    
    # Force YAML deletion even if Helm release exists
    ./delete.sh --mode psa --static
    
    # Delete specific Helm release
    ./delete.sh --helm-release my-device-plugin
    
    # Delete from specific namespace
    ./delete.sh --namespace custom-system

DELETION METHODS:
    1. Auto-Detection (Default): Automatically detects how the plugin was deployed
       - Checks for Helm releases first
       - Falls back to YAML deployments
       - Cleans up any orphaned resources
    
    2. Helm Deletion: Removes Helm release and all associated resources
       - Uses 'helm uninstall' command
       - Automatically handles all resources in the release
    
    3. YAML Deletion: Removes resources based on YAML files
       - Uses 'kubectl delete -f' command
       - Requires matching mode with original deployment
       - Manual cleanup of remaining resources

NOTES:
    - The script will always attempt to clean up any remaining resources
    - Use --static if you want to ensure YAML-based deletion
    - Auto-detection prioritizes Helm releases over YAML deployments

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
        log_error "Invalid mode: '$MODE'"
        log_error "Valid modes are: basic, psa, cm, cdi"
        exit 1
        ;;
esac

main "$@"
