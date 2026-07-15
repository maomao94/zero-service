package tool

// EstimateTokens 估算文本的 token 数量（近似值）
// 规则：
//   - 中文字符：约 2 tokens/字符
//   - 英文单词：约 1.3 tokens/单词
//   - 标点/空格：约 0.25 tokens/字符
//
// 注意：这是粗略估算，实际 token 数因模型而异。如需精确值请使用 tiktoken 等库。
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	var tokenCount float64
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case r >= 0x4E00 && r <= 0x9FFF: // CJK Unified Ideographs (中文)
			tokenCount += 2.0
		case r >= 0x3000 && r <= 0x303F: // CJK Symbols and Punctuation
			tokenCount += 0.25
		case r >= 0xFF00 && r <= 0xFFEF: // Halfwidth and Fullwidth Forms
			tokenCount += 2.0
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z': // English letters
			// 尝试计算完整单词
			wordLen := 1
			for i+1 < len(runes) && ((runes[i+1] >= 'a' && runes[i+1] <= 'z') || (runes[i+1] >= 'A' && runes[i+1] <= 'Z') || (runes[i+1] >= '0' && runes[i+1] <= '9')) {
				wordLen++
				i++
			}
			tokenCount += 1.3 * float64(wordLen)
			continue
		case r == ' ', r == '\t', r == '\n', r == '\r': // Whitespace
			tokenCount += 0.25
		case r < 128: // ASCII punctuation/symbols
			tokenCount += 0.25
		default: // Other characters (emoji, etc.)
			tokenCount += 2.0
		}
	}

	return int(tokenCount)
}

// EstimateMessagesTokens 估算消息列表的总 token 数（包含消息格式开销）
// 消息格式开销约 4 tokens/条（role + content wrapper）
func EstimateMessagesTokens(messages []string) int {
	total := 0
	for _, msg := range messages {
		total += 4 + EstimateTokens(msg) // 4 tokens overhead per message
	}
	return total
}
