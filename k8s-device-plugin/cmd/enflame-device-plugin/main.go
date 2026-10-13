/*
 * Copyright (c) 2020, NVIDIA CORPORATION.  All rights reserved.
 * Copyright (c) 2024, ENFLAME INC.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	// efinfo "github.com/ENFLAME/go-eflib/pkg/eflib/info"
	"github.com/fsnotify/fsnotify"
	"github.com/urfave/cli/v2"

	// "k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	spec "enflame/k8s-device-plugin/api/config/v1"
	"enflame/k8s-device-plugin/internal/info"
	"enflame/k8s-device-plugin/internal/logger"
	"enflame/k8s-device-plugin/internal/plugin"
	"enflame/k8s-device-plugin/internal/rm"
	"enflame/k8s-device-plugin/internal/types"
	"enflame/k8s-device-plugin/internal/utils"
	"enflame/k8s-device-plugin/internal/watch"
)

func main() {
	var configFile string

	c := cli.NewApp()
	c.Name = "ENFLAME Device Plugin"
	c.Usage = "ENFLAME device plugin for Kubernetes"
	c.Version = info.GetVersionString()
	c.Action = func(ctx *cli.Context) error {
		return start(ctx, c.Flags)
	}

	c.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "mig-strategy",
			Value:   spec.MigStrategyNone,
			Usage:   "the desired strategy for exposing MIG devices on GCUs that support it:\n\t\t[none | single | mixed]",
			EnvVars: []string{"MIG_STRATEGY"},
		},
		&cli.BoolFlag{
			Name:    "fail-on-init-error",
			Value:   true,
			Usage:   "fail the plugin if an error is encountered during initialization, otherwise block indefinitely",
			EnvVars: []string{"FAIL_ON_INIT_ERROR"},
		},
		&cli.StringFlag{
			Name:    "enflame-driver-root",
			Value:   "/",
			Usage:   "the root path for the ENFLAME driver installation (typical values are '/' or '/run/enflame/driver')",
			EnvVars: []string{"ENFLAME_DRIVER_ROOT"},
		},
		&cli.BoolFlag{
			Name:    "pass-device-specs",
			Value:   false,
			Usage:   "pass the list of DeviceSpecs to the kubelet on Allocate()",
			EnvVars: []string{"PASS_DEVICE_SPECS"},
		},
		&cli.StringSliceFlag{
			Name:    "device-list-strategy",
			Value:   cli.NewStringSlice(string(spec.DeviceListStrategyEnvvar)),
			Usage:   "the desired strategy for passing the device list to the underlying runtime:\n\t\t[envvar | volume-mounts | cdi-annotations]",
			EnvVars: []string{"DEVICE_LIST_STRATEGY"},
		},
		&cli.StringFlag{
			Name:    "device-id-strategy",
			Value:   spec.DeviceIDStrategyIndex,
			Usage:   "the desired strategy for passing device IDs to the underlying runtime:\n\t\t[uuid | index]",
			EnvVars: []string{"DEVICE_ID_STRATEGY"},
		},
		&cli.BoolFlag{
			Name:    "gds-enabled",
			Usage:   "ensure that containers are started with ENFLAME_GDS=enabled",
			EnvVars: []string{"GDS_ENABLED"},
		},
		&cli.BoolFlag{
			Name:    "mofed-enabled",
			Usage:   "ensure that containers are started with ENFLAME_MOFED=enabled",
			EnvVars: []string{"MOFED_ENABLED"},
		},
		&cli.StringFlag{
			Name:        "config-file",
			Usage:       "the path to a config file as an alternative to command line options or environment variables",
			Destination: &configFile,
			EnvVars:     []string{"CONFIG_FILE"},
		},
		&cli.StringFlag{
			Name:    "cdi-annotation-prefix",
			Value:   spec.DefaultCDIAnnotationPrefix,
			Usage:   "the prefix to use for CDI container annotation keys",
			EnvVars: []string{"CDI_ANNOTATION_PREFIX"},
		},
		&cli.StringFlag{
			Name:    "enflame-ctk-path",
			Value:   spec.DefaultEnflameCTKPath,
			Usage:   "the path to use for the enflame-ctk in the generated CDI specification",
			EnvVars: []string{"ENFLAME_CTK_PATH"},
		},
		&cli.StringFlag{
			Name:    "container-driver-root",
			Value:   spec.DefaultContainerDriverRoot,
			Usage:   "the path where the ENFLAME driver root is mounted in the container; used for generating CDI specifications",
			EnvVars: []string{"CONTAINER_DRIVER_ROOT"},
		},
		&cli.StringFlag{
			Name:    "mps-root",
			Usage:   "the path on the host where MPS-specific mounts and files are created by the MPS control daemon manager",
			EnvVars: []string{"MPS_ROOT"},
		},
		&cli.BoolFlag{
			Name:    "pcie-switch-affinity",
			Usage:   "whether to enable pcie switch affinity scheduling policy",
			EnvVars: []string{"PCIE_SWITCH_AFFINITY"},
		},
		&cli.StringSliceFlag{
			Name:    "custom-register-resource",
			Value:   cli.NewStringSlice(string(spec.DefaultRegisterResource)),
			Usage:   "customize the resource name to be registered by k8s-device-plugin",
			EnvVars: []string{"CUSTOM_REGISTER_RESOURCE"},
		},
		&cli.StringFlag{
			Name:    "cdi-config-path",
			Value:   spec.DefaultCDIConfigPath,
			Usage:   "the absolute path to the folder where the cdi config files is located",
			EnvVars: []string{"CDI_CONFIG_PATH"},
		},
		&cli.BoolFlag{
			Name:    "register-all-cdi-config",
			Value:   true,
			Usage:   "under the premise of registering custom resources, whether to register all resource types in the cdi config files",
			EnvVars: []string{"REGISTER_ALL_CDI_CONFIG"},
		},
		&cli.BoolFlag{
			Name:    "recreate-cdi-config-file",
			Value:   true,
			Usage:   "whether to recreate the cdi config files when the cdi config files already exist",
			EnvVars: []string{"RECREATE_CDI_CONFIG_FILE"},
		},
		&cli.BoolFlag{
			Name:    "by-card-scheduling",
			Usage:   "whether to mount whole GCU cards according to an external scheduler's result written on pod annotations",
			EnvVars: []string{"BY_CARD_SCHEDULING"},
		},
		&cli.StringFlag{
			Name:    "scheduler-device-annotation-key",
			Value:   plugin.DefaultSchedulerDeviceAnnotationKey,
			Usage:   "pod annotation key holding the scheduler-assigned device indices (by-card scheduling)",
			EnvVars: []string{"SCHEDULER_DEVICE_ANNOTATION_KEY"},
		},
		&cli.StringFlag{
			Name:    "scheduler-assumed-time-annotation-key",
			Value:   plugin.DefaultSchedulerAssumedTimeAnnotationKey,
			Usage:   "pod annotation key holding the scheduler assumed-time used to order pending pods (by-card scheduling)",
			EnvVars: []string{"SCHEDULER_ASSUMED_TIME_ANNOTATION_KEY"},
		},
		&cli.BoolFlag{
			Name:    "reconcile-stock-pods",
			Usage:   "on startup, backfill real allocated GCU cards (from kubelet podresources) onto pods scheduled before by-card was enabled",
			EnvVars: []string{"RECONCILE_STOCK_PODS"},
		},
		&cli.StringFlag{
			Name:    "pod-source",
			Value:   plugin.PodSourceApiServer,
			Usage:   "where by-card lists this node's pending pods from on each Allocate:\n\t\t[apiserver | informer | kubelet] (kubelet = node-local, zero apiserver, for very large clusters)",
			EnvVars: []string{"POD_SOURCE"},
		},
	}

	err := c.Run(os.Args)
	if err != nil {
		logger.Error(err, "run enflame device plugin failed")
		os.Exit(1)
	}
}

func validateFlags(config *spec.Config) error {
	// EF_CLOSE
	/*
		deviceListStrategies, err := spec.NewDeviceListStrategies(*config.Flags.Plugin.DeviceListStrategy)
		if err != nil {
			return fmt.Errorf("invalid --device-list-strategy option: %v", err)
		}

		hasEfml, _ := efinfo.New().HasEfml()
		if deviceListStrategies.IsCDIEnabled() && !hasEfml {
			return fmt.Errorf("CDI --device-list-strategy options are only supported on EFML-based systems")
		}
	*/
	if *config.Flags.Plugin.DeviceIDStrategy != spec.DeviceIDStrategyUUID && *config.Flags.Plugin.DeviceIDStrategy != spec.DeviceIDStrategyIndex {
		err := fmt.Errorf("invalid --device-id-strategy option: %v", *config.Flags.Plugin.DeviceIDStrategy)
		logger.Error(err)
		return err
	}

	// by-card and pcie-switch-affinity both override the kubelet card choice and
	// are mutually exclusive.
	pcieAffinityOn := config.Flags.Plugin.PcieSwitchAffinity != nil && *config.Flags.Plugin.PcieSwitchAffinity
	byCardOn := config.Flags.Plugin.ByCardScheduling != nil && *config.Flags.Plugin.ByCardScheduling
	if pcieAffinityOn && byCardOn {
		err := fmt.Errorf("--pcie-switch-affinity and --by-card-scheduling are mutually exclusive")
		logger.Error(err)
		return err
	}

	// by-card pending-pod source (large-scale optimization, design §5.5).
	if byCardOn && config.Flags.Plugin.PodSource != nil {
		switch *config.Flags.Plugin.PodSource {
		case "", plugin.PodSourceApiServer, plugin.PodSourceInformer, plugin.PodSourceKubelet:
		default:
			err := fmt.Errorf("invalid --pod-source option %q (want apiserver|informer|kubelet)", *config.Flags.Plugin.PodSource)
			logger.Error(err)
			return err
		}
	}

	if config.Sharing.SharingStrategy() == spec.SharingStrategyMPS {
		if *config.Flags.MigStrategy == spec.MigStrategyMixed {
			err := fmt.Errorf("using --mig-strategy=mixed is not supported with MPS")
			logger.Error(err)
			return err
		}
		if config.Flags.MpsRoot == nil || *config.Flags.MpsRoot == "" {
			err := fmt.Errorf("using MPS requires --mps-root to be specified")
			logger.Error(err)
			return err
		}
	}

	return nil
}

