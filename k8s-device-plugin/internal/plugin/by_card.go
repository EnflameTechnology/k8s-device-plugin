// Copyright (c) 2026, ENFLAME INC.  All rights reserved.
// EF_SUPPORT
//
// By-card ("按调度结果挂卡") whole-card allocation policy.
//
// Instead of the device plugin choosing which physical GCU cards to mount, an
// external scheduler decides the cards and writes them onto the pod annotation;
// the plugin reads that decision and mounts exactly those cards. Whole-card
// only (vgcu / sharing out of scope).
//
// Occupancy is validated against the live kubelet podresources view exposed by
// the PodInformer (IsDeviceInUse), so the check reflects real allocations rather
// than self-written annotations.
package plugin

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"

	"enflame/k8s-device-plugin/internal/eflib/interfaces"
	"enflame/k8s-device-plugin/internal/informer"
	"enflame/k8s-device-plugin/internal/logger"
	"enflame/k8s-device-plugin/internal/utils"
)

const (
	// Default scheduler annotation keys; both are configurable and only need to
	// agree with the scheduler.
	DefaultSchedulerDeviceAnnotationKey      = "enflame.com/vgpu-assumed"
	DefaultSchedulerAssumedTimeAnnotationKey = "enflame.com/assumed-time"

	// How long Allocate waits for the scheduler annotation to land before failing
	// (shorter than the kubelet Allocate timeout so kubelet retries cleanly). RISK-5.
	byCardAnnotationWaitTimeout = 10 * time.Second
	byCardAnnotationPollPeriod  = 1 * time.Second

	// How long an in-process card reservation stays valid before it is pruned. By
	// then kubelet podresources (and the PodInformer DevicesCache that backs
	// IsDeviceInUse) have caught up with the allocation and the live occupancy
	// check takes over; the TTL only guards against orphaned reservations.
	byCardReservationTTL = 30 * time.Second
)

// ByCardScheduler implements the "allocate by external scheduler result" policy.
type ByCardScheduler struct {
	eflib          interfaces.HandleInterface
	podResource    *PodResource
	podInformer    *informer.PodInformer
	deviceAnnoKey  string
	assumedTimeKey string
	waitTimeout    time.Duration

	// mu guards reserved and serializes the occupancy check-and-reserve so two
	// concurrent Allocate calls cannot both claim the same physical card while the
	// async PodInformer DevicesCache (IsDeviceInUse) is still catching up.
	mu sync.Mutex
	// reserved maps a physical card index -> the pod that just claimed it in this
	// process, before the claim is visible via kubelet podresources. Pruned by TTL.
	reserved map[string]reservedCard
}

// reservedCard records an in-process card claim not yet reflected in kubelet podresources.
type reservedCard struct {
	uid string
	at  time.Time
}

// NewByCardScheduler builds a by-card allocator for a resource. The podInformer
// supplies the live (kubelet podresources-backed) device-in-use view.
func NewByCardScheduler(eflib interfaces.HandleInterface, resourceName string, podInformer *informer.PodInformer,
	deviceAnnoKey, assumedTimeKey, podSource string) (*ByCardScheduler, error) {
	podResource, err := NewPodResource(resourceName)
	if err != nil {
		return nil, err
	}
	// Large-scale optimization (design §5.5): route the per-Allocate pending-pod
	// listing through the configured source (apiserver default / informer / kubelet).
	if err := podResource.SetPendingPodSource(podSource, podInformer); err != nil {
		return nil, err
	}
	if strings.TrimSpace(deviceAnnoKey) == "" {
		deviceAnnoKey = DefaultSchedulerDeviceAnnotationKey
	}
	if strings.TrimSpace(assumedTimeKey) == "" {
		assumedTimeKey = DefaultSchedulerAssumedTimeAnnotationKey
	}
	logger.Info("deploy k8s device plugin with by-card(scheduler-result) schedule, resource: %s, deviceKey=%s, assumedTimeKey=%s",
		resourceName, deviceAnnoKey, assumedTimeKey)
	return &ByCardScheduler{
		eflib:          eflib,
		podResource:    podResource,
		podInformer:    podInformer,
		deviceAnnoKey:  deviceAnnoKey,
		assumedTimeKey: assumedTimeKey,
		waitTimeout:    byCardAnnotationWaitTimeout,
		reserved:       make(map[string]reservedCard),
	}, nil
}

