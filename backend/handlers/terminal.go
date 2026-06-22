package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	fiberws "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gorilla/websocket"
	"zadia-host/db"
	"zadia-host/lxd"
)

// --- Store de tokens à usage unique ---

type termToken struct {
	vpsID     int64
	expiresAt time.Time
}

var (
	tokenStore = map[string]termToken{}
	tokenMu    sync.Mutex
)

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// cleanExpiredTokens supprime les tokens expirés (appelé avant chaque opération).
func cleanExpiredTokens() {
	now := time.Now()
	for k, v := range tokenStore {
		if now.After(v.expiresAt) {
			delete(tokenStore, k)
		}
	}
}

// CreateTerminalToken — POST /api/vps/:id/terminal-token
// Génère un token valide 60s pour ouvrir le WebSocket du terminal.
func CreateTerminalToken(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID invalide"})
	}

	vps, err := db.GetVPSByID(id)
	if err != nil || vps == nil {
		return c.Status(404).JSON(fiber.Map{"error": "VPS introuvable"})
	}
	if vps.Status != "running" {
		return c.Status(503).JSON(fiber.Map{"error": "VPS non démarré"})
	}

	token, err := generateToken()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Erreur génération token"})
	}

	tokenMu.Lock()
	cleanExpiredTokens()
	tokenStore[token] = termToken{
		vpsID:     id,
		expiresAt: time.Now().Add(60 * time.Second),
	}
	tokenMu.Unlock()

	return c.JSON(fiber.Map{"token": token})
}

// --- WebSocket handler Fiber (port 8083) ---

// FiberTerminalHandler gère les connexions WebSocket terminal via Fiber.
// URL : /ws/terminal/:id?token=xxx
func FiberTerminalHandler(c *fiberws.Conn) {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		c.WriteMessage(fiberws.TextMessage, []byte("ID invalide")) //nolint:errcheck
		return
	}

	token := c.Query("token")
	if token == "" {
		c.WriteMessage(fiberws.TextMessage, []byte("Token manquant")) //nolint:errcheck
		return
	}

	tokenMu.Lock()
	entry, ok := tokenStore[token]
	if ok {
		delete(tokenStore, token) // usage unique
	}
	tokenMu.Unlock()

	if !ok || time.Now().After(entry.expiresAt) || entry.vpsID != id {
		c.WriteMessage(fiberws.TextMessage, []byte("Token invalide ou expiré")) //nolint:errcheck
		return
	}

	vps, err := db.GetVPSByID(id)
	if err != nil || vps == nil {
		c.WriteMessage(fiberws.TextMessage, []byte("VPS introuvable")) //nolint:errcheck
		return
	}
	if vps.Status != "running" {
		c.WriteMessage(fiberws.TextMessage, []byte("VPS non démarré")) //nolint:errcheck
		return
	}

	containerName := fmt.Sprintf("vps-%d", id)
	log.Printf("Terminal: connexion ouverte pour %s", containerName)

	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	execDone := make(chan struct{})
	go func() {
		defer close(execDone)
		if err := lxd.ExecInteractive(containerName, stdinR, stdoutW, 220, 50); err != nil {
			log.Printf("Terminal: exec terminé pour %s: %v", containerName, err)
			errMsg := fmt.Sprintf("\r\n\x1b[31m[Erreur LXD: %v]\x1b[0m\r\n", err)
			stdoutW.Write([]byte(errMsg)) //nolint:errcheck
		}
		stdoutW.Close()
	}()

	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		buf := make([]byte, 4096)
		for {
			n, err := stdoutR.Read(buf)
			if n > 0 {
				if werr := c.WriteMessage(fiberws.BinaryMessage, buf[:n]); werr != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
	}()

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			break
		}
		if _, err := stdinW.Write(msg); err != nil {
			break
		}
	}

	stdinW.Close()
	<-execDone
	<-readerDone

	log.Printf("Terminal: connexion fermée pour %s", containerName)
}

// --- WebSocket handler net/http (port 80, subdomain proxy) ---

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// TerminalHandler gère les connexions WebSocket vers le terminal d'un VPS.
// URL : /terminal/{id}?token=xxx (port 80, via StartSubdomainProxy)
func TerminalHandler(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/ws/terminal/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		http.Error(w, "ID manquant", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(pathParts[0], 10, 64)
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Token manquant", http.StatusUnauthorized)
		return
	}

	tokenMu.Lock()
	entry, ok := tokenStore[token]
	if ok {
		delete(tokenStore, token)
	}
	tokenMu.Unlock()

	if !ok || time.Now().After(entry.expiresAt) || entry.vpsID != id {
		http.Error(w, "Token invalide ou expiré", http.StatusUnauthorized)
		return
	}

	vps, err := db.GetVPSByID(id)
	if err != nil || vps == nil {
		http.Error(w, "VPS introuvable", http.StatusNotFound)
		return
	}
	if vps.Status != "running" {
		http.Error(w, "VPS non démarré", http.StatusServiceUnavailable)
		return
	}

	containerName := fmt.Sprintf("vps-%d", id)

	if r.Header.Get("Connection") == "" {
		r.Header.Set("Connection", "upgrade")
	}
	if r.Header.Get("Upgrade") == "" {
		r.Header.Set("Upgrade", "websocket")
	}

	ws, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Terminal: erreur upgrade WebSocket pour vps-%d: %v", id, err)
		return
	}
	defer ws.Close()

	log.Printf("Terminal: connexion ouverte (port 80) pour %s", containerName)

	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	execDone := make(chan struct{})
	go func() {
		defer close(execDone)
		if err := lxd.ExecInteractive(containerName, stdinR, stdoutW, 220, 50); err != nil {
			errMsg := fmt.Sprintf("\r\n\x1b[31m[Erreur LXD: %v]\x1b[0m\r\n", err)
			stdoutW.Write([]byte(errMsg)) //nolint:errcheck
		}
		stdoutW.Close()
	}()

	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		buf := make([]byte, 4096)
		for {
			n, err := stdoutR.Read(buf)
			if n > 0 {
				if werr := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
		ws.Close()
	}()

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			break
		}
		if _, err := stdinW.Write(msg); err != nil {
			break
		}
	}

	stdinW.Close()
	<-execDone
	<-readerDone

	log.Printf("Terminal: connexion fermée (port 80) pour %s", containerName)
}
