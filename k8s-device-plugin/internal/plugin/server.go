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

package plugin

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
	cdiapi "tags.cncf.io/container-device-interface/pkg/cdi"

	spec "enflame/k8s-device-plugin/api/config/v1"
	"enflame/k8s-device-plugin/cmd/mps-control-daemon/mps"
	"enflame/k8s-device-plugin/internal/cdi"
	"enflame/k8s-device-plugin/internal/eflib/interfaces"
	"enflame/k8s-device-plugin/internal/informer"
	"enflame/k8s-device-plugin/internal/logger"
	"enflame/k8s-device-plugin/internal/rm"
	"enflame/k8s-device-plugin/internal/types"
)

// Constants for use by the 'volume-mounts' device list strategy
const (
	deviceListAsVolumeMountsHostPath          = "/dev/null"
	deviceListAsVolumeMountsContainerPathRoot = "/var/run/enflame-container-devices"
)

// EnflameDevicePlugin implements the Kubernetes device plugin API
type EnflameDevicePlugin struct {
	rm                   rm.ResourceManager
	config               *spec.Config
	deviceListEnvvar     string
	deviceListStrategies spec.DeviceListStrategies
	socket               string

	cdiHandler          cdi.CDIInterface
	cdiEnabled          bool
	cdiAnnotationPrefix string

	server *grpc.Server
	health chan []*rm.Device
	stop   chan interface{}

	mpsDaemon      *mps.Daemon
	mpsHostRoot    mps.Root
	switchAffinity *PcieSwitchAffinity
	byCard         *ByCardScheduler
	podInformer    *informer.PodInformer
}

// NewEnflameDevicePlugin returns an initialized EnflameDevicePlugin
func NewEnflameDevicePlugin(config *spec.Config, resourceManager rm.ResourceManager, cdiHandler cdi.CDIInterface,
	eflib interfaces.HandleInterface) (*EnflameDevicePlugin, error) {
	// _, name := resourceManager.Resource().Split()
	// pluginName := "enflame-" + name
	pluginPath := filepath.Join(pluginapi.DevicePluginPath, resourceManager.Resource().GenerateSockName())

	var mpsDaemon *mps.Daemon
	var mpsHostRoot mps.Root
	if config.Sharing.SharingStrategy() == spec.SharingStrategyMPS {
		// TODO: It might make sense to pull this logic into a resource manager.
		for _, device := range resourceManager.Devices() {
			if device.IsMigDevice() {
				err := fmt.Errorf("sharing using MPS is not supported for MIG devices")
				logger.Error(err)
				return nil, err
			}
		}
		mpsDaemon = mps.NewDaemon(resourceManager, mps.ContainerRoot)
		mpsHostRoot = mps.Root(*config.Flags.CommandLineFlags.MpsRoot)
	}

	deviceListStrategies, err := spec.NewDeviceListStrategies(*config.Flags.Plugin.DeviceListStrategy)
	if err != nil {
		return nil, err
	}
	podInformer, err := informer.NewPodInformer(string(resourceManager.Resource()))
	if err != nil {
		return nil, err
	}
	if err := podInformer.Start(context.TODO()); err != nil {
		return nil, err
	}
	plugin := EnflameDevicePlugin{
		rm:                   resourceManager,
		config:               config,
		deviceListEnvvar:     types.ENFLAME_VISIBLE_DEVICES,
		deviceListStrategies: deviceListStrategies,
		socket:               pluginPath + ".sock",
		cdiHandler:           cdiHandler,
		cdiEnabled:           deviceListStrategies.IsCDIEnabled(),
		cdiAnnotationPrefix:  *config.Flags.Plugin.CDIAnnotationPrefix,

		mpsDaemon:   mpsDaemon,
		mpsHostRoot: mpsHostRoot,
		podInformer: podInformer,

		// These will be reinitialized every
		// time the plugin server is restarted.
		server: nil,
		health: nil,
		stop:   nil,
	}
	if *config.Flags.Plugin.PcieSwitchAffinity {
		logger.Info("deploy k8s device plugin with pcie switch affinity schedule, resource name: %s", resourceManager.Resource())
		switchAffinity, err := NewPcieSwitchAffinity(eflib, string(resourceManager.Resource()))
		if err != nil {
			return nil, err
		}
		plugin.switchAffinity = switchAffinity
	}
	if config.Flags.Plugin.ByCardScheduling != nil && *config.Flags.Plugin.ByCardScheduling {
		byCard, err := NewByCardScheduler(eflib, string(resourceManager.Resource()), plugin.podInformer,
			derefString(config.Flags.Plugin.SchedulerDeviceAnnotationKey),
			derefString(config.Flags.Plugin.SchedulerAssumedTimeAnnotationKey),
			derefString(config.Flags.Plugin.PodSource))
		if err != nil {
			return nil, err
		}
		plugin.byCard = byCard
	}
	return &plugin, nil
}

