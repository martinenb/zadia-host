package handlers

import (
	"archive/zip"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"zadia-host/db"
	lxdpkg "zadia-host/lxd"
)

func DeployProject(c *fiber.Ctx) error {
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

	// Récupérer le fichier ZIP
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Fichier ZIP requis"})
	}
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".zip") {
		return c.Status(400).JSON(fiber.Map{"error": "Seuls les fichiers ZIP sont acceptés"})
	}

	// Sauvegarder temporairement sur le host
	tmpZip := fmt.Sprintf("/tmp/zadia-deploy-%d-%d.zip", id, time.Now().UnixNano())
	if err := c.SaveFile(file, tmpZip); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Sauvegarde fichier: " + err.Error()})
	}

	// Extraire pour détection
	tmpDir := tmpZip + "-extracted"
	if err := extractZip(tmpZip, tmpDir); err != nil {
		os.Remove(tmpZip)
		return c.Status(500).JSON(fiber.Map{"error": "Extraction ZIP: " + err.Error()})
	}

	// Détecter le type de projet
	info := lxdpkg.DetectProject(tmpDir, vps.OS)

	if info.Framework == "unknown" {
		os.RemoveAll(tmpDir)
		os.Remove(tmpZip)
		return c.Status(400).JSON(fiber.Map{
			"error": "Type de projet non reconnu. Assurez-vous que votre ZIP contient un package.json, requirements.txt, index.php ou index.html",
		})
	}

	log.Printf("[DEPLOY] VPS %d — Projet détecté: %s", id, info.Label)

	// Mettre à jour le statut en DB
	db.UpdateVPSDeploy(id, "building", info.AppPort)

	containerName := fmt.Sprintf("vps-%d", vps.ID)

	// Lancer le déploiement en asynchrone
	go func() {
		defer os.RemoveAll(tmpDir)
		defer os.Remove(tmpZip)

		log.Printf("[DEPLOY] %s — Envoi du ZIP dans le conteneur...", containerName)

		// Pousser le ZIP dans le conteneur
		f, err := os.Open(tmpZip)
		if err != nil {
			log.Printf("[DEPLOY] ERREUR ouverture ZIP: %v", err)
			db.UpdateVPSDeploy(id, "error", 0)
			return
		}
		defer f.Close()

		lxdpkg.EnsureDirectory(containerName, "/root/app")
		if err := lxdpkg.PushBinaryFile(containerName, "/root/app/project.zip", f); err != nil {
			log.Printf("[DEPLOY] ERREUR push ZIP: %v", err)
			db.UpdateVPSDeploy(id, "error", 0)
			return
		}

		// Installer unzip dans le conteneur
		switch vps.OS {
		case "alpine":
			lxdpkg.ExecCommand(containerName, []string{"apk", "add", "--no-cache", "unzip"}, nil)
		default:
			lxdpkg.ExecCommand(containerName, []string{"apt-get", "install", "-y", "-qq", "unzip"}, nil)
		}

		// Extraire le ZIP dans /root/app/
		lxdpkg.ExecCommand(containerName, []string{"sh", "-c",
			"rm -rf /root/app/src && mkdir -p /root/app/src && unzip -o /root/app/project.zip -d /root/app/src && rm /root/app/project.zip"}, nil)

		// Setup : installation du runtime (node, python...)
		for _, cmd := range info.SetupCmds {
			log.Printf("[DEPLOY] Setup: %s", strings.Join(cmd, " "))
			if err := lxdpkg.ExecCommand(containerName, cmd, nil); err != nil {
				log.Printf("[DEPLOY] AVERTISSEMENT setup: %v", err)
			}
		}

		// Build (install deps + compilation)
		for _, cmd := range info.BuildCmds {
			log.Printf("[DEPLOY] Build: %s", strings.Join(cmd, " "))
			fullCmd := []string{"sh", "-c", "cd /root/app/src && " + strings.Join(cmd, " ")}
			if err := lxdpkg.ExecCommand(containerName, fullCmd, nil); err != nil {
				log.Printf("[DEPLOY] AVERTISSEMENT build: %v", err)
			}
		}

		// Tuer l'ancienne instance si elle tourne
		lxdpkg.ExecCommand(containerName, []string{"sh", "-c",
			fmt.Sprintf("fuser -k %d/tcp 2>/dev/null || pkill -f '%s' 2>/dev/null || true", info.AppPort, info.StartCmd)}, nil)

		// Démarrer l'application en arrière-plan
		bgCmd := fmt.Sprintf("cd /root/app/src && nohup %s > /root/app/output.log 2>&1 &", info.StartCmd)
		log.Printf("[DEPLOY] Démarrage: %s", info.StartCmd)

		// Récupérer les variables d'env
		envVars, _ := db.GetAllEnvVarsAsMap(id)
		if err := lxdpkg.ExecCommand(containerName, []string{"sh", "-c", bgCmd}, envVars); err != nil {
			log.Printf("[DEPLOY] ERREUR démarrage: %v", err)
			db.UpdateVPSDeploy(id, "error", 0)
			return
		}

		// Mettre à jour le proxy device si le port a changé
		if vps.HostPort > 0 && info.AppPort != 80 {
			// Le proxy device actuel pointe vers port 80, on le met à jour
			lxdpkg.UpdateProxyDevice(containerName, vps.HostPort, info.AppPort)
		}

		db.UpdateVPSDeploy(id, "running", info.AppPort)
		log.Printf("[DEPLOY] %s — Déploiement terminé (%s)", containerName, info.Label)
	}()

	return c.Status(202).JSON(fiber.Map{
		"status":    "building",
		"framework": info.Framework,
		"label":     info.Label,
		"message":   fmt.Sprintf("Déploiement %s en cours...", info.Label),
	})
}

// extractZip extrait un ZIP dans destDir, en gérant le préfixe commun
func extractZip(src, destDir string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// Détecter le préfixe commun
	names := make([]string, 0, len(r.File))
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	prefix := lxdpkg.StripZipPrefix(names)

	for _, f := range r.File {
		name := f.Name
		if prefix != "" {
			name = strings.TrimPrefix(name, prefix)
		}
		if name == "" {
			continue
		}

		destPath := filepath.Join(destDir, name)
		// Protection path traversal
		if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(destDir)) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(destPath, 0755)
			continue
		}

		os.MkdirAll(filepath.Dir(destPath), 0755)
		out, err := os.Create(destPath)
		if err != nil {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			out.Close()
			continue
		}
		buf := make([]byte, 32*1024)
		for {
			n, readErr := rc.Read(buf)
			if n > 0 {
				out.Write(buf[:n])
			}
			if readErr != nil {
				break
			}
		}
		rc.Close()
		out.Close()
	}
	return nil
}
