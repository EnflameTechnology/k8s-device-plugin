// Copyright (c) 2026, ENFLAME INC.  All rights reserved.
// EF_SUPPORT

package plugin

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

func TestParseAssumedDeviceIndices(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    []string
		wantErr bool
	}{
		{"single", "2", []string{"2"}, false},
		{"comma", "2,5,7", []string{"2", "5", "7"}, false},
		{"comma-spaces", " 2 , 5 ", []string{"2", "5"}, false},
		{"json-array", "[2,5,7]", []string{"2", "5", "7"}, false},
		{"trailing-comma", "2,5,", []string{"2", "5"}, false},
		{"empty", "", nil, true},
		{"blank", "   ", nil, true},
		{"non-int", "2,a", nil, true},
		{"bad-json", "[2,5", nil, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseAssumedDeviceIndices(c.raw)
			if c.wantErr {
				if err == nil {
					t.Fatalf("parseAssumedDeviceIndices(%q) expected error, got %v", c.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseAssumedDeviceIndices(%q) unexpected error: %v", c.raw, err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("parseAssumedDeviceIndices(%q) = %v, want %v", c.raw, got, c.want)
			}
		})
	}
}

func podWithAssumed(name, uid, assumedTime string) *corev1.Pod {
	anno := map[string]string{}
	if assumedTime != "" {
		anno[DefaultSchedulerAssumedTimeAnnotationKey] = assumedTime
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			UID:         k8stypes.UID(uid),
			Annotations: anno,
		},
	}
}

func TestMakePodOrderedByAssumedTime(t *testing.T) {
	key := DefaultSchedulerAssumedTimeAnnotationKey

	pods := []*corev1.Pod{
		podWithAssumed("late", "uid-late", "300"),
		podWithAssumed("missing", "uid-missing", ""),
		podWithAssumed("early", "uid-early", "100"),
		podWithAssumed("mid", "uid-mid", "200"),
	}
	got := makePodOrderedByAssumedTime(pods, key)
	wantOrder := []string{"early", "mid", "late", "missing"}
	for i, w := range wantOrder {
		if got[i].Name != w {
			t.Fatalf("order[%d] = %s, want %s (full: %v)", i, got[i].Name, w, names(got))
		}
	}

	tie := []*corev1.Pod{
		podWithAssumed("b", "uid-2", "100"),
		podWithAssumed("a", "uid-1", "100"),
	}
	got = makePodOrderedByAssumedTime(tie, key)
	if got[0].Name != "a" || got[1].Name != "b" {
		t.Fatalf("tie-break by UID failed, got %v", names(got))
	}
}

func names(pods []*corev1.Pod) []string {
	out := make([]string, len(pods))
	for i, p := range pods {
		out[i] = p.Name
	}
	return out
}
