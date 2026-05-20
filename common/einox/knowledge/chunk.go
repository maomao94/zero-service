package knowledge

import (
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
)

// SplitIntoDocuments 将长文本切成多个 *schema.Document（段落优先，控制单块大小）。
func SplitIntoDocuments(text string, chunkSize int) []*schema.Document {
	return SplitIntoDocumentsWithOverlap(text, chunkSize, 0)
}

// SplitIntoDocumentsWithOverlap 将长文本切成多个 *schema.Document，并在连续分块间保留 overlapRunes 个重叠字符。
func SplitIntoDocumentsWithOverlap(text string, chunkSize, overlapRunes int) []*schema.Document {
	if chunkSize <= 0 {
		chunkSize = 800
	}
	if overlapRunes < 0 {
		overlapRunes = 0
	}
	if overlapRunes >= chunkSize {
		overlapRunes = chunkSize - 1
	}
	var chunks []*schema.Document
	var buf strings.Builder
	var bufRunes int

	flush := func() {
		s := strings.TrimSpace(buf.String())
		if s != "" {
			chunks = append(chunks, &schema.Document{Content: s})
		}
		buf.Reset()
		bufRunes = 0
	}

	for _, para := range strings.Split(text, "\n\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		paraRunes := utf8.RuneCountInString(para)
		if bufRunes+paraRunes+2 > chunkSize && bufRunes > 0 {
			flush()
		}
		if paraRunes > chunkSize {
			if bufRunes > 0 {
				flush()
			}
			for _, line := range strings.Split(para, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				lineRunes := utf8.RuneCountInString(line)
				if lineRunes > chunkSize {
					flush()
					for _, part := range splitRunesWithOverlap(line, chunkSize, overlapRunes) {
						chunks = append(chunks, &schema.Document{Content: part})
					}
					continue
				}
				if bufRunes+lineRunes+1 > chunkSize && bufRunes > 0 {
					flush()
				}
				if bufRunes > 0 {
					buf.WriteByte('\n')
					bufRunes++
				}
				buf.WriteString(line)
				bufRunes += lineRunes
			}
		} else {
			if bufRunes > 0 {
				buf.WriteString("\n\n")
				bufRunes += 2
			}
			buf.WriteString(para)
			bufRunes += paraRunes
		}
	}
	flush()
	return chunks
}

func splitRunesWithOverlap(text string, chunkSize, overlapRunes int) []string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) == 0 {
		return nil
	}
	if len(runes) <= chunkSize {
		return []string{string(runes)}
	}
	step := chunkSize - overlapRunes
	if step <= 0 {
		step = chunkSize
	}
	parts := make([]string, 0, (len(runes)+step-1)/step)
	for start := 0; start < len(runes); start += step {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		parts = append(parts, string(runes[start:end]))
		if end == len(runes) {
			break
		}
	}
	return parts
}
