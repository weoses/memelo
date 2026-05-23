import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface AuthState {
  accessToken: string | null
  refreshToken: string | null
  setTokens: (access: string, refresh: string) => void
  setAccessToken: (access: string) => void
  logout: () => void
  isAuthenticated: () => boolean
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      accessToken: null,
      refreshToken: null,

      setTokens: (access, refresh) => set({ accessToken: access, refreshToken: refresh }),
      setAccessToken: (access) => set({ accessToken: access }),

      logout: () => set({ accessToken: null, refreshToken: null }),

      isAuthenticated: () => get().accessToken !== null,
    }),
    {
      name: 'memelo-auth',
      partialize: (s) => ({ accessToken: s.accessToken, refreshToken: s.refreshToken }),
    },
  ),
)
