// Copyright (c) 2026, ENFLAME INC.  All rights reserved.
// EF_SUPPORT

package plugin

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDedupSortedIndices(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"dedup-sort", []string{"5", "2", "2"}, []string{"2", "5"}},
		{"multi", []string{"3", "1", "0"}, []string{"0", "1", "3"}},
		{"empty", []string{}, []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := dedupSortedIndices(c.in)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("dedupSortedIndices(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestNeedReconcile(t *testing.T) {
	running := func(anno map[string]string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Annotations: anno},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		}
	}

	if !needReconcile(running(nil)) {
		t.Fatalf("pod without annotations should need reconcile")
	}
	if !needReconcile(running(map[string]string{AssignedContainers: "{}"})) {
		t.Fatalf("pod with empty assignedContainers should need reconcile")
	}
	if needReconcile(running(map[string]string{AssignedContainers: `{"c1":["2"]}`})) {
		t.Fatalf("pod already carrying assignedContainers should be skipped")
	}

	terminal := &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodSucceeded}}
	if needReconcile(terminal) {
		t.Fatalf("terminal pod should be skipped")
	}
}

func TestBuildReconcileAnnotations(t *testing.T) {
	perContainer := map[string][]string{
		"c1": {"2", "5"},
		"c2": {"7"},
	}
	got, err := buildReconcileAnnotations(perContainer, DefaultSchedulerDeviceAnnotationKey)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got[AssignedContainers] != `{"c1":["2","5"],"c2":["7"]}` {
		t.Fatalf("assignedContainers = %s", got[AssignedContainers])
	}
	if got[DefaultSchedulerDeviceAnnotationKey] != "2,5,7" {
		t.Fatalf("device key = %s, want 2,5,7", got[DefaultSchedulerDeviceAnnotationKey])
	}
}