func loadConfig(c *cli.Context, flags []cli.Flag) (*spec.Config, error) {
	config, err := spec.NewConfig(c, flags)
	if err != nil {
		return nil, fmt.Errorf("unable to finalize config: %v", err)
	}
	err = validateFlags(config)
	if err != nil {
		return nil, fmt.Errorf("unable to validate flags: %v", err)
	}
	config.Flags.GFD = nil
	return config, nil
}

func start(c *cli.Context, flags []cli.Flag) error {
	logger.Info("Starting FS watcher.")
	watcher, err := watch.Files(pluginapi.DevicePluginPath)
	if err != nil {
		return fmt.Errorf("failed to create FS watcher for %s: %v", pluginapi.DevicePluginPath, err)
	}
	defer watcher.Close()

	logger.Info("Starting OS watcher.")
	sigs := watch.Signals(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	var started bool
	var restartTimeout <-chan time.Time
	var plugins []plugin.Interface
restart:
	// If we are restarting, stop plugins from previous run.
	if started {
		err := stopPlugins(plugins)
		if err != nil {
			logger.Error(err, "error stopping plugins from previous run")
			return err
		}
	}

	// Load the configuration file
	logger.Info("Loading configuration.")
	config, err := loadConfig(c, flags)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}
	logger.Info("Starting Plugins.")
	plugins, restartPlugins, err := startPlugins(config, watcher)
	if err != nil {
		return fmt.Errorf("error starting plugins: %v", err)
	}
	started = true

	if restartPlugins {
		logger.Warn("Failed to start one or more plugins. Retrying in 30s...")
		restartTimeout = time.After(30 * time.Second)
	}

	// Start an infinite loop, waiting for several indicators to either log
	// some messages, trigger a restart of the plugins, or exit the program.
	restartByFsWatcher := false
	for {
		select {
		// If the restart timeout has expired, then restart the plugins
		case <-restartTimeout:
			goto restart

		// Detect a kubelet restart by watching for a newly created
		// 'pluginapi.KubeletSocket' file. When this occurs, restart this loop,
		// restarting all of the plugins in the process.
		case event := <-watcher.Events:
			if event.Name == pluginapi.KubeletSocket && event.Op&fsnotify.Create == fsnotify.Create {
				logger.Warn("inotify: %s created, restarting.", pluginapi.KubeletSocket)
				restartByFsWatcher = true
			}
			// If cdi config path is detected to be modified, the k8s-device-plugin will restart this loop
			if filepath.Dir(event.Name) == *config.Flags.CDIConfigPath && utils.StrList([]string{".json", ".yaml", ".yml"}).ContainValue(
				filepath.Ext(event.Name)) && event.Op&fsnotify.Chmod != fsnotify.Chmod {
				logger.Warn("fsnotify: cdi config file: %s has been %v, need restart %s", event.Name, event.Op, types.COMPONENT_NAME)
				restartByFsWatcher = true
				watcher.Remove(*config.Flags.CDIConfigPath)
			}
		default:
			// To prevent frequent restarts, decide whether to restart after consuming all fs watcher events
			if restartByFsWatcher {
				logger.Warn("This round of event consumption is completed, restarting: %s uniformly", types.COMPONENT_NAME)
				time.Sleep(1 * time.Second)
				goto restart
			}
			// bugfix:SCGSRE-1712, to avoid cpu 100%
			time.Sleep(1 * time.Second)

		// Watch for any other fs errors and log them.
		case err := <-watcher.Errors:
			logger.Error(err, "watch occured error")

		// Watch for any signals from the OS. On SIGHUP, restart this loop,
		// restarting all of the plugins in the process. On all other
		// signals, exit the loop and exit the program.
		case s := <-sigs:
			switch s {
			case syscall.SIGHUP:
				logger.Warn("Received SIGHUP, restarting.")
				goto restart
			default:
				logger.Warn("Received signal \"%v\", shutting down.", s)
				goto exit
			}
		}
	}
exit:
	err = stopPlugins(plugins)
	if err != nil {
		logger.Error(err, "error stopping plugins")
		return err
	}
	return nil
}

