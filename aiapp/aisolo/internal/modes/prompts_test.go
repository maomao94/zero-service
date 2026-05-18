package modes

import (
	"strings"
	"testing"
)

func TestRuntimeAgentPromptDoesNotAdvertiseADKHumanTools(t *testing.T) {
	prompt := RuntimeAgentPrompt()
	for _, name := range []string{"ask_confirm", "ask_single_choice", "ask_multi_choice", "ask_text_input", "ask_form_input", "ask_info_ack"} {
		if strings.Contains(prompt, name) {
			t.Fatalf("RuntimeAgentPrompt() contains ADK human tool %q", name)
		}
	}
}
