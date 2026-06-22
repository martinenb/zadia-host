package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"zadia-host/db"
)

func StartSubdomainProxy(port string) {
	mux := http.NewServeMux()
	// /ws/terminal/ est routé en priorité vers le handler WebSocket du terminal
	mux.HandleFunc("/ws/terminal/", TerminalHandler)
	mux.HandleFunc("/", handleSubdomainProxy)
	log.Printf("Proxy sous-domaines démarré sur le port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Printf("Erreur proxy sous-domaines: %v", err)
	}
}

func handleSubdomainProxy(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	// Extraire le subdomain (ex: "monprojet" de "monprojet.host.mcmr.eu")
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		http.Error(w, "Hôte invalide", http.StatusBadRequest)
		return
	}
	subdomain := parts[0]

	vps, err := db.GetVPSBySubdomain(subdomain)
	if err != nil || vps == nil {
		http.Error(w, "Projet introuvable", http.StatusNotFound)
		return
	}

	if vps.Status != "running" {
		http.Error(w, "VPS arrêté", http.StatusServiceUnavailable)
		return
	}

	if vps.HostPort == 0 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html><html><body style="font-family:sans-serif;text-align:center;padding:60px">
<h2>Aucun projet déployé</h2>
<p>Déployez votre premier projet depuis le <a href="http://host.mcmr.eu:8880/vps/%d">panel Zadia Host</a>.</p>
</body></html>`, vps.ID)
		return
	}

	if vps.DeployStatus == "building" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Refresh", "5")
		fmt.Fprintf(w, `<!DOCTYPE html><html><body style="font-family:sans-serif;text-align:center;padding:60px">
<h2>Déploiement en cours...</h2><p>Cette page se rafraîchira automatiquement.</p>
</body></html>`)
		return
	}

	// Proxyer vers le port LXD proxy device via host.docker.internal
	target, err := url.Parse(fmt.Sprintf("http://host.docker.internal:%d", vps.HostPort))
	if err != nil {
		http.Error(w, "Erreur configuration proxy", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Erreur proxy %s: %v", subdomain, err)
		http.Error(w, "Service temporairement indisponible", http.StatusBadGateway)
	}

	r.Header.Set("X-Forwarded-Host", r.Host)
	r.Header.Set("X-Real-IP", r.RemoteAddr)
	r.URL.Host = target.Host
	r.URL.Scheme = target.Scheme

	proxy.ServeHTTP(w, r)
}
