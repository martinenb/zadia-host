"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Loader2, Rocket, ExternalLink, CheckCircle2 } from "lucide-react"

interface DeploySectionProps {
  vpsId: number
  hostPort: number
}

export default function DeploySection({ vpsId, hostPort }: DeploySectionProps) {
  const [code, setCode] = useState("")
  const [filename, setFilename] = useState("index.html")
  const [command, setCommand] = useState("python3 -m http.server 80")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [success, setSuccess] = useState("")
  const [accessUrl, setAccessUrl] = useState("")

  const handleDeploy = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError("")
    setSuccess("")

    try {
      const res = await fetch(`http://localhost:8080/api/vps/${vpsId}/deploy`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ code, filename, command }),
      })

      const data = await res.json()
      if (!res.ok) throw new Error(data.error || "Erreur de déploiement")

      setSuccess("Code déployé avec succès !")
      setAccessUrl(data.access_url || `http://host.mcmr.eu:${hostPort}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erreur inconnue")
    } finally {
      setLoading(false)
    }
  }

  return (
    <form onSubmit={handleDeploy} className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="filename">Nom du fichier</Label>
          <Input
            id="filename"
            value={filename}
            onChange={e => setFilename(e.target.value)}
            placeholder="index.html"
            className="font-mono text-sm"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="command">Commande d'exécution</Label>
          <Input
            id="command"
            value={command}
            onChange={e => setCommand(e.target.value)}
            placeholder="python3 -m http.server 80"
            className="font-mono text-sm"
          />
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="code">Code source</Label>
        <Textarea
          id="code"
          value={code}
          onChange={e => setCode(e.target.value)}
          placeholder="Collez votre code source ici..."
          className="font-mono text-sm min-h-[300px] resize-y"
          required
        />
        <p className="text-xs text-muted-foreground">
          Pour les fichiers HTML, un footer Zadia Host sera automatiquement ajouté.
        </p>
      </div>

      {error && (
        <p className="text-sm text-red-400 bg-red-500/10 border border-red-500/20 rounded-md px-3 py-2">
          {error}
        </p>
      )}

      {success && (
        <div className="bg-green-500/10 border border-green-500/20 rounded-md px-3 py-3 space-y-2">
          <div className="flex items-center gap-2 text-sm text-green-400">
            <CheckCircle2 className="h-4 w-4" />
            {success}
          </div>
          {accessUrl && (
            <a
              href={accessUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1.5 text-xs text-blue-400 hover:text-blue-300 transition-colors"
            >
              <ExternalLink className="h-3 w-3" />
              {accessUrl}
            </a>
          )}
        </div>
      )}

      <Button type="submit" disabled={loading || !code.trim()} className="w-full">
        {loading ? (
          <><Loader2 className="mr-2 h-4 w-4 animate-spin" />Déploiement en cours...</>
        ) : (
          <><Rocket className="mr-2 h-4 w-4" />Déployer le code</>
        )}
      </Button>
    </form>
  )
}