// AllocateByCard binds this Allocate to the pending pod whose scheduler-assigned
// cards equal the device ids kubelet selected (requestIds), then mounts exactly
// those cards. Choice A (fail-safe): if no pending pod's assigned cards match the
// kubelet selection (the pod's assigned card was unavailable so kubelet picked a
// different one, or the annotation is not ready), the request is retried until
// the deadline and then FAILS - by-card never mounts a card other than the
// scheduler-assigned one, and never mounts a card already taken.
func (b *ByCardScheduler) AllocateByCard(requestIds []string) ([]string, error) {
	deadline := time.Now().Add(b.waitTimeout)
	var lastErr error
	for {
		// Re-list and re-match every round; never cache the first pick (RISK-5).
		assumePod, assignedMap, err := b.matchPendingPod(requestIds)
		if err != nil {
			lastErr = err
		} else {
			allocateIds, perr := parseAssumedDeviceIndices(assumePod.Annotations[b.deviceAnnoKey])
			if perr != nil {
				lastErr = fmt.Errorf("pod %s/%s scheduler annotation %q not ready: %v",
					assumePod.Namespace, assumePod.Name, b.deviceAnnoKey, perr)
			} else if verr := b.validateAndReserve(assumePod, allocateIds, len(requestIds)); verr != nil {
				// card taken/invalid/reserved in-flight. Retry until the deadline (a
				// transient in-use from a pod being deleted may clear); then fail (choice A).
				lastErr = verr
			} else {
				logger.Info("allocate device: %v for pod %s/%s by scheduler result (kubelet cards %v)",
					allocateIds, assumePod.Namespace, assumePod.Name, requestIds)
				if err := b.podResource.PatchPodAnnotations(assumePod, assignedMap, allocateIds); err != nil {
					b.releaseReservation(allocateIds, string(assumePod.UID))
					return nil, err
				}
				return allocateIds, nil
			}
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("by-card allocate timed out after %s: no pending pod is assigned the "+
				"kubelet-selected cards %v (assigned card unavailable or annotation not ready): %v",
				b.waitTimeout, requestIds, lastErr)
		}
		logger.Warn("by-card allocate not ready, retrying: %v", lastErr)
		time.Sleep(byCardAnnotationPollPeriod)
	}
}

// matchPendingPod finds the pending pod whose scheduler-assigned cards equal the
// device ids kubelet selected for this Allocate (requestIds). Binding by the
// kubelet-selected card set keeps the mounted cards consistent with kubelet
// podresources (no divergence). There is deliberately NO assumed-time fallback:
// if the kubelet selection matches no pending pod's assignment we do not mount a
// different card (choice A / fail-safe); the caller retries then fails.
func (b *ByCardScheduler) matchPendingPod(requestIds []string) (*corev1.Pod, map[string][]string, error) {
	want := normalizeCardSet(requestIds)
	allPods, err := b.podResource.GetPendingPodsInNode()
	if err != nil {
		return nil, nil, err
	}
	candidatePods := []*corev1.Pod{}
	for i := range allPods {
		if b.podResource.checkPodRequestGCU(&allPods[i]) {
			candidatePods = append(candidatePods, &allPods[i])
		}
	}
	candidatePods = makePodOrderedByAssumedTime(candidatePods, b.assumedTimeKey)
	for _, pod := range candidatePods {
		ids, perr := parseAssumedDeviceIndices(pod.Annotations[b.deviceAnnoKey])
		if perr != nil || normalizeCardSet(ids) != want {
			continue
		}
		containerName, assignedMap, cerr := b.podResource.PodExistContainerCanBind(len(requestIds), pod)
		if cerr != nil {
			return nil, nil, cerr
		}
		if containerName == "" {
			continue
		}
		assignedMap[containerName] = []string{}
		logger.Info("found assumed pod %s/%s container: %s by requested cards %v",
			pod.Namespace, pod.Name, containerName, requestIds)
		return pod, assignedMap, nil
	}
	return nil, nil, fmt.Errorf("no pending pod is assigned the kubelet-selected cards %v", requestIds)
}

// validateAndReserve atomically validates the cards (static checks + live kubelet
// podresources occupancy + the in-process reservation ledger) and, on success,
// reserves them for this pod. Holding mu across check-and-reserve makes two
// concurrent Allocate calls unable to both pass for the same card while the async
// IsDeviceInUse cache lags - this closes the TOCTOU that let a card be double-mounted.
func (b *ByCardScheduler) validateAndReserve(pod *corev1.Pod, allocateIds []string, reqDeviceCount int) error {
	uid := string(pod.UID)
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pruneReservationsLocked()
	if err := b.validateScheduledCards(allocateIds, reqDeviceCount); err != nil {
		return err
	}
	for _, id := range allocateIds {
		if r, ok := b.reserved[id]; ok && r.uid != uid {
			return fmt.Errorf("scheduler assigned device %s is already reserved in-flight by another pod (uid %s)", id, r.uid)
		}
	}
	now := time.Now()
	for _, id := range allocateIds {
		b.reserved[id] = reservedCard{uid: uid, at: now}
	}
	return nil
}

// releaseReservation drops this pod's in-process reservations (used when a later
// step such as the annotation patch fails after reserving).
func (b *ByCardScheduler) releaseReservation(allocateIds []string, uid string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, id := range allocateIds {
		if r, ok := b.reserved[id]; ok && r.uid == uid {
			delete(b.reserved, id)
		}
	}
}

// pruneReservationsLocked drops reservations older than the TTL (caller holds mu).
func (b *ByCardScheduler) pruneReservationsLocked() {
	for id, r := range b.reserved {
		if time.Since(r.at) > byCardReservationTTL {
			delete(b.reserved, id)
		}
	}
}

