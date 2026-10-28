import { ArrowUpRight, Copy } from 'lucide-react'
import type { CSSProperties } from 'react'
import type { ExternalRepo } from '../data/modules'

interface ExternalCardProps {
  repo: ExternalRepo
  onOpen: () => void
  onCopy: () => void
}

/** 外部仓库入口：横向卡片，跨满模块网格，不参与健康检测 */
export function ExternalCard({ repo, onOpen, onCopy }: ExternalCardProps) {
  const Icon = repo.icon

  return (
    <article className="external-card" style={{ '--module-accent': repo.accent } as CSSProperties}>
      <span className="module-icon" aria-hidden="true">
        <Icon size={20} strokeWidth={1.8} />
      </span>

      <div className="external-copy">
        <div className="external-title">
          <h3>{repo.name}</h3>
          <span className="external-badge">外部站点</span>
        </div>
        <p className="external-repo">{repo.repo}</p>
        <p className="external-desc">{repo.description}</p>
        <ul className="module-features">
          {repo.features.map((feature) => (
            <li key={feature}>{feature}</li>
          ))}
        </ul>
      </div>

      <div className="external-actions">
        <button type="button" className="btn btn-solid" onClick={onOpen}>
          打开 GitHub
          <ArrowUpRight size={15} />
        </button>
        <button type="button" className="btn btn-ghost" onClick={onCopy}>
          <Copy size={14} />
          复制地址
        </button>
      </div>
    </article>
  )
}
