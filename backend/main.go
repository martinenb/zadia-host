package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"zadia-host/db"
	"zadia-host/handlers"
)

func main() {
	// Connexion PostgreSQL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://zadia:zadia@localhost:5432/zadiahost?sslmode=disable"
	}

	if err := db.InitDB(dbURL); err != nil {
		log.Fatalf("Erreur initialisation DB: %v", err)
	}
	log.Println("Base de données PostgreSQL connectée")

	// Démarrer le proxy sous-domaines en arrière-plan
	proxyPort := os.Getenv("PROXY_PORT")
	if proxyPort == "" {
		proxyPort = "9090"
	}
	go handlers.StartSubdomainProxy(proxyPort)

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:3000",
		AllowHeaders: "Origin, Content-Type, Accept",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	api := app.Group("/api")

	// Routes VPS
	api.Get("/vps", handlers.GetAllVPS)
	api.Post("/vps", handlers.CreateVPS)
	api.Get("/vps/:id", handlers.GetVPS)
	api.Delete("/vps/:id", handlers.DeleteVPS)
	api.Post("/vps/:id/start", handlers.StartVPS)
	api.Post("/vps/:id/stop", handlers.StopVPS)
	api.Post("/vps/:id/deploy", handlers.DeployCode)

	// Routes Variables d'environnement
	api.Get("/vps/:id/env", handlers.GetEnvVars)
	api.Post("/vps/:id/env", handlers.CreateEnvVar)
	api.Delete("/vps/:id/env/:envId", handlers.DeleteEnvVar)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Serveur Zadia Host démarré sur le port %s", port)
	log.Fatal(app.Listen(":" + port))
}