func (plugin *EnflameDevicePlugin) initialize() {
	plugin.server = grpc.NewServer([]grpc.ServerOption{}...)
	plugin.health = make(chan []*rm.Device)
	plugin.stop = make(chan interface{})
}

func (plugin *EnflameDevicePlugin) cleanup() {
	close(plugin.stop)
	plugin.server = nil
	plugin.health = nil
	plugin.stop = nil
	if plugin.podInformer != nil {
		plugin.podInformer.Close()
	}
}

// Devices returns the full set of devices associated with the plugin.
func (plugin *EnflameDevicePlugin) Devices() rm.Devices {
	return plugin.rm.Devices()
}

// Start starts the gRPC server, registers the device plugin with the Kubelet,
// and starts the device healthchecks.
func (plugin *EnflameDevicePlugin) Start() error {
	plugin.initialize()

	// Backfill stock pods' card annotations before serving Allocate, so the
	// by-card occupancy/scheduler view includes pods scheduled before by-card was
	// enabled. Best-effort: a failure is logged but does not block the plugin.
	if plugin.config.Flags.Plugin.ReconcileStockPods != nil && *plugin.config.Flags.Plugin.ReconcileStockPods {
		logger.Info("Reconciling stock pods before serving Allocate for resource %s.", plugin.rm.Resource())
		if err := plugin.reconcileStockPods(); err != nil {
			logger.Error(err, "stock-pod reconcile failed (continuing)")
		}
	}

	if err := plugin.waitForMPSDaemon(); err != nil {
		return err
	}

	err := plugin.Serve()
	if err != nil {
		plugin.cleanup()
		return err
	}
	logger.Info("Starting to serve '%s' on %s", plugin.rm.Resource(), plugin.socket)

	err = plugin.Register()
	if err != nil {
		logger.Info("Could not register device plugin: %s", err)
		return errors.Join(err, plugin.Stop())
	}
	logger.Info("Registered device plugin for '%s' with Kubelet", plugin.rm.Resource())

	go plugin.rm.CheckHealth(plugin.stop, plugin.health, plugin.podInformer)
	return nil
}

func (plugin *EnflameDevicePlugin) waitForMPSDaemon() error {
	if plugin.config.Sharing.SharingStrategy() != spec.SharingStrategyMPS {
		return nil
	}
	// TODO: Check the .ready file here.
	// TODO: Have some retry strategy here.
	if err := plugin.mpsDaemon.AssertHealthy(); err != nil {
		return fmt.Errorf("error checking MPS daemon health: %w", err)
	}
	klog.InfoS("MPS daemon is healthy", "resource", plugin.rm.Resource())
	return nil
}

// Stop stops the gRPC server.
func (plugin *EnflameDevicePlugin) Stop() error {
	if plugin == nil || plugin.server == nil {
		return nil
	}
	logger.Info("Stopping to serve '%s' on %s", plugin.rm.Resource(), plugin.socket)
	plugin.server.Stop()
	if err := os.Remove(plugin.socket); err != nil && !os.IsNotExist(err) {
		return err
	}
	plugin.cleanup()
	return nil
}

// Serve starts the gRPC server of the device plugin.
func (plugin *EnflameDevicePlugin) Serve() error {
	os.Remove(plugin.socket)
	sock, err := net.Listen("unix", plugin.socket)
	if err != nil {
		logger.Error(err, "listen %s failed", plugin.socket)
		return err
	}

	pluginapi.RegisterDevicePluginServer(plugin.server, plugin)

	go func() {
		lastCrashTime := time.Now()
		restartCount := 0

		for {
			// quite if it has been restarted too often
			// i.e. if server has crashed more than 5 times and it didn't last more than one hour each time
			if restartCount > 5 {
				// quit
				logger.Error("GRPC server for '%s' has repeatedly crashed recently. Quitting", plugin.rm.Resource())
				os.Exit(1)
			}

			logger.Info("Starting GRPC server for '%s'", plugin.rm.Resource())
			err := plugin.server.Serve(sock)
			if err == nil {
				break
			}

			logger.Error("GRPC server for '%s' crashed", plugin.rm.Resource())

			timeSinceLastCrash := time.Since(lastCrashTime).Seconds()
			lastCrashTime = time.Now()
			if timeSinceLastCrash > 3600 {
				// it has been one hour since the last crash.. reset the count
				// to reflect on the frequency
				restartCount = 0
			} else {
				restartCount++
			}
		}
	}()

	// Wait for server to start by launching a blocking connection
	conn, err := plugin.dial(plugin.socket, 5*time.Second)
	if err != nil {
		logger.Error(err, "dial establishes the gRPC communication with the registered: %s failed", plugin.socket)
		return err
	}
	conn.Close()

	return nil
}

