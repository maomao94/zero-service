import { useMemo } from 'react'

interface JsonHighlightProps {
  text: string
  emptyText?: string
}

const TOKEN = /("(?:\\u[\da-fA-F]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(?:true|false|null)\b|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)/g

/**
 * 轻量 JSON 语法高亮：全部使用 React 文本节点，不注入 HTML，响应内容不会被当作 DOM 执行。
 * 超大响应降级为纯文本，避免逐 token 渲染拖慢页面。
 */
export default function JsonHighlight({ text, emptyText = '（空）' }: JsonHighlightProps) {
  const nodes = useMemo(() => {
    if (!text) return null
    if (text.length > 200_000) return text

    const result: React.ReactNode[] = []
    let lastIndex = 0
    let match: RegExpExecArray | null
    let index = 0

    TOKEN.lastIndex = 0
    while ((match = TOKEN.exec(text)) !== null) {
      if (match.index > lastIndex) {
        result.push(text.slice(lastIndex, match.index))
      }
      const token = match[0]
      let className = 'json-number'
      if (token.startsWith('"')) {
        className = match[2] ? 'json-key' : 'json-string'
      } else if (token === 'true' || token === 'false') {
        className = 'json-boolean'
      } else if (token === 'null') {
        className = 'json-null'
      }
      result.push(
        <span className={className} key={`${index}-${match.index}`}>
          {token}
        </span>,
      )
      lastIndex = match.index + token.length
      index += 1
    }

    if (lastIndex < text.length) {
      result.push(text.slice(lastIndex))
    }
    return result
  }, [text])

  if (!text) {
    return <div className="json-empty">{emptyText}</div>
  }

  return (
    <pre className="json-view">
      <code>{nodes}</code>
    </pre>
  )
}
