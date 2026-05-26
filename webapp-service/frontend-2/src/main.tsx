import { StrictMode, useState, useEffect } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import MediaPage from './MediaPage'
import LoginPage from './LoginPage'
import { useAuthStore } from './store/authStore'
import { BASE } from './api/client'

function App() {
  const isAuthenticated = useAuthStore(s => s.isAuthenticated())
  const [checking, setChecking] = useState(() => isAuthenticated)

  // On mount: verify stored tokens are still accepted by the server
  useEffect(() => {
    if (!checking) return
    const { refreshToken } = useAuthStore.getState()
    fetch(`${BASE}api/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
      .then(async res => {
        if (!res.ok) throw new Error()
        const data = await res.json() as { access_token: string }
        useAuthStore.getState().setAccessToken(data.access_token)
      })
      .catch(() => {
        useAuthStore.getState().logout()
        window.location.replace(
          `/login?redirect_url=${encodeURIComponent(window.location.href)}`
        )
      })
      .finally(() => setChecking(false))
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Guard: redirect based on auth state + current path
  useEffect(() => {
    if (checking) return
    if (!isAuthenticated && window.location.pathname !== '/login') {
      window.location.replace(
        `/login?redirect_url=${encodeURIComponent(window.location.href)}`
      )
    }
    if (isAuthenticated && window.location.pathname === '/login') {
      window.location.replace('/media')
    }
  }, [isAuthenticated, checking])

  if (checking) return null

  if (!isAuthenticated) {
    if (window.location.pathname === '/login') return <LoginPage />
    return null
  }

  return <MediaPage />
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
