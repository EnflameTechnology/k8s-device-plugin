// Copyright (c) 2024, ENFLAME INC.  All rights reserved.

package tests

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"enflame/k8s-device-plugin/internal/logger"
	"enflame/k8s-device-plugin/internal/utils"
)

// clear test env before eixt test case
func ClearFakeDevice(fakeDeviceList []string) error {
	logger.Info("test code: clear test environment start")
	for _, devPath := range fakeDeviceList {
		err := os.Remove(devPath)
		if err != nil && !os.IsNotExist(err) {
			logger.Error(err, "test code: clear fake device: %s failed, please delete it manually before re-running the test case", devPath)
			return err
		}
		logger.Info("test code: clear fake device: %s success", devPath)
	}
	logger.Info("test code: clear test environment finish")
	return nil
}

// create fake device for test
func CreateFakeDevice(devType string, count int) ([]string, error) {
	logger.Info("test code: init test environment start")
	fakeDeviceList := []string{}
	realDeviceList, err := getRealDeviceList(devType)
	if err != nil {
		return fakeDeviceList, err
	}
	if len(realDeviceList) >= count {
		logger.Info("test code: expect to create %d fake devices, but found real device list: %v, so need not create fake devices",
			count, realDeviceList)
		return fakeDeviceList, nil
	}
	logger.Info("test code: expect to create %d fake devices, found real device list: %v, will create %d fake devices",
		count, realDeviceList, count-len(realDeviceList))
	fakeIndex := 0
	for len(fakeDeviceList) != count-len(realDeviceList) {
		fakeDevicePath := fmt.Sprintf("/dev/%s%d", devType, fakeIndex)
		if !utils.FileIsExist(fakeDevicePath) {
			if _, err := os.Create(fakeDevicePath); err != nil {
				logger.Error(err, "test code: create fake device: %s failed", fakeDevicePath)
				return fakeDeviceList, err
			}
			fakeDeviceList = append(fakeDeviceList, fakeDevicePath)
			logger.Info("test code: create fake device: %s success", fakeDevicePath)
		}
		fakeIndex += 1
	}

	if len(realDeviceList) == 0 {
		gcuctlPath := "/dev/gcuctl"
		_, err = os.Create(gcuctlPath)
		if err != nil {
			logger.Error(err, "test code: create fake device: %s failed", gcuctlPath)
			return fakeDeviceList, err
		}
		fakeDeviceList = append(fakeDeviceList, gcuctlPath)
		logger.Info("test code: create fake device: %s success", gcuctlPath)
	}
	logger.Info("test code: init test environment finish")
	return fakeDeviceList, nil
}

func getRealDeviceList(devType string) ([]string, error) {
	realDeviceList := []string{}
	files, err := os.ReadDir("/dev")
	if err != nil {
		logger.Error(err, "read directory: /dev failed")
		return realDeviceList, err
	}
	for _, file := range files {
		fileName := file.Name()
		if file.IsDir() || !strings.HasPrefix(fileName, devType) {
			continue
		}
		if _, err := strconv.Atoi(fileName[len(devType):]); err == nil {
			realDeviceList = append(realDeviceList, "/dev/"+fileName)
		}
	}
	return realDeviceList, nil
}
