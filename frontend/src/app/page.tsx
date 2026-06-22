"use client"

import { useState, useEffect, useCallback } from "react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import CreateVPSForm from "@/components/CreateVPSForm"
import VPSCard from "@/components/VPSCard"
import { Plus, Server, RefreshCw } from "lucide-react"

interface VPS {
  id: number
  name: string
  os: string
  vcores: number
  ram_gb: number
  disk_gb: number
  status: string
  ip: string
  host_port: number
  created_at: string
}

export default function HomePage() {
  const [vpsList, setVpsList] = useState<VPS[]>([])
  const [showForm, setShowForm] = useState(false)
  const [loading, setLoading] = useState(true)

  const fetchVPS = useCallback(async () => {
    try {
      const res = await fetch("/api/vps")
      if (res.ok) {
        const data = await res.json()
        setVpsList(data || [])
      }
    } catch {} finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchVPS()
    const interval = setInterval(fetchVPS, 5000)
    return () => clearInterval(interval)
  }, [fetchVPS])

  const handleFormSuccess = () => {
    setShowForm(false)
    setTimeout(fetchVPS, 1000)
  }

  const running = vpsList.filter(v => v.status === "running").length
  const total = vpsList.length

  return (
    <div className="space-y-8">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Infrastructure</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {total === 0 ? "Aucun VPS configuré" : `${running}/${total} instances en ligne`}
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={fetchVPS}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Actualiser
          </Button>
          <Button size="sm" onClick={() => setShowForm(true)}>
            <Plus className="h-4 w-4 mr-2" />
            Nouveau VPS
          </Button>
        </div>
      </div>

      {showForm && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Créer un nouveau VPS</CardTitle>
          </CardHeader>
          <CardContent>
            <CreateVPSForm onSuccess={handleFormSuccess} onCancel={() => setShowForm(false)} />
          </CardContent>
        </Card>
      )}

      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {[1, 2, 3].map(i => (
            <div key={i} className="h-48 rounded-lg border border-border bg-card animate-pulse" />
          ))}
        </div>
      ) : vpsList.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-24 text-center">
          <div className="w-14 h-14 rounded-full bg-muted flex items-center justify-center mb-4">
            <Server className="h-7 w-7 text-muted-foreground" />
          </div>
          <h3 className="font-medium mb-1">Aucune instance</h3>
          <p className="text-sm text-muted-foreground mb-4">
            Créez votre premier VPS pour commencer.
          </p>
          <Button size="sm" onClick={() => setShowForm(true)}>
            <Plus className="h-4 w-4 mr-2" />
            Créer un VPS
          </Button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {vpsList.map(vps => (
            <VPSCard key={vps.id} vps={vps} onRefresh={fetchVPS} />
          ))}
        </div>
      )}
    </div>
  )
}
