// Copyright (c) 2019, NVIDIA CORPORATION. All rights reserved.
// Copyright (c) 2024, ENFLAME INC.  All rights reserved.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
	"k8s.io/klog/v2"

	spec "enflame/k8s-device-plugin/api/config/v1"
	"enflame/k8s-device-plugin/internal/flags"
	"enflame/k8s-device-plugin/internal/info"
	"enflame/k8s-device-plugin/internal/lm"
	"enflame/k8s-device-plugin/internal/resource"
	"enflame/k8s-device-plugin/internal/vgcu"
	"enflame/k8s-device-plugin/internal/watch"
)

// Config represents a collection of config options for GFD.
type Config struct {
	configFile string

	kubeClientConfig flags.KubeClientConfig
	nodeConfig       flags.NodeConfig

	// flags stores the CLI flags for later processing.
	flags []cli.Flag
}

func main() {
	config := &Config{}

	c := cli.NewApp()
	c.Name = "GCU Feature Discovery"
	c.Usage = "generate labels for ENFLAME devices"
	c.Version = info.GetVersionString()
	c.Action = func(ctx *cli.Context) error {
		return start(ctx, config)
	}

	config.flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "mig-strategy",
			Value:   spec.MigStrategyNone,
			Usage:   "the desired strategy for exposing MIG devices on GCUs that support it:\n\t\t[none | single | mixed]",
			EnvVars: []string{"GFD_MIG_STRATEGY", "MIG_STRATEGY"},
		},
		&cli.BoolFlag{
			Name:    "fail-on-init-error",
			Value:   true,
			Usage:   "fail the plugin if an error is encountered during initialization, otherwise block indefinitely",
			EnvVars: []string{"GFD_FAIL_ON_INIT_ERROR", "FAIL_ON_INIT_ERROR"},
		},
		&cli.BoolFlag{
			Name:    "oneshot",
			Value:   false,
			Usage:   "Label once and exit",
			EnvVars: []string{"GFD_ONESHOT"},
		},
		&cli.BoolFlag{
			Name:    "no-timestamp",
			Value:   false,
			Usage:   "Do not add the timestamp to the labels",
			EnvVars: []string{"GFD_NO_TIMESTAMP"},
		},
		&cli.DurationFlag{
			Name:    "sleep-interval",
			Value:   60 * time.Second,
			Usage:   "Time to sleep between labeling",
			EnvVars: []string{"GFD_SLEEP_INTERVAL"},
		},
		&cli.StringFlag{
			Name:    "output-file",
			Aliases: []string{"output", "o"},
			Value:   "/etc/kubernetes/node-feature-discovery/features.d/gfd",
			EnvVars: []string{"GFD_OUTPUT_FILE"},
		},
		&cli.StringFlag{
			Name:    "machine-type-file",
			Value:   "/sys/class/dmi/id/product_name",
			Usage:   "a path to a file that contains the DMI (SMBIOS) information for the node",
			EnvVars: []string{"GFD_MACHINE_TYPE_FILE"},
		},
		&cli.StringFlag{
			Name:        "config-file",
			Usage:       "the path to a config file as an alternative to command line options or environment variables",
			Destination: &config.configFile,
			EnvVars:     []string{"GFD_CONFIG_FILE", "CONFIG_FILE"},
		},
		&cli.BoolFlag{
			Name:    "use-node-feature-api",
			Usage:   "Use NFD NodeFeature API to publish labels",
			EnvVars: []string{"GFD_USE_NODE_FEATURE_API", "USE_NODE_FEATURE_API"},
		},
	}

	config.flags = append(config.flags, config.kubeClientConfig.Flags()...)
	config.flags = append(config.flags, config.nodeConfig.Flags()...)

	c.Flags = config.flags

	if err := c.Run(os.Args); err != nil {
		klog.Error(err)
		os.Exit(1)
	}
}

func validateFlags(config *spec.Config) error {
	return nil
}

// loadConfig loads the config from the spec file.
func (cfg *Config) loadConfig(c *cli.Context) (*spec.Config, error) {
	config, err := spec.NewConfig(c, cfg.flags)
	if err != nil {
		return nil, fmt.Errorf("unable to finalize config: %v", err)
	}
	err = validateFlags(config)
	if err != nil {
		return nil, fmt.Errorf("unable to validate flags: %v", err)
	}
	config.Flags.Plugin = nil

	return config, nil
}