// Register registers the device plugin for the given resourceName with Kubelet.
func (plugin *EnflameDevicePlugin) Register() error {
	conn, err := plugin.dial(pluginapi.KubeletSocket, 5*time.Second)
	if err != nil {
		logger.Error(err, "dial establishes the gRPC communication with the registered: %s failed", pluginapi.KubeletSocket)
		return err
	}
	defer conn.Close()

	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     path.Base(plugin.socket),
		ResourceName: string(plugin.rm.Resource()),
		Options: &pluginapi.DevicePluginOptions{
			GetPreferredAllocationAvailable: true,
		},
	}

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		logger.Error(err, "register plugin: %s to kubelet failed", plugin.rm.Resource())
		return err
	}
	return nil
}

// GetDevicePluginOptions returns the values of the optional settings for this plugin
func (plugin *EnflameDevicePlugin) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	options := &pluginapi.DevicePluginOptions{
		GetPreferredAllocationAvailable: true,
	}
	return options, nil
}

// ListAndWatch lists devices and update that list according to the health status
func (plugin *EnflameDevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	if err := s.Send(&pluginapi.ListAndWatchResponse{Devices: plugin.apiDevices()}); err != nil {
		logger.Error(err, "list watch send gcu device health list to kubelet failed")
		return err
	}

	for {
		select {
		case <-plugin.stop:
			return nil
		case reportList := <-plugin.health:
			for _, d := range reportList {
				if d.Health == pluginapi.Unhealthy {
					d.Health = pluginapi.Healthy
				} else {
					d.Health = pluginapi.Unhealthy
				}
			}
			if err := s.Send(&pluginapi.ListAndWatchResponse{Devices: plugin.apiDevices()}); err != nil {
				logger.Error(err, "list watch send gcu device health list to kubelet failed")
				return err
			}
			logger.Info("list watch send gcu device health list to kubelet success")
		}
	}
}

// GetPreferredAllocation returns the preferred allocation from the set of devices specified in the request
func (plugin *EnflameDevicePlugin) GetPreferredAllocation(ctx context.Context, r *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	response := &pluginapi.PreferredAllocationResponse{}
	for _, req := range r.ContainerRequests {
		var devices []string
		// by-card: steer kubelet towards the scheduler-assigned cards so kubelet
		// bookkeeping (podresources, gcu-exporter, device-in-use) matches the mount.
		if plugin.byCard != nil {
			devices = plugin.byCard.PreferredAllocation(req.AvailableDeviceIDs, int(req.AllocationSize))
		}
		if devices == nil {
			var err error
			devices, err = plugin.rm.GetPreferredAllocation(req.AvailableDeviceIDs, req.MustIncludeDeviceIDs, int(req.AllocationSize))
			if err != nil {
				return nil, fmt.Errorf("error getting list of preferred allocation devices: %v", err)
			}
		}

		resp := &pluginapi.ContainerPreferredAllocationResponse{
			DeviceIDs: devices,
		}

		response.ContainerResponses = append(response.ContainerResponses, resp)
	}
	return response, nil
}

// Allocate which return list of devices.
func (plugin *EnflameDevicePlugin) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	logger.Info("----------Allocate device start----------")
	responses := pluginapi.AllocateResponse{}
	for _, req := range reqs.ContainerRequests {
		logger.Info("kubelet allocate container device ids: %v", req.DevicesIDs)
		if err := plugin.rm.ValidateRequest(req.DevicesIDs); err != nil {
			return nil, fmt.Errorf("invalid allocation request for %q: %w", plugin.rm.Resource(), err)
		}
		response, err := plugin.getAllocateResponse(req.DevicesIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to get allocate response: %v", err)
		}
		responses.ContainerResponses = append(responses.ContainerResponses, response)
	}
	logger.Info("----------Allocate device success----------")
	return &responses, nil
}

