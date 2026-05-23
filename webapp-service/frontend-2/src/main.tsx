import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import MediaPage from './MediaPage'
import LoginPage from './LoginPage'
import { useAuthStore } from './store/authStore'

function App() {
  const isAuthenticated = useAuthStore(s => s.isAuthenticated())

  if (!isAuthenticated) {
    return <LoginPage onLogin={() => {}} />
  }

  return <MediaPage />
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
