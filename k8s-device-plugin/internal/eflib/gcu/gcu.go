// Copyright (c) 2024, ENFLAME INC.  All rights reserved.

package gcu

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	v1 "enflame/k8s-device-plugin/api/config/v1"
	"enflame/k8s-device-plugin/internal/eflib/interfaces"
	"enflame/k8s-device-plugin/internal/logger"
	"enflame/k8s-device-plugin/internal/types"
	"enflame/k8s-device-plugin/internal/utils"
)

type deviceRecord struct {
	DeviceType      string
	DeviceIDPathMap map[int]string
}

func NewRegisterDeviceLib(config *v1.Config) map[string]interfaces.HandleInterface {
	regDeviceLib := map[string]interfaces.HandleInterface{}
	for deviceType := range config.Resources {
		regDeviceLib[deviceType] = &deviceRecord{DeviceType: deviceType, DeviceIDPathMap: BuildDeviceIDPathMap(deviceType)}
	}
	return regDeviceLib
}

func (rer *deviceRecord) GetDeviceIDPathMap() map[int]string {
	return rer.DeviceIDPathMap
}

func (rer *deviceRecord) DeviceGetHandleByIndex(Index int) interfaces.GcuDeviceInterface {
	return &deviceLib{Index: Index, deviceType: rer.DeviceType}
}

// device interfaces
type deviceLib struct {
	Index      int
	deviceType string
}

func (rer *deviceLib) DeviceType() string {
	return rer.deviceType
}

func (rer *deviceLib) GetUUID() string {
	return fmt.Sprintf("%s-%d", strings.ToUpper(rer.deviceType), rer.Index)
}

func (rer *deviceLib) GetName() string {
	return fmt.Sprintf("%s%d", rer.deviceType, rer.Index)
}

func (rer *deviceLib) GetMinorNumber() (int, error) {
	path := fmt.Sprintf("%s%s%d", types.DevicePath, rer.deviceType, rer.Index)
	dev, err := os.Stat(path)
	if err != nil {
		logger.Error(err, "check device %s failed", path)
		return 0, err
	}

	stat, ok := dev.Sys().(*syscall.Stat_t)
	if !ok {
		err = errors.New("syscall error")
		logger.Error(err)
		return 0, err
	}

	major := stat.Rdev / 256
	minor := stat.Rdev % 256

	logger.Info("getDeviceID: path: %s, major: %d, minor: %d", path, major, minor)
	return int(minor), nil
}

func (rer *deviceLib) GetPaths() ([]string, error) {
	minor, err := rer.GetMinorNumber()
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("%s%s%d", types.DevicePath, rer.deviceType, minor)
	return []string{path}, nil
}

func BuildDeviceIDPathMap(deviceType string) map[int]string {
	deviceMap := make(map[int]string)
	for i := 0; i < types.DEVICE_COUNT_MAX; i++ {
		devPath := fmt.Sprintf("%s%s%d", types.DevicePath, deviceType, i)
		if utils.FileIsExist(devPath) {
			deviceMap[i] = devPath
		}
	}
	logger.Info("found %s device count: %d, detail: %v", deviceType, len(deviceMap), deviceMap)
	return deviceMap
}
