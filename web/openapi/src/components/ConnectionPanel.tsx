import { Eye, EyeOff, Link2, LoaderCircle, RefreshCw, ShieldCheck, Trash2, Wifi, WifiOff } from 'lucide-react'
import { useState } from 'react'
import type { HealthResult } from '../lib/http'
import { buildUrl } from '../lib/http'
import type { Settings } from '../types'

interface ConnectionPanelProps {
  settings: Settings
  onChange: (patch: Partial<Settings>) => void
  health: HealthResult | null
  checking: boolean
  onCheck: () => void
  onClear: () => void
}

const LOCAL_BASE = 'http://127.0.0.1:11005'

export default function ConnectionPanel({ settings, onChange, health, checking, onCheck, onClear }: ConnectionPanelProps) {
  const [showSecret, setShowSecret] = useState(false)
  const customBase = settings.baseUrl !== '' && settings.baseUrl !== LOCAL_BASE

  return (
    <section className="card connection-card">
      <div className="card-heading">
        <div>
          <span className="eyebrow">CONNECTION</span>
          <h2>连接设置</h2>
        </div>
        <span className={`health-pill ${health?.ok ? 'ok' : health ? 'bad' : ''}`}>
          {checking ? <LoaderCircle size={13} className="spin" /> : health?.ok ? <Wifi size={13} /> : <WifiOff size={13} />}
          {checking ? '检测中' : health?.ok ? `连接正常 · ${Math.round(health.durationMs)}ms` : health ? '不可达' : '未检测'}
        </span>
      </div>

      <div className="field">
        <label className="field-label" htmlFor="baseUrl">Base URL</label>
        <div className="input-with-icon">
          <Link2 size={15} />
          <input
            id="baseUrl"
            className="text-input"
            value={settings.baseUrl}
            placeholder="留空 = 当前站点（同源）"
            onChange={(event) => onChange({ baseUrl: event.target.value })}
            spellCheck={false}
          />
        </div>
        <div className="preset-row">
          <button
            type="button"
            className={`preset ${settings.baseUrl === '' ? 'active' : ''}`}
            onClick={() => onChange({ baseUrl: '' })}
          >
            当前站点
          </button>
          <button
            type="button"
            className={`preset ${settings.baseUrl === LOCAL_BASE ? 'active' : ''}`}
            onClick={() => onChange({ baseUrl: LOCAL_BASE })}
          >
            本地直连
          </button>
          <button
            type="button"
            className={`preset ${customBase ? 'active' : ''}`}
            onClick={() => onChange({ baseUrl: customBase ? settings.baseUrl : 'https://' })}
          >
            自定义
          </button>
        </div>
        <p className="field-hint">实际请求：<code>{buildUrl(settings.baseUrl, '/v1/gateway')}</code></p>
      </div>

      <div className="field">
        <label className="field-label" htmlFor="appId">App ID</label>
        <input
          id="appId"
          className="text-input mono"
          value={settings.appId}
          placeholder="app_xxxxxxxxxxxx"
          onChange={(event) => onChange({ appId: event.target.value.trim() })}
          spellCheck={false}
          autoComplete="off"
        />
      </div>

      <div className="field">
        <label className="field-label" htmlFor="secret">Secret</label>
        <div className="secret-input-group">
          <input
            id="secret"
            className="text-input mono"
            type={showSecret ? 'text' : 'password'}
            value={settings.secret}
            placeholder="sec_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
            onChange={(event) => onChange({ secret: event.target.value.trim() })}
            spellCheck={false}
            autoComplete="off"
          />
          <button type="button" className="input-action" onClick={() => setShowSecret((value) => !value)} aria-label={showSecret ? '隐藏密钥' : '显示密钥'}>
            {showSecret ? <EyeOff size={15} /> : <Eye size={15} />}
          </button>
        </div>
      </div>

      <div className="connection-note">
        <ShieldCheck size={15} />
        <span>凭据仅保存在当前浏览器 localStorage，仅限测试环境使用。</span>
      </div>

      <div className="connection-actions">
        <button type="button" className="button secondary" onClick={onCheck} disabled={checking}>
          {checking ? <LoaderCircle size={14} className="spin" /> : <RefreshCw size={14} />}
          验证连接
        </button>
        <button type="button" className="button ghost danger-text" onClick={onClear}>
          <Trash2 size={14} />
          清除凭据
        </button>
      </div>
    </section>
  )
}
