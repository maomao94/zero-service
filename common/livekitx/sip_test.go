package livekitx_test

// SIP API 集成测试 — 验证 livekitx 对 LiveKit SIP Server 的管理能力。
// 运行前确保 Docker Compose 环境已启动（deploy/livekit/start.sh）。
//
// 测试内容：
//   1. 创建 outbound trunk（指向 FreeSWITCH）
//   2. 创建 inbound trunk（接收来电）
//   3. 创建 dispatch rule（来电路由到固定 Room）
//   4. 列出 trunk 和 rule
//   5. 清理：删除 trunk 和 rule

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/livekit/protocol/livekit"
	"zero-service/common/livekitx"
)

const (
	// Docker 环境的 LiveKit Server 地址（自签证书已装入系统信任链，见 deploy/tls）
	sipTestURL = "http://127.0.0.1:7880"
	// API 密钥（必须和 deploy/livekit/livekit-server.yaml 一致）
	sipTestKey    = "devkeydevkeydevkeydevkeydevkeydevkey"
	sipTestSecret = "secretsecretsecretsecretsecretsecret"
	// FreeSWITCH 地址（Docker 容器名，容器内可解析）
	// 使用 external profile（5080），不需要认证
	freeSWITCHAddr = "freeswitch:5080"
	// FreeSWITCH 默认分机号码
	freeSWITCHNumber = "1000"
)

