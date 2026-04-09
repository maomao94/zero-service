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

import "encoding/json"

type Message struct {
	BeginRendering   *BeginRenderingMsg   `json:"beginRendering,omitempty"`
	SurfaceUpdate    *SurfaceUpdateMsg    `json:"surfaceUpdate,omitempty"`
	DataModelUpdate  *DataModelUpdateMsg  `json:"dataModelUpdate,omitempty"`
	DeleteSurface    *DeleteSurfaceMsg    `json:"deleteSurface,omitempty"`
	InterruptRequest *InterruptRequestMsg `json:"interruptRequest,omitempty"`
}

type BeginRenderingMsg struct {
	SurfaceID string `json:"surfaceId"`
	Root      string `json:"root"`
}

type SurfaceUpdateMsg struct {
	SurfaceID  string      `json:"surfaceId"`
	Components []Component `json:"components"`
}

type Component struct {
	ID        string         `json:"id"`
	Component ComponentValue `json:"component"`
}

type ComponentValue struct {
	Text   *TextComp   `json:"Text,omitempty"`
	Column *ColumnComp `json:"Column,omitempty"`
	Card   *CardComp   `json:"Card,omitempty"`
	Row    *RowComp    `json:"Row,omitempty"`
}

type TextComp struct {
	Value     string `json:"value,omitempty"`
	DataKey   string `json:"dataKey,omitempty"`
	UsageHint string `json:"usageHint,omitempty"`
}

type ColumnComp struct {
	Children []string `json:"children"`
}

type CardComp struct {
	Children []string `json:"children"`
}

type RowComp struct {
	Children []string `json:"children"`
}

type DataModelUpdateMsg struct {
	SurfaceID string        `json:"surfaceId"`
	Contents  []DataContent `json:"contents"`
}

type DataContent struct {
	Key         string `json:"key"`
	ValueString string `json:"valueString,omitempty"`
}

type DeleteSurfaceMsg struct {
	SurfaceID string `json:"surfaceId"`
}

type InterruptType string

const (
	InterruptTypeApproval InterruptType = "approval"
	InterruptTypeSingle   InterruptType = "single"
	InterruptTypeMultiple InterruptType = "multiple"
)

type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type InterruptRequestMsg struct {
	InterruptID string        `json:"interruptId"`
	Description string        `json:"description"`
	Type        InterruptType `json:"type,omitempty"`
	Options     []Option      `json:"options,omitempty"`
	Required    bool          `json:"required,omitempty"`
}

func Encode(msg Message) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
