package lxd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type ProjectInfo struct {
	Framework string     // "nextjs", "node", "python-flask", "python", "php", "static", "unknown"
	Label     string     // affiché dans l'UI
	SetupCmds [][]string // installation du runtime (node, python...)
	BuildCmds [][]string // build (npm run build, etc.)
	StartCmd  string     // commande de démarrage (avec PORT=80)
	AppPort   int        // port sur lequel l'app écoute dans le conteneur
}

type pkgJSON struct {
	Scripts map[string]string `json:"scripts"`
	Deps    map[string]string `json:"dependencies"`
	DevDeps map[string]string `json:"devDependencies"`
}

func hasDep(pkg pkgJSON, name string) bool {
	_, a := pkg.Deps[name]
	_, b := pkg.DevDeps[name]
	return a || b
}

func nodeSetupCmds(os string) [][]string {
	switch os {
	case "alpine":
		return [][]string{{"apk", "add", "--no-cache", "nodejs", "npm"}}
	default:
		return [][]string{
			{"apt-get", "install", "-y", "-qq", "curl"},
			{"bash", "-c", "curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && apt-get install -y -qq nodejs"},
		}
	}
}

func pythonSetupCmds(os string) [][]string {
	switch os {
	case "alpine":
		return [][]string{{"apk", "add", "--no-cache", "python3", "py3-pip"}}
	default:
		return [][]string{{"apt-get", "install", "-y", "-qq", "python3", "python3-pip"}}
	}
}

func phpSetupCmds(os string) [][]string {
	switch os {
	case "alpine":
		return [][]string{{"apk", "add", "--no-cache", "php", "php-fpm", "nginx"}}
	default:
		return [][]string{{"apt-get", "install", "-y", "-qq", "php", "php-cli", "nginx"}}
	}
}

