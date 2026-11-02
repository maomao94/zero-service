package gtwx

import (
	"testing"

	"zero-service/common/errx"
	"zero-service/third_party/extproto"
)

func TestFoldPolicyDisabledPassesThrough(t *testing.T) {
	policy := newFoldPolicy(FoldConfig{Codes: []int{100998}})

	code, msg := policy.apply(100998, "db down")
	if code != 100998 || msg != "db down" {
		t.Fatalf("apply() = %d/%q, want 100998/db down", code, msg)
	}
}

func TestFoldPolicyConfiguredCode(t *testing.T) {
	policy := newFoldPolicy(FoldConfig{Enabled: true, Codes: []int{100998}})

	code, msg := policy.apply(100998, "db down")
	if code != int(extproto.Code__1_00_UNKNOWN) || msg != errx.SystemBusyMessage {
		t.Fatalf("apply() = %d/%q, want 100999/%q", code, msg, errx.SystemBusyMessage)
	}
}

func TestFoldPolicyLeavesOtherCodes(t *testing.T) {
	policy := newFoldPolicy(FoldConfig{Enabled: true, Codes: []int{100998}})

	code, msg := policy.apply(100201, "db down")
	if code != 100201 || msg != "db down" {
		t.Fatalf("apply() = %d/%q, want 100201/db down", code, msg)
	}
}

func TestFoldPolicyKeepMsg(t *testing.T) {
	policy := newFoldPolicy(FoldConfig{Enabled: true, Codes: []int{100998}, KeepMsg: true})

	code, msg := policy.apply(100998, "db down")
	if code != int(extproto.Code__1_00_UNKNOWN) || msg != "db down" {
		t.Fatalf("apply() = %d/%q, want 100999/db down", code, msg)
	}
}

func TestFoldPolicyCustomTarget(t *testing.T) {
	policy := newFoldPolicy(FoldConfig{
		Enabled: true,
		Target:  int(extproto.Code__1_00_BIZ),
		Codes:   []int{100998},
	})

	code, _ := policy.apply(100998, "db down")
	if code != int(extproto.Code__1_00_BIZ) {
		t.Fatalf("apply() = %d, want %d", code, extproto.Code__1_00_BIZ)
	}
}
