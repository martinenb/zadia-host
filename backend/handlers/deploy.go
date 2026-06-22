package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"zadia-host/db"
	lxdpkg "zadia-host/lxd"
	"zadia-host/models"
)

const watermark = `<footer style="position:fixed;bottom:0;left:0;right:0;text-align:center;padding:10px 20px;font-family:sans-serif;color:#888;font-size:12px;background:rgba(0,0,0,0.05);">Hébergé sur <strong>Zadia Host</strong></footer>`

func DeployCode(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID invalide"})
	}

	vps, err := db.GetVPSByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "VPS non trouvé"})
	}

	if vps.Status != "running" {
		return c.Status(400).JSON(fiber.Map{"error": "Le VPS doit être en cours d'exécution"})
	}

	var req models.DeployRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Corps de requête invalide"})
	}

	if req.Code == "" || req.Filename == "" || req.Command == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Code, nom de fichier et commande requis"})
	}

	containerName := fmt.Sprintf("vps-%d", vps.ID)

	// Injecter le watermark si fichier HTML
	code := req.Code
	if strings.HasSuffix(strings.ToLower(req.Filename), ".html") {
		if idx := strings.LastIndex(code, "</body>"); idx != -1 {
			code = code[:idx] + watermark + "\n" + code[idx:]
		} else {
			code = code + "\n" + watermark
		}
	}

	// Récupérer les variables d'env depuis la DB
	envVars, err := db.GetAllEnvVarsAsMap(id)
	if err != nil {
		envVars = make(map[string]string)
	}

	// Fusionner avec les env_vars de la requête
	if req.EnvVars != nil {
		for k, v := range req.EnvVars {
			envVars[k] = v
		}
	}

	// Créer le répertoire /root/app/
	if err := lxdpkg.EnsureDirectory(containerName, "/root/app"); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Création répertoire: " + err.Error()})
	}

	// Pousser le fichier
	destPath := "/root/app/" + req.Filename
	if err := lxdpkg.PushFile(containerName, destPath, code); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Upload fichier: " + err.Error()})
	}

	// Construire la commande avec les variables d'env
	shellCmd := buildCommandWithEnv(req.Command, envVars)

	// Exécuter la commande en arrière-plan via nohup
	bgCommand := []string{"sh", "-c", fmt.Sprintf("cd /root/app && nohup %s > /root/app/output.log 2>&1 &", shellCmd)}
	if err := lxdpkg.ExecCommand(containerName, bgCommand, envVars); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Exécution commande: " + err.Error()})
	}

	accessURL := fmt.Sprintf("http://host.mcmr.eu:%d", vps.HostPort)
	return c.JSON(fiber.Map{
		"message":    "Code déployé avec succès",
		"access_url": accessURL,
		"file":       destPath,
	})
}

func buildCommandWithEnv(command string, env map[string]string) string {
	if len(env) == 0 {
		return command
	}
	var parts []string
	for k, v := range env {
		parts = append(parts, fmt.Sprintf("export %s=%q", k, v))
	}
	return strings.Join(parts, " && ") + " && " + command
}
