# Zadia Host

Panel d'hébergement hybride IaaS/PaaS basé sur LXD.

## Prérequis

- Go 1.22+
- Node.js 18+
- LXD installé et initialisé
- PostgreSQL 14+

## Configuration PostgreSQL

```sql
CREATE USER zadia WITH PASSWORD 'zadia';
CREATE DATABASE zadiahost OWNER zadia;
```

## Configuration LXD

```bash
sudo lxd init --auto
sudo lxc storage create default dir
```

## Backend (Go)

```bash
cd backend
go mod tidy
go run .
```

Le serveur démarre sur le port 8080.

Variable d'environnement optionnelle:
- `DATABASE_URL` : `postgres://zadia:zadia@localhost:5432/zadiahost?sslmode=disable`
- `PORT` : port d'écoute (défaut: 8080)

## Frontend (Next.js)

```bash
cd frontend
npm install
npm run dev
```

Le frontend démarre sur le port 3000.

## Accès

- Dashboard: http://localhost:3000
- API: http://localhost:8080/api/vps
- Apps hébergées: http://host.mcmr.eu:[port]

## Routes API

| Méthode | Route | Description |
|---------|-------|-------------|
| GET | /api/vps | Liste des VPS |
| POST | /api/vps | Créer un VPS |
| GET | /api/vps/:id | Détails d'un VPS |
| DELETE | /api/vps/:id | Supprimer un VPS |
| POST | /api/vps/:id/start | Démarrer un VPS |
| POST | /api/vps/:id/stop | Arrêter un VPS |
| POST | /api/vps/:id/deploy | Déployer du code |
| GET | /api/vps/:id/env | Variables d'env |
| POST | /api/vps/:id/env | Créer une variable |
| DELETE | /api/vps/:id/env/:envId | Supprimer une variable |
