// Copyright (c) 2024, ENFLAME INC.  All rights reserved.

package types

// common
const (
	COMPONENT_NAME = "k8s-device-plugin"

	TimeFormat = "2006-01-02 15:04:05"

	// for log
	LOGDIR   = "/var/log/enflame/k8s-device-plugin/"
	LOGPATH  = LOGDIR + "k8s-device-plugin.log"
	LOGINFO  = "INFO"
	LOGERROR = "ERROR"
	LOGWARN  = "WARNING"
	LOGDEBUG = "DEBUG"

	BusIDPrefix     = "/sys/module/enflame/drivers/pci:enflame/busid"
	DevicePciPrefix = "/sys/bus/pci/devices"

	// default device info
	DevicePath       = "/dev/"
	DEVICE_COUNT_MAX = 128
	PCIDevicePath    = "/sys/bus/pci/devices"

	PodResourceSocket = "unix:///var/lib/kubelet/pod-resources/kubelet.sock"
)

// flag name
const (
	MigStrategy        = "mig-strategy"
	PassDeviceSpecs    = "pass-device-specs"
	EnflameDriverRoot  = "enflame-driver-root"
	FailOnInitError    = "fail-on-init-error"
	DeviceListStrategy = "device-list-strategy"
	DeviceIdStrategy   = "device-id-strategy"
	GdsEnabled         = "gds-enabled"
	MofedEnabled       = "mofed-enabled"
	ConfigFile         = "config-file"
)

// values of flag: DeviceListStrategy
const (
	DeviceListStrategyEnvvar       = "envvar"
	DeviceListStrategyVolumeMounts = "volume-mounts"
)

// values of flag: DeviceIdStrategy
const (
	DeviceIDStrategyUUID  = "uuid"
	DeviceIDStrategyIndex = "index"
)

// env name
const (
	EnvEnflameDriverRoot  = "ENFLAME_DRIVER_ROOT"
	EnvPassDeviceSpecs    = "PASS_DEVICE_SPECS"
	EnvFailOnInitError    = "FAIL_ON_INIT_ERROR"
	EnvDeviceListStrategy = "DEVICE_LIST_STRATEGY"
	EnvDeviceIdStrategy   = "DEVICE_ID_STRATEGY"
	EnvGdsEnabled         = "GDS_ENABLED"
	EnvMofedEnabled       = "MOFED_ENABLED"
	EnvConfigFile         = "CONFIG_FILE"

	ENFLAME_VISIBLE_DEVICES = "ENFLAME_VISIBLE_DEVICES"
)

// health check
type DeviceState uint32

const (
	HEALTH_PASS  DeviceState = 0 // Success
	HEALTH_WARN  DeviceState = 1 // WARN
	HEALTH_DEBUG DeviceState = 2 // DEBUG
	HEALTH_FAIL  DeviceState = 3 // FAIL

	EnvDisableHealthChecks = "DP_DISABLE_HEALTHCHECKS"
	AllHealthChecks        = "all"

	DEVICE_UNAVAILABLE = "0"
)