func (plugin *EnflameDevicePlugin) getAllocateResponse(requestIds []string) (*pluginapi.ContainerAllocateResponse, error) {
	deviceIDs := plugin.deviceIDsFromAnnotatedDeviceIDs(requestIds)
	var err error
	if *plugin.config.Flags.Plugin.PcieSwitchAffinity && len(plugin.switchAffinity.switchTopoList) > 0 {
		deviceIDs, err = plugin.switchAffinity.AllocateByPcieSwitchAffinity(len(requestIds))
		if err != nil {
			return nil, err
		}
	} else if plugin.byCard != nil {
		// by-card: mount exactly the cards chosen by the external scheduler.
		deviceIDs, err = plugin.byCard.AllocateByCard(requestIds)
		if err != nil {
			return nil, err
		}
	}

	// Create an empty response that will be updated as required below.
	response := &pluginapi.ContainerAllocateResponse{
		Envs: make(map[string]string),
	}
	if plugin.cdiEnabled {
		logger.Info("device list strategy cdi-annotations enabled")
		responseID := uuid.New().String()
		if err := plugin.updateResponseForCDI(response, responseID, deviceIDs...); err != nil {
			return nil, fmt.Errorf("failed to get allocate response for CDI: %v", err)
		}
	}
	if plugin.deviceListStrategies.Includes(spec.DeviceListStrategyEnvvar) {
		logger.Info("device list strategy envvar enabled")
		plugin.updateResponseForDeviceListEnvvar(response, deviceIDs...)
	}
	if plugin.deviceListStrategies.Includes(spec.DeviceListStrategyVolumeMounts) {
		logger.Info("device list strategy volume-mounts enabled")
		plugin.updateResponseForDeviceMounts(response, deviceIDs...)
	}
	if plugin.config.Sharing.SharingStrategy() == spec.SharingStrategyMPS {
		plugin.updateResponseForMPS(response)
	}
	if *plugin.config.Flags.Plugin.PassDeviceSpecs {
		// Use the selected deviceIDs (not the kubelet-picked requestIds) so the
		// mounted /dev/gcuN device nodes match the env var when the plugin
		// overrides the card choice (affinity / by-card). RISK-1.
		response.Devices = append(response.Devices, plugin.apiDeviceSpecs(*plugin.config.Flags.EnflameDriverRoot, deviceIDs)...)
	}
	if *plugin.config.Flags.GDSEnabled {
		response.Envs["ENFLAME_GDS"] = "enabled"
	}
	if *plugin.config.Flags.MOFEDEnabled {
		response.Envs["ENFLAME_MOFED"] = "enabled"
	}
	logger.Info("Allocated devices: %v to container", deviceIDs)
	return response, nil
}

// updateResponseForMPS ensures that the ContainerAllocate response contains the information required to use MPS.
// This includes per-resource pipe and log directories as well as a global daemon-specific shm
// and assumes that an MPS control daemon has already been started.
func (plugin EnflameDevicePlugin) updateResponseForMPS(response *pluginapi.ContainerAllocateResponse) {
	// TODO: We should check that the deviceIDs are shared using MPS.
	response.Envs["TOPS_MPS_PIPE_DIRECTORY"] = plugin.mpsDaemon.PipeDir()

	resourceName := plugin.rm.Resource()
	response.Mounts = append(response.Mounts,
		&pluginapi.Mount{
			ContainerPath: plugin.mpsDaemon.PipeDir(),
			HostPath:      plugin.mpsHostRoot.PipeDir(resourceName),
		},
		&pluginapi.Mount{
			ContainerPath: plugin.mpsDaemon.ShmDir(),
			HostPath:      plugin.mpsHostRoot.ShmDir(resourceName),
		},
	)
}

// updateResponseForCDI updates the specified response for the given device IDs.
// This response contains the annotations required to trigger CDI injection in the container engine or enflame-container-runtime.
func (plugin *EnflameDevicePlugin) updateResponseForCDI(response *pluginapi.ContainerAllocateResponse, responseID string, deviceIDs ...string) error {
	var devices []string
	for _, id := range deviceIDs {
		devices = append(devices, plugin.cdiHandler.QualifiedName(plugin.rm.Resource().ResourceClass(), id))
	}
	if *plugin.config.Flags.GDSEnabled {
		devices = append(devices, plugin.cdiHandler.QualifiedName("gds", "all"))
	}
	if *plugin.config.Flags.MOFEDEnabled {
		devices = append(devices, plugin.cdiHandler.QualifiedName("mofed", "all"))
	}

	if len(devices) == 0 {
		return nil
	}

	if plugin.deviceListStrategies.Includes(spec.DeviceListStrategyCDIAnnotations) {
		annotations, err := plugin.getCDIDeviceAnnotations(responseID, devices...)
		if err != nil {
			return err
		}
		response.Annotations = annotations
	}
	if plugin.deviceListStrategies.Includes(spec.DeviceListStrategyCDICRI) {
		logger.Info("device list strategy cdi-cri enabled")
		for _, device := range devices {
			cdiDevice := pluginapi.CDIDevice{
				Name: device,
			}
			response.CDIDevices = append(response.CDIDevices, &cdiDevice)
		}
	}

	return nil
}

