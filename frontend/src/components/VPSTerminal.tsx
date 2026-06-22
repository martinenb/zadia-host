"use client"

import { useEffect, useRef, useState } from "react"
import { Loader2, Terminal, WifiOff, TriangleAlert } from "lucide-react"
import "@xterm/xterm/css/xterm.css"

interface VPSTerminalProps {
  vpsId: number
  status: string
}

type TerminalState = "idle" | "requesting" | "connecting" | "connected" | "disconnected" | "error"

export default function VPSTerminal({ vpsId, status }: VPSTerminalProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<import("@xterm/xterm").Terminal | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const fitRef = useRef<import("@xterm/addon-fit").FitAddon | null>(null)
  const roRef = useRef<ResizeObserver | null>(null)
  const [termState, setTermState] = useState<TerminalState>("idle")
  const [errorMsg, setErrorMsg] = useState("")

  // Nettoyage à la fermeture du composant
  useEffect(() => {
    return () => {
      roRef.current?.disconnect()
      wsRef.current?.close()
      termRef.current?.dispose()
    }
  }, [])

  const openTerminal = async () => {
    if (status !== "running") return
    setTermState("requesting")
    setErrorMsg("")

    try {
      // 1. Demander un token à usage unique au backend
      const res = await fetch(`/api/vps/${vpsId}/terminal-token`, { method: "POST" })
      if (!res.ok) {
        const data = await res.json().catch(() => ({}))
        throw new Error(data.error || `Erreur ${res.status}`)
      }
      const { token } = await res.json()

      setTermState("connecting")

      // 2. Import dynamique xterm (nécessite le navigateur)
      const { Terminal } = await import("@xterm/xterm")
      const { FitAddon } = await import("@xterm/addon-fit")

      // Nettoyage si déjà ouvert
      termRef.current?.dispose()
      wsRef.current?.close()

      const term = new Terminal({
        cursorBlink: true,
        fontSize: 13,
        fontFamily: "\"JetBrains Mono\", \"Fira Code\", Menlo, monospace",
        theme: {
          background: "#09090b",
          foreground: "#e4e4e7",
          cursor: "#a1a1aa",
          selectionBackground: "#3f3f46",
          black: "#09090b",
          brightBlack: "#52525b",
          red: "#f87171",
          brightRed: "#fca5a5",
          green: "#4ade80",
          brightGreen: "#86efac",
          yellow: "#facc15",
          brightYellow: "#fde68a",
          blue: "#60a5fa",
          brightBlue: "#93c5fd",
          magenta: "#c084fc",
          brightMagenta: "#d8b4fe",
          cyan: "#22d3ee",
          brightCyan: "#67e8f9",
          white: "#e4e4e7",
          brightWhite: "#f4f4f5",
        },
      })

      const fit = new FitAddon()
      term.loadAddon(fit)

      if (!containerRef.current) throw new Error("Conteneur terminal introuvable")
      term.open(containerRef.current)
      fit.fit()

      termRef.current = term
      fitRef.current = fit

      // 3. Connexion WebSocket avec le token (usage unique, expire en 60s)
      const host = window.location.hostname
      const ws = new WebSocket(`ws://${host}:8085/terminal/${vpsId}?token=${token}`)
      ws.binaryType = "arraybuffer"
      wsRef.current = ws

      ws.onopen = () => {
        setTermState("connected")
        term.focus()
      }

      ws.onmessage = (ev) => {
        if (ev.data instanceof ArrayBuffer) {
          term.write(new Uint8Array(ev.data))
        } else {
          term.write(ev.data)
        }
      }

      ws.onclose = () => {
        setTermState("disconnected")
        term.write("\r\n\x1b[33m[Session terminée — cliquez sur Reconnecter pour une nouvelle session]\x1b[0m\r\n")
      }

      ws.onerror = () => {
        setTermState("error")
        setErrorMsg("Impossible de se connecter au serveur terminal (port 8085).")
      }

      term.onData((data) => {
        if (ws.readyState === WebSocket.OPEN) ws.send(data)
      })

      // Redimensionnement auto
      roRef.current?.disconnect()
      const ro = new ResizeObserver(() => fitRef.current?.fit())
      if (containerRef.current) ro.observe(containerRef.current)
      roRef.current = ro

    } catch (err) {
      setTermState("error")
      setErrorMsg(err instanceof Error ? err.message : "Erreur inconnue")
    }
  }

  const reconnect = () => {
    roRef.current?.disconnect()
    roRef.current = null
    termRef.current?.dispose()
    termRef.current = null
    wsRef.current?.close()
    wsRef.current = null
    setTermState("idle")
  }

  if (status !== "running") {
    return (
      <div className="flex items-center justify-center h-48 text-center">
        <div>
          <WifiOff className="h-8 w-8 text-muted-foreground mx-auto mb-3" />
          <p className="text-sm text-muted-foreground">Le VPS doit être en ligne pour ouvrir un terminal.</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {/* Bouton d'ouverture — affiché seulement en état idle ou error avant mount */}
      {(termState === "idle" || termState === "error") && (
        <div className="flex flex-col items-center justify-center gap-4 py-10 rounded-lg border border-dashed border-border bg-muted/30">
          <Terminal className="h-8 w-8 text-muted-foreground" />
          <div className="text-center">
            <p className="text-sm font-medium">Terminal de recovery</p>
            <p className="text-xs text-muted-foreground mt-1">
              Shell bash isolé — aucun processus auto-démarré
            </p>
          </div>
          {termState === "error" && errorMsg && (
            <div className="flex items-center gap-2 text-xs text-red-400 bg-red-500/10 border border-red-500/20 rounded px-3 py-2 max-w-sm text-center">
              <TriangleAlert className="h-3.5 w-3.5 shrink-0" />
              {errorMsg}
            </div>
          )}
          <button
            onClick={openTerminal}
            className="flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors"
          >
            <Terminal className="h-4 w-4" />
            {termState === "error" ? "Réessayer" : "Ouvrir le terminal"}
          </button>
        </div>
      )}

      {(termState === "requesting" || termState === "connecting") && (
        <div className="flex items-center justify-center gap-2 py-6 text-muted-foreground text-sm">
          <Loader2 className="h-4 w-4 animate-spin" />
          {termState === "requesting" ? "Authentification..." : "Connexion au terminal..."}
        </div>
      )}

      {/* Terminal xterm.js — visible dès que connected ou disconnected */}
      <div
        className={termState === "connected" || termState === "disconnected" ? "block" : "hidden"}
      >
        {(termState === "connected" || termState === "disconnected") && (
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-muted-foreground flex items-center gap-1.5">
              <span className={`h-2 w-2 rounded-full ${termState === "connected" ? "bg-green-400" : "bg-zinc-500"}`} />
              {termState === "connected" ? "Connecté" : "Session terminée"}
            </span>
            <button
              onClick={reconnect}
              className="text-xs px-2.5 py-1 rounded border border-border text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
            >
              {termState === "disconnected" ? "Reconnecter" : "Fermer"}
            </button>
          </div>
        )}
        <div
          ref={containerRef}
          className="rounded overflow-hidden"
          style={{ height: "480px", backgroundColor: "#09090b" }}
        />
      </div>
    </div>
  )
}
