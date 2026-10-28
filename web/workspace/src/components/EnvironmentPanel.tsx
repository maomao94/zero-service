import { POLL_INTERVAL_MS } from '../data/modules'
import { isoToDateTime, siteOrigin } from '../lib/format'

export function EnvironmentPanel() {
  const mode = import.meta.env.DEV ? '开发' : '生产'

  return (
    <section className="panel" aria-labelledby="env-title">
      <header className="panel-head">
        <div>
          <h2 id="env-title">运行环境</h2>
          <p>当前页面与构建信息</p>
        </div>
      </header>

      <dl className="env-list">
        <div>
          <dt>运行模式</dt>
          <dd>
            {mode}
            <span className="env-tag" data-mode={import.meta.env.DEV ? 'dev' : 'prod'}>
              {import.meta.env.DEV ? 'dev' : 'prod'}
            </span>
          </dd>
        </div>
        <div>
          <dt>站点</dt>
          <dd className="mono">{siteOrigin()}</dd>
        </div>
        <div>
          <dt>版本</dt>
          <dd className="mono">v{__APP_VERSION__}</dd>
        </div>
        <div>
          <dt>构建时间</dt>
          <dd className="mono">{isoToDateTime(__BUILD_TIME__)}</dd>
        </div>
        <div>
          <dt>检测周期</dt>
          <dd className="mono">{POLL_INTERVAL_MS / 1000} s</dd>
        </div>
      </dl>
    </section>
  )
}
