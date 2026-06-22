package models

type VPS struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Subdomain    string `json:"subdomain"`
	Type         string `json:"type"` // "vps" ou "web"
	OS           string `json:"os"`
	VCores       int    `json:"vcores"`
	RAMGB        int    `json:"ram_gb"`
	DiskGB       int    `json:"disk_gb"`
	Status       string `json:"status"`
	IP           string `json:"ip"`
	HostPort     int    `json:"host_port"`
	SSHPort      int    `json:"ssh_port"`
	SSHPassword  string `json:"ssh_password"`
	DeployStatus string `json:"deploy_status"`
	AppPort      int    `json:"app_port"`
	CreatedAt    string `json:"created_at"`
}

type CreateVPSRequest struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	OS     string `json:"os"`
	VCores int    `json:"vcores"`
	RAMGB  int    `json:"ram_gb"`
	DiskGB int    `json:"disk_gb"`
}

type DeployRequest struct {
	Code     string            `json:"code"`
	Filename string            `json:"filename"`
	Command  string            `json:"command"`
	EnvVars  map[string]string `json:"env_vars"`
}

type EnvVar struct {
	ID    int64  `json:"id"`
	VPSID int64  `json:"vps_id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type CreateEnvVarRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
