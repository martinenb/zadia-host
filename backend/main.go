package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"zadia-host/db"
	"zadia-host/handlers"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://zadia:zadia@localhost:5432/zadiahost?sslmode=disable"
	}

	if err := db.InitDB(dbURL); err != nil {
		log.Fatalf("Erreur initialisation DB: %v", err)
	}
	log.Println("Base de données PostgreSQL connectée")

	proxyPort := os.Getenv("PROXY_PORT")
	if proxyPort == "" {
		proxyPort = "80"
	}
	go handlers.StartSubdomainProxy(proxyPort)

	app := fiber.New(fiber.Config{
		BodyLimit: 200 * 1024 * 1024,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	api := app.Group("/api")
	api.Get("/vps", handlers.GetAllVPS)
	api.Post("/vps", handlers.CreateVPS)
	api.Get("/vps/:id", handlers.GetVPS)
	api.Delete("/vps/:id", handlers.DeleteVPS)
	api.Post("/vps/:id/start", handlers.StartVPS)
	api.Post("/vps/:id/stop", handlers.StopVPS)
	api.Post("/vps/:id/setup-ssh", handlers.SetupSSH)
	api.Post("/vps/:id/deploy", handlers.DeployProject)
	api.Post("/vps/:id/terminal-token", handlers.CreateTerminalToken)

	api.Get("/vps/:id/env", handlers.GetEnvVars)
	api.Post("/vps/:id/env", handlers.CreateEnvVar)
	api.Delete("/vps/:id/env/:envId", handlers.DeleteEnvVar)

	// Fiber tourne sur un port interne non exposé (8084).
	// Le mux net/http public (port 8083) route :
	//   /ws/terminal/* → gorilla/websocket (WebSocket natif)
	//   tout le reste   → Fiber via reverse proxy localhost
	go func() {
		if err := app.Listen(":8084"); err != nil {
			log.Fatalf("Erreur Fiber: %v", err)
		}
	}()

	fiberProxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: "http",
		Host:   "localhost:8084",
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/ws/terminal/", handlers.TerminalHandler)
	mux.Handle("/", fiberProxy)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Serveur Zadia Host démarré sur le port %s (API + WebSocket terminal)", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
