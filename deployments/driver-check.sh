#!/bin/bash

set -euo pipefail

function check_devices() {
    local count=0

    for device in /sys/bus/pci/devices/*; do
        class_file="$device/class"
        vendor_file="$device/vendor"

        if [[ -f "$class_file" ]]; then
            class_val=$(cat "$class_file")
            if [[ "$class_val" != "0x12"* ]]; then
                continue
            fi
        else
            continue
        fi

        if [[ -f "$vendor_file" ]]; then
            vendor_val=$(cat "$vendor_file")
            if [[ "$vendor_val" != "0x1e36" && "$vendor_val" != "0x1ea0" ]]; then
                continue
            fi
        else
            continue
        fi

        ((count++))
    done

    echo -e "\033[34mfound device count: $count from pci information\033[0m"

    gcu_count=0
    for path in /dev/gcu*; do
        [ -e "$path" ] || continue
        name=$(basename "$path")
        if [[ "$name" =~ ^gcu[0-9]+$ ]]; then
            gcu_count=$((gcu_count + 1))
        fi
    done

    echo -e "\033[34mfound device count: $gcu_count from /dev directory\033[0m"

    if [[ $count -eq $gcu_count ]]; then
        return 0
    else
        return 1
    fi
}

while true; do
    if check_devices; then
        echo -e "\033[32mdevice count match success\033[0m"
        break
    fi
    echo -e "\033[31mdevice count match failed, retry after 1 minute\033[0m"
    sleep 60
done
