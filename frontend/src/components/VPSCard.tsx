"use client"

import Link from "next/link"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Cpu, HardDrive, MemoryStick, Play, Square, Trash2, ExternalLink } from "lucide-react"

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

interface VPSCardProps {
  vps: VPS
  onRefresh: () => void
}

function StatusBadge({ status }: { status: string }) {
  if (status === "running") return <Badge variant="success">En ligne</Badge>
  if (status === "stopped") return <Badge variant="error">Arrêté</Badge>
  if (status === "creating") return <Badge variant="warning">Création...</Badge>
  return <Badge variant="outline">{status}</Badge>
}

export default function VPSCard({ vps, onRefresh }: VPSCardProps) {
  const handleAction = async (action: "start" | "stop" | "delete") => {
    const method = action === "delete" ? "DELETE" : "POST"
    const url = action === "delete"
      ? `http://localhost:8080/api/vps/${vps.id}`
      : `http://localhost:8080/api/vps/${vps.id}/${action}`

    await fetch(url, { method })
    onRefresh()
  }

  return (
    <Card className="hover:border-border/80 transition-colors">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div>
            <Link href={`/vps/${vps.id}`} className="font-semibold hover:underline underline-offset-4">
              {vps.name}
            </Link>
            <p className="text-xs text-muted-foreground mt-0.5 capitalize">{vps.os}</p>
          </div>
          <StatusBadge status={vps.status} />
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="grid grid-cols-3 gap-2 text-xs text-muted-foreground">
          <div className="flex items-center gap-1.5">
            <Cpu className="h-3 w-3" />
            <span>{vps.vcores} vCore{vps.vcores > 1 ? "s" : ""}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <MemoryStick className="h-3 w-3" />
            <span>{vps.ram_gb} Go RAM</span>
          </div>
          <div className="flex items-center gap-1.5">
            <HardDrive className="h-3 w-3" />
            <span>{vps.disk_gb} Go</span>
          </div>
        </div>

        {vps.ip && vps.ip !== "en attente..." && (
          <p className="text-xs text-muted-foreground font-mono">IP: {vps.ip}</p>
        )}

        {vps.host_port > 0 && (
          <a
            href={`http://host.mcmr.eu:${vps.host_port}`}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-1.5 text-xs text-blue-400 hover:text-blue-300 transition-colors"
          >
            <ExternalLink className="h-3 w-3" />
            host.mcmr.eu:{vps.host_port}
          </a>
        )}

        <div className="flex gap-2 pt-1">
          <Button
            variant="outline"
            size="sm"
            className="flex-1 text-xs"
            onClick={() => handleAction(vps.status === "running" ? "stop" : "start")}
            disabled={vps.status === "creating"}
          >
            {vps.status === "running" ? (
              <><Square className="h-3 w-3 mr-1" />Arrêter</>
            ) : (
              <><Play className="h-3 w-3 mr-1" />Démarrer</>
            )}
          </Button>
          <Link href={`/vps/${vps.id}`}>
            <Button variant="secondary" size="sm" className="text-xs">Gérer</Button>
          </Link>
          <Button
            variant="ghost"
            size="icon"
            className="h-9 w-9 text-muted-foreground hover:text-destructive-foreground hover:bg-destructive"
            onClick={() => handleAction("delete")}
          >
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
