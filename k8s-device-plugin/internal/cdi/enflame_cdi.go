// Copyright (c) 2024, ENFLAME INC.  All rights reserved.

package cdi

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	cdiparser "tags.cncf.io/container-device-interface/pkg/parser"
	cdiSpec "tags.cncf.io/container-device-interface/specs-go"

	spec "enflame/k8s-device-plugin/api/config/v1"
	"enflame/k8s-device-plugin/internal/eflib/interfaces"
	"enflame/k8s-device-plugin/internal/logger"
	"enflame/k8s-device-plugin/internal/utils"
)

type cdiConfigList map[spec.ResourceName]cdiConfigInfo

type cdiConfigInfo struct {
	fileName  string
	cdiConfig *cdiSpec.Spec
}

type cdiHandler struct {
	resourceName          spec.ResourceName
	hasCDIConfig          bool
	enflameCTKPath        string
	cdiConfigPath         string
	cdiFileName           string
	recreateCdiConfigFile bool

	deviceNodes []*cdiSpec.DeviceNode
}

func NewHandler(config *spec.Config, deviceHandle map[string]interfaces.HandleInterface, cdiEnabled bool) (
	map[spec.ResourceName]CDIInterface, error) {
	if !cdiEnabled {
		return nil, nil
	}
	cdiConfigs, err := handleCDIConfigFiles(config)
	if err != nil {
		return nil, err
	}
	cdiHandlers := map[spec.ResourceName]CDIInterface{}
	for _, resources := range config.Resources {
		for _, resource := range resources {
			if len(deviceHandle[resource.Name.ResourceClass()].GetDeviceIDPathMap()) == 0 {
				logger.Warn("device type: %s not found, skip create cdi handler", resource.Name.ResourceClass())
				continue
			}
			_, hasCDIConfig := cdiConfigs[resource.Name]
			cdiHandlers[resource.Name] = &cdiHandler{
				resourceName:          resource.Name,
				hasCDIConfig:          hasCDIConfig,
				cdiConfigPath:         *config.Flags.CDIConfigPath,
				enflameCTKPath:        *config.Flags.Plugin.EnflameCTKPath,
				recreateCdiConfigFile: *config.Flags.Plugin.RecreateCdiConfigFile,
				deviceNodes: func() []*cdiSpec.DeviceNode {
					if hasCDIConfig {
						return cdiConfigs[resource.Name].cdiConfig.ContainerEdits.DeviceNodes
					}
					return nil
				}(),
				cdiFileName: cdiConfigs[resource.Name].fileName,
			}
		}
	}
	return cdiHandlers, nil
}

func (cdi *cdiHandler) CreateSpecFile() error {
	filePath := fmt.Sprintf("%s/%s.yaml", cdi.cdiConfigPath, cdi.resourceName.GenerateCDIFileName())
	if cdi.hasCDIConfig {
		if !cdi.recreateCdiConfigFile {
			logger.Warn("cdi config file of %s cdiHandler already exist, do nothing", cdi.resourceName)
			return nil
		}
		logger.Warn("will recreate cdi config file: %s for resource: %s with recreateCdiConfigFile: %v",
			cdi.cdiFileName, cdi.resourceName, cdi.recreateCdiConfigFile)
		if err := os.Remove(cdi.cdiFileName); err != nil {
			logger.Error(err, "remove cdi config file: %s failed", cdi.cdiFileName)
			return err
		}
		logger.Info("remove cdi config file: %s success for resource: %s", cdi.cdiFileName, cdi.resourceName)
		filePath = cdi.cdiFileName
	} else {
		logger.Info("cdi config file of resource: %s not found, will create it", cdi.resourceName)
	}
	if utils.FileIsExist(filePath) {
		err := fmt.Errorf("cdi config name: %s already exist, it must be unique", filePath)
		logger.Error(err)
		return err
	}
	cmd := fmt.Sprintf("chroot /host-root %s cdi generate --output=%s --vendor=%s --class=%s", cdi.enflameCTKPath, filePath,
		cdi.resourceName.ResourceVendor(), cdi.resourceName.ResourceClass())
	if err := utils.ExecCommand("sh", "-c", cmd); err != nil {
		return err
	}
	logger.Info("create cdi config file: %s for %s success", filePath, cdi.resourceName)

	// set device nodes
	content, err := utils.ReadYamlToJson(filePath)
	if err != nil {
		return err
	}
	cdiConfig := new(cdiSpec.Spec)
	if err := json.Unmarshal(content, cdiConfig); err != nil {
		logger.Error(err, "json unmarshal config file: %s to strcut failed, content: %s", filePath, string(content))
		return err
	}
	cdi.deviceNodes = cdiConfig.ContainerEdits.DeviceNodes
	logger.Info("set device nodes for cdiHandler of %s success", cdi.resourceName)
	return nil
}

