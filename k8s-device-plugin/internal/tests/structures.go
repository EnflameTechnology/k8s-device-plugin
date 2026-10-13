// Copyright (c) 2024, ENFLAME INC.  All rights reserved.

// This file provides various interface implementations for testing

package tests

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	v1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"
	restclient "k8s.io/client-go/rest"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

type RegistrationClient struct{}

func (rer *RegistrationClient) Register(ctx context.Context, in *pluginapi.RegisterRequest, opts ...grpc.CallOption) (*pluginapi.Empty, error) {
	return &pluginapi.Empty{}, nil
}

type ListAndWatchServer struct{}

func (rer *ListAndWatchServer) Send(*pluginapi.ListAndWatchResponse) error { return nil }
func (rer *ListAndWatchServer) Context() context.Context                   { return nil }
func (rer *ListAndWatchServer) RecvMsg(m any) error                        { return nil }
func (rer *ListAndWatchServer) SendHeader(metadata.MD) error               { return nil }
func (rer *ListAndWatchServer) SendMsg(m any) error                        { return nil }
func (rer *ListAndWatchServer) SetHeader(metadata.MD) error                { return nil }
func (rer *ListAndWatchServer) SetTrailer(metadata.MD)                     {}

type MockPods struct{}

func (rer *MockPods) Create(ctx context.Context, pod *v1.Pod, opts metav1.CreateOptions) (*v1.Pod, error) {
	return nil, nil
}

func (rer *MockPods) Update(ctx context.Context, pod *v1.Pod, opts metav1.UpdateOptions) (*v1.Pod, error) {
	return nil, nil
}

func (rer *MockPods) UpdateStatus(ctx context.Context, pod *v1.Pod, opts metav1.UpdateOptions) (*v1.Pod, error) {
	return nil, nil
}

func (rer *MockPods) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return nil
}

func (rer *MockPods) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return nil
}

func (rer *MockPods) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Pod, error) {
	return nil, nil
}

func (rer *MockPods) List(ctx context.Context, opts metav1.ListOptions) (*v1.PodList, error) {
	return nil, nil
}

func (rer *MockPods) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return nil, nil
}

func (rer *MockPods) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions,
	subresources ...string) (result *v1.Pod, err error) {
	return nil, nil
}

func (rer *MockPods) Apply(ctx context.Context, pod *corev1.PodApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Pod, err error) {
	return nil, nil
}
func (rer *MockPods) ApplyStatus(ctx context.Context, pod *corev1.PodApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Pod, err error) {
	return nil, nil
}

func (rer *MockPods) UpdateEphemeralContainers(ctx context.Context, podName string, pod *v1.Pod, opts metav1.UpdateOptions) (*v1.Pod, error) {
	return nil, nil
}

func (rer *MockPods) Bind(ctx context.Context, binding *v1.Binding, opts metav1.CreateOptions) error {
	return nil
}

func (rer *MockPods) Evict(ctx context.Context, eviction *policyv1beta1.Eviction) error {
	return nil
}

func (rer *MockPods) EvictV1(ctx context.Context, eviction *policyv1.Eviction) error {
	return nil
}

func (rer *MockPods) EvictV1beta1(ctx context.Context, eviction *policyv1beta1.Eviction) error {
	return nil
}
func (rer *MockPods) GetLogs(name string, opts *v1.PodLogOptions) *restclient.Request {
	return nil
}
func (rer *MockPods) ProxyGet(scheme, name, port, path string, params map[string]string) restclient.ResponseWrapper {
	return nil
}
