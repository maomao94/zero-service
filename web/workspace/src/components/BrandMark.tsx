/** 工作台标识：三层堆叠，呼应“多子系统统一入口” */
export function BrandMark({ size = 32 }: { size?: number }) {
  return (
    <span className="brand-mark" style={{ width: size, height: size }} aria-hidden="true">
      <svg viewBox="0 0 24 24" width={size * 0.64} height={size * 0.64} fill="none">
        <path d="M12 3.2 20.2 8 12 12.8 3.8 8 12 3.2Z" fill="currentColor" />
        <path d="M3.8 12.3 12 17.1l8.2-4.8" stroke="currentColor" strokeWidth="1.7" strokeLinejoin="round" opacity=".62" />
        <path d="M3.8 16.5 12 21.3l8.2-4.8" stroke="currentColor" strokeWidth="1.7" strokeLinejoin="round" opacity=".32" />
      </svg>
    </span>
  )
}
