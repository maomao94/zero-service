package gtwx

import (
	"testing"

	"zero-service/common/errx"
	"zero-service/third_party/extproto"
)

func TestFoldPolicyDisabledPassesThrough(t *testing.T) {
	policy := newFoldPolicy(FoldConfig{Codes: []int{100001}})

	code, msg := policy.apply(100001, "db down")
	if code != 100001 || msg != "db down" {
		t.Fatalf("apply() = %d/%q, want 100001/db down", code, msg)
	}
}

func TestFoldPolicyConfiguredCode(t *testing.T) {
	policy := newFoldPolicy(FoldConfig{Enabled: true, Codes: []int{100001}})

	code, msg := policy.apply(100001, "db down")
	if code != int(extproto.Code__1_00_UNKNOWN) || msg != errx.SystemBusyMessage {
		t.Fatalf("apply() = %d/%q, want 100002/%q", code, msg, errx.SystemBusyMessage)
	}
}

func TestFoldPolicyLeavesOtherCodes(t *testing.T) {
	policy := newFoldPolicy(FoldConfig{Enabled: true, Codes: []int{100001}})

	code, msg := policy.apply(100019, "db down")
	if code != 100019 || msg != "db down" {
		t.Fatalf("apply() = %d/%q, want 100019/db down", code, msg)
	}
}

func TestFoldPolicyKeepMsg(t *testing.T) {
	policy := newFoldPolicy(FoldConfig{Enabled: true, Codes: []int{100001}, KeepMsg: true})

	code, msg := policy.apply(100001, "db down")
	if code != int(extproto.Code__1_00_UNKNOWN) || msg != "db down" {
		t.Fatalf("apply() = %d/%q, want 100002/db down", code, msg)
	}
}

func TestFoldPolicyCustomTarget(t *testing.T) {
	policy := newFoldPolicy(FoldConfig{
		Enabled: true,
		Target:  int(extproto.Code__1_00_BIZ),
		Codes:   []int{100001},
	})

	code, _ := policy.apply(100001, "db down")
	if code != int(extproto.Code__1_00_BIZ) {
		t.Fatalf("apply() = %d, want %d", code, extproto.Code__1_00_BIZ)
	}
}

func TestFoldPolicyNeverFoldsThirdPartyAPI(t *testing.T) {
	policy := newFoldPolicy(FoldConfig{
		Enabled: true,
		Codes:   []int{int(extproto.Code__1_00_THIRD_PARTY_API)},
	})

	msg := "第三方接口错误，vendor=dji，code=312014，message=设备升级中"
	code, gotMsg := policy.apply(int(extproto.Code__1_00_THIRD_PARTY_API), msg)
	if code != int(extproto.Code__1_00_THIRD_PARTY_API) || gotMsg != msg {
		t.Fatalf("apply() = %d/%q, want %d/%q",
			code, gotMsg, int(extproto.Code__1_00_THIRD_PARTY_API), msg)
	}
}

func TestFoldPolicyInvalidTargetFallsBack(t *testing.T) {
	policy := newFoldPolicy(FoldConfig{
		Enabled: true,
		Target:  123456,
		Codes:   []int{100001},
	})

	code, msg := policy.apply(100001, "db down")
	if code != int(extproto.Code__1_00_UNKNOWN) || msg != "系统繁忙" {
		t.Fatalf("apply() = %d/%q, want 100002/系统繁忙", code, msg)
	}
}

func TestFoldPolicyIgnoresUndefinedCodes(t *testing.T) {
	policy := newFoldPolicy(FoldConfig{
		Enabled: true,
		Codes:   []int{123456},
	})

	code, msg := policy.apply(123456, "raw")
	if code != 123456 || msg != "raw" {
		t.Fatalf("apply() = %d/%q, want 123456/raw", code, msg)
	}
}
