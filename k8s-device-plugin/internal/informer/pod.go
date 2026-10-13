package informer

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	podresourcesapi "k8s.io/kubelet/pkg/apis/podresources/v1"

	"enflame/k8s-device-plugin/internal/kube"
	"enflame/k8s-device-plugin/internal/logger"
	"enflame/k8s-device-plugin/internal/types"
)

type PodInformer struct {
	sync.RWMutex
	ResourceName       string
	DevicesCache       map[string]PodInfo
	NodeName           string
	Clientset          *kubernetes.Clientset
	PodResourcesClient podresourcesapi.PodResourcesListerClient
	Conn               *grpc.ClientConn
	// podLister serves node-scoped pod reads from the local informer cache; set
	// by Start(). Used by the by-card "informer" pod-source so the high-frequency
	// Allocate pending-pod listing reads cache instead of hitting the apiserver.
	podLister listersv1.PodLister
}

type PodInfo struct {
	Name      string
	Namespace string
	UUID      k8stypes.UID
}

func NewPodInformer(resourceName string) (*PodInformer, error) {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		err := fmt.Errorf("please set env NODE_NAME")
		logger.Error(err)
		return nil, err
	}
	clientSet, err := kube.GetKubeClient()
	if err != nil {
		return nil, err
	}
	conn, err := grpc.NewClient(types.PodResourceSocket, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	podResourcesClient := podresourcesapi.NewPodResourcesListerClient(conn)
	return &PodInformer{
		ResourceName:       resourceName,
		DevicesCache:       make(map[string]PodInfo),
		NodeName:           nodeName,
		Clientset:          clientSet,
		PodResourcesClient: podResourcesClient,
		Conn:               conn,
	}, nil
}

func (rer *PodInformer) Close() {
	if rer.Conn != nil {
		if err := rer.Conn.Close(); err != nil {
			logger.Error(err, "close pod resource grpc socket failed")
		} else {
			logger.Info("close pod resource grpc socket success")
		}
	}
}

func (rer *PodInformer) Start(ctx context.Context) error {
	factory := informers.NewSharedInformerFactoryWithOptions(
		rer.Clientset,
		time.Minute, // Resync time interval
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = "spec.nodeName=" + rer.NodeName
		}),
	)

	podInformer := factory.Core().V1().Pods().Informer()
	rer.podLister = factory.Core().V1().Pods().Lister()
	// register event handle funcs
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if pod, ok := obj.(*v1.Pod); ok {
				rer.addOrUpdate(pod)
			}
		},
		UpdateFunc: func(oldObj, newObj any) {
			if _, ok := oldObj.(*v1.Pod); ok {
				if newPod, ok := newObj.(*v1.Pod); ok {
					rer.addOrUpdate(newPod)
				}
			}
		},
		DeleteFunc: func(obj any) {
			if pod, ok := obj.(*v1.Pod); ok {
				rer.delete(pod)
			}
		},
	})

	// start pod informer
	go podInformer.Run(ctx.Done())

	// wait first sync finish
	if !cache.WaitForCacheSync(ctx.Done(), podInformer.HasSynced) {
		err := fmt.Errorf("timed out waiting for pods local cache sync")
		logger.Error(err)
		return err
	}
	logger.Info("pod informer for resource: %s work start...", rer.ResourceName)
	return nil
}

// ListNodePods returns the pods on this node from the local informer cache
// (already node-scoped by the spec.nodeName field selector). It serves the
// by-card "informer" pod-source so the per-Allocate pending-pod listing avoids
// a direct apiserver List. The returned pods are shared cache objects and must
// not be mutated by callers.
func (rer *PodInformer) ListNodePods() ([]*v1.Pod, error) {
	if rer.podLister == nil {
		return nil, fmt.Errorf("pod informer lister not initialized; informer not started")
	}
	return rer.podLister.List(labels.Everything())
}

func (rer *PodInformer) addOrUpdate(pod *v1.Pod) {
	rer.Lock()
	defer rer.Unlock()
	if !rer.podRequestResource(pod) {
		return
	}

	resp, err := rer.PodResourcesClient.List(context.TODO(), &podresourcesapi.ListPodResourcesRequest{})
	if err != nil {
		logger.Error(err, "get pod(name: %s, namespace: %s) by pod resource api failed", pod.Name, pod.Namespace)
		return
	}
	for _, podResource := range resp.PodResources {
		if podResource.Name != pod.Name || podResource.Namespace != pod.Namespace {
			continue
		}
		for _, container := range podResource.Containers {
			for _, device := range container.Devices {
				if device.ResourceName == rer.ResourceName {
					for _, deviceID := range device.DeviceIds {
						if podInfo, ok := rer.DevicesCache[deviceID]; ok {
							if podInfo.UUID == pod.UID {
								continue
							}
							msg := fmt.Sprintf("internal error! found device: %s used by pod: %v in pod informer cache",
								deviceID, podInfo)
							logger.Error("%s, but it allocated for anothor pod(name: %s, namespace: %s, uuid: %s) now",
								msg, pod.Name, pod.Namespace, pod.UID)
							return
						}
						logger.Info("add device: %s used by pod(name: %s, namespace: %s, uuid: %s) to cache",
							deviceID, pod.Name, pod.Namespace, pod.UID)
						rer.DevicesCache[deviceID] = PodInfo{Name: pod.Name, Namespace: pod.Namespace, UUID: pod.UID}
					}
				}
			}
		}
		return
	}
}

func (rer *PodInformer) delete(pod *v1.Pod) {
	rer.Lock()
	defer rer.Unlock()
	if !rer.podRequestResource(pod) {
		return
	}
	for deviceID, podInfo := range rer.DevicesCache {
		if podInfo.UUID == pod.UID {
			logger.Info("delete device: %s used by pod(name: %s, namespace: %s, uuid: %s) from cache",
				deviceID, pod.Name, pod.Namespace, pod.UID)
			delete(rer.DevicesCache, deviceID)
		}
	}
}

func (rer *PodInformer) IsDeviceInUse(deviceID string) (bool, PodInfo) {
	rer.RLock()
	defer rer.RUnlock()
	podInfo, exists := rer.DevicesCache[deviceID]
	return exists, podInfo
}

func (rer *PodInformer) podRequestResource(pod *v1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		if val, ok := container.Resources.Limits[v1.ResourceName(rer.ResourceName)]; ok && val.Value() > 0 {
			return true
		}
	}
	return false
}
