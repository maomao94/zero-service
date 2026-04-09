/*
 * Copyright 2026 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package a2ui

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type toolCallInfo struct {
	Name string
	Args string
}

func StreamToWriter(w io.Writer, sessionID string, history []*schema.Message, events *adk.AsyncIterator[*adk.AgentEvent]) (string, string, error) {
	if w == nil {
		return "", "", errors.New("writer is nil")
	}
	if events == nil {
		return "", "", errors.New("events iterator is nil")
	}

	surfaceID := "chat-" + sessionID

	rootChildren := make([]string, 0, len(history))
	for i := range history {
		rootChildren = append(rootChildren, fmt.Sprintf("msg-%d-card", i))
	}

	if err := emit(w, Message{
		BeginRendering: &BeginRenderingMsg{SurfaceID: surfaceID, Root: "root-col"},
	}); err != nil {
		return "", "", fmt.Errorf("emit begin rendering: %w", err)
	}

	if err := emitHistory(w, surfaceID, history, rootChildren); err != nil {
		return "", "", fmt.Errorf("emit history: %w", err)
	}

	msgIdx := len(history)
	return streamEvents(w, surfaceID, &rootChildren, &msgIdx, events)
}

func streamEvents(w io.Writer, surfaceID string, rootChildren *[]string, msgIdx *int, events *adk.AsyncIterator[*adk.AgentEvent]) (string, string, error) {
	var lastContent strings.Builder
	var interruptID string

	for {
		event, ok := events.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			if err := emitToolChip(w, surfaceID, rootChildren, msgIdx, "error", event.Err.Error()); err != nil {
				return lastContent.String(), "", fmt.Errorf("emit error chip: %w", err)
			}
			return lastContent.String(), "", event.Err
		}

		if event.Action != nil && event.Action.Interrupted != nil {
			ictxs := event.Action.Interrupted.InterruptContexts
			var desc string
			for _, ic := range ictxs {
				if ic.IsRootCause {
					interruptID = ic.ID
					desc = fmt.Sprintf("%v", ic.Info)
					break
				}
			}
			if interruptID == "" && len(ictxs) > 0 {
				interruptID = ictxs[0].ID
				desc = fmt.Sprintf("%v", ictxs[0].Info)
			}
			if err := emitToolChip(w, surfaceID, rootChildren, msgIdx, "approval needed", desc); err != nil {
				return lastContent.String(), "", fmt.Errorf("emit approval chip: %w", err)
			}
			if err := emit(w, Message{
				InterruptRequest: &InterruptRequestMsg{
					InterruptID: interruptID,
					Description: desc,
				},
			}); err != nil {
				return lastContent.String(), "", fmt.Errorf("emit interrupt request: %w", err)
			}
			break
		}

		hasOutput := event.Output != nil && event.Output.MessageOutput != nil
		hasExit := event.Action != nil && event.Action.Exit

		if !hasOutput {
			if hasExit {
				break
			}
			continue
		}

		mo := event.Output.MessageOutput
		role := mo.Role
		if role == "" && mo.Message != nil {
			role = mo.Message.Role
		}

		switch role {
		case schema.Tool:
			content := drainToolResult(mo)
			if err := emitToolChip(w, surfaceID, rootChildren, msgIdx, "tool result", content); err != nil {
				return lastContent.String(), "", fmt.Errorf("emit tool chip: %w", err)
			}

		default:
			if mo.IsStreaming && mo.MessageStream != nil {
				textIdx := *msgIdx
				cardID := fmt.Sprintf("msg-%d-card", textIdx)
				colID := fmt.Sprintf("msg-%d-col", textIdx)
				roleID := fmt.Sprintf("msg-%d-role", textIdx)
				contentID := fmt.Sprintf("msg-%d-content", textIdx)
				dataKey := fmt.Sprintf("%s/msg-%d", surfaceID, textIdx)

				nameByIdx := map[int]string{}
				argsByIdx := map[int]*strings.Builder{}
				var tcOrder []int
				seenTCIdx := map[int]bool{}

				var shellEmitted bool
				var accContent strings.Builder

				for {
					chunk, recvErr := mo.MessageStream.Recv()
					if errors.Is(recvErr, io.EOF) {
						break
					}
					if recvErr != nil {
						break
					}

					for _, tc := range chunk.ToolCalls {
						idx := 0
						if tc.Index != nil {
							idx = *tc.Index
						}
						if !seenTCIdx[idx] {
							seenTCIdx[idx] = true
							tcOrder = append(tcOrder, idx)
						}
						if tc.Function.Name != "" && nameByIdx[idx] == "" {
							nameByIdx[idx] = tc.Function.Name
						}
						if tc.Function.Arguments != "" {
							if argsByIdx[idx] == nil {
								argsByIdx[idx] = &strings.Builder{}
							}
							argsByIdx[idx].WriteString(tc.Function.Arguments)
						}
					}

					if chunk.Content != "" {
						if !shellEmitted {
							*rootChildren = append(*rootChildren, cardID)
							*msgIdx++
							if shellErr := emitMessageShell(w, surfaceID, *rootChildren, cardID, colID, roleID, contentID, dataKey, roleToLabel(role)); shellErr != nil {
								return lastContent.String(), "", fmt.Errorf("emit message shell: %w", shellErr)
							}
							shellEmitted = true
						}
						accContent.WriteString(chunk.Content)
						if dataErr := emitDataUpdate(w, surfaceID, dataKey, accContent.String()); dataErr != nil {
							return lastContent.String(), "", fmt.Errorf("emit data update: %w", dataErr)
						}
					}
				}

				var toolCalls []toolCallInfo
				for _, i := range tcOrder {
					name := nameByIdx[i]
					if name == "" {
						continue
					}
					args := ""
					if ab := argsByIdx[i]; ab != nil {
						args = ab.String()
					}
					toolCalls = append(toolCalls, toolCallInfo{Name: name, Args: args})
				}

				for _, tc := range toolCalls {
					if err := emitToolChip(w, surfaceID, rootChildren, msgIdx, "tool call", formatToolCall(tc)); err != nil {
						return lastContent.String(), "", fmt.Errorf("emit tool call chip: %w", err)
					}
				}
				if shellEmitted {
					lastContent.Reset()
					lastContent.WriteString(accContent.String())
				}

			} else if mo.Message != nil {
				msg := mo.Message

				for _, tc := range msg.ToolCalls {
					if err := emitToolChip(w, surfaceID, rootChildren, msgIdx, "tool call", formatToolCall(toolCallInfo{
						Name: tc.Function.Name,
						Args: tc.Function.Arguments,
					})); err != nil {
						return lastContent.String(), "", fmt.Errorf("emit tool call chip: %w", err)
					}
				}
				if msg.Content != "" {
					if err := emitTextCard(w, surfaceID, rootChildren, msgIdx, roleToLabel(role), msg.Content); err != nil {
						return lastContent.String(), "", fmt.Errorf("emit text card: %w", err)
					}
					lastContent.Reset()
					lastContent.WriteString(msg.Content)
				}
			}
		}

		if hasExit {
			break
		}
	}

	return lastContent.String(), interruptID, nil
}

func drainToolResult(mo *adk.MessageVariant) string {
	if mo.IsStreaming && mo.MessageStream != nil {
		content, _ := consumeStream(mo.MessageStream)
		return content
	}
	if mo.Message != nil {
		return mo.Message.Content
	}
	return ""
}

func consumeStream(stream *schema.StreamReader[*schema.Message]) (string, []toolCallInfo) {
	nameByIdx := map[int]string{}
	argsByIdx := map[int]*strings.Builder{}
	var order []int
	seenIdx := map[int]bool{}
	var buf strings.Builder

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			break
		}
		if chunk.Content != "" {
			buf.WriteString(chunk.Content)
		}
		for _, tc := range chunk.ToolCalls {
			idx := 0
			if tc.Index != nil {
				idx = *tc.Index
			}
			if !seenIdx[idx] {
				seenIdx[idx] = true
				order = append(order, idx)
			}
			if tc.Function.Name != "" && nameByIdx[idx] == "" {
				nameByIdx[idx] = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				if argsByIdx[idx] == nil {
					argsByIdx[idx] = &strings.Builder{}
				}
				argsByIdx[idx].WriteString(tc.Function.Arguments)
			}
		}
	}

	var toolCalls []toolCallInfo
	for _, idx := range order {
		name := nameByIdx[idx]
		if name == "" {
			continue
		}
		args := ""
		if ab := argsByIdx[idx]; ab != nil {
			args = ab.String()
		}
		toolCalls = append(toolCalls, toolCallInfo{Name: name, Args: args})
	}
	return buf.String(), toolCalls
}

func emitHistory(w io.Writer, surfaceID string, history []*schema.Message, rootChildren []string) error {
	comps := []Component{
		{ID: "root-col", Component: ComponentValue{Column: &ColumnComp{Children: append([]string{}, rootChildren...)}}},
	}
	for i, msg := range history {
		cardID := fmt.Sprintf("msg-%d-card", i)
		colID := fmt.Sprintf("msg-%d-col", i)
		roleID := fmt.Sprintf("msg-%d-role", i)
		contentID := fmt.Sprintf("msg-%d-content", i)
		comps = append(comps,
			Component{ID: cardID, Component: ComponentValue{Card: &CardComp{Children: []string{colID}}}},
			Component{ID: colID, Component: ComponentValue{Column: &ColumnComp{Children: []string{roleID, contentID}}}},
			Component{ID: roleID, Component: ComponentValue{Text: &TextComp{Value: roleToLabel(msg.Role), UsageHint: "caption"}}},
			Component{ID: contentID, Component: ComponentValue{Text: &TextComp{Value: msg.Content, UsageHint: "body"}}},
		)
	}
	return emit(w, Message{SurfaceUpdate: &SurfaceUpdateMsg{SurfaceID: surfaceID, Components: comps}})
}

func emitMessageShell(w io.Writer, surfaceID string, rootChildren []string, cardID, colID, roleID, contentID, dataKey, roleLabel string) error {
	return emit(w, Message{
		SurfaceUpdate: &SurfaceUpdateMsg{
			SurfaceID: surfaceID,
			Components: []Component{
				{ID: "root-col", Component: ComponentValue{Column: &ColumnComp{Children: append([]string{}, rootChildren...)}}},
				{ID: cardID, Component: ComponentValue{Card: &CardComp{Children: []string{colID}}}},
				{ID: colID, Component: ComponentValue{Column: &ColumnComp{Children: []string{roleID, contentID}}}},
				{ID: roleID, Component: ComponentValue{Text: &TextComp{Value: roleLabel, UsageHint: "caption"}}},
				{ID: contentID, Component: ComponentValue{Text: &TextComp{DataKey: dataKey, UsageHint: "body"}}},
			},
		},
	})
}

func emitDataUpdate(w io.Writer, surfaceID, dataKey, content string) error {
	return emit(w, Message{
		DataModelUpdate: &DataModelUpdateMsg{
			SurfaceID: surfaceID,
			Contents:  []DataContent{{Key: dataKey, ValueString: content}},
		},
	})
}

func emitTextCard(w io.Writer, surfaceID string, rootChildren *[]string, msgIdx *int, roleLabel, content string) error {
	idx := *msgIdx
	cardID := fmt.Sprintf("msg-%d-card", idx)
	colID := fmt.Sprintf("msg-%d-col", idx)
	roleID := fmt.Sprintf("msg-%d-role", idx)
	contentID := fmt.Sprintf("msg-%d-content", idx)
	dataKey := fmt.Sprintf("%s/msg-%d", surfaceID, idx)

	*rootChildren = append(*rootChildren, cardID)
	*msgIdx++

	if err := emitMessageShell(w, surfaceID, *rootChildren, cardID, colID, roleID, contentID, dataKey, roleLabel); err != nil {
		return err
	}
	return emitDataUpdate(w, surfaceID, dataKey, content)
}

func emitToolChip(w io.Writer, surfaceID string, rootChildren *[]string, msgIdx *int, kind, text string) error {
	idx := *msgIdx
	cardID := fmt.Sprintf("msg-%d-card", idx)
	colID := fmt.Sprintf("msg-%d-col", idx)
	labelID := fmt.Sprintf("msg-%d-label", idx)
	textID := fmt.Sprintf("msg-%d-text", idx)

	*rootChildren = append(*rootChildren, cardID)
	*msgIdx++

	display := text
	if kind != "approval needed" && len([]rune(display)) > 300 {
		display = string([]rune(display)[:300]) + "…"
	}

	return emit(w, Message{
		SurfaceUpdate: &SurfaceUpdateMsg{
			SurfaceID: surfaceID,
			Components: []Component{
				{ID: "root-col", Component: ComponentValue{Column: &ColumnComp{Children: append([]string{}, *rootChildren...)}}},
				{ID: cardID, Component: ComponentValue{Card: &CardComp{Children: []string{colID}}}},
				{ID: colID, Component: ComponentValue{Column: &ColumnComp{Children: []string{labelID, textID}}}},
				{ID: labelID, Component: ComponentValue{Text: &TextComp{Value: kind, UsageHint: "caption"}}},
				{ID: textID, Component: ComponentValue{Text: &TextComp{Value: display, UsageHint: "body"}}},
			},
		},
	})
}

func formatToolCall(tc toolCallInfo) string {
	text := "🔧 " + tc.Name
	if tc.Args != "" {
		args := tc.Args
		if len([]rune(args)) > 400 {
			args = string([]rune(args)[:400]) + "…"
		}
		text += "\n" + args
	}
	return text
}

func roleToLabel(role schema.RoleType) string {
	switch role {
	case schema.User:
		return "You"
	case schema.Assistant:
		return "Agent"
	case schema.Tool:
		return "Tool"
	case schema.System:
		return "System"
	default:
		if role != "" {
			return string(role)
		}
		return "Agent"
	}
}

func emit(w io.Writer, msg Message) error {
	data, err := Encode(msg)
	if err != nil {
		return fmt.Errorf("encode message: %w", err)
	}
	_, err = w.Write(data)
	if err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	return nil
}
