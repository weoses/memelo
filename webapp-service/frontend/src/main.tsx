import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import MediaPage from './MediaPage.tsx'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <MediaPage />
  </StrictMode>,
)