func newSIPTestClient(t *testing.T) *livekitx.Client {
	t.Helper()
	client, err := livekitx.New(
		livekitx.WithURL(sipTestURL),
		livekitx.WithAPIKey(sipTestKey, sipTestSecret),
	)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

// TestSIPTrunkCRUD 测试 SIP Trunk 的创建、列出、删除。
func TestSIPTrunkCRUD(t *testing.T) {
	client := newSIPTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	sip := client.API().SIP()

	// 1. 创建 outbound trunk
	t.Log("创建 outbound trunk...")
	outRes, err := sip.CreateSIPOutboundTrunk(ctx, &livekit.CreateSIPOutboundTrunkRequest{
		Trunk: &livekit.SIPOutboundTrunkInfo{
			Name:         "test-outbound-trunk",
			Address:      freeSWITCHAddr,
			Numbers:      []string{freeSWITCHNumber},
			AuthUsername: "freeswitch",
			AuthPassword: "freeswitch",
		},
	})
	if err != nil {
		t.Fatalf("CreateSIPOutboundTrunk: %v", err)
	}
	t.Logf("outbound trunk created: %s", outRes.SipTrunkId)

	// 2. 创建 inbound trunk
	t.Log("创建 inbound trunk...")
	inRes, err := sip.CreateSIPInboundTrunk(ctx, &livekit.CreateSIPInboundTrunkRequest{
		Trunk: &livekit.SIPInboundTrunkInfo{
			Name:    "test-inbound-trunk",
			Numbers: []string{freeSWITCHNumber},
		},
	})
	if err != nil {
		t.Fatalf("CreateSIPInboundTrunk: %v", err)
	}
	t.Logf("inbound trunk created: %s", inRes.SipTrunkId)

	// 3. 列出 trunk
	t.Log("列出 trunk...")
	listRes, err := sip.ListSIPTrunk(ctx, &livekit.ListSIPTrunkRequest{})
	if err != nil {
		t.Fatalf("ListSIPTrunk: %v", err)
	}
	t.Logf("trunk count: %d", len(listRes.Items))
	for _, trunk := range listRes.Items {
		t.Logf("  trunk: %s (%s)", trunk.SipTrunkId, trunk.Name)
	}

	// 4. 清理
	t.Log("删除 trunk...")
	_, err = sip.DeleteSIPTrunk(ctx, &livekit.DeleteSIPTrunkRequest{SipTrunkId: outRes.SipTrunkId})
	if err != nil {
		t.Errorf("DeleteSIPTrunk (outbound): %v", err)
	}
	_, err = sip.DeleteSIPTrunk(ctx, &livekit.DeleteSIPTrunkRequest{SipTrunkId: inRes.SipTrunkId})
	if err != nil {
		t.Errorf("DeleteSIPTrunk (inbound): %v", err)
	}
	t.Log("trunk 清理完成")
}

// TestSIPDispatchRuleCRUD 测试 Dispatch Rule 的创建、列出、删除。
func TestSIPDispatchRuleCRUD(t *testing.T) {
	client := newSIPTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	sip := client.API().SIP()

	// 1. 先创建一个 trunk（dispatch rule 需要关联 trunk）
	t.Log("创建 trunk...")
	trunkRes, err := sip.CreateSIPOutboundTrunk(ctx, &livekit.CreateSIPOutboundTrunkRequest{
		Trunk: &livekit.SIPOutboundTrunkInfo{
			Name:    "test-rule-trunk",
			Address: freeSWITCHAddr,
			Numbers: []string{freeSWITCHNumber},
		},
	})
	if err != nil {
		t.Fatalf("CreateSIPOutboundTrunk: %v", err)
	}
	t.Logf("trunk created: %s", trunkRes.SipTrunkId)

	// 2. 创建 dispatch rule（fixed 模式，所有来电进入 sip-test-room）
	t.Log("创建 dispatch rule...")
	ruleRes, err := sip.CreateSIPDispatchRule(ctx, &livekit.CreateSIPDispatchRuleRequest{
		Rule: &livekit.SIPDispatchRule{
			Rule: &livekit.SIPDispatchRule_DispatchRuleDirect{
				DispatchRuleDirect: &livekit.SIPDispatchRuleDirect{
					RoomName: "sip-test-room",
				},
			},
		},
		TrunkIds: []string{trunkRes.SipTrunkId},
		Name:     "test-dispatch-rule",
	})
	if err != nil {
		t.Fatalf("CreateSIPDispatchRule: %v", err)
	}
	t.Logf("dispatch rule created: %s", ruleRes.SipDispatchRuleId)

	// 3. 列出 dispatch rule
	t.Log("列出 dispatch rule...")
	listRes, err := sip.ListSIPDispatchRule(ctx, &livekit.ListSIPDispatchRuleRequest{})
	if err != nil {
		t.Fatalf("ListSIPDispatchRule: %v", err)
	}
	t.Logf("dispatch rule count: %d", len(listRes.Items))
	for _, rule := range listRes.Items {
		t.Logf("  rule: %s (%s)", rule.SipDispatchRuleId, rule.Name)
	}

	// 4. 清理
	t.Log("删除 dispatch rule...")
	_, err = sip.DeleteSIPDispatchRule(ctx, &livekit.DeleteSIPDispatchRuleRequest{
		SipDispatchRuleId: ruleRes.SipDispatchRuleId,
	})
	if err != nil {
		t.Errorf("DeleteSIPDispatchRule: %v", err)
	}
	_, err = sip.DeleteSIPTrunk(ctx, &livekit.DeleteSIPTrunkRequest{SipTrunkId: trunkRes.SipTrunkId})
	if err != nil {
		t.Errorf("DeleteSIPTrunk: %v", err)
	}
	t.Log("清理完成")
}

// TestSIPDial 测试 SIP 外呼拨号。
// 注意：需要 FreeSWITCH 中有注册的分机才能真正接通。
func TestSIPDial(t *testing.T) {
	client := newSIPTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	sip := client.API().SIP()

	// 1. 创建 trunk
	t.Log("创建 trunk...")
	trunkRes, err := sip.CreateSIPOutboundTrunk(ctx, &livekit.CreateSIPOutboundTrunkRequest{
		Trunk: &livekit.SIPOutboundTrunkInfo{
			Name:         "test-dial-trunk",
			Address:      freeSWITCHAddr,
			Numbers:      []string{freeSWITCHNumber},
			AuthUsername: "freeswitch",
			AuthPassword: "freeswitch",
		},
	})
	if err != nil {
		t.Fatalf("CreateSIPOutboundTrunk: %v", err)
	}
	defer func() {
		_, _ = sip.DeleteSIPTrunk(ctx, &livekit.DeleteSIPTrunkRequest{SipTrunkId: trunkRes.SipTrunkId})
	}()
	t.Logf("trunk created: %s", trunkRes.SipTrunkId)

	// 2. 尝试拨号到 FreeSWITCH 分机 1001
	//    注意：如果没有分机注册，会返回 SIP 错误（404/480 等），这是预期行为
	roomName := fmt.Sprintf("sip-call-%d", time.Now().UnixMilli())
	t.Logf("尝试拨号 1001，房间: %s", roomName)
	participant, err := sip.CreateSIPParticipant(ctx, &livekit.CreateSIPParticipantRequest{
		SipTrunkId:          trunkRes.SipTrunkId,
		SipCallTo:           "1001",
		RoomName:            roomName,
		ParticipantIdentity: "sip-1001",
		ParticipantName:     "Test SIP Call",
		WaitUntilAnswered:   false, // 不等待接听，立即返回
	})
	if err != nil {
		// 拨号失败是预期的（没有分机注册）
		t.Logf("拨号失败（预期，无分机注册）: %v", err)
	} else {
		t.Logf("拨号成功: identity=%s, room=%s, callID=%s",
			participant.ParticipantIdentity, participant.RoomName, participant.SipCallId)
	}
}
