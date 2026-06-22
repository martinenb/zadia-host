package lxd

import (
	"fmt"
	"io"
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

	req := api.ContainersPost{
		Name: name,
		Source: api.ContainerSource{
			Type:  "image",
			Alias: alias,
		},
		ContainerPut: api.ContainerPut{
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

	op, err := client.CreateContainer(req)
	if err != nil {
		return fmt.Errorf("création conteneur: %w", err)
	}
	return op.Wait()
}

func StartContainer(name string) error {
	client, err := ConnectLXD()
	if err != nil {
		return fmt.Errorf("connexion LXD: %w", err)
	}

	req := api.ContainerStatePut{
		Action:  "start",
		Timeout: 60,
	}
	op, err := client.UpdateContainerState(name, req, "")
	if err != nil {
		return fmt.Errorf("démarrage conteneur: %w", err)
	}
	return op.Wait()
}

func StopContainer(name string) error {
	client, err := ConnectLXD()
	if err != nil {
		return fmt.Errorf("connexion LXD: %w", err)
	}

	req := api.ContainerStatePut{
		Action:  "stop",
		Timeout: 60,
		Force:   true,
	}
	op, err := client.UpdateContainerState(name, req, "")
	if err != nil {
		return fmt.Errorf("arrêt conteneur: %w", err)
	}
	return op.Wait()
}

func DeleteContainer(name string) error {
	client, err := ConnectLXD()
	if err != nil {
		return fmt.Errorf("connexion LXD: %w", err)
	}

	op, err := client.DeleteContainer(name)
	if err != nil {
		return fmt.Errorf("suppression conteneur: %w", err)
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
		state, _, err := client.GetContainerState(name)
		if err != nil {
			return "", fmt.Errorf("état conteneur: %w", err)
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

	container, etag, err := client.GetContainer(name)
	if err != nil {
		return fmt.Errorf("récupération conteneur: %w", err)
	}

	if container.Devices == nil {
		container.Devices = make(map[string]map[string]string)
	}

	container.Devices["proxy-web"] = map[string]string{
		"type":    "proxy",
		"listen":  fmt.Sprintf("tcp:0.0.0.0:%d", hostPort),
		"connect": "tcp:127.0.0.1:80",
		"bind":    "host",
	}

	op, err := client.UpdateContainer(name, container.Writable(), etag)
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

	args := lxdclient.ContainerFileArgs{
		Content:   io.NopCloser(strings.NewReader(content)),
		UID:       0,
		GID:       0,
		Mode:      0644,
		Type:      "file",
		WriteMode: "overwrite",
	}

	return client.CreateContainerFile(containerName, destPath, args)
}

func ExecCommand(containerName string, command []string, env map[string]string) error {
	client, err := ConnectLXD()
	if err != nil {
		return fmt.Errorf("connexion LXD: %w", err)
	}

	req := api.ContainerExecPost{
		Command:     command,
		WaitForWS:   false,
		Interactive: false,
		Environment: env,
	}

	op, err := client.ExecContainer(containerName, req, nil)
	if err != nil {
		return fmt.Errorf("exécution commande: %w", err)
	}
	return op.Wait()
}

func EnsureDirectory(containerName, path string) error {
	return ExecCommand(containerName, []string{"mkdir", "-p", path}, nil)
}
