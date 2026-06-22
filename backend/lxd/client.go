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

type imageSource struct {
	server   string
	protocol string
	alias    string
}

// osToSource retourne le serveur, protocole et alias image LXD pour un OS donné.
// Ubuntu utilise son propre serveur simplestreams (alias court "22.04").
// Les autres OS passent par images.lxd.canonical.com.
func osToSource(os string) imageSource {
	switch os {
	case "ubuntu":
		return imageSource{
			server:   "https://cloud-images.ubuntu.com/releases",
			protocol: "simplestreams",
			alias:    "22.04",
		}
	case "debian":
		return imageSource{
			server:   "https://images.lxd.canonical.com",
			protocol: "simplestreams",
			alias:    "debian/bookworm/amd64",
		}
	case "alpine":
		return imageSource{
			server:   "https://images.lxd.canonical.com",
			protocol: "simplestreams",
			alias:    "alpine/3.19/amd64",
		}
	default:
		return imageSource{
			server:   "https://cloud-images.ubuntu.com/releases",
			protocol: "simplestreams",
			alias:    "22.04",
		}
	}
}

func CreateContainer(name, os string, vcores, ramGB, diskGB int) error {
	client, err := ConnectLXD()
	if err != nil {
		return fmt.Errorf("connexion LXD: %w", err)
	}

	src := osToSource(os)

	req := api.InstancesPost{
		Name: name,
		Type: api.InstanceTypeContainer,
		Source: api.InstanceSource{
			Type:     "image",
			Alias:    src.alias,
			Server:   src.server,
			Protocol: src.protocol,
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

func UpdateProxyDevice(name string, hostPort, appPort int) error {
	client, err := ConnectLXD()
	if err != nil {
		return err
	}
	instance, etag, err := client.GetInstance(name)
	if err != nil {
		return err
	}
	if instance.Devices == nil {
		instance.Devices = make(map[string]map[string]string)
	}
	instance.Devices["proxy-web"] = map[string]string{
		"type":    "proxy",
		"listen":  fmt.Sprintf("tcp:0.0.0.0:%d", hostPort),
		"connect": fmt.Sprintf("tcp:127.0.0.1:%d", appPort),
		"bind":    "host",
	}
	op, err := client.UpdateInstance(name, instance.Writable(), etag)
	if err != nil {
		return err
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

// PushBinaryFile pousse un fichier binaire (ex: ZIP) dans le conteneur
func PushBinaryFile(containerName, destPath string, r io.ReadSeeker) error {
	client, err := ConnectLXD()
	if err != nil {
		return fmt.Errorf("connexion LXD: %w", err)
	}
	args := lxdclient.InstanceFileArgs{
		Content:   r,
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

// SetupSSH installe et configure openssh dans le conteneur, démarre sshd
func SetupSSH(containerName, password, os string) error {
	// 1. Installer openssh selon l'OS
	switch os {
	case "alpine":
		if err := ExecCommand(containerName, []string{"apk", "add", "--no-cache", "openssh"}, nil); err != nil {
			return fmt.Errorf("installation openssh: %w", err)
		}
	default: // ubuntu, debian
		ExecCommand(containerName, []string{"apt-get", "update", "-qq"}, nil)
		if err := ExecCommand(containerName, []string{"apt-get", "install", "-y", "-qq", "openssh-server"}, nil); err != nil {
			return fmt.Errorf("installation openssh-server: %w", err)
		}
	}
	// 2. Définir le mot de passe root
	if err := ExecCommand(containerName, []string{"sh", "-c", "echo 'root:" + password + "' | chpasswd"}, nil); err != nil {
		return fmt.Errorf("mot de passe root: %w", err)
	}
	// 3. Configurer sshd (PermitRootLogin + PasswordAuthentication)
	sshdExtra := "PermitRootLogin yes\nPasswordAuthentication yes\n"
	EnsureDirectory(containerName, "/etc/ssh/sshd_config.d")
	PushFile(containerName, "/etc/ssh/sshd_config.d/zadia.conf", sshdExtra)
	// Fallback via sed si sshd_config.d n'est pas supporté
	ExecCommand(containerName, []string{"sh", "-c",
		`grep -q "sshd_config.d" /etc/ssh/sshd_config || ` +
			`{ sed -i 's/^#*PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config && ` +
			`sed -i 's/^#*PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config; }`}, nil)
	// 4. Générer les clés host si absentes
	ExecCommand(containerName, []string{"ssh-keygen", "-A"}, nil)
	// 5. Créer /run/sshd (requis sur certaines distros)
	ExecCommand(containerName, []string{"mkdir", "-p", "/run/sshd"}, nil)
	// 6. Démarrer sshd (il se daemonise tout seul)
	if err := ExecCommand(containerName, []string{"/usr/sbin/sshd"}, nil); err != nil {
		ExecCommand(containerName, []string{"sh", "-c", "which sshd && $(which sshd)"}, nil)
	}
	return nil
}

// AddSSHProxyDevice ajoute un proxy device LXD pour le SSH (host:sshPort → container:22)
func AddSSHProxyDevice(name string, sshPort int) error {
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
	instance.Devices["proxy-ssh"] = map[string]string{
		"type":    "proxy",
		"listen":  fmt.Sprintf("tcp:0.0.0.0:%d", sshPort),
		"connect": "tcp:127.0.0.1:22",
		"bind":    "host",
	}
	op, err := client.UpdateInstance(name, instance.Writable(), etag)
	if err != nil {
		return fmt.Errorf("ajout proxy SSH: %w", err)
	}
	return op.Wait()
}
