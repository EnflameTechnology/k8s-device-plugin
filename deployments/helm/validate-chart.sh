#!/bin/bash

# Validation script for Enflame Device Plugin Helm Chart
# This script performs basic validation of the chart structure

CHART_DIR="./enflame-device-plugin"
ERRORS=0

echo "🔍 Validating Enflame Device Plugin Helm Chart..."
echo

# Check if chart directory exists
if [ ! -d "$CHART_DIR" ]; then
    echo "❌ Chart directory not found: $CHART_DIR"
    exit 1
fi

# Check required files
echo "📋 Checking required files..."
required_files=(
    "Chart.yaml"
    "values.yaml"
    "templates/_helpers.tpl"
    "templates/daemonset.yaml"
    "templates/serviceaccount.yaml"
    "templates/clusterrole.yaml"
    "templates/clusterrolebinding.yaml"
    "README.md"
)

for file in "${required_files[@]}"; do
    if [ -f "$CHART_DIR/$file" ]; then
        echo "✅ $file"
    else
        echo "❌ Missing: $file"
        ERRORS=$((ERRORS + 1))
    fi
done

# Check variants configuration in values.yaml
echo
echo "📋 Checking variant configurations..."
if [ -f "$CHART_DIR/values.yaml" ]; then
    variants=("basic" "cdi" "pcie-switch-affinity" "compat-with-cpumanager")
    for variant in "${variants[@]}"; do
        if command -v yq >/dev/null 2>&1; then
            if yq eval ".variants.$variant" "$CHART_DIR/values.yaml" >/dev/null 2>&1; then
                echo "✅ Variant '$variant' configuration found"
            else
                echo "❌ Variant '$variant' configuration missing"
                ERRORS=$((ERRORS + 1))
            fi
        else
            # Basic grep check if yq is not available
            if grep -q "^  $variant:" "$CHART_DIR/values.yaml"; then
                echo "✅ Variant '$variant' configuration found"
            else
                echo "❌ Variant '$variant' configuration missing"
                ERRORS=$((ERRORS + 1))
            fi
        fi
    done
fi

# Basic YAML syntax check (if yq is available)
echo
echo "🔍 Checking YAML syntax..."
if command -v yq >/dev/null 2>&1; then
    for yaml_file in "$CHART_DIR"/*.yaml "$CHART_DIR"/templates/*.yaml; do
        if [ -f "$yaml_file" ]; then
            if yq eval '.' "$yaml_file" >/dev/null 2>&1; then
                echo "✅ $(basename "$yaml_file") - Valid YAML"
            else
                echo "❌ $(basename "$yaml_file") - Invalid YAML"
                ERRORS=$((ERRORS + 1))
            fi
        fi
    done
else
    echo "⚠️  yq not available, skipping YAML syntax validation"
fi

# Check Chart.yaml structure
echo
echo "🔍 Checking Chart.yaml structure..."
if [ -f "$CHART_DIR/Chart.yaml" ]; then
    required_fields=("apiVersion" "name" "description" "version" "appVersion")
    for field in "${required_fields[@]}"; do
        if command -v yq >/dev/null 2>&1; then
            if yq eval ".$field" "$CHART_DIR/Chart.yaml" >/dev/null 2>&1; then
                echo "✅ Chart.$field present"
            else
                echo "❌ Chart.$field missing"
                ERRORS=$((ERRORS + 1))
            fi
        else
            # Basic grep check if yq is not available
            if grep -q "^$field:" "$CHART_DIR/Chart.yaml"; then
                echo "✅ Chart.$field present"
            else
                echo "❌ Chart.$field missing"
                ERRORS=$((ERRORS + 1))
            fi
        fi
    done
fi

# Summary
echo
echo "📊 Validation Summary:"
if [ $ERRORS -eq 0 ]; then
    echo "✅ All validations passed! Chart structure looks good."
    echo
    echo "🚀 Next steps:"
    echo "   1. Install Helm if not already installed"
    echo "   2. Run: helm lint $CHART_DIR"
    echo "   3. Test template generation: helm template test-release $CHART_DIR"
    echo "   4. Test different variants:"
    echo "      helm template test-release $CHART_DIR --set variant=cdi"
    echo "      helm template test-release $CHART_DIR --set variant=pcie-switch-affinity"
    echo "      helm template test-release $CHART_DIR --set variant=compat-with-cpumanager"
else
    echo "❌ Found $ERRORS error(s). Please fix them before proceeding."
    exit 1
fi 