import { Globe2, Menu, Search } from 'lucide-react'
import { useNow } from '../hooks/useNow'
import { clockTime } from '../lib/format'
import { searchShortcut } from '../lib/platform'

interface TopBarProps {
  navOpen: boolean
  onOpenNav: () => void
  onOpenSearch: () => void
}

export function TopBar({ navOpen, onOpenNav, onOpenSearch }: TopBarProps) {
  const now = useNow(1000)

  return (
    <header className="topbar">
      <div className="topbar-left">
        <button
          type="button"
          className="topbar-menu"
          onClick={onOpenNav}
          aria-label="打开导航"
          aria-expanded={navOpen}
        >
          <Menu size={18} />
        </button>
        <span className="topbar-title">控制台</span>
        <span className="topbar-sep" aria-hidden="true" />
        <span className="topbar-host">
          <Globe2 size={13} />
          {location.host}
        </span>
      </div>

      <div className="topbar-right">
        <button type="button" className="search-trigger" onClick={onOpenSearch}>
          <Search size={15} />
          <span className="search-trigger-text">搜索模块与端点</span>
          <kbd>{searchShortcut}</kbd>
        </button>
        <time className="topbar-clock" dateTime={new Date(now).toISOString()}>
          {clockTime(now)}
        </time>
        <span className="env-pill" data-mode={import.meta.env.DEV ? 'dev' : 'prod'}>
          <i aria-hidden="true" />
          {import.meta.env.DEV ? '开发环境' : '生产环境'}
        </span>
      </div>
    </header>
  )
}
