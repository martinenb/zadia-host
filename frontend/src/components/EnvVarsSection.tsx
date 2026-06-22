"use client"

import { useState, useEffect, useCallback } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Plus, Trash2, Loader2, FileCode2, Check, X, RefreshCw, ChevronDown, ChevronUp } from "lucide-react"

interface EnvVar {
  id: number
  vps_id: number
  key: string
}

interface EnvVarsSectionProps {
  vpsId: number
}

function parseDotEnv(text: string): { key: string; value: string }[] {
  const result: { key: string; value: string }[] = []
  for (const raw of text.split("\n")) {
    const line = raw.trim()
    if (!line || line.startsWith("#")) continue
    const idx = line.indexOf("=")
    if (idx < 1) continue
    const key = line.slice(0, idx).trim()
    let value = line.slice(idx + 1).trim()
    // Retirer les guillemets entourant la valeur
    if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
      value = value.slice(1, -1)
    }
    if (key) result.push({ key, value })
  }
  return result
}

export default function EnvVarsSection({ vpsId }: EnvVarsSectionProps) {
  const [envVars, setEnvVars] = useState<EnvVar[]>([])
  const [newKey, setNewKey] = useState("")
  const [newValue, setNewValue] = useState("")
  const [addLoading, setAddLoading] = useState(false)
  const [addError, setAddError] = useState("")

  // Remplacement d'une variable existante
  const [replacingId, setReplacingId] = useState<number | null>(null)
  const [replaceValue, setReplaceValue] = useState("")
  const [replaceLoading, setReplaceLoading] = useState(false)

  // Import .env
  const [showImport, setShowImport] = useState(false)
  const [importText, setImportText] = useState("")
  const [parsedVars, setParsedVars] = useState<{ key: string; value: string }[]>([])
  const [importLoading, setImportLoading] = useState(false)
  const [importDone, setImportDone] = useState(0)

  const fetchEnvVars = useCallback(async () => {
    try {
      const res = await fetch(`/api/vps/${vpsId}/env`)
      if (res.ok) {
        const data = await res.json()
        // On ne stocke que la clé et l'ID — jamais la valeur en état
        setEnvVars((data || []).map((ev: { id: number; vps_id: number; key: string }) => ({
          id: ev.id,
          vps_id: ev.vps_id,
          key: ev.key,
        })))
      }
    } catch {}
  }, [vpsId])

  useEffect(() => {
    fetchEnvVars()
  }, [fetchEnvVars])

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newKey.trim() || !newValue) return
    setAddLoading(true)
    setAddError("")
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
      setAddError(err instanceof Error ? err.message : "Erreur")
    } finally {
      setAddLoading(false)
    }
  }

  const handleDelete = async (envId: number) => {
    await fetch(`/api/vps/${vpsId}/env/${envId}`, { method: "DELETE" })
    if (replacingId === envId) setReplacingId(null)
    fetchEnvVars()
  }

  const handleReplace = async (envId: number, key: string) => {
    if (!replaceValue) return
    setReplaceLoading(true)
    // Supprimer l'ancienne + créer la nouvelle
    await fetch(`/api/vps/${vpsId}/env/${envId}`, { method: "DELETE" })
    await fetch(`/api/vps/${vpsId}/env`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key, value: replaceValue }),
    })
    setReplacingId(null)
    setReplaceValue("")
    setReplaceLoading(false)
    fetchEnvVars()
  }

  const handleParseImport = () => {
    setParsedVars(parseDotEnv(importText))
  }

  const handleImport = async () => {
    if (parsedVars.length === 0) return
    setImportLoading(true)
    let count = 0
    for (const { key, value } of parsedVars) {
      // Si la clé existe déjà, supprimer d'abord
      const existing = envVars.find(ev => ev.key === key)
      if (existing) {
        await fetch(`/api/vps/${vpsId}/env/${existing.id}`, { method: "DELETE" })
      }
      const res = await fetch(`/api/vps/${vpsId}/env`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ key, value }),
      })
      if (res.ok) count++
    }
    setImportDone(count)
    setImportText("")
    setParsedVars([])
    setShowImport(false)
    setImportLoading(false)
    fetchEnvVars()
  }

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between">
        <div>
          <h3 className="text-sm font-semibold text-foreground">Variables d'environnement</h3>
          <p className="text-xs text-muted-foreground mt-0.5">
            Injectées lors du déploiement. Les valeurs sont masquées.
          </p>
        </div>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 text-xs gap-1.5 text-muted-foreground"
          onClick={() => { setShowImport(!showImport); setParsedVars([]); setImportText("") }}
        >
          <FileCode2 className="h-3.5 w-3.5" />
          Importer .env
          {showImport ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
        </Button>
      </div>

      {/* Import .env panel */}
      {showImport && (
        <div className="border border-border rounded-md p-3 space-y-3 bg-muted/20">
          <p className="text-xs text-muted-foreground">
            Collez le contenu de votre fichier <code className="bg-muted px-1 rounded">.env</code> ci-dessous.
            Les variables existantes avec le même nom seront remplacées.
          </p>
          <textarea
            className="w-full h-32 text-xs font-mono bg-background border border-border rounded-md px-3 py-2 resize-y outline-none focus:ring-1 focus:ring-ring"
            placeholder={"DATABASE_URL=postgres://user:pass@host/db\nSECRET_KEY=abc123\nAPI_TOKEN=xyz"}
            value={importText}
            onChange={e => { setImportText(e.target.value); setParsedVars([]) }}
            spellCheck={false}
          />
          {parsedVars.length > 0 && (
            <div className="space-y-1">
              <p className="text-xs text-muted-foreground">{parsedVars.length} variable{parsedVars.length > 1 ? "s" : ""} détectée{parsedVars.length > 1 ? "s" : ""} :</p>
              <div className="flex flex-wrap gap-1">
                {parsedVars.map(v => (
                  <span key={v.key} className="text-xs font-mono bg-muted px-1.5 py-0.5 rounded border border-border">
                    {v.key}
                  </span>
                ))}
              </div>
            </div>
          )}
          <div className="flex gap-2">
            {parsedVars.length === 0 ? (
              <Button size="sm" variant="outline" className="text-xs h-7" onClick={handleParseImport} disabled={!importText.trim()}>
                Analyser
              </Button>
            ) : (
              <Button size="sm" className="text-xs h-7" onClick={handleImport} disabled={importLoading}>
                {importLoading
                  ? <><Loader2 className="h-3 w-3 mr-1.5 animate-spin" />Import...</>
                  : <><Check className="h-3 w-3 mr-1.5" />Importer {parsedVars.length} variable{parsedVars.length > 1 ? "s" : ""}</>
                }
              </Button>
            )}
            <Button size="sm" variant="ghost" className="text-xs h-7" onClick={() => { setShowImport(false); setParsedVars([]); setImportText("") }}>
              Annuler
            </Button>
          </div>
        </div>
      )}

      {/* Confirmation import */}
      {importDone > 0 && (
        <p className="text-xs text-green-400 flex items-center gap-1">
          <Check className="h-3 w-3" />
          {importDone} variable{importDone > 1 ? "s" : ""} importée{importDone > 1 ? "s" : ""}
        </p>
      )}

      {/* Liste des variables */}
      {envVars.length > 0 ? (
        <div className="space-y-1.5">
          {envVars.map((ev) => (
            <div key={ev.id} className="rounded-md border border-border bg-muted/20 overflow-hidden">
              <div className="flex items-center gap-2 px-2 py-1.5">
                <code className="text-xs font-mono text-foreground shrink-0">{ev.key}</code>
                <span className="text-xs text-muted-foreground font-mono flex-1 tracking-widest select-none">
                  ••••••••••••
                </span>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-6 text-xs px-2 text-muted-foreground hover:text-foreground"
                  onClick={() => { setReplacingId(replacingId === ev.id ? null : ev.id); setReplaceValue("") }}
                >
                  <RefreshCw className="h-3 w-3 mr-1" />
                  Remplacer
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6 text-muted-foreground hover:text-destructive shrink-0"
                  onClick={() => handleDelete(ev.id)}
                >
                  <Trash2 className="h-3 w-3" />
                </Button>
              </div>
              {replacingId === ev.id && (
                <div className="flex items-center gap-2 px-2 pb-2 pt-0">
                  <Input
                    autoFocus
                    type="password"
                    placeholder="Nouvelle valeur..."
                    value={replaceValue}
                    onChange={e => setReplaceValue(e.target.value)}
                    className="font-mono text-xs h-7 flex-1"
                    onKeyDown={e => e.key === "Enter" && handleReplace(ev.id, ev.key)}
                  />
                  <Button
                    size="sm"
                    className="h-7 text-xs px-2"
                    onClick={() => handleReplace(ev.id, ev.key)}
                    disabled={!replaceValue || replaceLoading}
                  >
                    {replaceLoading ? <Loader2 className="h-3 w-3 animate-spin" /> : <Check className="h-3 w-3" />}
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-7 text-xs px-2"
                    onClick={() => { setReplacingId(null); setReplaceValue("") }}
                  >
                    <X className="h-3 w-3" />
                  </Button>
                </div>
              )}
            </div>
          ))}
        </div>
      ) : (
        <p className="text-xs text-muted-foreground text-center py-4 border border-dashed border-border rounded-md">
          Aucune variable configurée
        </p>
      )}

      {/* Ajout manuel */}
      <form onSubmit={handleAdd} className="space-y-2 pt-1 border-t border-border">
        <div className="grid grid-cols-2 gap-2">
          <div className="space-y-1">
            <Label className="text-xs text-muted-foreground">Clé</Label>
            <Input
              placeholder="DATABASE_URL"
              value={newKey}
              onChange={e => setNewKey(e.target.value)}
              className="font-mono text-xs h-8"
              autoComplete="off"
            />
          </div>
          <div className="space-y-1">
            <Label className="text-xs text-muted-foreground">Valeur</Label>
            <Input
              type="password"
              placeholder="••••••••"
              value={newValue}
              onChange={e => setNewValue(e.target.value)}
              className="font-mono text-xs h-8"
              autoComplete="new-password"
            />
          </div>
        </div>
        {addError && <p className="text-xs text-red-400">{addError}</p>}
        <Button type="submit" size="sm" variant="outline" className="text-xs h-7" disabled={addLoading || !newKey.trim() || !newValue}>
          {addLoading ? <Loader2 className="h-3 w-3 mr-1.5 animate-spin" /> : <Plus className="h-3 w-3 mr-1.5" />}
          Ajouter
        </Button>
      </form>
    </div>
  )
}
