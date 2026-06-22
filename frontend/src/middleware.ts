import { NextResponse } from "next/server"
import type { NextRequest } from "next/server"

const ALLOWED_IP = "217.154.102.163"

export function middleware(request: NextRequest) {
  // CF-Connecting-IP : IP réelle du visiteur injectée par Cloudflare
  // X-Forwarded-For  : fallback si accès direct sans Cloudflare
  const cfIp = request.headers.get("cf-connecting-ip")
  const forwarded = request.headers.get("x-forwarded-for")?.split(",")[0].trim()
  const realIp = request.headers.get("x-real-ip")

  const clientIp = cfIp ?? forwarded ?? realIp ?? "unknown"

  if (clientIp !== ALLOWED_IP) {
    return new NextResponse(
      `<!DOCTYPE html>
<html lang="fr">
<head>
  <meta charset="UTF-8" />
  <title>Accès refusé — Zadia Host</title>
  <style>
    body { background: #0a0a0a; color: #888; font-family: sans-serif;
           display: flex; align-items: center; justify-content: center;
           height: 100vh; margin: 0; }
    .box { text-align: center; }
    h1 { color: #fff; font-size: 1.5rem; margin-bottom: .5rem; }
    code { background: #1a1a1a; padding: 2px 8px; border-radius: 4px;
           font-size: .85rem; color: #f87171; }
  </style>
</head>
<body>
  <div class="box">
    <h1>Accès refusé</h1>
    <p>Cette interface est réservée aux administrateurs.</p>
    <p><code>${clientIp}</code></p>
  </div>
</body>
</html>`,
      {
        status: 403,
        headers: { "Content-Type": "text/html; charset=utf-8" },
      }
    )
  }

  return NextResponse.next()
}

export const config = {
  // Appliqué à toutes les routes sauf assets statiques Next.js
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
}
