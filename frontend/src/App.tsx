import { Link, NavLink, Route, Routes } from "react-router-dom"
import {
  Archive,
  KeyRound,
  LayoutDashboard,
  LogOut,
  Plus,
  Shield,
  Users,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Dashboard } from "@/features/wireguard/Dashboard"
import { TrashPage } from "@/features/wireguard/TrashPage"
import { ProtocolSelector } from "@/features/vpn/ProtocolSelector"
import { OpenVPNRoadmapPage } from "@/features/vpn/OpenVPNRoadmapPage"
import { ProtocolRoadmapPage } from "@/features/vpn/ProtocolRoadmapPage"
import { UsersPage } from "@/features/admin/UsersPage"
import { RolesPage } from "@/features/admin/RolesPage"
import { PermissionsPage } from "@/features/admin/PermissionsPage"
import { LoginPage } from "@/features/auth/LoginPage"
import { useAuth } from "@/features/auth/AuthContext"

function App() {
  const { isAuthenticated, username, logout, hasPermission } = useAuth()

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
            {hasPermission("users.view") && (
              <Button variant="outline" size="sm" asChild>
                <NavLink to="/users">
                  <Users />
                  Users
                </NavLink>
              </Button>
            )}
            {hasPermission("roles.view") && (
              <Button variant="outline" size="sm" asChild>
                <NavLink to="/roles">
                  <Shield />
                  Roles
                </NavLink>
              </Button>
            )}
            {hasPermission("permissions.view") && (
              <Button variant="outline" size="sm" asChild>
                <NavLink to="/permissions">
                  <KeyRound />
                  Permissions
                </NavLink>
              </Button>
            )}
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
          <Route path="/vpn/:protocol" element={<ProtocolRoadmapPage />} />
          <Route path="/users" element={<UsersPage />} />
          <Route path="/roles" element={<RolesPage />} />
          <Route path="/permissions" element={<PermissionsPage />} />
        </Routes>
      </main>
    </div>
  )
}

export default App
