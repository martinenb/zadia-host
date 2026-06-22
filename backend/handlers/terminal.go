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

// --- WebSocket handler ---

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// TerminalHandler gère les connexions WebSocket vers le terminal d'un VPS.
// URL : /terminal/{id}?token=xxx
// Le token est à usage unique et expire après 60s.
func TerminalHandler(w http.ResponseWriter, r *http.Request) {
	// Extraire l'ID depuis l'URL : /terminal/123
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/terminal/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		http.Error(w, "ID manquant", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(pathParts[0], 10, 64)
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	// Valider le token à usage unique
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Token manquant", http.StatusUnauthorized)
		return
	}

	tokenMu.Lock()
	entry, ok := tokenStore[token]
	if ok {
		delete(tokenStore, token) // consommé immédiatement (usage unique)
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

	// Upgrade HTTP → WebSocket
	ws, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Terminal: erreur upgrade WebSocket pour vps-%d: %v", id, err)
		return
	}
	defer ws.Close()

	log.Printf("Terminal: connexion ouverte pour %s", containerName)

	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	execDone := make(chan struct{})
	go func() {
		defer close(execDone)
		if err := lxd.ExecInteractive(containerName, stdinR, stdoutW, 220, 50); err != nil {
			log.Printf("Terminal: exec terminé pour %s: %v", containerName, err)
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

	log.Printf("Terminal: connexion fermée pour %s", containerName)
}
