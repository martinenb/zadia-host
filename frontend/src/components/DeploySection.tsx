"use client"

import { useState, useRef, useCallback } from "react"
import { Button } from "@/components/ui/button"
import { Loader2, Upload, CheckCircle2, AlertCircle, ExternalLink, FolderArchive } from "lucide-react"

interface DeploySectionProps {
  vpsId: number
  subdomain: string
  deployStatus: string
  onDeployStart: () => void
}

export default function DeploySection({ vpsId, subdomain, deployStatus, onDeployStart }: DeploySectionProps) {
  const [file, setFile] = useState<File | null>(null)
  const [dragging, setDragging] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState("")
  const [result, setResult] = useState<{ label: string; framework: string } | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const handleFile = (f: File) => {
    if (!f.name.endsWith(".zip")) {
      setError("Seuls les fichiers ZIP sont acceptés")
      return
    }
    setFile(f)
    setError("")
    setResult(null)
  }

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setDragging(false)
    const f = e.dataTransfer.files[0]
    if (f) handleFile(f)
  }, [])

  const handleDeploy = async () => {
    if (!file) return
    setUploading(true)
    setError("")

    try {
      const form = new FormData()
      form.append("file", file)
      const res = await fetch(`/api/vps/${vpsId}/deploy`, { method: "POST", body: form })
      const data = await res.json()
      if (!res.ok) throw new Error(data.error || "Erreur de déploiement")
      setResult({ label: data.label, framework: data.framework })
      onDeployStart()
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erreur inconnue")
    } finally {
      setUploading(false)
    }
  }

  const formatSize = (bytes: number) => {
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} Ko`
    return `${(bytes / 1024 / 1024).toFixed(1)} Mo`
  }

  const isBuilding = deployStatus === "building"
  const isRunning = deployStatus === "running"

  return (
    <div className="space-y-4">
      {/* Zone de drop */}
      <div
        className={`border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors ${
          dragging ? "border-primary bg-primary/5" : "border-border hover:border-border/60"
        }`}
        onDragOver={e => { e.preventDefault(); setDragging(true) }}
        onDragLeave={() => setDragging(false)}
        onDrop={handleDrop}
        onClick={() => inputRef.current?.click()}
      >
        <input
          ref={inputRef}
          type="file"
          accept=".zip"
          className="hidden"
          onChange={e => e.target.files?.[0] && handleFile(e.target.files[0])}
        />
        {file ? (
          <div className="space-y-1">
            <FolderArchive className="h-8 w-8 mx-auto text-primary" />
            <p className="text-sm font-medium">{file.name}</p>
            <p className="text-xs text-muted-foreground">{formatSize(file.size)}</p>
          </div>
        ) : (
          <div className="space-y-2">
            <Upload className="h-8 w-8 mx-auto text-muted-foreground" />
            <div>
              <p className="text-sm font-medium">Glissez votre projet ici</p>
              <p className="text-xs text-muted-foreground mt-1">
                ZIP contenant votre projet (Next.js, Python, PHP, HTML...)
              </p>
            </div>
          </div>
        )}
      </div>

      {/* Info frameworks supportés */}
      <div className="flex flex-wrap gap-1.5">
        {["Next.js", "Vite", "React", "Python", "Flask", "FastAPI", "PHP", "HTML"].map(f => (
          <span key={f} className="text-xs px-2 py-0.5 rounded-full bg-muted text-muted-foreground border border-border">
            {f}
          </span>
        ))}
      </div>

      {/* Erreur */}
      {error && (
        <div className="flex items-start gap-2 text-sm text-red-400 bg-red-500/10 border border-red-500/20 rounded-md px-3 py-2">
          <AlertCircle className="h-4 w-4 mt-0.5 shrink-0" />
          {error}
        </div>
      )}

      {/* Statut building */}
      {isBuilding && (
        <div className="flex items-center gap-2 text-sm text-yellow-400 bg-yellow-500/10 border border-yellow-500/20 rounded-md px-3 py-2">
          <Loader2 className="h-4 w-4 animate-spin shrink-0" />
          Installation des dépendances et démarrage en cours...
        </div>
      )}

      {/* Succès */}
      {(isRunning || result) && (
        <div className="bg-green-500/10 border border-green-500/20 rounded-md px-3 py-3 space-y-2">
          <div className="flex items-center gap-2 text-sm text-green-400">
            <CheckCircle2 className="h-4 w-4" />
            {result ? `${result.label} déployé avec succès` : "Projet en ligne"}
          </div>
          {subdomain && (
            <a
              href={`http://${subdomain}.host.mcmr.eu`}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1.5 text-xs text-blue-400 hover:text-blue-300 transition-colors"
            >
              <ExternalLink className="h-3 w-3" />
              {subdomain}.host.mcmr.eu
            </a>
          )}
        </div>
      )}

      <Button
        onClick={handleDeploy}
        disabled={!file || uploading || isBuilding}
        className="w-full"
      >
        {uploading ? (
          <><Loader2 className="mr-2 h-4 w-4 animate-spin" />Upload en cours...</>
        ) : isBuilding ? (
          <><Loader2 className="mr-2 h-4 w-4 animate-spin" />Build en cours...</>
        ) : (
          <><Upload className="mr-2 h-4 w-4" />Déployer le projet</>
        )}
      </Button>

      <p className="text-xs text-muted-foreground">
        N&apos;incluez pas <code className="bg-muted px-1 rounded">node_modules</code> ou <code className="bg-muted px-1 rounded">.git</code> dans votre ZIP.
      </p>
    </div>
  )
}
