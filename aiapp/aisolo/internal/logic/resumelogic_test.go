package logic

import (
	"context"
	"testing"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResumeLogic(t *testing.T) {
	svcCtx := &svc.ServiceContext{}
	ctx := context.Background()

	logic := NewResumeLogic(ctx, svcCtx)

	assert.NotNil(t, logic)
	assert.Equal(t, ctx, logic.ctx)
	assert.Equal(t, svcCtx, logic.svcCtx)
}

func TestResume_Validation(t *testing.T) {
	tests := []struct {
		name        string
		request     *aisolo.ResumeRequest
		expectError string
	}{
		{
			name:        "missing session_id",
			request:     &aisolo.ResumeRequest{InterruptId: "int-1"},
			expectError: "session_id is required",
		},
		{
			name:        "missing interrupt_id",
			request:     &aisolo.ResumeRequest{SessionId: "sess-1"},
			expectError: "interrupt_id is required",
		},
		{
			name: "valid request",
			request: &aisolo.ResumeRequest{
				SessionId:   "sess-1",
				InterruptId: "int-1",
			},
			expectError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			svcCtx := &svc.ServiceContext{}
			logic := NewResumeLogic(ctx, svcCtx)

			resp, err := logic.Resume(tt.request)

			if tt.expectError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectError)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, "sess-1", resp.SessionId)
			}
		})
	}
}

func TestResume_ActionApproval(t *testing.T) {
	ctx := context.Background()
	svcCtx := &svc.ServiceContext{}
	logic := NewResumeLogic(ctx, svcCtx)

	req := &aisolo.ResumeRequest{
		SessionId:   "test-session",
		InterruptId: "test-interrupt",
		UserId:      "test-user",
		Action:      aisolo.ResumeAction_RESUME_ACTION_APPROVE,
		SelectedIds: []string{"option-1", "option-2"},
	}

	resp, err := logic.Resume(req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "test-session", resp.SessionId)
}

func TestResume_ActionDeny(t *testing.T) {
	ctx := context.Background()
	svcCtx := &svc.ServiceContext{}
	logic := NewResumeLogic(ctx, svcCtx)

	req := &aisolo.ResumeRequest{
		SessionId:   "test-session",
		InterruptId: "test-interrupt",
		Action:      aisolo.ResumeAction_RESUME_ACTION_DENY,
		Reason:      "User rejected the action",
	}

	resp, err := logic.Resume(req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestResume_DefaultAction(t *testing.T) {
	ctx := context.Background()
	svcCtx := &svc.ServiceContext{}
	logic := NewResumeLogic(ctx, svcCtx)

	req := &aisolo.ResumeRequest{
		SessionId:   "test-session",
		InterruptId: "test-interrupt",
		Action:      aisolo.ResumeAction_RESUME_ACTION_UNSPECIFIED,
	}

	resp, err := logic.Resume(req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestResume_AnonymousUser(t *testing.T) {
	ctx := context.Background()
	svcCtx := &svc.ServiceContext{}
	logic := NewResumeLogic(ctx, svcCtx)

	req := &aisolo.ResumeRequest{
		SessionId:   "test-session",
		InterruptId: "test-interrupt",
	}

	resp, err := logic.Resume(req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestApprovalResult(t *testing.T) {
	t.Run("approve with selected ids", func(t *testing.T) {
		result := &ApprovalResult{
			Approved:    true,
			SelectedIds: []string{"id-1", "id-2"},
		}
		assert.True(t, result.Approved)
		assert.Len(t, result.SelectedIds, 2)
	})

	t.Run("deny with reason", func(t *testing.T) {
		result := &ApprovalResult{
			Approved:         false,
			DisapproveReason: "Not allowed",
		}
		assert.False(t, result.Approved)
		assert.Equal(t, "Not allowed", result.DisapproveReason)
	})
}
