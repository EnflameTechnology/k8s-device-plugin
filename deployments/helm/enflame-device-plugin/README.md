# Enflame Device Plugin Helm Chart

This Helm chart deploys the Enflame Kubernetes Device Plugin across your cluster, enabling Kubernetes to manage Enflame GCU (Graphics Computing Unit) resources.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- Enflame drivers installed on nodes
- Container runtime that supports device plugins

## Installation

### Quick Start (Basic Variant)

```bash
helm install enflame-device-plugin ./enflame-device-plugin
```

### Installing Different Variants

The chart supports multiple deployment variants. Simply set the `variant` parameter:

#### 1. Basic Variant (Default)
Standard deployment with minimal permissions and restricted security context.

```bash
helm install enflame-device-plugin ./enflame-device-plugin
# or explicitly
helm install enflame-device-plugin ./enflame-device-plugin --set variant=basic
```

#### 2. CDI Variant
Uses Container Device Interface (CDI) for device management.

```bash
helm install enflame-device-plugin ./enflame-device-plugin --set variant=cdi
```

#### 3. PCIe Switch Affinity Variant
Optimized for PCIe switch topology awareness.

```bash
helm install enflame-device-plugin ./enflame-device-plugin --set variant=pcie-switch-affinity
```

#### 4. CPU Manager Compatibility Variant
Compatible with Kubernetes CPU Manager.

```bash
helm install enflame-device-plugin ./enflame-device-plugin --set variant=compat-with-cpumanager
```

## Configuration

### Variant Comparison

| Feature | Basic | CDI | PCIe Switch Affinity | CPU Manager Compat |
|---------|-------|-----|---------------------|-------------------|
| Security Mode | Restricted | Privileged | Restricted | Privileged |
| RBAC Permissions | Basic | Basic | Extended | Basic |
| Device List Strategy | Default | CDI Annotations | Default | Default |
| PCIe Affinity | No | No | Yes | No |
| Device Specs Passing | No | No | No | Yes |
| Additional Tolerations | No | No | No | Yes |

### Customizing Configuration

You can override any configuration using `--set` flags or custom values files:

```bash
# Custom image tag
helm install enflame-device-plugin ./enflame-device-plugin \
  --set image.tag=v2.1.0

# Custom variant with additional overrides
helm install enflame-device-plugin ./enflame-device-plugin \
  --set variant=cdi \
  --set image.pullPolicy=Always \
  --set overrides.env.CUSTOM_REGISTER_RESOURCE="custom.com/gcu"
```

### Using Custom Values File

Create a custom values file for more complex configurations:

```yaml
# custom-values.yaml
variant: "cdi"
image:
  tag: "v2.1.0"
  pullPolicy: Always
overrides:
  env:
    CUSTOM_REGISTER_RESOURCE: "custom.com/gcu"
```

```bash
helm install enflame-device-plugin ./enflame-device-plugin -f custom-values.yaml
```

### Environment Variables

The chart supports all Enflame device plugin environment variables. Common ones include:

- `FAIL_ON_INIT_ERROR`: Whether to fail on initialization errors (default: "false")
- `CUSTOM_REGISTER_RESOURCE`: Custom resource names to register
- `CDI_CONFIG_PATH`: Path to CDI configuration (default: "/etc/cdi")
- `DEVICE_LIST_STRATEGY`: Device listing strategy (set automatically for CDI variant)
- `PCIE_SWITCH_AFFINITY`: Enable PCIe switch affinity (set automatically for PCIe variant)
- `PASS_DEVICE_SPECS`: Pass device specifications (set automatically for CPU Manager variant)

Override environment variables using:

```bash
helm install enflame-device-plugin ./enflame-device-plugin \
  --set overrides.env.FAIL_ON_INIT_ERROR="true"
```

## Upgrading

```bash
# Upgrade to new chart version
helm upgrade enflame-device-plugin ./enflame-device-plugin

# Change variants
helm upgrade enflame-device-plugin ./enflame-device-plugin --set variant=pcie-switch-affinity
```

## Uninstalling

```bash
helm uninstall enflame-device-plugin
```

## Verification

After installation, verify the deployment:

1. Check DaemonSet status:
   ```bash
   kubectl get daemonset enflame-device-plugin-daemonset -n kube-system
   ```

2. Check pod status:
   ```bash
   kubectl get pods -n kube-system -l name=enflame-device-plugin-ds
   ```

3. Verify node resources:
   ```bash
   kubectl describe nodes | grep enflame.com
   ```

4. Check logs:
   ```bash
   kubectl logs -n kube-system -l name=enflame-device-plugin-ds
   ```

## Available Variants

### Basic (Default)
- **Description**: Standard deployment with minimal permissions
- **Security**: Restricted security context
- **RBAC**: Basic pod permissions (get, list, watch)
- **Use Case**: General purpose deployments

### CDI
- **Description**: Container Device Interface variant
- **Security**: Privileged security context
- **RBAC**: Basic pod permissions
- **Special**: Uses CDI annotations for device management
- **Use Case**: Advanced container runtimes with CDI support

### PCIe Switch Affinity
- **Description**: PCIe switch topology awareness
- **Security**: Restricted security context
- **RBAC**: Extended permissions (update, patch pods, create bindings)
- **Special**: Enables PCIe switch affinity scheduling
- **Use Case**: High-performance computing with optimal PCIe topology

### CPU Manager Compatibility
- **Description**: Compatible with Kubernetes CPU Manager
- **Security**: Privileged security context
- **RBAC**: Basic pod permissions
- **Special**: Passes device specifications, additional tolerations
- **Use Case**: Environments using Kubernetes CPU Manager

## Troubleshooting

### Common Issues

1. **Pods not running**: Check node labels and tolerations
2. **No resources detected**: Verify Enflame drivers are installed
3. **Permission errors**: Ensure correct variant is used
4. **CDI errors**: Verify CDI configuration files exist on nodes

### Debugging Commands

```bash
# Check deployment status
helm status enflame-device-plugin

# Describe pods for detailed info
kubectl describe pods -n kube-system -l name=enflame-device-plugin-ds

# Check logs
kubectl logs -n kube-system -l name=enflame-device-plugin-ds -f

# Verify RBAC permissions
kubectl auth can-i get pods --as=system:serviceaccount:kube-system:enflame-k8s-device-plugin
```

## Development

### Testing

```bash
# Validate chart
helm lint ./enflame-device-plugin

# Generate templates for review
helm template enflame-device-plugin ./enflame-device-plugin

# Test different variants
helm template enflame-device-plugin ./enflame-device-plugin --set variant=cdi
```

### Values Structure

The chart uses a simple structure:
- `variant`: Selects the deployment variant
- `image`: Container image configuration  
- `namespace`: Deployment namespace
- `overrides`: Override any variant-specific settings
- `variants`: Predefined configurations for each variant
- `global`: Shared configuration used by all variants
