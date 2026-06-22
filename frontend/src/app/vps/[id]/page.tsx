"use client"

import { useState, useEffect, useCallback } from "react"
import { useParams, useRouter } from "next/navigation"
import Link from "next/link"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import DeploySection from "@/components/DeploySection"
import EnvVarsSection from "@/components/EnvVarsSection"
import { ArrowLeft, Play, Square, Trash2, Cpu, HardDrive, MemoryStick, ExternalLink, Loader2, Terminal } from "lucide-react"

interface VPS {
  id: number
  name: string
  subdomain: string
  os: string
  vcores: number
  ram_gb: number
  disk_gb: number
  status: string
  ip: string
  host_port: number
  ssh_port: number
  ssh_password: string
  deploy_status: string
  app_port: number
  created_at: string
}

function StatusBadge({ status }: { status: string }) {
  if (status === "running") return <Badge variant="success">En ligne</Badge>
  if (status === "stopped") return <Badge variant="error">Arrêté</Badge>
  if (status === "creating") return <Badge variant="warning">Création...</Badge>
  return <Badge variant="outline">{status}</Badge>
}

function SSHPassword({ password }: { password: string }) {
  const [visible, setVisible] = useState(false)
  return (
    <div className="flex items-center gap-2">
      <code className="text-xs bg-muted px-2 py-1.5 rounded font-mono flex-1 select-all">
        {visible ? password : "••••••••••••••"}
      </code>
      <Button variant="ghost" size="sm" className="h-7 text-xs" onClick={() => setVisible(!visible)}>
        {visible ? "Masquer" : "Afficher"}
      </Button>
    </div>
  )
}