func startPlugins(config *spec.Config, watcher *fsnotify.Watcher) ([]plugin.Interface, bool, error) {
	spec.DisableResourceNamingInConfig(config)

	// Update the configuration file with default resources.
	logger.Info("Updating config with default resource matching patterns.")
	err := rm.AddDefaultResourcesToConfig(config)
	if err != nil {
		return nil, false, fmt.Errorf("unable to add default resources to config: %v", err)
	}
	if err := spec.CheckUnsupportedFeatures(config); err != nil {
		return nil, false, err
	}

	// Print the config to the output.
	configJSON, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return nil, false, fmt.Errorf("failed to marshal config to JSON: %v", err)
	}
	logger.Info("Running with config:\n%v", string(configJSON))

	// Get the set of plugins.
	logger.Info("Retrieving plugins.")
	pluginManager, cdiEnabled, err := NewPluginManager(config)
	if err != nil {
		return nil, false, fmt.Errorf("error creating plugin manager: %v", err)
	}
	if cdiEnabled {
		logger.Info("add %s to FS watcher", *config.Flags.CDIConfigPath)
		if err := watcher.Add(*config.Flags.CDIConfigPath); err != nil {
			return nil, false, err
		}
	}
	plugins, err := pluginManager.GetPlugins()
	if err != nil {
		return nil, false, fmt.Errorf("error getting plugins: %v", err)
	}
	logger.Info("Retrieving plugins success.")
	// Loop through all plugins, starting them if they have any devices
	// to serve. If even one plugin fails to start properly, try
	// starting them all again.
	started := 0
	for _, p := range plugins {
		// Just continue if there are no devices to serve for plugin p.
		if len(p.Devices()) == 0 {
			continue
		}

		// Start the gRPC server for plugin p and connect it with the kubelet.
		if err := p.Start(); err != nil {
			return plugins, true, nil
		}
		started++
	}

	if started == 0 {
		logger.Warn("No devices found. Waiting indefinitely.")
	}

	return plugins, false, nil
}

func stopPlugins(plugins []plugin.Interface) error {
	logger.Info("Stopping plugins.")
	var errs error
	for _, p := range plugins {
		errs = errors.Join(errs, p.Stop())
	}
	return errs
}
