package runtime

import (
	"slices"

	"github.com/cloudwego/eino/schema"
)

// Messages builds the common system/history/user prompt shape used by simple runners.
func Messages(system string, history []*schema.Message, user string) []*schema.Message {
	msgs := make([]*schema.Message, 0, len(history)+2)
	if system != "" {
		msgs = append(msgs, schema.SystemMessage(system))
	}
	msgs = append(msgs, history...)
	if user != "" {
		msgs = append(msgs, schema.UserMessage(user))
	}
	return msgs
}

func LastText(msgs []*schema.Message) string {
	for _, msg := range slices.Backward(msgs) {
		if msg == nil || msg.Content == "" {
			continue
		}
		return msg.Content
	}
	return ""
}
