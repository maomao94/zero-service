package planscope

import "testing"

func TestScopeLogMessage(t *testing.T) {
	tests := []struct {
		name  string
		scope Scope
		want  string
	}{
		{name: "cron lock", scope: CronLockScope(), want: "[cron-plan] message"},
		{name: "cron exec", scope: ExecCron(nil), want: "[cron-plan] message"},
		{name: "cron trigger", scope: TriggerScope(nil, nil), want: "[cron-plan] message"},
		{name: "rpc", scope: ExecScope(nil), want: "message"},
		{name: "callback", scope: CallbackScope(nil, nil, nil), want: "message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.scope.LogMessage("message"); got != tt.want {
				t.Fatalf("LogMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCronPlanLogMessage(t *testing.T) {
	if got, want := CronPlanLogMessage("message"), "[cron-plan] message"; got != want {
		t.Fatalf("CronPlanLogMessage() = %q, want %q", got, want)
	}
}
