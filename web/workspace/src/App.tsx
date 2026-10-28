import { Copy, Link2, Moon, RefreshCw, Sun } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { ActivityPanel } from './components/ActivityPanel'
import { CommandPalette } from './components/CommandPalette'
import type { CommandItem } from './components/CommandPalette'
import { EndpointTable } from './components/EndpointTable'
import { ExternalCard } from './components/ExternalCard'
import { EnvironmentPanel } from './components/EnvironmentPanel'
import { ModuleCard } from './components/ModuleCard'
import { Rail } from './components/Rail'
import { StatusBoard } from './components/StatusBoard'
import { Toast } from './components/Toast'
import { TopBar } from './components/TopBar'
import { endpoints, externalRepos, liveBaseUrl, modules, POLL_INTERVAL_MS } from './data/modules'
import type { ModuleDefinition } from './data/modules'
import { useHealth } from './hooks/useHealth'
import { useNow } from './hooks/useNow'
import { useTheme } from './hooks/useTheme'
import { useToast } from './hooks/useToast'
import { copyText } from './lib/clipboard'
import { hostOf, isoToDateTime } from './lib/format'

export default function App() {
  const { health, events, snapshots, checking, lastCheckAt, nextCheckAt, checkAll, clearEvents } = useHealth()
  const { theme, toggleTheme } = useTheme()
  const { toast, show } = useToast()
  const now = useNow(1000)

  const [navOpen, setNavOpen] = useState(false)
  const [paletteOpen, setPaletteOpen] = useState(false)

  const openUrl = useCallback((url: string) => {
    window.open(url, '_blank', 'noopener,noreferrer')
    setNavOpen(false)
  }, [])

  const openModule = useCallback((module: ModuleDefinition) => openUrl(module.url), [openUrl])

  const copy = useCallback(
    async (text: string, label: string) => {
      const copied = await copyText(text)
      show(copied ? `已复制 ${label}` : '复制失败，请手动选择地址')
    },
    [show],
  )

  const copyAll = useCallback(() => {
    void copy(modules.map((module) => module.url).join('\n'), '全部模块地址')
  }, [copy])

  // 端点挂载在 Live 站点下，复制时要拼 Live 基址而不是工作台自身端口
  const copyEndpoint = useCallback(
    (path: string) => {
      void copy(`${liveBaseUrl.replace(/\/$/, '')}${path}`, path)
    },
    [copy],
  )

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault()
        setPaletteOpen((open) => !open)
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [])

  useEffect(() => {
    if (!navOpen) return
    const previous = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setNavOpen(false)
    }
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.body.style.overflow = previous
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [navOpen])

  const online = modules.filter((module) => health[module.id]?.status === 'online').length
  const latencies = modules
    .map((module) => health[module.id])
    .filter((state) => state?.status === 'online' && state.latency !== null)
    .map((state) => state.latency as number)
  const average = latencies.length
    ? Math.round(latencies.reduce((sum, value) => sum + value, 0) / latencies.length)
    : null

  const commands = useMemo<CommandItem[]>(
    () => [
      ...modules.map((module) => ({
        id: `open-${module.id}`,
        label: `打开 ${module.name}`,
        hint: hostOf(module.url),
        group: '模块',
        icon: module.icon,
        run: () => openModule(module),
      })),
      ...externalRepos.map((repo) => ({
        id: `open-${repo.id}`,
        label: `打开 ${repo.name}`,
        hint: repo.repo,
        group: '代码仓库',
        icon: repo.icon,
        run: () => openUrl(repo.url),
      })),
      ...endpoints.map((endpoint) => ({
        id: `copy-${endpoint.path}`,
        label: `复制 ${endpoint.path}`,
        hint: endpoint.title,
        group: '端点',
        icon: Link2,
        run: () => copyEndpoint(endpoint.path),
      })),
      {
        id: 'check-all',
        label: '立即检测全部模块',
        hint: 'health',
        group: '操作',
        icon: RefreshCw,
        run: () => void checkAll(),
      },
      {
        id: 'copy-all',
        label: '复制全部模块地址',
        hint: `${modules.length} 个地址`,
        group: '操作',
        icon: Copy,
        run: copyAll,
      },
      {
        id: 'toggle-theme',
        label: theme === 'dark' ? '切换浅色主题' : '切换深色主题',
        hint: 'appearance',
        group: '操作',
        icon: theme === 'dark' ? Sun : Moon,
        run: toggleTheme,
      },
    ],
    [checkAll, copy, copyAll, copyEndpoint, openModule, openUrl, theme, toggleTheme],
  )

  return (
    <div className="shell">
      <Rail
        health={health}
        open={navOpen}
        theme={theme}
        online={online}
        total={modules.length}
        average={average}
        checking={checking}
        nextCheckAt={nextCheckAt}
        now={now}
        events={events}
        onOpenModule={openModule}
        onToggleTheme={toggleTheme}
        onClose={() => setNavOpen(false)}
      />

      {navOpen && (
        <button
          type="button"
          className="rail-scrim"
          onClick={() => setNavOpen(false)}
          aria-label="关闭导航"
        />
      )}

      <div className="main">
        <TopBar
          navOpen={navOpen}
          onOpenNav={() => setNavOpen(true)}
          onOpenSearch={() => setPaletteOpen(true)}
        />

        <main className="content">
          <div className="page-head">
            <div className="page-head-copy">
              <h1>控制台总览</h1>
              <p>统一入口与运行状态，覆盖会议、网关、开放平台与代码仓库。</p>
            </div>
            <div className="page-actions">
              <button type="button" className="btn btn-ghost" onClick={copyAll}>
                <Copy size={14} />
                复制全部地址
              </button>
              <button
                type="button"
                className="btn btn-solid"
                onClick={() => void checkAll()}
                disabled={checking}
              >
                <RefreshCw size={15} className={checking ? 'spin' : undefined} />
                {checking ? '检测中…' : '立即检测'}
              </button>
            </div>
          </div>

          <StatusBoard
            online={online}
            total={modules.length}
            average={average}
            checking={checking}
            lastCheckAt={lastCheckAt}
            nextCheckAt={nextCheckAt}
            now={now}
            snapshots={snapshots}
          />

          <section className="section" aria-labelledby="modules-title">
            <div className="section-head">
              <div>
                <h2 id="modules-title">服务模块</h2>
                <p>点击打开对应子系统，状态每 {POLL_INTERVAL_MS / 1000} 秒自动刷新</p>
              </div>
            </div>

            <div className="module-grid">
              {modules.map((module) => (
                <ModuleCard
                  key={module.id}
                  module={module}
                  health={health[module.id]}
                  onOpen={() => openModule(module)}
                  onCopy={() => void copy(module.url, module.name)}
                />
              ))}
              {externalRepos.map((repo) => (
                <ExternalCard
                  key={repo.id}
                  repo={repo}
                  onOpen={() => openUrl(repo.url)}
                  onCopy={() => void copy(repo.url, repo.name)}
                />
              ))}
            </div>
          </section>

          <div className="lower-grid">
            <EndpointTable baseUrl={liveBaseUrl} onCopy={copyEndpoint} />
            <div className="side-stack">
              <EnvironmentPanel />
              <ActivityPanel events={events} onClear={clearEvents} />
            </div>
          </div>

          <footer className="foot">
            <span>Zero Service · 工作台 v{__APP_VERSION__}</span>
            <span>
              构建 {isoToDateTime(__BUILD_TIME__)} · {import.meta.env.DEV ? '开发模式' : '生产模式'}
            </span>
          </footer>
        </main>
      </div>

      <CommandPalette open={paletteOpen} items={commands} onClose={() => setPaletteOpen(false)} />
      <Toast toast={toast} />
    </div>
  )
}