func (cdi *cdiHandler) QualifiedName(class string, id string) string {
	return cdiparser.QualifiedName(cdi.resourceName.ResourceVendor(), class, id)
}

func (cdi *cdiHandler) GetDeviceNodes() []*cdiSpec.DeviceNode {
	return cdi.deviceNodes
}

func handleCDIConfigFiles(config *spec.Config) (cdiConfigList, error) {
	cdiConfigs, err := getCDIConfigList(*config.Flags.CDIConfigPath)
	if err != nil {
		return nil, err
	}
	if *config.Flags.Plugin.RegisterAllCdiConfig {
		logger.Warn("in addition to the custom register resource: %v to be registered, all resource types in the cdi config file will be registered",
			config.Resources)
		for resourceName := range cdiConfigs {
			if err := config.Resources.AddResource("*", resourceName); err != nil {
				return nil, err
			}
		}
	}
	resourceJSON, err := json.MarshalIndent(config.Resources, "", "    ")
	if err != nil {
		logger.Error(err, "failed to marshal config.Resources: %v to JSON", config.Resources)
		return nil, err
	}
	logger.Info("following resource kind will be registered:\n%s", string(resourceJSON))
	return cdiConfigs, nil
}

func getCDIConfigList(cdiConfigPath string) (cdiConfigList, error) {
	cdiConfigs := cdiConfigList{}
	files, err := os.ReadDir(cdiConfigPath)
	if err != nil {
		logger.Error(err, "read cdi config directory: %s failed", cdiConfigPath)
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		filePath := fmt.Sprintf("%s/%s", cdiConfigPath, file.Name())
		var content []byte
		var err error
		if strings.HasSuffix(filePath, ".json") {
			content, err = os.ReadFile(filePath)
			if err != nil {
				logger.Error(err, "read file: %s failed", filePath)
				return nil, err
			}
		} else if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
			content, err = utils.ReadYamlToJson(filePath)
			if err != nil {
				return nil, err
			}
		} else {
			logger.Warn("file: %s is not json or yaml format, skip it", filePath)
		}
		if content == nil {
			continue
		}
		cdiConfig := new(cdiSpec.Spec)
		if err := json.Unmarshal(content, cdiConfig); err != nil {
			logger.Error(err, "json unmarshal config file: %s to strcut failed, content: %s", filePath, string(content))
			return nil, err
		}
		if _, exist := cdiConfigs[spec.ResourceName(cdiConfig.Kind)]; exist {
			err := fmt.Errorf("existence of CDI config files of the same kind: %s is not allowed, check path: %s",
				cdiConfig.Kind, cdiConfigPath)
			logger.Error(err)
			return nil, err
		}
		logger.Info("found cdi config file: %s, kind: %s", filePath, cdiConfig.Kind)
		cdiConfigs[spec.ResourceName(cdiConfig.Kind)] = cdiConfigInfo{
			fileName:  filePath,
			cdiConfig: cdiConfig,
		}
	}
	return cdiConfigs, nil
}
