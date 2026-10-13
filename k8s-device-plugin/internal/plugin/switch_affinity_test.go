// Copyright (c) 2024, ENFLAME INC.  All rights reserved.
// EF_SUPPORT

package plugin

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"

	. "github.com/agiledragon/gomonkey"
	. "github.com/smartystreets/goconvey/convey"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	coretypev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	confv1 "enflame/k8s-device-plugin/api/config/v1"
	"enflame/k8s-device-plugin/internal/eflib/gcu"
	"enflame/k8s-device-plugin/internal/tests"
)

func TestMakeSwitchTopoList(t *testing.T) {
	Convey("test MakeSwitchTopoList\n", t, func() {
		fakeDeviceCount, deviceType := 8, "gcu"
		fakeDeviceList, err := tests.CreateFakeDevice(deviceType, fakeDeviceCount)
		defer func() {
			So(tests.ClearFakeDevice(fakeDeviceList), ShouldEqual, nil)
		}()
		So(err, ShouldEqual, nil)
		Convey("1. test MakeSwitchTopoList success\n", func() {
			defineOutputs := []OutputCell{
				{Values: Params{[]byte("0000:11:00.0"), nil}},
				{Values: Params{[]byte("0000:12:00.0"), nil}},
				{Values: Params{[]byte("0000:15:00.0"), nil}},
				{Values: Params{[]byte("0000:19:00.0"), nil}},
				{Values: Params{[]byte("0000:9c:00.0"), nil}},
				{Values: Params{[]byte("0000:9d:00.0"), nil}},
				{Values: Params{[]byte("0000:a0:00.0"), nil}},
				{Values: Params{[]byte("0000:a4:00.0"), nil}},
			}
			patchesReadFile := ApplyFuncSeq(os.ReadFile, defineOutputs)
			defer patchesReadFile.Reset()

			outputs := []OutputCell{
				{Values: Params{[]byte("/sys/devices/pci0000:0c/0000:0c:02.0/0000:0d:00.0/0000:0e:00.0/0000:0f:00.0/0000:10:00.0/0000:11:00.0"), nil}},
				{Values: Params{[]byte("/sys/devices/pci0000:0c/0000:0c:02.0/0000:0d:00.0/0000:0e:00.0/0000:0f:00.0/0000:10:10.0/0000:12:00.0"), nil}},
				{Values: Params{[]byte("/sys/devices/pci0000:0c/0000:0c:02.0/0000:0d:00.0/0000:0e:04.0/0000:13:00.0/0000:14:00.0/0000:15:00.0"), nil}},
				{Values: Params{[]byte("/sys/devices/pci0000:0c/0000:0c:02.0/0000:0d:00.0/0000:0e:08.0/0000:16:00.0/0000:17:10.0/0000:19:00.0"), nil}},
				{Values: Params{[]byte("/sys/devices/pci0000:97/0000:97:02.0/0000:98:00.0/0000:99:00.0/0000:9a:00.0/0000:9b:00.0/0000:9c:00.0"), nil}},
				{Values: Params{[]byte("/sys/devices/pci0000:97/0000:97:02.0/0000:98:00.0/0000:99:00.0/0000:9a:00.0/0000:9b:10.0/0000:9d:00.0"), nil}},
				{Values: Params{[]byte("/sys/devices/pci0000:97/0000:97:02.0/0000:98:00.0/0000:99:04.0/0000:9e:00.0/0000:9f:00.0/0000:a0:00.0"), nil}},
				{Values: Params{[]byte("/sys/devices/pci0000:97/0000:97:02.0/0000:98:00.0/0000:99:08.0/0000:a2:00.0/0000:a3:10.0/0000:a4:00.0"), nil}},
			}
			patcheAssume := ApplyMethodSeq(reflect.TypeOf(new(exec.Cmd)), "Output", outputs)
			defer patcheAssume.Reset()

			config := &confv1.Config{
				Resources: confv1.Resources{
					"gcu": []confv1.Resource{
						{
							Pattern: "*",
							Name:    "enflame.com/gcu",
						},
						{
							Pattern: "*",
							Name:    "enflame-tech.com/gcu",
						},
					},
					"vgcu": []confv1.Resource{
						{
							Pattern: "*",
							Name:    "enflame.com/vgcu",
						},
					},
				},
			}
			switchTopoList, err := MakeSwitchTopoList(gcu.NewRegisterDeviceLib(config)["gcu"])
			So(err, ShouldEqual, nil)
			So(len(switchTopoList), ShouldEqual, 28)
			So(switchTopoList[0].Score, ShouldEqual, 5)
			So(switchTopoList[0].Device1.Name, ShouldEqual, "gcu4")
			So(switchTopoList[0].Device2.Name, ShouldEqual, "gcu5")

			So(switchTopoList[1].Score, ShouldEqual, 5)
			So(switchTopoList[1].Device1.Name, ShouldEqual, "gcu0")
			So(switchTopoList[1].Device2.Name, ShouldEqual, "gcu1")
		})
	})
}

