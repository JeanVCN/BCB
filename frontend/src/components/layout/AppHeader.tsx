import type { ReactNode } from 'react'

export function AppHeader({
  eyebrow,
  title,
  meta,
  children,
}: {
  eyebrow: string
  title: string
  meta?: ReactNode
  children: ReactNode
}) {
  return (
    <header className="app-header">
      <div>
        <span className="eyebrow">{eyebrow}</span>
        <h1>{title}</h1>
        {meta}
      </div>
      <nav className="top-nav" aria-label="Atalhos do painel">
        {children}
      </nav>
    </header>
  )
}
