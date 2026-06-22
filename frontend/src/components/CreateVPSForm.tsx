"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Loader2 } from "lucide-react"

interface CreateVPSFormProps {
  onSuccess: () => void
  onCancel: () => void
}

export default function CreateVPSForm({ onSuccess, onCancel }: CreateVPSFormProps) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [form, setForm] = useState({
    name: "",
    os: "ubuntu",
    vcores: 1,
    ram_gb: 1,
    disk_gb: 10,
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError("")

    try {
      const res = await fetch("/api/vps", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(form),
      })
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || "Erreur lors de la création")
      }
      onSuccess()
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erreur inconnue")
    } finally {
      setLoading(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="name">Nom du VPS</Label>
        <Input
          id="name"
          placeholder="mon-serveur-01"
          value={form.name}
          onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
          required
        />
      </div>

      <div className="space-y-2">
        <Label>Système d'exploitation</Label>
        <Select value={form.os} onValueChange={v => setForm(f => ({ ...f, os: v }))}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="ubuntu">Ubuntu 22.04 LTS</SelectItem>
            <SelectItem value="debian">Debian 12</SelectItem>
            <SelectItem value="alpine">Alpine 3.19</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="space-y-2">
          <Label htmlFor="vcores">vCores</Label>
          <Input
            id="vcores"
            type="number"
            min={1}
            max={8}
            value={form.vcores}
            onChange={e => setForm(f => ({ ...f, vcores: parseInt(e.target.value) || 1 }))}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="ram">RAM (Go)</Label>
          <Input
            id="ram"
            type="number"
            min={1}
            max={16}
            value={form.ram_gb}
            onChange={e => setForm(f => ({ ...f, ram_gb: parseInt(e.target.value) || 1 }))}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="disk">Disque (Go)</Label>
          <Input
            id="disk"
            type="number"
            min={10}
            max={100}
            value={form.disk_gb}
            onChange={e => setForm(f => ({ ...f, disk_gb: parseInt(e.target.value) || 10 }))}
          />
        </div>
      </div>

      {error && (
        <p className="text-sm text-red-400 bg-red-500/10 border border-red-500/20 rounded-md px-3 py-2">
          {error}
        </p>
      )}

      <div className="flex gap-3 pt-2">
        <Button type="submit" disabled={loading} className="flex-1">
          {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          {loading ? "Création en cours..." : "Créer le VPS"}
        </Button>
        <Button type="button" variant="outline" onClick={onCancel}>
          Annuler
        </Button>
      </div>
    </form>
  )
}
