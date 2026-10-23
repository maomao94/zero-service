import { Check, Copy } from 'lucide-react'
import { useState } from 'react'
import { copyText } from '../lib/clipboard'

interface CopyButtonProps {
  text: string
  label?: string
  className?: string
  onResult?: (ok: boolean) => void
}

export default function CopyButton({ text, label = '复制', className = '', onResult }: CopyButtonProps) {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    const ok = await copyText(text)
    setCopied(ok)
    onResult?.(ok)
    if (ok) window.setTimeout(() => setCopied(false), 1600)
  }

  return (
    <button type="button" className={`copy-button ${className}`} onClick={handleCopy} disabled={!text}>
      {copied ? <Check size={13} /> : <Copy size={13} />}
      <span>{copied ? '已复制' : label}</span>
    </button>
  )
}
