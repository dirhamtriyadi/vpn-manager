import { Link, NavLink, Route, Routes } from "react-router-dom"
import { Archive, LayoutDashboard } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Dashboard } from "@/features/wireguard/Dashboard"
import { TrashPage } from "@/features/wireguard/TrashPage"

function App() {
  return (
    <div className="min-h-screen bg-muted/30">
      <header className="border-b bg-background">
        <div className="container flex min-h-16 flex-wrap items-center justify-between gap-3 py-3">
          <Link to="/" className="flex items-baseline gap-3">
            <h1 className="text-lg font-semibold">WireGuard Panel</h1>
            <span className="hidden text-sm text-muted-foreground sm:inline">
              Manage your VPN concentrator — no CLI
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
              <NavLink to="/trash">
                <Archive />
                Trash
              </NavLink>
            </Button>
          </nav>
        </div>
      </header>
      <main className="container py-8">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/trash" element={<TrashPage />} />
        </Routes>
      </main>
    </div>
  )
}

export default App