func (plugin *EnflameDevicePlugin) getCDIDeviceAnnotations(id string, devices ...string) (map[string]string, error) {
	annotations, err := cdiapi.UpdateAnnotations(map[string]string{}, "enflame-device-plugin", id, devices)
	if err != nil {
		return nil, fmt.Errorf("failed to add CDI annotations: %v", err)
	}
	logger.Info("cdi device annotations: %v", annotations)

	if plugin.cdiAnnotationPrefix == spec.DefaultCDIAnnotationPrefix {
		return annotations, nil
	}
	logger.Info("use custom cdi prefix: %s", plugin.cdiAnnotationPrefix)

	// update annotations if a custom CDI prefix is configured
	updatedAnnotations := make(map[string]string)
	for k, v := range annotations {
		newKey := plugin.cdiAnnotationPrefix + strings.TrimPrefix(k, spec.DefaultCDIAnnotationPrefix)
		updatedAnnotations[newKey] = v
	}
	logger.Info("cdi device updated annotations: %v", updatedAnnotations)
	return updatedAnnotations, nil
}

// PreStartContainer is unimplemented for this plugin
func (plugin *EnflameDevicePlugin) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

// dial establishes the gRPC communication with the registered device plugin.
func (plugin *EnflameDevicePlugin) dial(unixSocketPath string, timeout time.Duration) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	c, err := grpc.DialContext(ctx, unixSocketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		// TODO: We need to switch to grpc.WithContextDialer.
		//nolint:staticcheck
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (plugin *EnflameDevicePlugin) deviceIDsFromAnnotatedDeviceIDs(ids []string) []string {
	var deviceIDs []string
	if *plugin.config.Flags.Plugin.DeviceIDStrategy == spec.DeviceIDStrategyUUID {
		deviceIDs = rm.AnnotatedIDs(ids).GetIDs()
	}
	if *plugin.config.Flags.Plugin.DeviceIDStrategy == spec.DeviceIDStrategyIndex {
		deviceIDs = plugin.rm.Devices().Subset(ids).GetIndices()
	}
	return deviceIDs
}

func (plugin *EnflameDevicePlugin) apiDevices() []*pluginapi.Device {
	return plugin.rm.Devices().GetPluginDevices()
}

// updateResponseForDeviceListEnvvar sets the environment variable for the requested devices.
func (plugin *EnflameDevicePlugin) updateResponseForDeviceListEnvvar(response *pluginapi.ContainerAllocateResponse, deviceIDs ...string) {
	response.Envs[plugin.deviceListEnvvar] = strings.Join(deviceIDs, ",")
}

// updateResponseForDeviceMounts sets the mounts required to request devices if volume mounts are used.
func (plugin *EnflameDevicePlugin) updateResponseForDeviceMounts(response *pluginapi.ContainerAllocateResponse, deviceIDs ...string) {
	plugin.updateResponseForDeviceListEnvvar(response, deviceListAsVolumeMountsContainerPathRoot)
	for _, id := range deviceIDs {
		mount := &pluginapi.Mount{
			HostPath:      deviceListAsVolumeMountsHostPath,
			ContainerPath: filepath.Join(deviceListAsVolumeMountsContainerPathRoot, id),
		}
		response.Mounts = append(response.Mounts, mount)
	}
}

func (plugin *EnflameDevicePlugin) apiDeviceSpecs(driverRoot string, ids []string) []*pluginapi.DeviceSpec {
	optional := map[string]bool{
		"/dev/gcuctl": true,
	}

	paths := plugin.rm.GetDevicePaths(ids)

	var specs []*pluginapi.DeviceSpec
	for _, p := range paths {
		if optional[p] {
			if _, err := os.Stat(p); err != nil {
				continue
			}
		}
		spec := &pluginapi.DeviceSpec{
			ContainerPath: p,
			HostPath:      filepath.Join(driverRoot, p),
			Permissions:   "rw",
		}
		specs = append(specs, spec)
	}
	logger.Info("pass device spec content: %v", specs)
	return specs
}
