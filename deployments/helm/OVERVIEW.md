# Enflame Device Plugin Helm Chart Overview

## 🎯 Project Summary

This project converts the existing YAML manifests in `../yaml/` directory into a unified, flexible Helm chart that supports all deployment variants through a single, simple configuration.

## 📁 Chart Structure

```
enflame-device-plugin/
├── Chart.yaml                              # Chart metadata
├── values.yaml                             # Single configuration file with all variants
├── README.md                               # Comprehensive usage guide
└── templates/
    ├── _helpers.tpl                        # Reusable template helpers
    ├── serviceaccount.yaml                 # ServiceAccount resource
    ├── clusterrole.yaml                    # ClusterRole with variant-specific rules
    ├── clusterrolebinding.yaml             # ClusterRoleBinding
    ├── daemonset.yaml                      # Main DaemonSet with variant logic
    └── NOTES.txt                           # Post-install instructions
```

## 🔄 Conversion Mapping

### Original YAML Files → Helm Variants

| Original YAML File | Helm Variant | Command |
|-------------------|---------------|---------|
| `enflame-device-plugin.yaml` | `basic` (default) | `helm install ... --set variant=basic` |
| `enflame-device-plugin-cdi.yaml` | `cdi` | `helm install ... --set variant=cdi` |
| `enflame-device-plugin-pcie-switch-affinity.yaml` | `pcie-switch-affinity` | `helm install ... --set variant=pcie-switch-affinity` |
| `enflame-device-plugin-compat-with-cpumanager.yaml` | `compat-with-cpumanager` | `helm install ... --set variant=compat-with-cpumanager` |

## 🛠️ Key Features

### 1. **Simple Configuration**
- **Single values file**: All configurations in one place
- **Variant selection**: Just set `variant: "cdi"` to switch modes
- **Override capability**: Use `overrides` section for custom settings

### 2. **Easy Deployment**
```bash
# Basic deployment
helm install enflame-device-plugin ./enflame-device-plugin

# CDI variant
helm install enflame-device-plugin ./enflame-device-plugin --set variant=cdi

# Custom overrides
helm install enflame-device-plugin ./enflame-device-plugin \
  --set variant=cdi \
  --set overrides.env.CUSTOM_REGISTER_RESOURCE="custom.com/gcu"
```

### 3. **Intelligent Templates**
- Templates automatically select appropriate settings based on variant
- Security contexts switch between privileged and restricted modes
- RBAC rules adapt to variant requirements
- Environment variables merge variant-specific and custom overrides

### 4. **Production Ready**
- Comprehensive documentation and examples
- Built-in validation script
- Clear upgrade and rollback procedures

## 🚀 Usage Examples

### Basic Usage
```bash
# Default basic variant
helm install enflame-device-plugin ./enflame-device-plugin

# Specific variant
helm install enflame-device-plugin ./enflame-device-plugin --set variant=pcie-switch-affinity

# With custom image
helm install enflame-device-plugin ./enflame-device-plugin \
  --set variant=cdi \
  --set image.tag=v2.1.0
```

### Custom Values File
```yaml
# custom-values.yaml
variant: "cdi"
image:
  tag: "v2.1.0"
overrides:
  env:
    CUSTOM_REGISTER_RESOURCE: "custom.com/gcu"
```

```bash
helm install enflame-device-plugin ./enflame-device-plugin -f custom-values.yaml
```

## 📊 Configuration Matrix

| Configuration | Basic | CDI | PCIe Switch | CPU Manager |
|--------------|-------|-----|------------|-------------|
| **ServiceAccount** | `enflame-k8s-device-plugin` | `enflame-k8s-device-plugin` | `k8s-device-plugin` | `enflame-k8s-device-plugin` |
| **Security Mode** | Restricted | Privileged | Restricted | Privileged |
| **RBAC** | Basic | Basic | Extended | Basic |
| **Device Strategy** | Default | CDI Annotations | Default | Default |
| **Special Features** | None | CDI Support | PCIe Affinity | Device Specs |

## 🎨 Benefits Over Original YAML

1. **Extreme Simplification**: One values file instead of 4 separate YAML files
2. **One-Command Switching**: Change variants with `--set variant=X`
3. **Zero Duplication**: Single source of truth for all configurations
4. **Easy Customization**: Override any setting without editing core files
5. **Better Maintenance**: Changes propagate to all variants automatically
6. **User-Friendly**: Simple parameter changes instead of file management

## ✅ Validation

The chart has been validated and simplified:
- ✅ Single values.yaml with all variants
- ✅ Template logic handles all configurations
- ✅ No duplicate values files needed
- ✅ Simple variant switching mechanism

## 🔮 Simplified Workflow

### Before (Complex)
```bash
# Had to choose different files
helm install plugin ./chart -f values-cdi.yaml
helm install plugin ./chart -f values-pcie-switch-affinity.yaml
helm install plugin ./chart -f values-compat-with-cpumanager.yaml
```

### After (Simple)
```bash
# Just set variant parameter
helm install plugin ./chart --set variant=cdi
helm install plugin ./chart --set variant=pcie-switch-affinity
helm install plugin ./chart --set variant=compat-with-cpumanager
```

## 🎯 Design Philosophy

**Maximum Simplicity with Full Functionality**

- **Users want**: Easy deployment with variant selection
- **Users get**: Single parameter (`variant`) controls everything
- **Developers want**: Maintainable, non-duplicated configuration
- **Developers get**: One values file, intelligent templates

## 📞 Support

For questions about this simplified Helm chart:
- Review the README.md for detailed examples
- Use the validation script: `./validate-chart.sh`
- Test with: `helm template ... --set variant=X`
- All original functionality is preserved, just easier to use! 