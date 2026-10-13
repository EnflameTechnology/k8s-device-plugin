/*
 * Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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

package v1

import (
	"fmt"
	"io"
	"os"

	cli "github.com/urfave/cli/v2"
	"sigs.k8s.io/yaml"

	"enflame/k8s-device-plugin/internal/logger"
)

// Version indicates the version of the 'Config' struct used to hold configuration information.
const Version = "v1"

// Config is a versioned struct used to hold configuration information.
type Config struct {
	Version   string    `json:"version"             yaml:"version"`
	Flags     Flags     `json:"flags,omitempty"     yaml:"flags,omitempty"`
	Resources Resources `json:"resources,omitempty" yaml:"resources,omitempty"`
	Sharing   Sharing   `json:"sharing,omitempty"   yaml:"sharing,omitempty"`
}

// NewConfig builds out a Config struct from a config file (or command line flags).
// The data stored in the config will be populated in order of precedence from
// (1) command line, (2) environment variable, (3) config file.
func NewConfig(c *cli.Context, flags []cli.Flag) (*Config, error) {
	config := &Config{Version: Version}

	if configFile := c.String("config-file"); configFile != "" {
		var err error
		config, err = parseConfig(configFile)
		if err != nil {
			return nil, fmt.Errorf("unable to parse config file: %v", err)
		}
	}

	config.Flags.UpdateFromCLIFlags(c, flags)

	// We explicitly set sharing.mps.failRequestsGreaterThanOne = true
	// This can be relaxed in certain cases -- such as a single GCU -- but
	// requires additional logic around when it's OK to combine requests and
	// makes the semantics of a request unclear.
	if config.Sharing.MPS != nil {
		config.Sharing.MPS.FailRequestsGreaterThanOne = true
	}

	return config, nil
}

// EF_CLOSE
/*
// logger is used to issue warning in API functions without requiring an explicit implementation.
type logger interface {
	Warning(...interface{})
	Warningf(string, ...interface{})
}
*/

// DisableResourceNamingInConfig temporarily disable the resource renaming feature of the plugin.
// This may be reenabled in a future release.
func DisableResourceNamingInConfig(config *Config) {
	// Disable resource renaming through config.Resource
	if len(config.Resources) > 0 {
		logger.Warn("Customizing the 'resources' field is not yet supported in the config. Ignoring...")
	}
	config.Resources = make(Resources)

	// Disable renaming / device selection in Sharing.TimeSlicing.Resources
	config.Sharing.TimeSlicing.disableResoureRenaming("timeSlicing")
	// Disable renaming / device selection in Sharing.MPS.Resources
	config.Sharing.MPS.disableResoureRenaming("mps")
}

// parseConfig parses a config file as either YAML of JSON and unmarshals it into a Config struct.
func parseConfig(configFile string) (*Config, error) {
	reader, err := os.Open(configFile)
	if err != nil {
		logger.Error(err, "open config file: %s failed", configFile)
		return nil, err
	}
	defer reader.Close()

	config, err := parseConfigFrom(reader)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	return config, nil
}

func parseConfigFrom(reader io.Reader) (*Config, error) {
	var err error
	var configYaml []byte

	configYaml, err = io.ReadAll(reader)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(configYaml, &config)
	if err != nil {
		logger.Error(err, "unmarshal config: %s failed", string(configYaml))
		return nil, err
	}

	if config.Version == "" {
		config.Version = Version
	}

	if config.Version != Version {
		err := fmt.Errorf("unknown version: %v", config.Version)
		logger.Error(err)
		return nil, err
	}

	return &config, nil
}

// EF_SUPPORT
func CheckUnsupportedFeatures(config *Config) error {
	// gcu share
	if len(config.Sharing.TimeSlicing.Resources) > 0 {
		err := fmt.Errorf("the gcu time-slice feature is not supported currently")
		logger.Error(err)
		return err
	}
	if config.Sharing.MPS != nil {
		err := fmt.Errorf("the gcu mps share feature is not supported currently")
		logger.Error(err)
		return err
	}
	// cdi
	// for _, strategy := range *config.Flags.Plugin.DeviceListStrategy {
	// 	if strategy != DeviceListStrategyEnvvar {
	// 		err := fmt.Errorf("the device list strategy: %s is not supported currently", strategy)
	// 		logger.Error(err)
	// 		return err
	// 	}
	// }
	// uuid
	if *config.Flags.Plugin.DeviceIDStrategy != DeviceIDStrategyIndex {
		err := fmt.Errorf("the device id strategy: %s is not supported currently", *config.Flags.Plugin.DeviceIDStrategy)
		logger.Error(err)
		return err
	}

	// mig
	if *config.Flags.MigStrategy != MigStrategyNone {
		err := fmt.Errorf("the mig strategy is not supported currently")
		logger.Error(err)
		return err
	}
	// other fields
	if *config.Flags.GDSEnabled || *config.Flags.MOFEDEnabled {
		err := fmt.Errorf("the GDSEnabled and MOFEDEnabled is not supported currently")
		logger.Error(err)
		return err
	}
	return nil
}
