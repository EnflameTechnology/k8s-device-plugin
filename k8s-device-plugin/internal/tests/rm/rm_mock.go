// Copyright (c) 2024, ENFLAME INC.  All rights reserved.

package rm

import (
	spec "enflame/k8s-device-plugin/api/config/v1"
	"enflame/k8s-device-plugin/internal/informer"
	"enflame/k8s-device-plugin/internal/rm"
)

type ResourceManager struct{ ResourceName spec.ResourceName }

func (rer *ResourceManager) Resource() spec.ResourceName                                      { return rer.ResourceName }
func (rer *ResourceManager) Devices() rm.Devices                                              { return map[string]*rm.Device{} }
func (rer *ResourceManager) GetDevicePaths([]string) []string                                 { return []string{} }
func (rer *ResourceManager) CheckHealth(stop <-chan interface{}, reportList chan<- []*rm.Device, podInformer *informer.PodInformer) {}
func (rer *ResourceManager) ValidateRequest(rm.AnnotatedIDs) error                            { return nil }
func (rer *ResourceManager) GetPreferredAllocation(available, required []string, size int) ([]string, error) {
	return []string{}, nil
}
