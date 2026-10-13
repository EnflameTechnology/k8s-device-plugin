// Copyright (c) 2026, ENFLAME INC.  All rights reserved.
// EF_SUPPORT
//
// Stock-pod reconcile (requirement part 3).
//
// When a cluster switches to by-card scheduling, pods already running under the
// old scheme carry no card annotation. The scheduler then cannot see which cards
// they occupy and may hand those cards to new pods. This backfills, for each such
// stock pod, the real allocated cards (read from the kubelet podresources API)
// onto the scheduler device key (e.g. enflame.com/vgpu-assumed) and onto
// k8s-device-plugin/assignedContainers.
//
// The plugin's own by-card occupancy check already sees stock pods via the
// podresources-backed PodInformer, so this is primarily for the scheduler's
// ledger. It runs once when the plugin starts (a DaemonSet rolling update runs it
// once per node) before the plugin serves Allocate; it is idempotent.
package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	podresourcesapi "k8s.io/kubelet/pkg/apis/podresources/v1"

	"enflame/k8s-device-plugin/internal/logger"
)

// reconcileStockPods backfills card annotations for this node's stock pods of the
// plugin's resource. A single pod failure is logged and skipped.
func (plugin *EnflameDevicePlugin) reconcileStockPods() error {
	resourceName := string(plugin.rm.Resource())
	deviceKey := DefaultSchedulerDeviceAnnotationKey
	if k := plugin.config.Flags.Plugin.SchedulerDeviceAnnotationKey; k != nil && strings.TrimSpace(*k) != "" {
		deviceKey = *k
	}
	if plugin.podInformer == nil {
		return fmt.Errorf("pod informer is nil, cannot read podresources for reconcile")
	}

	podResource, err := NewPodResource(resourceName)
	if err != nil {
		return err
	}
	mounted, err := podResourcesMountedCards(plugin.podInformer.PodResourcesClient, resourceName)
	if err != nil {
		return fmt.Errorf("get real mounted cards from podresources failed: %w", err)
	}
	pods, err := podResource.ListRunningGCUPods()
	if err != nil {
		return fmt.Errorf("list running gcu pods failed: %w", err)
	}
	logger.Info("stock-pod reconcile: resource %s, %d running pods, %d pods with cards in podresources",
		resourceName, len(pods), len(mounted))

	reconciled, skipped, failed := 0, 0, 0
	for _, pod := range pods {
		if !needReconcile(pod) {
			skipped++
			continue
		}
		perContainer, ok := mounted[pod.Namespace+"/"+pod.Name]
		if !ok || len(perContainer) == 0 {
			logger.Warn("stock-pod reconcile: pod %s/%s has no %s cards in podresources, skip", pod.Namespace, pod.Name, resourceName)
			skipped++
			continue
		}
		annotations, err := buildReconcileAnnotations(perContainer, deviceKey)
		if err != nil {
			logger.Error(err, "stock-pod reconcile: build annotations for pod %s/%s failed", pod.Namespace, pod.Name)
			failed++
			continue
		}
		if err := podResource.PatchPodAnnotationKeys(pod, annotations); err != nil {
			logger.Error(err, "stock-pod reconcile: patch pod %s/%s failed", pod.Namespace, pod.Name)
			failed++
			continue
		}
		logger.Info("stock-pod reconcile: pod %s/%s backfilled annotations %v", pod.Namespace, pod.Name, annotations)
		reconciled++
	}
	logger.Info("stock-pod reconcile done: resource %s reconciled=%d skipped=%d failed=%d",
		resourceName, reconciled, skipped, failed)
	return nil
}

// podResourcesMountedCards queries the kubelet podresources API and returns, per
// "namespace/name" pod, the device indices kubelet allocated to each container
// for the given resource.
func podResourcesMountedCards(client podresourcesapi.PodResourcesListerClient, resourceName string) (map[string]map[string][]string, error) {
	resp, err := client.List(context.TODO(), &podresourcesapi.ListPodResourcesRequest{})
	if err != nil {
		return nil, err
	}
	out := map[string]map[string][]string{}
	for _, pr := range resp.GetPodResources() {
		key := pr.GetNamespace() + "/" + pr.GetName()
		for _, c := range pr.GetContainers() {
			ids := []string{}
			for _, d := range c.GetDevices() {
				if d.GetResourceName() == resourceName {
					ids = append(ids, d.GetDeviceIds()...)
				}
			}
			if len(ids) == 0 {
				continue
			}
			if out[key] == nil {
				out[key] = map[string][]string{}
			}
			out[key][c.GetName()] = dedupSortedIndices(ids)
		}
	}
	return out, nil
}

// needReconcile reports whether a stock pod still needs its cards backfilled. A
// pod already carrying a non-empty assignedContainers annotation is left alone.
func needReconcile(pod *corev1.Pod) bool {
	if isPodTerminal(pod) {
		return false
	}
	v, ok := pod.Annotations[AssignedContainers]
	if ok && strings.TrimSpace(v) != "" && strings.TrimSpace(v) != "{}" {
		return false
	}
	return true
}

// buildReconcileAnnotations produces the annotation patch:
//   - AssignedContainers: {"container":["2","5"]} (per-container, plugin view)
//   - deviceAnnoKey:      "2,5" (pod-level flat unique sorted, scheduler view)
func buildReconcileAnnotations(perContainer map[string][]string, deviceAnnoKey string) (map[string]string, error) {
	assigned := map[string][]string{}
	all := []string{}
	for container, ids := range perContainer {
		assigned[container] = ids
		all = append(all, ids...)
	}
	flat := dedupSortedIndices(all)
	if len(flat) == 0 {
		return nil, fmt.Errorf("no cards to backfill")
	}
	assignedJSON, err := json.Marshal(assigned)
	if err != nil {
		return nil, fmt.Errorf("marshal assignedContainers: %w", err)
	}
	return map[string]string{
		AssignedContainers: string(assignedJSON),
		deviceAnnoKey:      strings.Join(flat, ","),
	}, nil
}

// dedupSortedIndices removes duplicates and sorts numeric index strings
// numerically (non-numeric kept and sorted after numbers).
func dedupSortedIndices(ids []string) []string {
	seen := map[string]bool{}
	uniq := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		uniq = append(uniq, id)
	}
	sort.Slice(uniq, func(i, j int) bool {
		ai, aerr := strconv.Atoi(uniq[i])
		bi, berr := strconv.Atoi(uniq[j])
		if aerr == nil && berr == nil {
			return ai < bi
		}
		if aerr == nil {
			return true
		}
		if berr == nil {
			return false
		}
		return uniq[i] < uniq[j]
	})
	return uniq
}
