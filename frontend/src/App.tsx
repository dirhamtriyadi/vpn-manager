import { Link, NavLink, Route, Routes } from "react-router-dom"
import { Archive, LayoutDashboard, LogOut, Plus } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Dashboard } from "@/features/wireguard/Dashboard"
import { TrashPage } from "@/features/wireguard/TrashPage"
import { ProtocolSelector } from "@/features/vpn/ProtocolSelector"
import { OpenVPNRoadmapPage } from "@/features/vpn/OpenVPNRoadmapPage"
import { LoginPage } from "@/features/auth/LoginPage"
import { useAuth } from "@/features/auth/AuthContext"

function App() {
  const { isAuthenticated, username, logout } = useAuth()

  if (!isAuthenticated) {
    return <LoginPage />
  }

  return (
    <div className="min-h-screen bg-muted/30">
      <header className="border-b bg-background">
        <div className="container flex min-h-16 flex-wrap items-center justify-between gap-3 py-3">
          <Link to="/" className="flex items-baseline gap-3">
            <h1 className="text-lg font-semibold">VPN Manager</h1>
            <span className="hidden text-sm text-muted-foreground sm:inline">
              Manage WireGuard now, with multi-protocol support staged
            </span>
          </Link>
          <nav className="flex items-center gap-2">
            <Button variant="outline" size="sm" asChild>
              <NavLink to="/" end>
                <LayoutDashboard />
                Dashboard
              </NavLink>
            </Button>
            <Button variant="outline" size="sm" asChild>
              <NavLink to="/vpn/new">
                <Plus />
                New VPN
              </NavLink>
            </Button>
            <Button variant="outline" size="sm" asChild>
              <NavLink to="/trash">
                <Archive />
                Trash
              </NavLink>
            </Button>
            {username && (
              <span className="hidden text-sm text-muted-foreground sm:inline">
                {username}
              </span>
            )}
            <Button variant="outline" size="sm" onClick={logout}>
              <LogOut />
              Keluar
            </Button>
          </nav>
        </div>
      </header>
      <main className="container py-8">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/trash" element={<TrashPage />} />
          <Route path="/vpn/new" element={<ProtocolSelector />} />
          <Route path="/vpn/openvpn" element={<OpenVPNRoadmapPage />} />
        </Routes>
      </main>
    </div>
  )
}

export default App
