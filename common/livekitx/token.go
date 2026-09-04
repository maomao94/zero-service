package livekitx

import (
	"strings"
	"time"

	"github.com/livekit/protocol/auth"
)

// JoinTokenOptions 定义指定房间和身份的视频入会授权。
type JoinTokenOptions struct {
	APIKey            string
	APISecret         string
	Room              string
	Identity          string
	Name              string
	ValidFor          time.Duration
	CanPublish        bool
	CanSubscribe      bool
	CanPublishData    bool
	CanPublishSources []string
}

// NewSIPToken 生成只包含 SIP 权限的管理/呼叫 token，不混用 VideoGrant。
func NewSIPToken(apiKey, apiSecret string, admin, call bool, validFor time.Duration) (string, error) {
	if strings.TrimSpace(apiKey) == "" || strings.TrimSpace(apiSecret) == "" || (!admin && !call) {
		return "", ErrInvalidTokenOptions
	}
	if validFor <= 0 {
		validFor = time.Hour
	}
	return auth.NewAccessToken(apiKey, apiSecret).SetValidFor(validFor).
		SetSIPGrant(&auth.SIPGrant{Admin: admin, Call: call}).ToJWT()
}

// NewJoinToken 生成指定房间和身份的最小视频入会 token。
func NewJoinToken(opts JoinTokenOptions) (string, error) {
	if strings.TrimSpace(opts.APIKey) == "" || strings.TrimSpace(opts.APISecret) == "" ||
		strings.TrimSpace(opts.Room) == "" || strings.TrimSpace(opts.Identity) == "" {
		return "", ErrInvalidTokenOptions
	}
	if opts.ValidFor <= 0 {
		opts.ValidFor = time.Hour
	}
	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     opts.Room,
	}
	grant.SetCanPublish(opts.CanPublish)
	grant.SetCanSubscribe(opts.CanSubscribe)
	grant.SetCanPublishData(opts.CanPublishData)
	if len(opts.CanPublishSources) > 0 {
		grant.CanPublishSources = opts.CanPublishSources
	}
	return auth.NewAccessToken(opts.APIKey, opts.APISecret).
		SetIdentity(opts.Identity).
		SetName(opts.Name).
		SetValidFor(opts.ValidFor).
		SetVideoGrant(grant).ToJWT()
}