export default function VPSDetailPage() {
  const params = useParams()
  const router = useRouter()
  const id = params.id as string

  const [vps, setVps] = useState<VPS | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState("")

  const fetchVPS = useCallback(async () => {
    try {
      const res = await fetch(`/api/vps/${id}`)
      if (res.ok) {
        const data = await res.json()
        setVps(data)
      }
    } catch {} finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    fetchVPS()
    const interval = setInterval(fetchVPS, 5000)
    return () => clearInterval(interval)
  }, [fetchVPS])

  const handleAction = async (action: "start" | "stop" | "delete") => {
    setActionLoading(action)
    try {
      const method = action === "delete" ? "DELETE" : "POST"
      const url = action === "delete"
        ? `/api/vps/${id}`
        : `/api/vps/${id}/${action}`
      await fetch(url, { method })
      if (action === "delete") {
        router.push("/")
      } else {
        fetchVPS()
      }
    } finally {
      setActionLoading("")
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!vps) {
    return (
      <div className="text-center py-24">
        <p className="text-muted-foreground">VPS introuvable.</p>
        <Link href="/"><Button variant="outline" className="mt-4">Retour au dashboard</Button></Link>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link href="/">
          <Button variant="ghost" size="icon" className="h-8 w-8">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </Link>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-semibold">{vps.name}</h1>
            <StatusBadge status={vps.status} />
          </div>
          <p className="text-sm text-muted-foreground mt-0.5 font-mono">
            {vps.ip && vps.ip !== "en attente..." ? vps.ip : "Pas d'IP assignée"}
          </p>
        </div>
        <div className="flex gap-2">
          {vps.subdomain && (
            <a href={`http://${vps.subdomain}.host.mcmr.eu`} target="_blank" rel="noopener noreferrer">
              <Button variant="outline" size="sm">
                <ExternalLink className="h-4 w-4 mr-2" />
                Voir l&apos;app
              </Button>
            </a>
          )}
          <Button
            variant="outline"
            size="sm"
            disabled={!!actionLoading || vps.status === "creating"}
            onClick={() => handleAction(vps.status === "running" ? "stop" : "start")}
          >
            {actionLoading === "start" || actionLoading === "stop" ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : vps.status === "running" ? (
              <><Square className="h-4 w-4 mr-2" />Arrêter</>
            ) : (
              <><Play className="h-4 w-4 mr-2" />Démarrer</>
            )}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            disabled={!!actionLoading}
            onClick={() => handleAction("delete")}
          >
            {actionLoading === "delete" ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <><Trash2 className="h-4 w-4 mr-2" />Supprimer</>
            )}
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Colonne gauche */}
        <div className="space-y-6">
          {/* Ressources */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium">Ressources</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Cpu className="h-4 w-4" />vCores
                </div>
                <span className="font-medium">{vps.vcores}</span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <MemoryStick className="h-4 w-4" />RAM
                </div>
                <span className="font-medium">{vps.ram_gb} Go</span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <HardDrive className="h-4 w-4" />Disque
                </div>
                <span className="font-medium">{vps.disk_gb} Go</span>
              </div>
              <div className="pt-2 border-t border-border">
                <p className="text-xs text-muted-foreground">
                  OS: <span className="capitalize text-foreground">{vps.os}</span>
                </p>
                {vps.host_port > 0 && (
                  <p className="text-xs text-muted-foreground mt-1">
                    Port: <span className="text-foreground font-mono">{vps.host_port}</span>
                  </p>
                )}
              </div>
            </CardContent>
          </Card>

          {/* SSH */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium flex items-center gap-2">
                <Terminal className="h-4 w-4" />
                Accès SSH
              </CardTitle>
            </CardHeader>
            <CardContent>
              {vps.ssh_port > 0 ? (
                <div className="space-y-3">
                  <div>
                    <p className="text-xs text-muted-foreground mb-1">Commande</p>
                    <code className="text-xs bg-muted px-2 py-1.5 rounded block font-mono select-all break-all">
                      ssh root@host.mcmr.eu -p {vps.ssh_port}
                    </code>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground mb-1">Mot de passe</p>
                    <SSHPassword password={vps.ssh_password} />
                  </div>
                </div>
              ) : (
                <div className="text-center py-3 space-y-3">
                  <p className="text-xs text-muted-foreground">SSH non configuré sur ce VPS.</p>
                  {vps.status === "running" && (
                    <Button
                      size="sm"
                      variant="outline"
                      className="text-xs"
                      disabled={actionLoading === "ssh"}
                      onClick={async () => {
                        setActionLoading("ssh")
                        await fetch(`/api/vps/${id}/setup-ssh`, { method: "POST" })
                        // Attendre ~30s que le setup se termine, puis rafraîchir
                        setTimeout(() => { fetchVPS(); setActionLoading("") }, 30000)
                        fetchVPS()
                      }}
                    >
                      {actionLoading === "ssh" ? (
                        <><Loader2 className="h-3 w-3 mr-1.5 animate-spin" />Configuration en cours (~30s)...</>
                      ) : (
                        <><Terminal className="h-3 w-3 mr-1.5" />Configurer SSH</>
                      )}
                    </Button>
                  )}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Variables d'environnement */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium">Variables d&apos;environnement</CardTitle>
              <CardDescription className="text-xs">
                Injectées automatiquement lors du déploiement
              </CardDescription>
            </CardHeader>
            <CardContent>
              <EnvVarsSection vpsId={vps.id} />
            </CardContent>
          </Card>
        </div>

        {/* Déploiement - colonne droite (2/3) */}
        <div className="lg:col-span-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm font-medium">Déploiement ZIP</CardTitle>
              <CardDescription className="text-xs">
                Déposez votre projet en ZIP — détection automatique du framework
              </CardDescription>
            </CardHeader>
            <CardContent>
              {vps.status === "running" ? (
                <DeploySection
                  vpsId={vps.id}
                  subdomain={vps.subdomain}
                  deployStatus={vps.deploy_status}
                  onDeployStart={fetchVPS}
                />
              ) : (
                <div className="flex items-center justify-center py-12 text-center">
                  <div>
                    <p className="text-sm text-muted-foreground">
                      Le VPS doit être en ligne pour déployer du code.
                    </p>
                    {vps.status === "stopped" && (
                      <Button
                        variant="outline"
                        size="sm"
                        className="mt-3"
                        onClick={() => handleAction("start")}
                      >
                        <Play className="h-4 w-4 mr-2" />
                        Démarrer le VPS
                      </Button>
                    )}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
