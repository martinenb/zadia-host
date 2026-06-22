import type { Metadata } from "next"
import { Inter } from "next/font/google"
import "./globals.css"
import Link from "next/link"

const inter = Inter({ subsets: ["latin"], variable: "--font-inter" })

export const metadata: Metadata = {
  title: "Zadia Host",
  description: "Panel d'hébergement hybride IaaS / PaaS",
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="fr" className="dark">
      <body className={`${inter.variable} font-sans antialiased`}>
        <header className="border-b border-border/50 sticky top-0 z-50 bg-background/80 backdrop-blur-sm">
          <div className="max-w-7xl mx-auto px-6 py-4 flex items-center gap-3">
            <div className="w-7 h-7 bg-white rounded-md flex items-center justify-center shadow-sm">
              <span className="text-black font-bold text-sm">Z</span>
            </div>
            <span className="font-semibold text-foreground tracking-tight">Zadia Host</span>
            <nav className="ml-8 flex gap-6 text-sm text-muted-foreground">
              <Link href="/" className="hover:text-foreground transition-colors">Dashboard</Link>
            </nav>
            <div className="ml-auto">
              <span className="text-xs text-muted-foreground px-2 py-1 rounded border border-border">
                host.mcmr.eu
              </span>
            </div>
          </div>
        </header>
        <main className="max-w-7xl mx-auto px-6 py-8">{children}</main>
        <footer className="border-t border-border/50 mt-16">
          <div className="max-w-7xl mx-auto px-6 py-4 text-center text-xs text-muted-foreground">
            Zadia Host — Panel d'hébergement hybride
          </div>
        </footer>
      </body>
    </html>
  )
}