// normalizeCardSet returns a canonical (numerically sorted, comma-joined) form of
// a card-index list so two lists compare equal as sets regardless of order.
func normalizeCardSet(ids []string) string {
	nums := make([]int, 0, len(ids))
	for _, s := range ids {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return "invalid:" + strings.Join(ids, ",")
		}
		nums = append(nums, n)
	}
	sort.Ints(nums)
	parts := make([]string, len(nums))
	for i, n := range nums {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ",")
}

// PreferredAllocation steers kubelet's device selection towards the
// scheduler-assigned cards, so that kubelet's own bookkeeping (podresources,
// consumed by gcu-exporter and by the device-in-use check) matches the cards
// by-card actually mounts. It is advisory: on any miss it returns nil and lets
// kubelet fall back to the default policy, because the authoritative mount
// still happens in AllocateByCard.
func (b *ByCardScheduler) PreferredAllocation(available []string, size int) []string {
	assumePod, _, err := b.podResource.GetAssumePodByAssumedTime(size, b.assumedTimeKey)
	if err != nil {
		logger.Warn("by-card preferred allocation: no assumed pod for device count %d yet: %v", size, err)
		return nil
	}
	allocateIds, perr := parseAssumedDeviceIndices(assumePod.Annotations[b.deviceAnnoKey])
	if perr != nil {
		logger.Warn("by-card preferred allocation: pod %s/%s annotation %q not ready: %v",
			assumePod.Namespace, assumePod.Name, b.deviceAnnoKey, perr)
		return nil
	}
	if len(allocateIds) != size {
		logger.Warn("by-card preferred allocation: pod %s/%s assigned %d cards but kubelet asked %d, defer to kubelet",
			assumePod.Namespace, assumePod.Name, len(allocateIds), size)
		return nil
	}
	availSet := make(map[string]bool, len(available))
	for _, id := range available {
		availSet[id] = true
	}
	for _, id := range allocateIds {
		if !availSet[id] {
			logger.Warn("by-card preferred allocation: pod %s/%s assigned card %s not in kubelet available set %v, defer to kubelet",
				assumePod.Namespace, assumePod.Name, id, available)
			return nil
		}
	}
	logger.Info("by-card preferred allocation: recommend cards %v for pod %s/%s",
		allocateIds, assumePod.Namespace, assumePod.Name)
	return allocateIds
}

// validateScheduledCards enforces the scheduler/plugin contract (RISK-3 / RISK-7):
//   - count matches the kubelet request,
//   - every index is a real host-physical GCU on this node,
//   - the device node exists,
//   - the card is not already in use by a running pod (per kubelet podresources).
func (b *ByCardScheduler) validateScheduledCards(allocateIds []string, reqDeviceCount int) error {
	if len(allocateIds) != reqDeviceCount {
		return fmt.Errorf("scheduler assigned %d cards %v but kubelet requested %d for resource %s",
			len(allocateIds), allocateIds, reqDeviceCount, b.podResource.ResourceName)
	}
	deviceIDPathMap := b.eflib.GetDeviceIDPathMap()
	seen := map[int]bool{}
	for _, idStr := range allocateIds {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return fmt.Errorf("invalid device index %q in scheduler annotation: %v", idStr, err)
		}
		if seen[id] {
			return fmt.Errorf("scheduler annotation contains duplicate device index %d: %v", id, allocateIds)
		}
		seen[id] = true
		path, ok := deviceIDPathMap[id]
		if !ok {
			return fmt.Errorf("scheduler assigned device index %d is not a host-physical GCU on this node (known indices: %v)",
				id, deviceIDPathMap)
		}
		if !utils.FileIsExist(path) {
			return fmt.Errorf("scheduler assigned device index %d path %s does not exist", id, path)
		}
		if b.podInformer != nil {
			if inUse, podInfo := b.podInformer.IsDeviceInUse(idStr); inUse {
				return fmt.Errorf("scheduler assigned device index %d is already in use by pod %s/%s (uid %s) per kubelet podresources",
					id, podInfo.Namespace, podInfo.Name, podInfo.UUID)
			}
		}
	}
	return nil
}

// parseAssumedDeviceIndices parses the scheduler device annotation into a list of
// physical GCU index strings. Accepts comma-separated ("2" / "2,5,7") and
// JSON-array ("[2,5,7]") forms.
func parseAssumedDeviceIndices(raw string) ([]string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, fmt.Errorf("empty device annotation")
	}
	var parts []string
	if strings.HasPrefix(s, "[") {
		var arr []int
		if err := json.Unmarshal([]byte(s), &arr); err != nil {
			return nil, fmt.Errorf("parse json array %q: %v", s, err)
		}
		for _, v := range arr {
			parts = append(parts, strconv.Itoa(v))
		}
	} else {
		parts = strings.Split(s, ",")
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, err := strconv.Atoi(p); err != nil {
			return nil, fmt.Errorf("invalid device index %q in annotation %q: %v", p, s, err)
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no device index parsed from annotation %q", s)
	}
	return out, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
