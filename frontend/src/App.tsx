import { Dashboard } from "@/features/wireguard/Dashboard"

function App() {
  return (
    <div className="min-h-screen bg-muted/30">
      <header className="border-b bg-background">
        <div className="container flex h-16 items-center">
          <h1 className="text-lg font-semibold">WireGuard Panel</h1>
          <span className="ml-3 hidden text-sm text-muted-foreground sm:inline">
            Manage your VPN concentrator — no CLI
          </span>
        </div>
      </header>
      <main className="container py-8">
        <Dashboard />
      </main>
    </div>
  )
}

export default App
