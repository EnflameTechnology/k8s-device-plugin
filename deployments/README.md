# Enflame Device Plugin Deployment Scripts

This directory contains intelligent deployment and deletion scripts that automatically choose the best deployment method available.

## 🚀 Quick Start

### Deploy the Device Plugin
```bash
# Basic deployment (auto-detects Helm)
./deploy.sh

# CDI variant deployment
./deploy.sh --mode cdi

# PCIe Switch Affinity variant
./deploy.sh --mode psa

# CPU Manager compatibility variant
./deploy.sh --mode cm
```

### Delete the Device Plugin
```bash
# Auto-detect and delete
./delete.sh

# Force YAML deletion method
./delete.sh --static
```

## 📋 Available Scripts

### `deploy.sh` - Intelligent Deployment Script

**Features:**
- ✅ **Auto-detection**: Automatically detects if Helm is available
- ✅ **Fallback support**: Uses YAML files if Helm is not available
- ✅ **Mode mapping**: Maps legacy modes to new Helm variants
- ✅ **Enhanced options**: Supports custom image tags, namespaces, etc.

**Usage Examples:**
```bash
# Auto-detect deployment method
./deploy.sh --mode cdi

# Force YAML deployment even if Helm exists
./deploy.sh --mode psa --static

# Helm deployment with custom settings
./deploy.sh --mode cm --image-tag v2.1.0 --helm-release my-plugin

# Deploy to custom namespace
./deploy.sh --mode cdi --namespace ai-system
```

### `delete.sh` - Intelligent Deletion Script

**Features:**
- ✅ **Auto-detection**: Detects how the plugin was originally deployed
- ✅ **Smart cleanup**: Removes resources completely regardless of deployment method
- ✅ **Orphan cleanup**: Cleans up any remaining resources after deletion
- ✅ **Safe operation**: Uses `--ignore-not-found` to avoid errors

**Usage Examples:**
```bash
# Auto-detect and delete (recommended)
./delete.sh

# Delete specific Helm release
./delete.sh --helm-release my-plugin

# Force YAML deletion method
./delete.sh --mode psa --static

# Delete from custom namespace
./delete.sh --namespace ai-system
```

## 🔄 Deployment Methods

### 1. Helm Chart Deployment (Preferred)

**When used:**
- Helm is installed and available in PATH
- Not forced to use YAML with `--static`

**Benefits:**
- Unified configuration with variant selection
- Easy upgrades and rollbacks with `helm upgrade`
- Advanced configuration options
- Better resource management

**Command example:**
```bash
# Internally runs: helm install enflame-device-plugin helm/enflame-device-plugin --set variant=cdi
./deploy.sh --mode cdi
```

### 2. YAML File Deployment (Fallback)

**When used:**
- Helm is not available
- Forced with `--static` flag
- Fallback when Helm deployment fails

**Benefits:**
- Works without Helm installation
- Direct kubectl apply method
- Legacy compatibility with existing workflows

**Command example:**
```bash
# Internally runs: kubectl create -f yaml/enflame-device-plugin-cdi.yaml
./deploy.sh --mode cdi --static
```

## 📊 Mode Mapping

| Script Mode | YAML File | Helm Variant | Description |
|-------------|-----------|--------------|-------------|
| `basic` | `enflame-device-plugin.yaml` | `basic` | Standard deployment |
| `cdi` | `enflame-device-plugin-cdi.yaml` | `cdi` | Container Device Interface |
| `psa` | `enflame-device-plugin-pcie-switch-affinity.yaml` | `pcie-switch-affinity` | PCIe Switch Affinity |
| `cm` | `enflame-device-plugin-compat-with-cpumanager.yaml` | `compat-with-cpumanager` | CPU Manager Compatibility |

## 🛠️ Advanced Usage

### Custom Helm Configuration
```bash
# Deploy with specific image and release name
./deploy.sh \
  --mode cdi \
  --image-tag v2.1.0 \
  --image-pull-policy Always \
  --helm-release production-device-plugin \
  --namespace kube-system
```

### Force Specific Deployment Method
```bash
# Always use YAML even if Helm is available
./deploy.sh --mode psa --static

# Always use Helm (will fail if not available)
./deploy.sh --mode cdi  # (default behavior when Helm exists)
```

### Custom Namespace Operations
```bash
# Deploy to custom namespace
./deploy.sh --mode cdi --namespace ai-infrastructure

# Delete from custom namespace
./delete.sh --namespace ai-infrastructure
```

## 🔍 Troubleshooting

### Check Deployment Status
```bash
# For Helm deployments
helm status enflame-device-plugin
helm list -n kube-system

# For all deployments
kubectl get pods -n kube-system -l name=enflame-device-plugin-ds
kubectl get daemonset -n kube-system
```

### Common Issues

1. **"kubectl not found"**
   - Install kubectl: https://kubernetes.io/docs/tasks/tools/
   
2. **"Helm not found" (using fallback)**
   - This is basic - script will use YAML files
   - Install Helm for better experience: https://helm.sh/docs/intro/install/
   
3. **"YAML file not found"**
   - Ensure you're running script from the `deployments/` directory
   - Check that `yaml/` directory contains the required files
   
4. **"Helm deployment failed"**
   - Check Helm chart exists: `ls helm/enflame-device-plugin/`
   - Verify chart syntax: `helm lint helm/enflame-device-plugin/`
   - Use `--static` as fallback

### Debug Mode
```bash
# Enable debug output
set -x
./deploy.sh --mode cdi
set +x
```

## 📁 Directory Structure

```bash
deployments/
├── deploy.sh                          # Intelligent deployment script
├── delete.sh                          # Intelligent deletion script
├── README.md                          # This documentation
├── yaml/                              # YAML deployment files
│   ├── rbac.yaml
│   ├── enflame-device-plugin.yaml
│   ├── enflame-device-plugin-cdi.yaml
│   ├── enflame-device-plugin-pcie-switch-affinity.yaml
│   └── enflame-device-plugin-compat-with-cpumanager.yaml
└── helm/                              # Helm chart directory
    └── enflame-device-plugin/         # Unified Helm chart
        ├── Chart.yaml
        ├── values.yaml
        ├── templates/
        └── README.md
```

## 🎯 Migration from Old Scripts

### Before (Manual Selection)
```bash
# Had to manually choose YAML files
kubectl create -f yaml/enflame-device-plugin-cdi.yaml
kubectl delete -f yaml/enflame-device-plugin-cdi.yaml
```

### After (Intelligent Scripts)
```bash
# Scripts automatically choose best method
./deploy.sh --mode cdi
./delete.sh
```

## 📞 Support

For issues with the deployment scripts, check:
1. Script help: `./deploy.sh --help` or `./delete.sh --help`
2. Kubernetes connectivity: `kubectl get nodes`
3. Helm installation: `helm version`
4. Chart validation: `helm lint helm/enflame-device-plugin/`

## 🚀 Recommended Workflow

1. **First-time setup:**
   ```bash
   ./deploy.sh --mode cdi
   ```

2. **Verify deployment:**
   ```bash
   kubectl get pods -n kube-system -l name=enflame-device-plugin-ds
   kubectl describe nodes | grep enflame.com
   ```

3. **Upgrade (Helm only):**
   ```bash
   # Will automatically use Helm if originally deployed with Helm
   ./deploy.sh --mode cdi --image-tag v2.1.0
   ```

4. **Cleanup:**
   ```bash
   ./delete.sh
   ```

The scripts provide a seamless transition between YAML and Helm deployment methods while maintaining full backward compatibility!