func start(c *cli.Context, cfg *Config) error {
	defer func() {
		klog.Info("Exiting")
	}()

	klog.Info("Starting OS watcher.")
	sigs := watch.Signals(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		// Load the configuration file
		klog.Info("Loading configuration.")
		config, err := cfg.loadConfig(c)
		if err != nil {
			return fmt.Errorf("unable to load config: %v", err)
		}
		spec.DisableResourceNamingInConfig(config)

		// Print the config to the output.
		configJSON, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config to JSON: %v", err)
		}
		klog.Infof("\nRunning with config:\n%v", string(configJSON))

		manager := resource.NewManager(config)
		vgcul := vgcu.NewVGCULib(vgcu.NewEnflamePCILib())

		var clientSets flags.ClientSets
		if config.Flags.UseNodeFeatureAPI != nil && *config.Flags.UseNodeFeatureAPI {
			cs, err := cfg.kubeClientConfig.NewClientSets()
			if err != nil {
				return fmt.Errorf("failed to create clientsets: %w", err)
			}
			clientSets = cs
		}

		klog.Info("Start running")
		d := &gfd{
			manager: manager,
			vgcu:    vgcul,
			config:  config,

			labelOutputer: lm.NewOutputer(
				config,
				cfg.nodeConfig,
				clientSets,
			),
		}
		restart, err := d.run(sigs)
		if err != nil {
			return err
		}

		if !restart {
			return nil
		}
	}
}

type gfd struct {
	manager resource.Manager
	vgcu    vgcu.Interface
	config  *spec.Config

	labelOutputer lm.Outputer
}

func (d *gfd) run(sigs chan os.Signal) (bool, error) {
	defer func() {
		if d.config.Flags.UseNodeFeatureAPI != nil && *d.config.Flags.UseNodeFeatureAPI {
			return
		}
		if d.config.Flags.GFD.Oneshot != nil && *d.config.Flags.GFD.Oneshot {
			return
		}
		if d.config.Flags.GFD.OutputFile != nil && *d.config.Flags.GFD.OutputFile == "" {
			return
		}
		err := removeOutputFile(*d.config.Flags.GFD.OutputFile)
		if err != nil {
			klog.Warningf("Error removing output file: %v", err)
		}
	}()

	timestampLabeler := lm.NewTimestampLabeler(d.config)
rerun:
	loopLabelers, err := lm.NewLabelers(d.manager, d.vgcu, d.config)
	if err != nil {
		return false, err
	}

	labelers := lm.Merge(
		timestampLabeler,
		loopLabelers,
	)

	labels, err := labelers.Labels()
	if err != nil {
		return false, fmt.Errorf("error generating labels: %v", err)
	}

	if len(labels) <= 1 {
		klog.Warning("No labels generated from any source")
	}

	klog.Info("Creating Labels")
	if err := d.labelOutputer.Output(labels); err != nil {
		return false, err
	}

	if *d.config.Flags.GFD.Oneshot {
		return false, nil
	}

	klog.Info("Sleeping for ", *d.config.Flags.GFD.SleepInterval)
	rerunTimeout := time.After(time.Duration(*d.config.Flags.GFD.SleepInterval))

	for {
		select {
		case <-rerunTimeout:
			goto rerun

		// Watch for any signals from the OS. On SIGHUP trigger a reload of the config.
		// On all other signals, exit the loop and exit the program.
		case s := <-sigs:
			switch s {
			case syscall.SIGHUP:
				klog.Info("Received SIGHUP, restarting.")
				return true, nil
			default:
				klog.Infof("Received signal %v, shutting down.", s)
				return false, nil
			}
		}
	}
}

func removeOutputFile(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to retrieve absolute path of output file: %v", err)
	}

	absDir := filepath.Dir(absPath)
	tmpDir := filepath.Join(absDir, "gfd-tmp")

	err = os.RemoveAll(tmpDir)
	if err != nil {
		return fmt.Errorf("failed to remove temporary output directory: %v", err)
	}

	err = os.Remove(absPath)
	if err != nil {
		return fmt.Errorf("failed to remove output file: %v", err)
	}

	return nil
}
