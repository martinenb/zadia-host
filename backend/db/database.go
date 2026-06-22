package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"zadia-host/models"
)

var DB *sql.DB

func InitDB(connStr string) error {
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("ouverture DB: %w", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("ping DB: %w", err)
	}

	createVPSTable := `
	CREATE TABLE IF NOT EXISTS vps (
		id         SERIAL PRIMARY KEY,
		name       VARCHAR(255) NOT NULL,
		os         VARCHAR(100) NOT NULL,
		vcores     INT NOT NULL,
		ram_gb     INT NOT NULL,
		disk_gb    INT NOT NULL,
		status     VARCHAR(50) DEFAULT 'creating',
		ip         VARCHAR(45) DEFAULT '',
		host_port  INT DEFAULT 0,
		created_at TIMESTAMP DEFAULT NOW()
	);`

	if _, err = DB.Exec(createVPSTable); err != nil {
		return fmt.Errorf("création table vps: %w", err)
	}

	createEnvVarsTable := `
	CREATE TABLE IF NOT EXISTS env_vars (
		id     SERIAL PRIMARY KEY,
		vps_id INT REFERENCES vps(id) ON DELETE CASCADE,
		key    VARCHAR(255) NOT NULL,
		value  TEXT
	);`

	if _, err = DB.Exec(createEnvVarsTable); err != nil {
		return fmt.Errorf("création table env_vars: %w", err)
	}

	return nil
}

func GetAllVPS() ([]models.VPS, error) {
	rows, err := DB.Query(`SELECT id, name, os, vcores, ram_gb, disk_gb, status, ip, host_port, created_at FROM vps ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("requête GetAllVPS: %w", err)
	}
	defer rows.Close()

	var list []models.VPS
	for rows.Next() {
		var v models.VPS
		if err := rows.Scan(&v.ID, &v.Name, &v.OS, &v.VCores, &v.RAMGB, &v.DiskGB, &v.Status, &v.IP, &v.HostPort, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan VPS: %w", err)
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

func GetVPSByID(id int64) (*models.VPS, error) {
	var v models.VPS
	row := DB.QueryRow(`SELECT id, name, os, vcores, ram_gb, disk_gb, status, ip, host_port, created_at FROM vps WHERE id = $1`, id)
	if err := row.Scan(&v.ID, &v.Name, &v.OS, &v.VCores, &v.RAMGB, &v.DiskGB, &v.Status, &v.IP, &v.HostPort, &v.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("VPS %d introuvable", id)
		}
		return nil, fmt.Errorf("scan GetVPSByID: %w", err)
	}
	return &v, nil
}

func CreateVPS(vps *models.VPS) (int64, error) {
	var id int64
	err := DB.QueryRow(
		`INSERT INTO vps (name, os, vcores, ram_gb, disk_gb, status) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		vps.Name, vps.OS, vps.VCores, vps.RAMGB, vps.DiskGB, vps.Status,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert VPS: %w", err)
	}
	return id, nil
}

func UpdateVPSStatus(id int64, status, ip string) error {
	_, err := DB.Exec(`UPDATE vps SET status = $1, ip = $2 WHERE id = $3`, status, ip, id)
	if err != nil {
		return fmt.Errorf("update VPS status: %w", err)
	}
	return nil
}

func UpdateVPSHostPort(id int64, port int) error {
	_, err := DB.Exec(`UPDATE vps SET host_port = $1 WHERE id = $2`, port, id)
	if err != nil {
		return fmt.Errorf("update VPS host_port: %w", err)
	}
	return nil
}

func DeleteVPS(id int64) error {
	_, err := DB.Exec(`DELETE FROM vps WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete VPS: %w", err)
	}
	return nil
}

func GetEnvVars(vpsID int64) ([]models.EnvVar, error) {
	rows, err := DB.Query(`SELECT id, vps_id, key, value FROM env_vars WHERE vps_id = $1 ORDER BY id ASC`, vpsID)
	if err != nil {
		return nil, fmt.Errorf("requête GetEnvVars: %w", err)
	}
	defer rows.Close()

	var list []models.EnvVar
	for rows.Next() {
		var ev models.EnvVar
		if err := rows.Scan(&ev.ID, &ev.VPSID, &ev.Key, &ev.Value); err != nil {
			return nil, fmt.Errorf("scan EnvVar: %w", err)
		}
		list = append(list, ev)
	}
	return list, rows.Err()
}

func CreateEnvVar(vpsID int64, key, value string) (int64, error) {
	var id int64
	err := DB.QueryRow(
		`INSERT INTO env_vars (vps_id, key, value) VALUES ($1, $2, $3) RETURNING id`,
		vpsID, key, value,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert EnvVar: %w", err)
	}
	return id, nil
}

func DeleteEnvVar(id int64) error {
	_, err := DB.Exec(`DELETE FROM env_vars WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete EnvVar: %w", err)
	}
	return nil
}

func GetAllEnvVarsAsMap(vpsID int64) (map[string]string, error) {
	rows, err := DB.Query(`SELECT key, value FROM env_vars WHERE vps_id = $1`, vpsID)
	if err != nil {
		return nil, fmt.Errorf("requête GetAllEnvVarsAsMap: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("scan EnvVar map: %w", err)
		}
		result[k] = v
	}
	return result, rows.Err()
}
