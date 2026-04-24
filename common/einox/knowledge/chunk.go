package knowledge

import (
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
)

// SplitIntoDocuments 将长文本切成多个 *schema.Document（段落优先，控制单块大小）。
func SplitIntoDocuments(text string, chunkSize int) []*schema.Document {
	if chunkSize <= 0 {
		chunkSize = 800
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
			for _, line := range strings.Split(para, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				lineRunes := utf8.RuneCountInString(line)
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