// DetectProject analyse un répertoire et retourne le type de projet détecté
func DetectProject(dir, vpsOS string) ProjectInfo {
	// Node.js → package.json
	if data, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
		var pkg pkgJSON
		json.Unmarshal(data, &pkg)

		setup := nodeSetupCmds(vpsOS)
		installCmd := [][]string{{"npm", "install", "--legacy-peer-deps"}}

		if hasDep(pkg, "next") {
			// Next.js : build puis start
			hasBuild := pkg.Scripts["build"] != ""
			buildCmds := [][]string{}
			if hasBuild {
				buildCmds = [][]string{{"npm", "run", "build"}}
			}
			return ProjectInfo{
				Framework: "nextjs",
				Label:     "Next.js",
				SetupCmds: append(setup, installCmd...),
				BuildCmds: buildCmds,
				StartCmd:  "PORT=80 npm run start",
				AppPort:   80,
			}
		}
		if hasDep(pkg, "vite") {
			return ProjectInfo{
				Framework: "vite",
				Label:     "Vite",
				SetupCmds: append(setup, installCmd...),
				BuildCmds: [][]string{{"npm", "run", "build"}},
				StartCmd:  "npx serve dist -p 80",
				AppPort:   80,
			}
		}
		if hasDep(pkg, "react-scripts") {
			return ProjectInfo{
				Framework: "cra",
				Label:     "Create React App",
				SetupCmds: append(setup, installCmd...),
				BuildCmds: [][]string{{"npm", "run", "build"}},
				StartCmd:  "npx serve -s build -l 80",
				AppPort:   80,
			}
		}
		// Node générique
		startScript := pkg.Scripts["start"]
		if startScript == "" {
			startScript = "node index.js"
		}
		return ProjectInfo{
			Framework: "node",
			Label:     "Node.js",
			SetupCmds: append(setup, installCmd...),
			StartCmd:  "PORT=80 " + startScript,
			AppPort:   80,
		}
	}

	// Python → requirements.txt ou pyproject.toml
	if _, err := os.Stat(filepath.Join(dir, "requirements.txt")); err == nil {
		setup := pythonSetupCmds(vpsOS)
		installCmd := [][]string{{"pip3", "install", "-r", "requirements.txt"}}

		// Détecter le framework Python
		reqs, _ := os.ReadFile(filepath.Join(dir, "requirements.txt"))
		reqStr := strings.ToLower(string(reqs))

		if strings.Contains(reqStr, "fastapi") || strings.Contains(reqStr, "uvicorn") {
			entry := findPythonEntry(dir, "main", "app")
			return ProjectInfo{
				Framework: "fastapi",
				Label:     "FastAPI",
				SetupCmds: append(setup, installCmd...),
				StartCmd:  "uvicorn " + entry + ":app --host 0.0.0.0 --port 80",
				AppPort:   80,
			}
		}
		if strings.Contains(reqStr, "flask") {
			entry := findPythonEntry(dir, "app", "main", "wsgi")
			return ProjectInfo{
				Framework: "flask",
				Label:     "Flask",
				SetupCmds: append(setup, installCmd...),
				StartCmd:  "FLASK_APP=" + entry + ".py flask run --host=0.0.0.0 --port=80",
				AppPort:   80,
			}
		}
		if strings.Contains(reqStr, "django") {
			return ProjectInfo{
				Framework: "django",
				Label:     "Django",
				SetupCmds: append(setup, installCmd...),
				StartCmd:  "python3 manage.py runserver 0.0.0.0:80",
				AppPort:   80,
			}
		}
		// Python générique
		entry := findPythonEntry(dir, "main", "app", "server", "index")
		return ProjectInfo{
			Framework: "python",
			Label:     "Python",
			SetupCmds: append(setup, installCmd...),
			StartCmd:  "python3 " + entry + ".py",
			AppPort:   80,
		}
	}

	// PHP → index.php ou composer.json
	if _, err := os.Stat(filepath.Join(dir, "index.php")); err == nil {
		setup := phpSetupCmds(vpsOS)
		return ProjectInfo{
			Framework: "php",
			Label:     "PHP",
			SetupCmds: setup,
			StartCmd:  "php -S 0.0.0.0:80",
			AppPort:   80,
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "composer.json")); err == nil {
		setup := phpSetupCmds(vpsOS)
		return ProjectInfo{
			Framework: "php",
			Label:     "PHP (Composer)",
			SetupCmds: setup,
			StartCmd:  "php -S 0.0.0.0:80 -t public",
			AppPort:   80,
		}
	}

	// HTML statique → index.html
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err == nil {
		setup := [][]string{}
		switch vpsOS {
		case "alpine":
			setup = [][]string{{"apk", "add", "--no-cache", "python3"}}
		default:
			setup = [][]string{{"apt-get", "install", "-y", "-qq", "python3"}}
		}
		return ProjectInfo{
			Framework: "static",
			Label:     "HTML statique",
			SetupCmds: setup,
			StartCmd:  "python3 -m http.server 80",
			AppPort:   80,
		}
	}

	return ProjectInfo{
		Framework: "unknown",
		Label:     "Projet non reconnu",
		AppPort:   80,
	}
}

// findPythonEntry cherche le fichier d'entrée Python (parmi les candidats)
func findPythonEntry(dir string, candidates ...string) string {
	for _, name := range candidates {
		if _, err := os.Stat(filepath.Join(dir, name+".py")); err == nil {
			return name
		}
	}
	return candidates[0]
}

// StripZipPrefix détecte si toutes les entrées d'un ZIP ont un préfixe commun
// (ex: si l'utilisateur a zippé un dossier au lieu des fichiers directement)
func StripZipPrefix(entries []string) string {
	if len(entries) == 0 {
		return ""
	}
	// Trouver le préfixe commun
	prefix := ""
	for _, e := range entries {
		parts := strings.SplitN(e, "/", 2)
		if len(parts) < 2 {
			return "" // fichier à la racine, pas de préfixe
		}
		if prefix == "" {
			prefix = parts[0] + "/"
		} else if !strings.HasPrefix(e, prefix) {
			return ""
		}
	}
	return prefix
}
