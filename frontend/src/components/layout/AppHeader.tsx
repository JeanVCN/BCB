import type { ReactNode } from 'react'

export function AppHeader({
  eyebrow,
  title,
  children,
}: {
  eyebrow: string
  title: string
  children: ReactNode
}) {
  return (
    <header className="app-header">
      <div>
        <span className="eyebrow">{eyebrow}</span>
        <h1>{title}</h1>
      </div>
      <nav className="top-nav" aria-label="Atalhos do painel">
        {children}
      </nav>
    </header>
  )
}
