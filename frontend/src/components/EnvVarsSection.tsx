"use client"

import { useState, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Plus, Trash2, Loader2 } from "lucide-react"

interface EnvVar {
  id: number
  vps_id: number
  key: string
  value: string
}

interface EnvVarsSectionProps {
  vpsId: number
}

export default function EnvVarsSection({ vpsId }: EnvVarsSectionProps) {
  const [envVars, setEnvVars] = useState<EnvVar[]>([])
  const [newKey, setNewKey] = useState("")
  const [newValue, setNewValue] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")

  const fetchEnvVars = async () => {
    try {
      const res = await fetch(`/api/vps/${vpsId}/env`)
      if (res.ok) {
        const data = await res.json()
        setEnvVars(data || [])
      }
    } catch {}
  }

  useEffect(() => {
    fetchEnvVars()
  }, [vpsId])

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newKey.trim()) return
    setLoading(true)
    setError("")
    try {
      const res = await fetch(`/api/vps/${vpsId}/env`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ key: newKey.trim(), value: newValue }),
      })
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || "Erreur")
      }
      setNewKey("")
      setNewValue("")
      fetchEnvVars()
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erreur")
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (envId: number) => {
    await fetch(`/api/vps/${vpsId}/env/${envId}`, { method: "DELETE" })
    fetchEnvVars()
  }

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-sm font-semibold text-foreground mb-1">Variables d'environnement</h3>
        <p className="text-xs text-muted-foreground">
          Ces variables seront injectées automatiquement lors de chaque déploiement.
        </p>
      </div>

      {envVars.length > 0 && (
        <div className="space-y-2">
          {envVars.map((ev) => (
            <div key={ev.id} className="flex items-center gap-2 p-2 rounded-md border border-border bg-muted/30">
              <Badge variant="outline" className="font-mono text-xs shrink-0">
                {ev.key}
              </Badge>
              <span className="text-xs text-muted-foreground font-mono flex-1 truncate">
                {ev.value || "(vide)"}
              </span>
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6 text-muted-foreground hover:text-destructive"
                onClick={() => handleDelete(ev.id)}
              >
                <Trash2 className="h-3 w-3" />
              </Button>
            </div>
          ))}
        </div>
      )}

      {envVars.length === 0 && (
        <p className="text-xs text-muted-foreground text-center py-4 border border-dashed border-border rounded-md">
          Aucune variable configurée
        </p>
      )}

      <form onSubmit={handleAdd} className="space-y-2">
        <div className="grid grid-cols-2 gap-2">
          <div className="space-y-1">
            <Label className="text-xs">Clé</Label>
            <Input
              placeholder="DATABASE_URL"
              value={newKey}
              onChange={e => setNewKey(e.target.value)}
              className="font-mono text-xs h-8"
            />
          </div>
          <div className="space-y-1">
            <Label className="text-xs">Valeur</Label>
            <Input
              placeholder="valeur_secrète"
              value={newValue}
              onChange={e => setNewValue(e.target.value)}
              className="font-mono text-xs h-8"
              type="text"
            />
          </div>
        </div>
        {error && <p className="text-xs text-red-400">{error}</p>}
        <Button type="submit" size="sm" variant="outline" disabled={loading || !newKey.trim()}>
          {loading ? <Loader2 className="h-3 w-3 mr-1.5 animate-spin" /> : <Plus className="h-3 w-3 mr-1.5" />}
          Ajouter la variable
        </Button>
      </form>
    </div>
  )
}
