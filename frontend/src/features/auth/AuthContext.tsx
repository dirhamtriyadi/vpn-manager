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
import { getMe, login as loginRequest, type CurrentUser } from "./api"

const USER_KEY = "vpn_manager_user"

interface AuthState {
  isAuthenticated: boolean
  /** Resolved once GET /auth/me returns; null until then or when signed out. */
  user: CurrentUser | null
  username: string | null
  /** True while the initial /auth/me lookup for a stored token is in flight. */
  loading: boolean
  /** Effective permission check (wildcard "*" satisfies everything). */
  hasPermission: (permission: string) => boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthState | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setTokenState] = useState<string | null>(() => getToken())
  const [username, setUsername] = useState<string | null>(() =>
    localStorage.getItem(USER_KEY),
  )
  const [user, setUser] = useState<CurrentUser | null>(null)
  // Only block the UI for the initial lookup when we boot with a stored token.
  const [loading, setLoading] = useState<boolean>(() => Boolean(getToken()))

  const clearSession = useCallback(() => {
    clearToken()
    localStorage.removeItem(USER_KEY)
    setTokenState(null)
    setUsername(null)
    setUser(null)
    setLoading(false)
  }, [])

  // When the API rejects a stored token (401), drop the session everywhere.
  useEffect(() => {
    setUnauthorizedHandler(clearSession)
    return () => setUnauthorizedHandler(null)
  }, [clearSession])

  // Resolve the full user (roles + effective permissions) whenever we hold a
  // token. A failure other than 401 (handled by the interceptor) just leaves
  // permissions empty so permission-gated UI stays hidden.
  useEffect(() => {
    if (!token) {
      setUser(null)
      setLoading(false)
      return
    }
    let active = true
    setLoading(true)
    getMe()
      .then((me) => {
        if (!active) return
        setUser(me)
        setUsername(me.username)
        localStorage.setItem(USER_KEY, me.username)
      })
      .catch(() => {
        /* 401 clears the session via the interceptor; other errors leave
           permissions empty. */
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [token])

  const login = useCallback(async (usernameArg: string, password: string) => {
    const result = await loginRequest(usernameArg, password)
    setToken(result.token)
    localStorage.setItem(USER_KEY, result.username)
    setTokenState(result.token)
    setUsername(result.username)
    // The /auth/me effect above fires off the token change and fills in
    // roles/permissions.
  }, [])

  const hasPermission = useCallback(
    (permission: string) => {
      if (!user) return false
      return user.permissions.includes("*") || user.permissions.includes(permission)
    },
    [user],
  )

  const value = useMemo<AuthState>(
    () => ({
      isAuthenticated: Boolean(token),
      user,
      username,
      loading,
      hasPermission,
      login,
      logout: clearSession,
    }),
    [token, user, username, loading, hasPermission, login, clearSession],
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
