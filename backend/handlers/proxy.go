package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"

	"zadia-host/db"
)

var subdomainRegex = regexp.MustCompile(`^vps-(\d+)\.`)

func StartSubdomainProxy(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleSubdomainProxy)

	log.Printf("Proxy sous-domaines démarré sur le port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Printf("Erreur proxy sous-domaines: %v", err)
	}
}

func handleSubdomainProxy(w http.ResponseWriter, r *http.Request) {
	host := r.Host

	matches := subdomainRegex.FindStringSubmatch(host)
	if len(matches) < 2 {
		http.Error(w, "VPS introuvable", http.StatusNotFound)
		return
	}

	vpsID, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		http.Error(w, "ID VPS invalide", http.StatusBadRequest)
		return
	}

	vps, err := db.GetVPSByID(vpsID)
	if err != nil || vps == nil || vps.HostPort == 0 {
		http.Error(w, "VPS non trouvé ou inactif", http.StatusNotFound)
		return
	}

	if vps.Status != "running" {
		http.Error(w, "VPS arrêté", http.StatusServiceUnavailable)
		return
	}

	target, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", vps.HostPort))
	if err != nil {
		http.Error(w, "Erreur configuration proxy", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Erreur proxy vps-%d: %v", vpsID, err)
		http.Error(w, "Le service est temporairement indisponible", http.StatusBadGateway)
	}

	// Conserver l'Host original pour les apps qui en ont besoin
	r.Header.Set("X-Forwarded-Host", r.Host)
	r.Header.Set("X-Real-IP", r.RemoteAddr)
	r.URL.Host = target.Host
	r.URL.Scheme = target.Scheme

	proxy.ServeHTTP(w, r)
}