func TestAllocateByPcieSwitchAffinity(t *testing.T) {
	Convey("test AllocateByPcieSwitchAffinity\n", t, func() {
		// create fake gcu device
		fakeDeviceCount, deviceType := 8, "gcu"
		fakeDeviceList, err := tests.CreateFakeDevice(deviceType, fakeDeviceCount)
		defer func() {
			So(tests.ClearFakeDevice(fakeDeviceList), ShouldEqual, nil)
		}()
		So(err, ShouldEqual, nil)

		// patch kube client
		patcheBuildConfig := ApplyFunc(clientcmd.BuildConfigFromFlags, func(_, _ string) (*rest.Config, error) {
			return &rest.Config{}, nil
		})
		defer patcheBuildConfig.Reset()

		patcheGetClient := ApplyFunc(kubernetes.NewForConfig, func(_ *rest.Config) (*kubernetes.Clientset, error) {
			return &kubernetes.Clientset{}, nil
		})
		defer patcheGetClient.Reset()

		// set env node name
		err = os.Setenv("NODE_NAME", "node-1")
		So(err, ShouldEqual, nil)
		defer func() {
			err := os.Unsetenv("NODE_NAME")
			So(err, ShouldEqual, nil)
		}()

		// build switch topo list
		defineOutputs := []OutputCell{
			{Values: Params{[]byte("0000:11:00.0"), nil}},
			{Values: Params{[]byte("0000:12:00.0"), nil}},
			{Values: Params{[]byte("0000:15:00.0"), nil}},
			{Values: Params{[]byte("0000:19:00.0"), nil}},
			{Values: Params{[]byte("0000:9c:00.0"), nil}},
			{Values: Params{[]byte("0000:9d:00.0"), nil}},
			{Values: Params{[]byte("0000:a0:00.0"), nil}},
			{Values: Params{[]byte("0000:a4:00.0"), nil}},
		}
		patchesReadFile := ApplyFuncSeq(os.ReadFile, defineOutputs)
		defer patchesReadFile.Reset()

		outputs := []OutputCell{
			{Values: Params{[]byte("/sys/devices/pci0000:0c/0000:0c:02.0/0000:0d:00.0/0000:0e:00.0/0000:0f:00.0/0000:10:00.0/0000:11:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:0c/0000:0c:02.0/0000:0d:00.0/0000:0e:00.0/0000:0f:00.0/0000:10:10.0/0000:12:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:0c/0000:0c:02.0/0000:0d:00.0/0000:0e:04.0/0000:13:00.0/0000:14:00.0/0000:15:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:0c/0000:0c:02.0/0000:0d:00.0/0000:0e:08.0/0000:16:00.0/0000:17:10.0/0000:19:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:97/0000:97:02.0/0000:98:00.0/0000:99:00.0/0000:9a:00.0/0000:9b:00.0/0000:9c:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:97/0000:97:02.0/0000:98:00.0/0000:99:00.0/0000:9a:00.0/0000:9b:10.0/0000:9d:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:97/0000:97:02.0/0000:98:00.0/0000:99:04.0/0000:9e:00.0/0000:9f:00.0/0000:a0:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:97/0000:97:02.0/0000:98:00.0/0000:99:08.0/0000:a2:00.0/0000:a3:10.0/0000:a4:00.0"), nil}},
		}
		patcheAssume := ApplyMethodSeq(reflect.TypeOf(new(exec.Cmd)), "Output", outputs)
		defer patcheAssume.Reset()

		// patch pod list interface
		patcheCoreV1 := ApplyMethod(reflect.TypeOf(new(kubernetes.Clientset)), "CoreV1",
			func(_ *kubernetes.Clientset) coretypev1.CoreV1Interface {
				return &coretypev1.CoreV1Client{}
			})
		defer patcheCoreV1.Reset()

		patcheCoreV1Pods := ApplyMethod(reflect.TypeOf(new(coretypev1.CoreV1Client)), "Pods",
			func(_ *coretypev1.CoreV1Client, _ string) coretypev1.PodInterface {
				return &tests.MockPods{}
			})
		defer patcheCoreV1Pods.Reset()

		config := &confv1.Config{
			Resources: confv1.Resources{
				"gcu": []confv1.Resource{
					{
						Pattern: "*",
						Name:    "enflame.com/gcu",
					},
					{
						Pattern: "*",
						Name:    "enflame-tech.com/gcu",
					},
				},
				"vgcu": []confv1.Resource{
					{
						Pattern: "*",
						Name:    "enflame.com/vgcu",
					},
				},
			},
		}
		psa, err := NewPcieSwitchAffinity(gcu.NewRegisterDeviceLib(config)["gcu"], "enflame.com/gcu")
		So(err, ShouldEqual, nil)

		Convey("1. test create pod1, request 2 GCU success\n", func() {
			// candidate pod
			pod1 := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod-1",
					Namespace:         "kube-system",
					UID:               "pod-1-uid",
					CreationTimestamp: metav1.Now(),
				},
				Spec: v1.PodSpec{
					NodeName: "node-1",
					Containers: []v1.Container{
						{
							Name: "pod1-c1",
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"enflame.com/gcu": *resource.NewQuantity(2, ""),
								},
							}},
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
				},
			}
			outputsListPod := []OutputCell{
				// mock candidate pods
				{Values: Params{nil, fmt.Errorf("apiserver list pods failed")}},
				{Values: Params{&v1.PodList{Items: []v1.Pod{pod1}}, nil}},
				// mock list allocated pods
				{Values: Params{&v1.PodList{Items: []v1.Pod{}}, nil}},
			}
			patchePodsList := ApplyMethodSeq(reflect.TypeOf(new(tests.MockPods)), "List", outputsListPod)
			defer patchePodsList.Reset()

			clusterPod1 := pod1
			clusterPod1.Annotations = map[string]string{AssignedContainers: `{"pod1-c1": ["0", "1"]}`}
			patcheGetPod := ApplyMethod(reflect.TypeOf(new(tests.MockPods)), "Get",
				func(_ *tests.MockPods, _ context.Context, _ string, _ metav1.GetOptions) (*v1.Pod, error) {
					return &clusterPod1, nil
				})
			defer patcheGetPod.Reset()

			res, err := psa.AllocateByPcieSwitchAffinity(2)
			So(err, ShouldEqual, nil)
			So(res, ShouldResemble, []string{"0", "1"})
		})

		Convey("2. test create pod2, request 2 GCU success\n", func() {
			// candidate pod will be seleted
			pod2 := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod-2",
					Namespace:         "kube-system",
					UID:               "pod-2-uid",
					CreationTimestamp: metav1.Time{Time: time.Unix(time.Now().Unix(), 0)},
				},
				Spec: v1.PodSpec{
					NodeName: "node-1",
					Containers: []v1.Container{
						{
							Name: "pod2-c1",
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"enflame.com/gcu": *resource.NewQuantity(2, ""),
								},
							},
						},
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
				},
			}
			// candidate pod other node cann not be selected
			pod3 := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod-3",
					Namespace:         "kube-system",
					UID:               "pod-3-uid",
					CreationTimestamp: metav1.Time{Time: time.Unix(time.Now().Unix(), 0)},
				},
				Spec: v1.PodSpec{
					NodeName: "node-2",
					Containers: []v1.Container{
						{
							Name: "pod3-c1",
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"enflame.com/gcu": *resource.NewQuantity(2, ""),
								},
							},
						},
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
				},
			}

			// allocatedPod1 already allocated gcu0 and gcu1
			allocatedPod1 := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "allocatedPod1",
					Namespace:         "kube-system",
					UID:               "allocatedPod1-uid",
					CreationTimestamp: metav1.Time{Time: time.Unix(time.Now().Unix(), 0)},
					Annotations:       map[string]string{AssignedContainers: `{"allocatedPod1-c1": ["0", "1"]}`},
				},
				Spec: v1.PodSpec{
					NodeName: "node-1",
					Containers: []v1.Container{
						{
							Name: "allocatedPod1-c1",
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"enflame.com/gcu": *resource.NewQuantity(2, ""),
								},
							},
						},
					},
				},
			}
			// allocatedPod2 already allocated gcu4 and gcu5, but it's node is node-2
			allocatedPod2 := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "allocatedPod2",
					Namespace:         "kube-system",
					UID:               "allocatedPod2-uid",
					CreationTimestamp: metav1.Time{Time: time.Unix(time.Now().Unix(), 0)},
					Annotations:       map[string]string{AssignedContainers: `{"allocatedPod1-c1": ["4", "5"]}`},
				},
				Spec: v1.PodSpec{
					NodeName: "node-2",
					Containers: []v1.Container{
						{
							Name: "allocatedPod2-c1",
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"enflame.com/gcu": *resource.NewQuantity(2, ""),
								},
							},
						},
					},
				},
			}
			outputsListPod := []OutputCell{
				// mock candidate pods
				{Values: Params{&v1.PodList{Items: []v1.Pod{pod2, pod3}}, nil}},
				// mock list allocated pods
				{Values: Params{&v1.PodList{Items: []v1.Pod{allocatedPod1, allocatedPod2}}, nil}},
			}
			patchePodsList := ApplyMethodSeq(reflect.TypeOf(new(tests.MockPods)), "List", outputsListPod)
			defer patchePodsList.Reset()

			clusterPod2 := pod2
			clusterPod2.Annotations = map[string]string{AssignedContainers: `{"pod2-c1": ["4", "5"]}`}
			patcheGetPod := ApplyMethod(reflect.TypeOf(new(tests.MockPods)), "Get",
				func(_ *tests.MockPods, _ context.Context, _ string, _ metav1.GetOptions) (*v1.Pod, error) {
					return &clusterPod2, nil
				})
			defer patcheGetPod.Reset()

			patcheStat := ApplyFunc(os.Stat, func(_ string) (fs.FileInfo, error) {
				return nil, nil
			})
			defer patcheStat.Reset()

			res, err := psa.AllocateByPcieSwitchAffinity(2)
			So(err, ShouldEqual, nil)
			So(res, ShouldResemble, []string{"4", "5"})
		})

		Convey("3. test create pod1, request 1 GCU success\n", func() {
			// candidate pod will be seleted
			pod1 := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod-1",
					Namespace:         "kube-system",
					UID:               "pod-1-uid",
					CreationTimestamp: metav1.Time{Time: time.Unix(time.Now().Unix(), 0)},
				},
				Spec: v1.PodSpec{
					NodeName: "node-1",
					Containers: []v1.Container{
						{
							Name: "pod1-c1",
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"enflame.com/gcu": *resource.NewQuantity(1, ""),
								},
							},
						},
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
				},
			}
			// pod2 create time later, will not be select
			pod2 := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod-2",
					Namespace:         "kube-system",
					UID:               "pod-2-uid",
					CreationTimestamp: metav1.Time{Time: time.Unix(time.Now().Unix()+1, 0)},
				},
				Spec: v1.PodSpec{
					NodeName: "node-1",
					Containers: []v1.Container{
						{
							Name: "pod2-c1",
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"enflame.com/gcu": *resource.NewQuantity(1, ""),
								},
							},
						},
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
				},
			}
			outputsListPod := []OutputCell{
				// mock candidate pods
				{Values: Params{&v1.PodList{Items: []v1.Pod{pod2, pod1}}, nil}},
				// mock list allocated pods
				{Values: Params{&v1.PodList{Items: []v1.Pod{}}, nil}},
			}
			patchePodsList := ApplyMethodSeq(reflect.TypeOf(new(tests.MockPods)), "List", outputsListPod)
			defer patchePodsList.Reset()

			clusterPod1 := pod1
			clusterPod1.Annotations = map[string]string{AssignedContainers: `{"pod1-c1": ["2"]}`}
			patcheGetPod := ApplyMethod(reflect.TypeOf(new(tests.MockPods)), "Get",
				func(_ *tests.MockPods, _ context.Context, _ string, _ metav1.GetOptions) (*v1.Pod, error) {
					return &clusterPod1, nil
				})
			defer patcheGetPod.Reset()

			patcheStat := ApplyFunc(os.Stat, func(_ string) (fs.FileInfo, error) {
				return nil, nil
			})
			defer patcheStat.Reset()

			res, err := psa.AllocateByPcieSwitchAffinity(1)
			So(err, ShouldEqual, nil)
			So(res, ShouldResemble, []string{"2"})
		})

		Convey("4. test create pod, c1 request 1 GCU and has been allocated, c2 request 4 GCU, c3 request 1 GCU success\n", func() {
			// candidate pod will be seleted
			pod1 := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod-1",
					Namespace:         "kube-system",
					UID:               "pod-1-uid",
					CreationTimestamp: metav1.Time{Time: time.Unix(time.Now().Unix(), 0)},
					Annotations:       map[string]string{AssignedContainers: `{"pod1-c1": ["2"]}`},
				},
				Spec: v1.PodSpec{
					NodeName: "node-1",
					Containers: []v1.Container{
						{
							Name: "pod1-c1",
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"enflame.com/gcu": *resource.NewQuantity(1, ""),
								},
							},
						},
						{
							Name: "pod1-c2",
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"enflame.com/gcu": *resource.NewQuantity(4, ""),
								},
							},
						},
						{
							Name: "pod1-c3",
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"enflame.com/gcu": *resource.NewQuantity(1, ""),
								},
							},
						},
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
				},
			}
			outputsListPod := []OutputCell{
				// mock candidate pods
				{Values: Params{&v1.PodList{Items: []v1.Pod{pod1}}, nil}},
				// mock current node pod list to select allocated pods
				{Values: Params{&v1.PodList{Items: []v1.Pod{pod1}}, nil}},
			}
			patchePodsList := ApplyMethodSeq(reflect.TypeOf(new(tests.MockPods)), "List", outputsListPod)
			defer patchePodsList.Reset()

			clusterPod1 := pod1
			clusterPod1.Annotations = map[string]string{AssignedContainers: `{"pod1-c1": ["2"], "pod1-c3": ["3"]}`}
			patcheGetPod := ApplyMethod(reflect.TypeOf(new(tests.MockPods)), "Get",
				func(_ *tests.MockPods, _ context.Context, _ string, _ metav1.GetOptions) (*v1.Pod, error) {
					return &clusterPod1, nil
				})
			defer patcheGetPod.Reset()

			patcheStat := ApplyFunc(os.Stat, func(_ string) (fs.FileInfo, error) {
				return nil, nil
			})
			defer patcheStat.Reset()

			res, err := psa.AllocateByPcieSwitchAffinity(1)
			So(err, ShouldEqual, nil)
			So(res, ShouldResemble, []string{"3"})
		})
	})
}

