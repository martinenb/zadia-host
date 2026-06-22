package handlers

import (
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"zadia-host/db"
	lxdpkg "zadia-host/lxd"
	"zadia-host/models"
)

func sanitizeSubdomain(name string) string {
	name = strings.ToLower(name)
	re := regexp.MustCompile(`[^a-z0-9-]`)
	name = re.ReplaceAllString(name, "-")
	re2 := regexp.MustCompile(`-+`)
	name = re2.ReplaceAllString(name, "-")
	return strings.Trim(name, "-")
}

func generatePassword() string {
	const chars = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 14)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func GetAllVPS(c *fiber.Ctx) error {
	vpsList, err := db.GetAllVPS()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(vpsList)
}

func GetVPS(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID invalide"})
	}
	vps, err := db.GetVPSByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "VPS non trouvé"})
	}
	return c.JSON(vps)
}

func CreateVPS(c *fiber.Ctx) error {
	var req models.CreateVPSRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Corps de requête invalide"})
	}

	if req.Name == "" || req.OS == "" || req.VCores <= 0 || req.RAMGB <= 0 || req.DiskGB <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "Champs requis manquants"})
	}

	// Sanitiser le nom en subdomain
	subdomain := sanitizeSubdomain(req.Name)
	if subdomain == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Nom de projet invalide pour un sous-domaine"})
	}

	// Vérifier l'unicité du subdomain
	if db.SubdomainExists(subdomain) {
		return c.Status(409).JSON(fiber.Map{"error": fmt.Sprintf("Le sous-domaine '%s' est déjà utilisé", subdomain)})
	}

	vps := &models.VPS{
		Name:      req.Name,
		Subdomain: subdomain,
		OS:        req.OS,
		VCores:    req.VCores,
		RAMGB:     req.RAMGB,
		DiskGB:    req.DiskGB,
		Status:    "creating",
	}

	id, err := db.CreateVPS(vps)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Erreur DB: " + err.Error()})
	}
	vps.ID = id

	containerName := fmt.Sprintf("vps-%d", id)
	hostPort := 8080 + rand.Intn(900)

	go func() {
		log.Printf("[LXD] Création de %s (OS:%s CPU:%d RAM:%dGB Disk:%dGB)", containerName, req.OS, req.VCores, req.RAMGB, req.DiskGB)
		if err := lxdpkg.CreateContainer(containerName, req.OS, req.VCores, req.RAMGB, req.DiskGB); err != nil {
			log.Printf("[LXD] ERREUR création %s: %v", containerName, err)
			db.UpdateVPSStatus(id, "error", "")
			return
		}
		log.Printf("[LXD] %s créé, démarrage...", containerName)

		if err := lxdpkg.StartContainer(containerName); err != nil {
			log.Printf("[LXD] ERREUR démarrage %s: %v", containerName, err)
			db.UpdateVPSStatus(id, "error", "")
			return
		}
		log.Printf("[LXD] %s démarré, attente IP...", containerName)

		time.Sleep(3 * time.Second)

		if err := lxdpkg.AddProxyDevice(containerName, hostPort); err != nil {
			log.Printf("[LXD] AVERTISSEMENT proxy device %s port %d: %v", containerName, hostPort, err)
		}
		db.UpdateVPSHostPort(id, hostPort)

		ip, err := lxdpkg.GetContainerIP(containerName)
		if err != nil {
			log.Printf("[LXD] AVERTISSEMENT IP %s: %v", containerName, err)
			ip = "en attente..."
		}

		log.Printf("[LXD] %s prêt — IP:%s port:%d", containerName, ip, hostPort)
		db.UpdateVPSStatus(id, "running", ip)

		// Générer les credentials SSH
		sshPass := generatePassword()
		sshPort := 20000 + rand.Intn(5000)
		log.Printf("[LXD] %s — Configuration SSH (port %d)...", containerName, sshPort)
		if err := lxdpkg.SetupSSH(containerName, sshPass, req.OS); err != nil {
			log.Printf("[LXD] AVERTISSEMENT SSH %s: %v", containerName, err)
		} else {
			if err := lxdpkg.AddSSHProxyDevice(containerName, sshPort); err != nil {
				log.Printf("[LXD] AVERTISSEMENT proxy SSH %s: %v", containerName, err)
			} else {
				db.UpdateVPSSSH(id, sshPort, sshPass)
				log.Printf("[LXD] %s — SSH prêt (port %d)", containerName, sshPort)
			}
		}
	}()

	return c.Status(202).JSON(fiber.Map{
		"id":        id,
		"subdomain": subdomain,
		"message":   "VPS en cours de création",
		"status":    "creating",
	})
}

func StartVPS(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID invalide"})
	}

	vps, err := db.GetVPSByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "VPS non trouvé"})
	}

	containerName := fmt.Sprintf("vps-%d", vps.ID)
	if err := lxdpkg.StartContainer(containerName); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	ip, _ := lxdpkg.GetContainerIP(containerName)
	db.UpdateVPSStatus(id, "running", ip)

	return c.JSON(fiber.Map{"message": "VPS démarré"})
}

func StopVPS(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID invalide"})
	}

	vps, err := db.GetVPSByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "VPS non trouvé"})
	}

	containerName := fmt.Sprintf("vps-%d", vps.ID)
	if err := lxdpkg.StopContainer(containerName); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	db.UpdateVPSStatus(id, "stopped", "")
	return c.JSON(fiber.Map{"message": "VPS arrêté"})
}

func DeleteVPS(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID invalide"})
	}

	vps, err := db.GetVPSByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "VPS non trouvé"})
	}

	containerName := fmt.Sprintf("vps-%d", vps.ID)
	// Tenter d'arrêter puis supprimer (ignorer les erreurs si déjà arrêté)
	lxdpkg.StopContainer(containerName)
	if err := lxdpkg.DeleteContainer(containerName); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Suppression LXD: " + err.Error()})
	}

	if err := db.DeleteVPS(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Suppression DB: " + err.Error()})
	}

	return c.JSON(fiber.Map{"message": "VPS supprimé"})
}
