#!/bin/bash
#
# Copyright 2024 Enflame. All Rights Reserved.
#

PATH_ORIG=${PATH}

CURRENT_DIR=$(cd $(dirname $0); pwd)

if [ -f "../../g_version.txt" ]; then
    source ../../g_version.txt
fi
PKG_VER=${ENFLAME_DEVICE_PLUGIN_VERSION:-"2.0.0"}

function usage() {
  cat <<EOF
Usage: ./build.sh [command]
Version: $PKG_VER
Command:
    all       Build and package the k8s-device-plugin:$PKG_VER.
    clean     Clean the k8s-device-plugin:$PKG_VER package.
    help      Show help.
Example:
    ./build.sh all
    ./build.sh clean
EOF
}

function build_all() {
    clean_all
    echo -e "\033[33mBuilding k8s-device-plugin:$PKG_VER release package start...\033[0m"
    cd k8s-device-plugin; make all; cd -
}

function package_all() {
    PKG_DIR="dist/k8s-device-plugin_${PKG_VER}"
    mkdir -p $PKG_DIR/bin

    mv k8s-device-plugin/enflame-device-plugin ${PKG_DIR}/bin
    mv k8s-device-plugin/config-manager ${PKG_DIR}/bin
    cp -rf LICENSE ${PKG_DIR}
    cp -rf -L deployments/* ${PKG_DIR}

    echo -e "\033[33mBuilding k8s-device-plugin:$PKG_VER release package successfully!\033[0m"
}

function clean_all() {
    if [ -d "./dist" ]; then
        rm -rf ./dist
    fi
    cd k8s-device-plugin; make clean; cd -
    echo -e "\033[33mClean k8s-device-plugin:$PKG_VER release package successfully.\033[0m"
}

function main() {
    [[ "$EUID" -ne 0 ]] && { echo "[INFO] Run as non-root"; }
    [[ "$#" -lt 1 ]] && { echo -e "\033[31mError! Please enter the specified parameters according to the usage\033[0m";usage >&2; exit 1; }

    cd ${CURRENT_DIR}

    case $1 in
    "clean")
        clean_all
        ;;
    "all")
        build_all
        package_all
        ;;
    "help")
        usage
        ;;
    *)
        echo -e "\033[31mError! Unknown parameter: $1. Please enter the specified parameters according to the usage\033[0m"
        usage
        exit 1
        ;;
    esac
}

main "$@"