func TestAllocateByPcieSwitchAffinity16Card(t *testing.T) {
	Convey("test AllocateByPcieSwitchAffinity16Card\n", t, func() {
		// create fake gcu device
		fakeDeviceCount, deviceType := 16, "gcu"
		fakeDeviceList, err := tests.CreateFakeDevice(deviceType, fakeDeviceCount)
		defer func() {
			So(tests.ClearFakeDevice(fakeDeviceList), ShouldEqual, nil)
		}()
		So(err, ShouldEqual, nil)

		// patch kube client
		patcheBuildConfig := ApplyFunc(clientcmd.BuildConfigFromFlags, func(_, _ string) (*rest.Config, error) {
			return &rest.Config{}, nil
		})
		defer patcheBuildConfig.Reset()

		patcheGetClient := ApplyFunc(kubernetes.NewForConfig, func(_ *rest.Config) (*kubernetes.Clientset, error) {
			return &kubernetes.Clientset{}, nil
		})
		defer patcheGetClient.Reset()

		// set env node name
		err = os.Setenv("NODE_NAME", "node-1")
		So(err, ShouldEqual, nil)
		defer func() {
			err := os.Unsetenv("NODE_NAME")
			So(err, ShouldEqual, nil)
		}()

		defineOutputs := []OutputCell{
			{Values: Params{[]byte("0000:11:00.0"), nil}},
			{Values: Params{[]byte("0000:12:00.0"), nil}},
			{Values: Params{[]byte("0000:15:00.0"), nil}},
			{Values: Params{[]byte("0000:19:00.0"), nil}},
			{Values: Params{[]byte("0000:9c:00.0"), nil}},
			{Values: Params{[]byte("0000:9d:00.0"), nil}},
			{Values: Params{[]byte("0000:a0:00.0"), nil}},
			{Values: Params{[]byte("0000:a4:00.0"), nil}},

			{Values: Params{[]byte("0000:3d:00.0"), nil}},
			{Values: Params{[]byte("0000:3c:00.0"), nil}},
			{Values: Params{[]byte("0000:45:00.0"), nil}},
			{Values: Params{[]byte("0000:6t:00.0"), nil}},
			{Values: Params{[]byte("0000:21:00.0"), nil}},
			{Values: Params{[]byte("0000:sd:00.0"), nil}},
			{Values: Params{[]byte("0000:fd:00.0"), nil}},
			{Values: Params{[]byte("0000:d4:00.0"), nil}},
		}
		patchesReadFile := ApplyFuncSeq(os.ReadFile, defineOutputs)
		defer patchesReadFile.Reset()

		outputs := []OutputCell{
			{Values: Params{[]byte("/sys/devices/pci0000:0c/0000:0c:02.0/0000:0d:00.0/0000:0e:00.0/0000:0f:00.0/0000:10:00.0/0000:11:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:0c/0000:0c:02.0/0000:0d:00.0/0000:0e:00.0/0000:0f:00.0/0000:10:10.0/0000:12:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:0c/0000:0c:02.0/0000:0d:00.0/0000:0e:04.0/0000:13:00.0/0000:14:00.0/0000:15:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:0c/0000:0c:02.0/0000:0d:00.0/0000:0e:08.0/0000:16:00.0/0000:17:10.0/0000:19:00.0"), nil}},

			{Values: Params{[]byte("/sys/devices/pci0000:97/0000:97:02.0/0000:98:00.0/0000:99:00.0/0000:9a:00.0/0000:9b:00.0/0000:9c:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:97/0000:97:02.0/0000:98:00.0/0000:99:00.0/0000:9a:00.0/0000:9b:10.0/0000:9d:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:97/0000:97:02.0/0000:98:00.0/0000:99:04.0/0000:9e:00.0/0000:9f:00.0/0000:a0:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:97/0000:97:02.0/0000:98:00.0/0000:99:08.0/0000:a2:00.0/0000:a3:10.0/0000:a4:00.0"), nil}},

			{Values: Params{[]byte("/sys/devices/pci0000:k4/0000:k4:02.0/0000:0f:00.0/0000:as:05.0/0000:gr:00.0/0000:11:01.0/0000:3d:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:k4/0000:k4:02.0/0000:0f:00.0/0000:as:05.0/0000:gr:00.0/0000:11:02.0/0000:3c:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:k4/0000:k4:02.0/0000:0f:00.0/0000:as:05.0/0000:gr:00.0/0000:11:03.0/0000:45:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:k4/0000:k4:02.0/0000:0f:00.0/0000:as:07.0/0000:1g:00.0/0000:17:30.0/0000:6t:00.0"), nil}},

			{Values: Params{[]byte("/sys/devices/pci0000:sy/0000:sy:02.0/0000:67:00.0/0000:9f:05.0/0000:3e:00.0/0000:fb:10.0/0000:21:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:sy/0000:sy:02.0/0000:67:00.0/0000:49:00.0/0000:wf:00.0/0000:7j:10.0/0000:sd:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:sy/0000:sy:02.0/0000:67:00.0/0000:9f:05.0/0000:3e:00.0/0000:ff:40.0/0000:fd:00.0"), nil}},
			{Values: Params{[]byte("/sys/devices/pci0000:sy/0000:sy:02.0/0000:67:00.0/0000:49:08.0/0000:gh:00.0/0000:ty:10.0/0000:d4:00.0"), nil}},
		}
		patcheAssume := ApplyMethodSeq(reflect.TypeOf(new(exec.Cmd)), "Output", outputs)
		defer patcheAssume.Reset()

		// patch pod list interface
		patcheCoreV1 := ApplyMethod(reflect.TypeOf(new(kubernetes.Clientset)), "CoreV1",
			func(_ *kubernetes.Clientset) coretypev1.CoreV1Interface {
				return &coretypev1.CoreV1Client{}
			})
		defer patcheCoreV1.Reset()

		patcheCoreV1Pods := ApplyMethod(reflect.TypeOf(new(coretypev1.CoreV1Client)), "Pods",
			func(_ *coretypev1.CoreV1Client, _ string) coretypev1.PodInterface {
				return &tests.MockPods{}
			})
		defer patcheCoreV1Pods.Reset()

		config := &confv1.Config{
			Resources: confv1.Resources{
				"gcu": []confv1.Resource{
					{
						Pattern: "*",
						Name:    "enflame.com/gcu",
					},
					{
						Pattern: "*",
						Name:    "enflame-tech.com/gcu",
					},
				},
				"vgcu": []confv1.Resource{
					{
						Pattern: "*",
						Name:    "enflame.com/vgcu",
					},
				},
			},
		}
		psa, err := NewPcieSwitchAffinity(gcu.NewRegisterDeviceLib(config)["gcu"], "enflame.com/gcu")
		So(err, ShouldEqual, nil)

		Convey("1. test create pod, request 3 GCU success\n", func() {
			// candidate pod will be seleted
			pod1 := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod-1",
					Namespace:         "kube-system",
					UID:               "pod-1-uid",
					CreationTimestamp: metav1.Time{Time: time.Unix(time.Now().Unix(), 0)},
				},
				Spec: v1.PodSpec{
					NodeName: "node-1",
					Containers: []v1.Container{
						{
							Name: "pod1-c1",
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"enflame.com/gcu": *resource.NewQuantity(3, ""),
								},
							},
						},
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
				},
			}
			outputsListPod := []OutputCell{
				// mock candidate pods
				{Values: Params{&v1.PodList{Items: []v1.Pod{pod1}}, nil}},
				// mock list allocated pods
				{Values: Params{&v1.PodList{Items: []v1.Pod{pod1}}, nil}},
			}
			patchePodsList := ApplyMethodSeq(reflect.TypeOf(new(tests.MockPods)), "List", outputsListPod)
			defer patchePodsList.Reset()

			clusterPod1 := pod1
			clusterPod1.Annotations = map[string]string{AssignedContainers: `{"pod1-c1": ["8", "9", "10"]}`}
			patcheGetPod := ApplyMethod(reflect.TypeOf(new(tests.MockPods)), "Get",
				func(_ *tests.MockPods, _ context.Context, _ string, _ metav1.GetOptions) (*v1.Pod, error) {
					return &clusterPod1, nil
				})
			defer patcheGetPod.Reset()

			res, err := psa.AllocateByPcieSwitchAffinity(3)
			So(err, ShouldEqual, nil)
			So(res, ShouldResemble, []string{"8", "9", "10"})
		})

		Convey("2. test create pod, request 1 GCU success\n", func() {
			// candidate pod will be seleted
			pod1 := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod-1",
					Namespace:         "kube-system",
					UID:               "pod-1-uid",
					CreationTimestamp: metav1.Time{Time: time.Unix(time.Now().Unix(), 0)},
				},
				Spec: v1.PodSpec{
					NodeName: "node-1",
					Containers: []v1.Container{
						{
							Name: "pod1-c1",
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"enflame.com/gcu": *resource.NewQuantity(1, ""),
								},
							},
						},
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
				},
			}
			outputsListPod := []OutputCell{
				// mock candidate pods
				{Values: Params{&v1.PodList{Items: []v1.Pod{pod1}}, nil}},
				// mock list allocated pods
				{Values: Params{&v1.PodList{Items: []v1.Pod{pod1}}, nil}},
			}
			patchePodsList := ApplyMethodSeq(reflect.TypeOf(new(tests.MockPods)), "List", outputsListPod)
			defer patchePodsList.Reset()

			clusterPod1 := pod1
			clusterPod1.Annotations = map[string]string{AssignedContainers: `{"pod1-c1": ["2"]}`}
			patcheGetPod := ApplyMethod(reflect.TypeOf(new(tests.MockPods)), "Get",
				func(_ *tests.MockPods, _ context.Context, _ string, _ metav1.GetOptions) (*v1.Pod, error) {
					return &clusterPod1, nil
				})
			defer patcheGetPod.Reset()

			patcheStat := ApplyFunc(os.Stat, func(_ string) (fs.FileInfo, error) {
				return nil, nil
			})
			defer patcheStat.Reset()

			res, err := psa.AllocateByPcieSwitchAffinity(1)
			So(err, ShouldEqual, nil)
			So(res, ShouldResemble, []string{"2"})
		})
	})
}
