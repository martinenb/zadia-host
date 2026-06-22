package lxd

import (
	"fmt"
	"strings"
	"time"

	lxdclient "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
)

const lxdSocket = "/var/snap/lxd/common/lxd/unix.socket"

func ConnectLXD() (lxdclient.InstanceServer, error) {
	return lxdclient.ConnectLXDUnix(lxdSocket, nil)
}

// OSToAlias mappe le nom OS vers l'alias image LXD
func OSToAlias(os string) string {
	switch os {
	case "ubuntu":
		return "ubuntu:22.04"
	case "debian":
		return "debian:12"
	case "alpine":
		return "alpine:3.19"
	default:
		return "ubuntu:22.04"
	}
}

func CreateContainer(name, os string, vcores, ramGB, diskGB int) error {
	client, err := ConnectLXD()
	if err != nil {
		return fmt.Errorf("connexion LXD: %w", err)
	}

	alias := OSToAlias(os)

	req := api.InstancesPost{
		Name: name,
		Type: api.InstanceTypeContainer,
		Source: api.InstanceSource{
			Type:  "image",
			Alias: alias,
		},
		InstancePut: api.InstancePut{
			Config: map[string]string{
				"limits.cpu":    fmt.Sprintf("%d", vcores),
				"limits.memory": fmt.Sprintf("%dGB", ramGB),
			},
			Devices: map[string]map[string]string{
				"root": {
					"type": "disk",
					"path": "/",
					"pool": "default",
					"size": fmt.Sprintf("%dGB", diskGB),
				},
			},
		},
	}

	op, err := client.CreateInstance(req)
	if err != nil {
		return fmt.Errorf("création instance: %w", err)
	}
	return op.Wait()
}

func StartContainer(name string) error {
	client, err := ConnectLXD()
	if err != nil {
		return fmt.Errorf("connexion LXD: %w", err)
	}

	req := api.InstanceStatePut{
		Action:  "start",
		Timeout: 60,
	}
	op, err := client.UpdateInstanceState(name, req, "")
	if err != nil {
		return fmt.Errorf("démarrage instance: %w", err)
	}
	return op.Wait()
}

func StopContainer(name string) error {
	client, err := ConnectLXD()
	if err != nil {
		return fmt.Errorf("connexion LXD: %w", err)
	}

	req := api.InstanceStatePut{
		Action:  "stop",
		Timeout: 60,
		Force:   true,
	}
	op, err := client.UpdateInstanceState(name, req, "")
	if err != nil {
		return fmt.Errorf("arrêt instance: %w", err)
	}
	return op.Wait()
}

func DeleteContainer(name string) error {
	client, err := ConnectLXD()
	if err != nil {
		return fmt.Errorf("connexion LXD: %w", err)
	}

	op, err := client.DeleteInstance(name, false)
	if err != nil {
		return fmt.Errorf("suppression instance: %w", err)
	}
	return op.Wait()
}

func GetContainerIP(name string) (string, error) {
	client, err := ConnectLXD()
	if err != nil {
		return "", fmt.Errorf("connexion LXD: %w", err)
	}

	// Tentatives répétées pour attendre que l'IP soit assignée
	for i := 0; i < 10; i++ {
		state, _, err := client.GetInstanceState(name)
		if err != nil {
			return "", fmt.Errorf("état instance: %w", err)
		}

		for _, net := range state.Network {
			for _, addr := range net.Addresses {
				if addr.Family == "inet" && addr.Address != "127.0.0.1" {
					return addr.Address, nil
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
	return "", fmt.Errorf("aucune IP trouvée pour %s", name)
}

func AddProxyDevice(name string, hostPort int) error {
	client, err := ConnectLXD()
	if err != nil {
		return fmt.Errorf("connexion LXD: %w", err)
	}

	instance, etag, err := client.GetInstance(name)
	if err != nil {
		return fmt.Errorf("récupération instance: %w", err)
	}

	if instance.Devices == nil {
		instance.Devices = make(map[string]map[string]string)
	}

	instance.Devices["proxy-web"] = map[string]string{
		"type":    "proxy",
		"listen":  fmt.Sprintf("tcp:0.0.0.0:%d", hostPort),
		"connect": "tcp:127.0.0.1:80",
		"bind":    "host",
	}

	op, err := client.UpdateInstance(name, instance.Writable(), etag)
	if err != nil {
		return fmt.Errorf("ajout proxy device: %w", err)
	}
	return op.Wait()
}

func PushFile(containerName, destPath, content string) error {
	client, err := ConnectLXD()
	if err != nil {
		return fmt.Errorf("connexion LXD: %w", err)
	}

	args := lxdclient.InstanceFileArgs{
		Content:   strings.NewReader(content),
		UID:       0,
		GID:       0,
		Mode:      0644,
		Type:      "file",
		WriteMode: "overwrite",
	}

	return client.CreateInstanceFile(containerName, destPath, args)
}

func ExecCommand(containerName string, command []string, env map[string]string) error {
	client, err := ConnectLXD()
	if err != nil {
		return fmt.Errorf("connexion LXD: %w", err)
	}

	req := api.InstanceExecPost{
		Command:     command,
		WaitForWS:   false,
		Interactive: false,
		Environment: env,
	}

	op, err := client.ExecInstance(containerName, req, nil)
	if err != nil {
		return fmt.Errorf("exécution commande: %w", err)
	}
	return op.Wait()
}

func EnsureDirectory(containerName, path string) error {
	return ExecCommand(containerName, []string{"mkdir", "-p", path}, nil)
}
