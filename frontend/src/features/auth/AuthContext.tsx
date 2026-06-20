import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react"
import { clearToken, getToken, setToken, setUnauthorizedHandler } from "@/lib/api"
import { login as loginRequest } from "./api"

const USER_KEY = "vpn_manager_user"

interface AuthState {
  isAuthenticated: boolean
  username: string | null
  login: (username: string, password: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthState | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setTokenState] = useState<string | null>(() => getToken())
  const [username, setUsername] = useState<string | null>(() =>
    localStorage.getItem(USER_KEY),
  )

  const clearSession = useCallback(() => {
    clearToken()
    localStorage.removeItem(USER_KEY)
    setTokenState(null)
    setUsername(null)
  }, [])

  // When the API rejects a stored token (401), drop the session everywhere.
  useEffect(() => {
    setUnauthorizedHandler(clearSession)
    return () => setUnauthorizedHandler(null)
  }, [clearSession])

  const login = useCallback(async (user: string, password: string) => {
    const result = await loginRequest(user, password)
    setToken(result.token)
    localStorage.setItem(USER_KEY, result.username)
    setTokenState(result.token)
    setUsername(result.username)
  }, [])

  const value = useMemo<AuthState>(
    () => ({
      isAuthenticated: Boolean(token),
      username,
      login,
      logout: clearSession,
    }),
    [token, username, login, clearSession],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider")
  }
  return ctx
}
