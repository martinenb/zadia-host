package models

type VPS struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	OS        string `json:"os"`
	VCores    int    `json:"vcores"`
	RAMGB     int    `json:"ram_gb"`
	DiskGB    int    `json:"disk_gb"`
	Status    string `json:"status"`
	IP        string `json:"ip"`
	HostPort  int    `json:"host_port"`
	CreatedAt string `json:"created_at"`
}

type CreateVPSRequest struct {
	Name   string `json:"name"`
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
