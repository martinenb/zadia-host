#!/bin/bash
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log()  { echo -e "${GREEN}[ZADIA]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC}  $1"; }
fail() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

log "=== Zadia Host — Setup initial ==="

# ── 1. LXD installé ?
if ! command -v lxc &>/dev/null; then
  log "Installation de LXD via snap..."
  snap install lxd
fi
log "LXD version : $(lxc --version)"

# ── 2. LXD initialisé ?
if ! lxc storage list 2>/dev/null | grep -q "default"; then
  log "Initialisation de LXD..."
  lxd init --auto
  # Créer le storage pool 'default' si toujours absent
  lxc storage list 2>/dev/null | grep -q "default" || lxc storage create default dir
  log "Storage pool 'default' créé."
else
  log "LXD déjà initialisé (storage pool 'default' trouvé)."
fi

# ── 3. Permissions du socket LXD
SOCKET=/var/snap/lxd/common/lxd/unix.socket
if [ ! -S "$SOCKET" ]; then
  fail "Socket LXD introuvable : $SOCKET — relance 'snap start lxd'"
fi
log "Socket LXD trouvé."
chmod 660 "$SOCKET"
chown root:lxd "$SOCKET" 2>/dev/null || true

# ── 4. Ajouter root au groupe lxd
if ! groups root | grep -q lxd; then
  usermod -aG lxd root
  log "root ajouté au groupe lxd."
fi

# ── 5. Vérifier que Docker est installé
if ! command -v docker &>/dev/null; then
  fail "Docker n'est pas installé."
fi
log "Docker version : $(docker --version)"

# ── 6. Lancer le stack
log "Démarrage du stack Zadia Host..."
docker compose down 2>/dev/null || true
docker compose up -d --build

log ""
log "✓ Zadia Host est prêt !"
log "  Panel    : http://host.mcmr.eu:8880"
log "  API      : http://localhost:8083/api/vps"
log "  VPS apps : http://vps-{id}.host.mcmr.eu:9090"
