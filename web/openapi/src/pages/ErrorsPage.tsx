import { BookOpen, GitBranch, Search, ShieldAlert } from 'lucide-react'
import { useMemo, useState } from 'react'
import { ERROR_DOCS } from '../lib/errors'

const GROUPS = ['全部', '成功', '开放平台', '参数', '数据', '认证', '业务', '基础设施', '网关', '系统'] as const

export default function ErrorsPage() {
  const [query, setQuery] = useState('')
  const [group, setGroup] = useState<(typeof GROUPS)[number]>('全部')

  const rows = useMemo(() => {
    const keyword = query.trim().toLowerCase()
    return ERROR_DOCS.filter((doc) => {
      if (group !== '全部' && doc.group !== group) return false
      if (!keyword) return true
      return [doc.code, doc.http, doc.title, doc.description, doc.suggestion]
        .filter(Boolean)
        .join(' ')
        .toLowerCase()
        .includes(keyword)
    })
  }, [group, query])

  return (
    <div className="page-stack">
      <section className="page-intro">
        <div>
          <span className="eyebrow">REFERENCE</span>
          <h1>错误码速查</h1>
          <p>与 <code>third_party/extproto.proto</code>、<code>common/errx/codec.go</code> 同源维护，覆盖开放平台与常用错误。</p>
        </div>
        <div className="intro-stats">
          <span><BookOpen size={15} /> {ERROR_DOCS.length} 个错误码</span>
          <span><ShieldAlert size={15} /> BC=30 开放平台</span>
        </div>
      </section>

      <section className="card folding-note">
        <div className="folding-icon"><GitBranch size={20} /></div>
        <div>
          <b>基础设施错误折叠规则</b>
          <p><code>100201</code>、<code>100301</code>、<code>100303</code>、<code>1006xx</code>、<code>100998</code> 对第三方统一返回 <code>100999 系统繁忙</code>，真实原因只写服务端日志。</p>
        </div>
      </section>

      <section className="card errors-card">
        <div className="errors-toolbar">
          <div className="search-box">
            <Search size={14} />
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="搜索错误码、标题或说明"
              aria-label="搜索错误码"
            />
          </div>
          <div className="group-tabs">
            {GROUPS.map((item) => (
              <button
                type="button"
                key={item}
                className={group === item ? 'active' : ''}
                onClick={() => setGroup(item)}
              >
                {item}
              </button>
            ))}
          </div>
        </div>

        <div className="table-scroll">
          <table className="errors-table">
            <thead>
              <tr>
                <th>错误码</th>
                <th>语义 HTTP</th>
                <th>含义</th>
                <th>说明与处理建议</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((doc) => (
                <tr key={doc.code}>
                  <td><code className="code-tag">{doc.code || '0'}</code></td>
                  <td><span className={`http-tag http-${doc.http >= 500 ? 'error' : doc.http >= 400 ? 'warn' : 'ok'}`}>{doc.http}</span></td>
                  <td>
                    <b>{doc.title}</b>
                    {doc.folded && <span className="folded-tag">折叠</span>}
                  </td>
                  <td>
                    <p>{doc.description}</p>
                    {doc.suggestion && <small>{doc.suggestion}</small>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!rows.length && <div className="history-empty"><Search size={20} /><span>没有匹配的错误码。</span></div>}
      </section>
    </div>
  )
}
