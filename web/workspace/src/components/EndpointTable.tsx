import { Copy } from 'lucide-react'
import { endpoints } from '../data/modules'
import { hostOf } from '../lib/format'

interface EndpointTableProps {
  baseUrl: string
  onCopy: (path: string) => void
}

export function EndpointTable({ baseUrl, onCopy }: EndpointTableProps) {
  return (
    <section className="panel endpoints" aria-labelledby="endpoints-title">
      <header className="panel-head">
        <div>
          <h2 id="endpoints-title">端点速查</h2>
          <p>路径相对 Live 站点，复制即为完整地址</p>
        </div>
        <span className="panel-meta mono">{hostOf(baseUrl)}</span>
      </header>

      <div className="table-scroll">
        <table className="endpoint-table">
          <thead>
            <tr>
              <th scope="col">路径</th>
              <th scope="col">用途</th>
              <th scope="col">上游</th>
              <th scope="col">
                <span className="sr-only">操作</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {endpoints.map((endpoint) => (
              <tr key={endpoint.path}>
                <td>
                  <code className="endpoint-path">{endpoint.path}</code>
                </td>
                <td>
                  <span className="endpoint-title">{endpoint.title}</span>
                  <span className="endpoint-note">{endpoint.note}</span>
                </td>
                <td>
                  <code className="endpoint-upstream">{endpoint.upstream}</code>
                </td>
                <td className="endpoint-action">
                  <button
                    type="button"
                    className="icon-btn"
                    onClick={() => onCopy(endpoint.path)}
                    aria-label={`复制 ${endpoint.path}`}
                    title="复制完整地址"
                  >
                    <Copy size={14} />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}
