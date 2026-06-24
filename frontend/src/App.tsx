import { useState } from 'react'
import type { Session } from './api'
import './App.css'
import { roles } from './domain'
import { AdminDashboard } from './features/admin/AdminDashboard'
import { AuthScreen } from './features/auth/AuthScreen'
import { ClientHome } from './features/client/ClientHome'

function App() {
  const [session, setSession] = useState<Session | null>(() => {
    const stored = sessionStorage.getItem('bcb-session')
    return stored ? (JSON.parse(stored) as Session) : null
  })

  function saveSession(next: Session | null) {
    setSession(next)
    if (next) sessionStorage.setItem('bcb-session', JSON.stringify(next))
    else sessionStorage.removeItem('bcb-session')
  }

  if (session?.user.role === roles.admin) {
    return <AdminDashboard session={session} onLogout={() => saveSession(null)} />
  }

  if (session?.user.role === roles.client) {
    return <ClientHome session={session} onLogout={() => saveSession(null)} />
  }

  return <AuthScreen onSession={saveSession} />
}

export default App
