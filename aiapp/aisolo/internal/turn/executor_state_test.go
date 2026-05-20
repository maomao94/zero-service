package turn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/session"
	einoxagent "zero-service/common/einox/agent"
	"zero-service/common/einox/memory"
	"zero-service/common/einox/protocol"
	einoxruntime "zero-service/common/einox/runtime"
)

func TestMarkSessionIdleClearsRunningState(t *testing.T) {
	ctx := context.Background()
	store := session.NewMemoryStore()
	sess := &session.Session{
		ID:            "sess-1",
		UserID:        "user-1",
		Status:        aisolo.SessionStatus_SESSION_STATUS_RUNNING,
		InterruptID:   "interrupt-1",
		RunOwner:      "worker-1",
		RunLeaseUntil: time.Now().Add(time.Minute),
	}
	if err := store.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	executor := &Executor{sessions: store}
	if err := executor.markSessionIdle(ctx, sess); err != nil {
		t.Fatalf("markSessionIdle() error = %v", err)
	}

	got, err := store.GetSession(ctx, sess.UserID, sess.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got.Status == aisolo.SessionStatus_SESSION_STATUS_RUNNING {
		t.Fatalf("session still RUNNING after failure recovery")
	}
	if got.Status != aisolo.SessionStatus_SESSION_STATUS_IDLE {
		t.Fatalf("Status = %v, want IDLE", got.Status)
	}
	if got.InterruptID != "" {
		t.Fatalf("InterruptID = %q, want empty", got.InterruptID)
	}
	if got.RunOwner != "" || !got.RunLeaseUntil.IsZero() {
		t.Fatalf("run lease not cleared: owner=%q lease=%v", got.RunOwner, got.RunLeaseUntil)
	}
}

func TestMarkSessionInterruptedRestoresResumeState(t *testing.T) {
	ctx := context.Background()
	store := session.NewMemoryStore()
	sess := &session.Session{
		ID:            "sess-1",
		UserID:        "user-1",
		Status:        aisolo.SessionStatus_SESSION_STATUS_RUNNING,
		InterruptID:   "interrupt-1",
		RunOwner:      "worker-1",
		RunLeaseUntil: time.Now().Add(time.Minute),
	}
	if err := store.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	executor := &Executor{sessions: store}
	if err := executor.markSessionInterrupted(ctx, sess, "interrupt-1"); err != nil {
		t.Fatalf("markSessionInterrupted() error = %v", err)
	}

	got, err := store.GetSession(ctx, sess.UserID, sess.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got.Status == aisolo.SessionStatus_SESSION_STATUS_RUNNING {
		t.Fatalf("session still RUNNING after resume failure recovery")
	}
	if got.Status != aisolo.SessionStatus_SESSION_STATUS_INTERRUPTED {
		t.Fatalf("Status = %v, want INTERRUPTED", got.Status)
	}
	if got.InterruptID != "interrupt-1" {
		t.Fatalf("InterruptID = %q, want original interrupt", got.InterruptID)
	}
	if got.RunOwner != "" || !got.RunLeaseUntil.IsZero() {
		t.Fatalf("run lease not cleared: owner=%q lease=%v", got.RunOwner, got.RunLeaseUntil)
	}
}

func TestMarkSessionRunningReturnsUpdateError(t *testing.T) {
	ctx := context.Background()
	base := session.NewMemoryStore()
	sess := &session.Session{
		ID:     "sess-1",
		UserID: "user-1",
		Mode:   aisolo.AgentMode_AGENT_MODE_AGENT,
		Status: aisolo.SessionStatus_SESSION_STATUS_IDLE,
	}
	if err := base.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	store := &failingSessionStore{Store: base, updateErr: fmt.Errorf("update failed")}
	executor := &Executor{sessions: store, runInstanceID: "worker-1", runLeaseTTL: time.Minute}

	err := executor.markSessionRunning(ctx, sess, true)
	if err == nil {
		t.Fatal("markSessionRunning() error = nil, want update failure")
	}
	if store.updateCalls != 1 {
		t.Fatalf("AcquireRun calls = %d, want 1", store.updateCalls)
	}
	got, err := base.GetSession(ctx, "user-1", "sess-1")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got.Status != aisolo.SessionStatus_SESSION_STATUS_IDLE {
		t.Fatalf("persisted status = %v, want original IDLE", got.Status)
	}
}

func TestAskUsesADKPoolForDefaultAgentModeWhenRuntimeRunnerConfigured(t *testing.T) {
	ctx := context.Background()
	sessions := session.NewMemoryStore()
	messages := memory.NewMemoryStorage()
	sess := &session.Session{
		ID:     "sess-runtime",
		UserID: "user-1",
		Mode:   aisolo.AgentMode_AGENT_MODE_AGENT,
		Status: aisolo.SessionStatus_SESSION_STATUS_IDLE,
	}
	if err := sessions.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	agent, err := einoxagent.NewChatModelAgent(ctx, einoxruntime.StaticChatModel{Chunks: []string{"hello ", "adk"}})
	if err != nil {
		t.Fatalf("NewChatModelAgent() error = %v", err)
	}
	runtimeErr := fmt.Errorf("runtime should not run")
	runner, err := einoxruntime.NewRunner(einoxruntime.StaticChatModel{Err: runtimeErr})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	pool := &singleAgentPool{agent: agent}
	writer := &collectingEventWriter{}
	executor := New(Config{
		Pool:                pool,
		Messages:            messages,
		Sessions:            sessions,
		RuntimeRunner:       runner,
		RuntimeSystemPrompt: "runtime system",
		RunInstanceID:       "worker-1",
		RunLeaseTTL:         time.Minute,
		RunNullLeaseGrace:   time.Minute,
	})

	err = executor.Ask(ctx, protocol.NewEmitter(writer, sess.ID, "turn-runtime"), AskInput{
		SessionID: sess.ID,
		UserID:    sess.UserID,
		Message:   "hello",
	})
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if pool.calls != 1 || len(pool.modes) != 1 || pool.modes[0] != aisolo.AgentMode_AGENT_MODE_AGENT {
		t.Fatalf("pool calls = %d modes = %#v, want one AGENT call", pool.calls, pool.modes)
	}
	assertEventTypes(t, writer.events,
		protocol.EventTurnStart,
		protocol.EventMessageStart,
		protocol.EventMessageDelta,
		protocol.EventMessageDelta,
		protocol.EventMessageEnd,
		protocol.EventTurnEnd,
	)
	end := eventData[protocol.TurnEndData](t, writer.events[len(writer.events)-1])
	if end.LastMessage != "hello adk" {
		t.Fatalf("TurnEnd.LastMessage = %q, want hello adk", end.LastMessage)
	}
	stored, err := messages.GetMessages(ctx, sess.UserID, sess.ID, 0)
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}
	if len(stored) != 2 || stored[0].Role != string(schema.User) || stored[0].Content != "hello" || stored[1].Role != string(schema.Assistant) || stored[1].Content != "hello adk" {
		t.Fatalf("stored messages = %#v, want user + ADK assistant", stored)
	}
	got, err := sessions.GetSession(ctx, sess.UserID, sess.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got.Status != aisolo.SessionStatus_SESSION_STATUS_IDLE || got.MessageCount != 2 || got.LastMessage != "hello adk" {
		t.Fatalf("session = %#v, want IDLE with assistant last message", got)
	}
}

func TestAskRuntimeHelperPassesSystemAndLimitedPreviousHistory(t *testing.T) {
	ctx := context.Background()
	sessions := session.NewMemoryStore()
	messages := memory.NewMemoryStorage()
	sess := &session.Session{
		ID:     "sess-runtime-history",
		UserID: "user-1",
		Mode:   aisolo.AgentMode_AGENT_MODE_AGENT,
		Status: aisolo.SessionStatus_SESSION_STATUS_IDLE,
	}
	if err := sessions.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	for _, msg := range []*memory.ConversationMessage{
		{ID: "m-1", SessionID: sess.ID, UserID: sess.UserID, Role: string(schema.User), Content: "old q1", CreatedAt: time.Unix(1, 0)},
		{ID: "m-2", SessionID: sess.ID, UserID: sess.UserID, Role: string(schema.Assistant), Content: "old a1", CreatedAt: time.Unix(2, 0)},
		{ID: "m-3", SessionID: sess.ID, UserID: sess.UserID, Role: string(schema.User), Content: "old q2", CreatedAt: time.Unix(3, 0)},
		{ID: "m-4", SessionID: sess.ID, UserID: sess.UserID, Role: string(schema.Assistant), Content: "old a2", CreatedAt: time.Unix(4, 0)},
	} {
		if err := messages.SaveMessage(ctx, msg); err != nil {
			t.Fatalf("SaveMessage() error = %v", err)
		}
	}
	modelCalls := &einoxruntime.ModelCalls{}
	runner, err := einoxruntime.NewRunner(einoxruntime.StaticChatModel{Chunks: []string{"runtime ok"}, Calls: modelCalls})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	executor := New(Config{
		Messages:                  messages,
		Sessions:                  sessions,
		RuntimeRunner:             runner,
		RuntimeSystemPrompt:       "runtime system",
		RuntimeMaxHistoryMessages: 2,
		RunInstanceID:             "worker-1",
		RunLeaseTTL:               time.Minute,
		RunNullLeaseGrace:         time.Minute,
	})

	err = executor.askRuntime(ctx, protocol.NewEmitter(&discardEventWriter{}, sess.ID, "turn-runtime-history"), AskInput{
		SessionID: sess.ID,
		UserID:    sess.UserID,
		Message:   "new question",
	}, sess, aisolo.AgentMode_AGENT_MODE_AGENT, time.Now())
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	got := modelCalls.StreamInput
	if len(got) != 4 {
		t.Fatalf("runtime model messages = %#v, want system + two history + current user", got)
	}
	if got[0].Role != schema.System || got[0].Content != "runtime system" {
		t.Fatalf("system message = %#v, want runtime system", got[0])
	}
	if got[1].Content != "old q2" || got[2].Content != "old a2" || got[3].Content != "new question" {
		t.Fatalf("runtime model messages = %#v, want limited previous history plus current input", got)
	}
}

func TestAskRuntimeHelperReturnsRestoreErrorWithRunError(t *testing.T) {
	ctx := context.Background()
	base := session.NewMemoryStore()
	messages := memory.NewMemoryStorage()
	sess := &session.Session{
		ID:     "sess-runtime-error",
		UserID: "user-1",
		Mode:   aisolo.AgentMode_AGENT_MODE_AGENT,
		Status: aisolo.SessionStatus_SESSION_STATUS_IDLE,
	}
	if err := base.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	runErr := fmt.Errorf("runtime failed")
	restoreErr := fmt.Errorf("restore idle failed")
	runner, err := einoxruntime.NewRunner(einoxruntime.StaticChatModel{Err: runErr})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	executor := New(Config{
		Messages:          messages,
		Sessions:          &runtimeRestoreFailStore{Store: base, restoreErr: restoreErr},
		RuntimeRunner:     runner,
		RunInstanceID:     "worker-1",
		RunLeaseTTL:       time.Minute,
		RunNullLeaseGrace: time.Minute,
	})

	err = executor.askRuntime(ctx, protocol.NewEmitter(&discardEventWriter{}, sess.ID, "turn-runtime-error"), AskInput{
		SessionID: sess.ID,
		UserID:    sess.UserID,
		Message:   "hello",
	}, sess, aisolo.AgentMode_AGENT_MODE_AGENT, time.Now())
	if !errors.Is(err, runErr) {
		t.Fatalf("Ask() error = %v, want runtime failure", err)
	}
	if !errors.Is(err, restoreErr) {
		t.Fatalf("Ask() error = %v, want restore failure joined", err)
	}
}

func TestResumeAlwaysUsesADKPoolEvenWhenRuntimeRunnerConfigured(t *testing.T) {
	ctx := context.Background()
	sessions := session.NewMemoryStore()
	sess := &session.Session{
		ID:          "sess-resume",
		UserID:      "user-1",
		Mode:        aisolo.AgentMode_AGENT_MODE_AGENT,
		Status:      aisolo.SessionStatus_SESSION_STATUS_INTERRUPTED,
		InterruptID: "interrupt-1",
	}
	if err := sessions.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if err := sessions.SaveInterrupt(ctx, &session.InterruptRecord{
		InterruptID: "interrupt-1",
		SessionID:   sess.ID,
		UserID:      sess.UserID,
		Kind:        aisolo.InterruptKind_INTERRUPT_KIND_APPROVAL,
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveInterrupt() error = %v", err)
	}
	runner, err := einoxruntime.NewRunner(einoxruntime.StaticChatModel{Response: "runtime should not run"})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	executor := New(Config{
		Pool:              failingModePool{},
		Messages:          memory.NewMemoryStorage(),
		Sessions:          sessions,
		RuntimeRunner:     runner,
		RunInstanceID:     "worker-1",
		RunLeaseTTL:       time.Minute,
		RunNullLeaseGrace: time.Minute,
	})

	err = executor.Resume(ctx, protocol.NewEmitter(&discardEventWriter{}, sess.ID, "turn-resume"), ResumeInput{
		SessionID:   sess.ID,
		UserID:      sess.UserID,
		InterruptID: sess.InterruptID,
		Action:      aisolo.ResumeAction_RESUME_ACTION_YES,
	})
	if err == nil || !strings.Contains(err.Error(), "pool should not be used") {
		t.Fatalf("Resume() error = %v, want ADK pool path failure", err)
	}
}

func TestADKRunOptionsPassesAgentModelOptions(t *testing.T) {
	ctx := context.Background()
	temp := float32(0.42)
	modelCalls := &einoxruntime.ModelCalls{}
	agent, err := einoxagent.NewChatModelAgent(ctx,
		einoxruntime.StaticChatModel{Chunks: []string{"ok"}, Calls: modelCalls},
		einoxagent.WithModelOption(model.WithTemperature(temp)),
	)
	if err != nil {
		t.Fatalf("NewChatModelAgent() error = %v", err)
	}

	writer := &collectingEventWriter{}
	iter := agent.Runner().Run(ctx, []adk.Message{schema.UserMessage("hello")}, adkRunOptions(agent, adk.WithCheckPointID("cp-1"))...)
	_, err = protocol.PipeEvents(protocol.NewEmitter(writer, "session-1", "turn-1"), iter, protocol.PipeOptions{})
	if err != nil {
		t.Fatalf("PipeEvents() error = %v", err)
	}
	if modelCalls.StreamInput == nil {
		t.Fatal("model stream was not called")
	}
	gotOpts := model.GetCommonOptions(&model.Options{}, modelCalls.StreamOptions...)
	if gotOpts.Temperature == nil || *gotOpts.Temperature != temp {
		t.Fatalf("Temperature = %#v, want %v", gotOpts.Temperature, temp)
	}
}

func TestAskSkipsRuntimeRunnerForNonDefaultMode(t *testing.T) {
	ctx := context.Background()
	sessions := session.NewMemoryStore()
	messages := memory.NewMemoryStorage()
	sess := &session.Session{
		ID:     "sess-plan",
		UserID: "user-1",
		Mode:   aisolo.AgentMode_AGENT_MODE_PLAN,
		Status: aisolo.SessionStatus_SESSION_STATUS_IDLE,
	}
	if err := sessions.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	runner, err := einoxruntime.NewRunner(einoxruntime.StaticChatModel{Response: "runtime should not run"})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	executor := New(Config{
		Pool:              failingModePool{},
		Messages:          messages,
		Sessions:          sessions,
		RuntimeRunner:     runner,
		RunInstanceID:     "worker-1",
		RunLeaseTTL:       time.Minute,
		RunNullLeaseGrace: time.Minute,
	})

	err = executor.Ask(ctx, protocol.NewEmitter(&discardEventWriter{}, sess.ID, "turn-plan"), AskInput{
		SessionID: sess.ID,
		UserID:    sess.UserID,
		Message:   "hello",
	})
	if err == nil || !strings.Contains(err.Error(), "pool should not be used") {
		t.Fatalf("Ask() error = %v, want ADK pool path failure", err)
	}
	stored, err := messages.GetMessages(ctx, sess.UserID, sess.ID, 0)
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}
	if len(stored) != 0 {
		t.Fatalf("stored messages = %#v, want none before ADK build failure", stored)
	}
}

func TestRuntimeHistoryExcludesCurrentUserMessage(t *testing.T) {
	history := []*schema.Message{
		schema.UserMessage("old question"),
		schema.AssistantMessage("old answer", nil),
		schema.UserMessage("new question"),
	}
	got := runtimeHistory(history, "new question", 0)
	if len(got) != 2 || got[len(got)-1].Content != "old answer" {
		t.Fatalf("runtimeHistory() = %#v, want previous history only", got)
	}
}

func TestRuntimeHistoryLimitsPreviousMessages(t *testing.T) {
	history := []*schema.Message{
		schema.UserMessage("q1"),
		schema.AssistantMessage("a1", nil),
		schema.UserMessage("q2"),
		schema.AssistantMessage("a2", nil),
		schema.UserMessage("new question"),
	}
	got := runtimeHistory(history, "new question", 2)
	if len(got) != 2 || got[0].Content != "q2" || got[1].Content != "a2" {
		t.Fatalf("runtimeHistory() = %#v, want last two previous messages", got)
	}
}

func TestEmitRuntimeEventsSkipsRunnerTurnStart(t *testing.T) {
	writer := &collectingEventWriter{}
	em := protocol.NewEmitter(writer, "session-test-001", "turn-test-001")
	turnEnd := protocol.TurnEndData{LastMessage: "done"}
	turnEndJSON, err := json.Marshal(turnEnd)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	got, err := emitRuntimeEvents(em, []protocol.Event{
		{Type: protocol.EventTurnStart},
		{Type: protocol.EventMessageStart, Data: json.RawMessage(`{"role":"assistant"}`)},
		{Type: protocol.EventTurnEnd, Data: turnEndJSON},
	})
	if err != nil {
		t.Fatalf("emitRuntimeEvents() error = %v", err)
	}
	if len(writer.events) != 1 || writer.events[0].Type != protocol.EventMessageStart {
		t.Fatalf("emitted events = %#v, want only message_start", writer.events)
	}
	if got.LastMessage != "done" {
		t.Fatalf("TurnEnd.LastMessage = %q, want done", got.LastMessage)
	}
}

func TestConcurrentAskRunAcquireAllowsOneWinner(t *testing.T) {
	ctx := context.Background()
	base := session.NewMemoryStore()
	sess := &session.Session{
		ID:     "sess-1",
		UserID: "user-1",
		Mode:   aisolo.AgentMode_AGENT_MODE_AGENT,
		Status: aisolo.SessionStatus_SESSION_STATUS_IDLE,
	}
	if err := base.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	store := newBarrierGetSessionStore(base, 2)
	messages := newBlockingSaveStorage(fmt.Errorf("history failed"))
	executor := &Executor{
		sessions:          store,
		pool:              staticModePool{},
		messages:          messages,
		runInstanceID:     "worker-1",
		runLeaseTTL:       time.Minute,
		runNullLeaseGrace: time.Minute,
	}

	results := make(chan error, 2)
	go func() {
		results <- executor.Ask(ctx, protocol.NewEmitter(&discardEventWriter{}, "sess-1", "turn-1"), AskInput{
			SessionID: "sess-1",
			UserID:    "user-1",
			Message:   "first",
		})
	}()
	go func() {
		results <- executor.Ask(ctx, protocol.NewEmitter(&discardEventWriter{}, "sess-1", "turn-2"), AskInput{
			SessionID: "sess-1",
			UserID:    "user-1",
			Message:   "second",
		})
	}()
	store.waitLoaded(t)
	store.release()
	messages.waitStarted(t)

	select {
	case err := <-results:
		if err == nil || !strings.Contains(err.Error(), "run acquire conflict") {
			t.Fatalf("losing Ask error = %v, want run acquire conflict", err)
		}
	case <-time.After(time.Second):
		t.Fatal("losing Ask did not fail while winner held RUNNING")
	}
	messages.release()
	select {
	case err := <-results:
		if err == nil || !strings.Contains(err.Error(), "history failed") {
			t.Fatalf("winning Ask error = %v, want history failure", err)
		}
	case <-time.After(time.Second):
		t.Fatal("winning Ask did not finish after releasing SaveMessage")
	}
	if messages.saveCalls != 1 {
		t.Fatalf("SaveMessage calls = %d, want only first Ask to save", messages.saveCalls)
	}
	got, err := base.GetSession(ctx, "user-1", "sess-1")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got.Status != aisolo.SessionStatus_SESSION_STATUS_IDLE {
		t.Fatalf("Status = %v, want IDLE after first Ask restore", got.Status)
	}
}

func TestConcurrentResumeRunAcquireAllowsOneWinner(t *testing.T) {
	ctx := context.Background()
	store := session.NewMemoryStore()
	sess := &session.Session{
		ID:          "sess-1",
		UserID:      "user-1",
		Mode:        aisolo.AgentMode_AGENT_MODE_AGENT,
		Status:      aisolo.SessionStatus_SESSION_STATUS_INTERRUPTED,
		InterruptID: "interrupt-1",
	}
	if err := store.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	executor := &Executor{sessions: store, runInstanceID: "worker-1", runLeaseTTL: time.Minute}

	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			loaded, err := store.GetSession(ctx, "user-1", "sess-1")
			if err != nil {
				results <- err
				return
			}
			<-start
			results <- executor.markSessionRunning(ctx, loaded, false)
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	conflicts := 0
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		if strings.Contains(err.Error(), "run acquire conflict") {
			conflicts++
			continue
		}
		t.Fatalf("unexpected acquire error: %v", err)
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("acquire results successes=%d conflicts=%d, want 1/1", successes, conflicts)
	}
	got, err := store.GetSession(ctx, "user-1", "sess-1")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got.Status != aisolo.SessionStatus_SESSION_STATUS_RUNNING {
		t.Fatalf("Status = %v, want RUNNING", got.Status)
	}
	if got.InterruptID != "interrupt-1" {
		t.Fatalf("InterruptID = %q, want preserved interrupt", got.InterruptID)
	}
}

func TestMemoryStoreAcquireRunUsesCurrentStatus(t *testing.T) {
	ctx := context.Background()
	store := session.NewMemoryStore()
	sess := &session.Session{ID: "sess-1", UserID: "user-1", Status: aisolo.SessionStatus_SESSION_STATUS_IDLE}
	if err := store.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	loaded, err := store.GetSession(ctx, "user-1", "sess-1")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	loaded.Status = aisolo.SessionStatus_SESSION_STATUS_INTERRUPTED

	err = store.AcquireRun(ctx, loaded, aisolo.SessionStatus_SESSION_STATUS_INTERRUPTED, "worker-1", time.Now().Add(time.Minute), false)
	if err == nil || !strings.Contains(err.Error(), "run acquire conflict") {
		t.Fatalf("AcquireRun() error = %v, want conflict against persisted IDLE", err)
	}
}

func TestStaleRunningRecoveryFailureDoesNotProceed(t *testing.T) {
	ctx := context.Background()
	base := session.NewMemoryStore()
	sess := &session.Session{
		ID:            "sess-1",
		UserID:        "user-1",
		Mode:          aisolo.AgentMode_AGENT_MODE_AGENT,
		Status:        aisolo.SessionStatus_SESSION_STATUS_RUNNING,
		RunLeaseUntil: time.Now().Add(-time.Minute),
	}
	if err := base.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	store := &failingSessionStore{Store: base, updateErr: fmt.Errorf("recover failed")}
	executor := &Executor{sessions: store}

	err := executor.markSessionIdle(ctx, sess)
	if err == nil {
		t.Fatal("markSessionIdle() error = nil, want recovery failure")
	}
	got, err := base.GetSession(ctx, "user-1", "sess-1")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got.Status != aisolo.SessionStatus_SESSION_STATUS_RUNNING {
		t.Fatalf("persisted status = %v, want RUNNING", got.Status)
	}
}

func TestPersistInterruptReturnsSaveError(t *testing.T) {
	ctx := context.Background()
	base := session.NewMemoryStore()
	sess := &session.Session{
		ID:     "sess-1",
		UserID: "user-1",
		Mode:   aisolo.AgentMode_AGENT_MODE_AGENT,
		Status: aisolo.SessionStatus_SESSION_STATUS_RUNNING,
	}
	if err := base.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	store := &failingSessionStore{Store: base, saveInterruptErr: fmt.Errorf("save interrupt failed")}
	executor := &Executor{sessions: store}

	err := executor.persistInterrupt(ctx, sess, protocol.RunResult{
		HasInterrupt:  true,
		InterruptID:   "interrupt-1",
		InterruptKind: protocol.InterruptApproval,
		Interrupt: &protocol.InterruptData{
			InterruptID: "interrupt-1",
			Kind:        protocol.InterruptApproval,
			Question:    "approve?",
		},
	})
	if err == nil {
		t.Fatal("persistInterrupt() error = nil, want save failure")
	}
	if _, getErr := base.GetInterrupt(ctx, "interrupt-1"); getErr == nil {
		t.Fatal("interrupt record was saved despite failing SaveInterrupt wrapper")
	}
	got, err := base.GetSession(ctx, "user-1", "sess-1")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got.Status != aisolo.SessionStatus_SESSION_STATUS_RUNNING {
		t.Fatalf("Status = %v, want original RUNNING", got.Status)
	}
	if got.InterruptID != "" {
		t.Fatalf("InterruptID = %q, want empty after failed SaveInterrupt", got.InterruptID)
	}
}

func TestAskRestoresIdleWhenLoadHistoryFails(t *testing.T) {
	ctx := context.Background()
	base := session.NewMemoryStore()
	sess := &session.Session{
		ID:     "sess-1",
		UserID: "user-1",
		Mode:   aisolo.AgentMode_AGENT_MODE_AGENT,
		Status: aisolo.SessionStatus_SESSION_STATUS_IDLE,
	}
	if err := base.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	executor := &Executor{
		sessions: base,
		pool:     staticModePool{},
		messages: failingMemoryStorage{getErr: fmt.Errorf("history failed")},
		metrics:  nil,
	}

	err := executor.Ask(ctx, protocol.NewEmitter(&discardEventWriter{}, "sess-1", "turn-1"), AskInput{
		SessionID: "sess-1",
		UserID:    "user-1",
		Message:   "hello",
	})
	if err == nil {
		t.Fatal("Ask() error = nil, want load history failure")
	}
	got, err := base.GetSession(ctx, "user-1", "sess-1")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got.Status != aisolo.SessionStatus_SESSION_STATUS_IDLE {
		t.Fatalf("Status = %v, want IDLE", got.Status)
	}
	if got.RunOwner != "" || !got.RunLeaseUntil.IsZero() {
		t.Fatalf("run lease not cleared: owner=%q lease=%v", got.RunOwner, got.RunLeaseUntil)
	}
}

func TestAskDoesNotSaveUserMessageWhenMarkRunningFails(t *testing.T) {
	ctx := context.Background()
	base := session.NewMemoryStore()
	sess := &session.Session{
		ID:     "sess-1",
		UserID: "user-1",
		Mode:   aisolo.AgentMode_AGENT_MODE_AGENT,
		Status: aisolo.SessionStatus_SESSION_STATUS_IDLE,
	}
	if err := base.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	store := &failingSessionStore{Store: base, updateErr: fmt.Errorf("mark running failed")}
	messages := &countingMemoryStorage{}
	executor := &Executor{
		sessions:          store,
		pool:              staticModePool{},
		messages:          messages,
		runInstanceID:     "worker-1",
		runLeaseTTL:       time.Minute,
		runNullLeaseGrace: time.Minute,
	}

	err := executor.Ask(ctx, protocol.NewEmitter(&discardEventWriter{}, "sess-1", "turn-1"), AskInput{
		SessionID: "sess-1",
		UserID:    "user-1",
		Message:   "hello",
	})
	if err == nil {
		t.Fatal("Ask() error = nil, want mark running failure")
	}
	if messages.saveCalls != 0 {
		t.Fatalf("SaveMessage calls = %d, want 0", messages.saveCalls)
	}
	got, err := base.GetSession(ctx, "user-1", "sess-1")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got.Status != aisolo.SessionStatus_SESSION_STATUS_IDLE {
		t.Fatalf("Status = %v, want original IDLE", got.Status)
	}
}

type failingSessionStore struct {
	session.Store
	updateErr        error
	saveInterruptErr error
	updateCalls      int
}

type runtimeRestoreFailStore struct {
	session.Store
	restoreErr error
}

func (s *runtimeRestoreFailStore) UpdateSession(ctx context.Context, sess *session.Session) error {
	if sess.Status == aisolo.SessionStatus_SESSION_STATUS_IDLE && s.restoreErr != nil {
		return s.restoreErr
	}
	return s.Store.UpdateSession(ctx, sess)
}

func (s *failingSessionStore) UpdateSession(ctx context.Context, sess *session.Session) error {
	s.updateCalls++
	if s.updateErr != nil {
		return s.updateErr
	}
	return s.Store.UpdateSession(ctx, sess)
}

func (s *failingSessionStore) AcquireRun(ctx context.Context, sess *session.Session, expected aisolo.SessionStatus, owner string, leaseUntil time.Time, clearInterrupt bool) error {
	s.updateCalls++
	if s.updateErr != nil {
		return s.updateErr
	}
	return s.Store.AcquireRun(ctx, sess, expected, owner, leaseUntil, clearInterrupt)
}

func (s *failingSessionStore) SaveInterrupt(ctx context.Context, rec *session.InterruptRecord) error {
	if s.saveInterruptErr != nil {
		return s.saveInterruptErr
	}
	return s.Store.SaveInterrupt(ctx, rec)
}

type failingMemoryStorage struct {
	getErr error
}

func (s failingMemoryStorage) SaveMessage(context.Context, *memory.ConversationMessage) error {
	return nil
}

func (s failingMemoryStorage) GetMessages(context.Context, string, string, int) ([]*memory.ConversationMessage, error) {
	return nil, s.getErr
}

func (s failingMemoryStorage) DeleteSession(context.Context, string, string) error { return nil }

func (s failingMemoryStorage) Close() error { return nil }

type countingMemoryStorage struct {
	saveCalls int
}

func (s *countingMemoryStorage) SaveMessage(context.Context, *memory.ConversationMessage) error {
	s.saveCalls++
	return nil
}

func (s *countingMemoryStorage) GetMessages(context.Context, string, string, int) ([]*memory.ConversationMessage, error) {
	return nil, nil
}

func (s *countingMemoryStorage) DeleteSession(context.Context, string, string) error { return nil }

func (s *countingMemoryStorage) Close() error { return nil }

type barrierGetSessionStore struct {
	session.Store
	target    int
	mu        sync.Mutex
	loaded    int
	ready     chan struct{}
	releaseCh chan struct{}
	once      sync.Once
}

func newBarrierGetSessionStore(store session.Store, target int) *barrierGetSessionStore {
	return &barrierGetSessionStore{Store: store, target: target, ready: make(chan struct{}), releaseCh: make(chan struct{})}
}

func (s *barrierGetSessionStore) GetSession(ctx context.Context, userID, sessionID string) (*session.Session, error) {
	sess, err := s.Store.GetSession(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.loaded++
	if s.loaded == s.target {
		s.once.Do(func() { close(s.ready) })
	}
	shouldWait := s.loaded <= s.target
	s.mu.Unlock()
	if shouldWait {
		<-s.releaseCh
	}
	return sess, nil
}

func (s *barrierGetSessionStore) waitLoaded(t *testing.T) {
	t.Helper()
	select {
	case <-s.ready:
	case <-time.After(time.Second):
		t.Fatal("Ask calls did not both load the session")
	}
}

func (s *barrierGetSessionStore) release() { close(s.releaseCh) }

type blockingSaveStorage struct {
	started   chan struct{}
	releaseCh chan struct{}
	err       error
	once      sync.Once
	saveCalls int
}

func newBlockingSaveStorage(err error) *blockingSaveStorage {
	return &blockingSaveStorage{
		started:   make(chan struct{}),
		releaseCh: make(chan struct{}),
		err:       err,
	}
}

func (s *blockingSaveStorage) SaveMessage(context.Context, *memory.ConversationMessage) error {
	s.once.Do(func() { close(s.started) })
	<-s.releaseCh
	s.saveCalls++
	return nil
}

func (s *blockingSaveStorage) GetMessages(context.Context, string, string, int) ([]*memory.ConversationMessage, error) {
	return nil, s.err
}

func (s *blockingSaveStorage) DeleteSession(context.Context, string, string) error { return nil }

func (s *blockingSaveStorage) Close() error { return nil }

func (s *blockingSaveStorage) waitStarted(t *testing.T) {
	t.Helper()
	select {
	case <-s.started:
	case <-time.After(time.Second):
		t.Fatal("winning Ask did not reach blocked SaveMessage")
	}
}

func (s *blockingSaveStorage) release() { close(s.releaseCh) }

type staticModePool struct{}

func (staticModePool) Get(context.Context, aisolo.AgentMode) (*einoxagent.Agent, error) {
	return &einoxagent.Agent{}, nil
}

type singleAgentPool struct {
	agent *einoxagent.Agent
	calls int
	modes []aisolo.AgentMode
}

func (p *singleAgentPool) Get(_ context.Context, mode aisolo.AgentMode) (*einoxagent.Agent, error) {
	p.calls++
	p.modes = append(p.modes, mode)
	return p.agent, nil
}

type failingModePool struct{}

func (failingModePool) Get(context.Context, aisolo.AgentMode) (*einoxagent.Agent, error) {
	return nil, fmt.Errorf("pool should not be used")
}

type collectingEventWriter struct {
	events []protocol.Event
}

func (w *collectingEventWriter) Write(p []byte) (int, error) {
	event, err := protocol.Decode(p)
	if err != nil {
		return 0, err
	}
	w.events = append(w.events, event)
	return len(p), nil
}

func assertEventTypes(t *testing.T, events []protocol.Event, want ...protocol.EventType) {
	t.Helper()
	if len(events) != len(want) {
		t.Fatalf("event count = %d, want %d: %#v", len(events), len(want), events)
	}
	for i, event := range events {
		if event.Type != want[i] {
			t.Fatalf("event[%d].Type = %q, want %q", i, event.Type, want[i])
		}
	}
}

func eventData[T any](t *testing.T, event protocol.Event) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(event.Data, &out); err != nil {
		t.Fatalf("decode event data: %v", err)
	}
	return out
}

type discardEventWriter struct{}

func (w *discardEventWriter) Write(p []byte) (int, error) { return len(p), nil }
