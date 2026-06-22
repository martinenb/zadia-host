package lxd

import (
	"io"

	lxdclient "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
)

// ExecInteractive ouvre un shell interactif dans le conteneur et connecte
// stdin/stdout aux pipes fournis. Bloque jusqu'à la fin de la session.
// Mode recovery : /bin/bash --norc --noprofile (aucun processus auto-démarré).
func ExecInteractive(containerName string, stdin io.ReadCloser, stdout io.WriteCloser, width, height int) error {
	client, err := ConnectLXD()
	if err != nil {
		return err
	}

	if width <= 0 {
		width = 220
	}
	if height <= 0 {
		height = 50
	}

	req := api.InstanceExecPost{
		Command:     []string{"/bin/bash", "--norc", "--noprofile"},
		WaitForWS:   true,
		Interactive: true,
		Environment: map[string]string{
			"TERM":  "xterm-256color",
			"SHELL": "/bin/bash",
		},
		Width:  width,
		Height: height,
	}

	dataDone := make(chan bool)
	args := &lxdclient.InstanceExecArgs{
		Stdin:    stdin,
		Stdout:   stdout,
		Stderr:   stdout,
		DataDone: dataDone,
	}

	op, err := client.ExecInstance(containerName, req, args)
	if err != nil {
		return err
	}

	op.Wait() //nolint:errcheck
	<-dataDone
	return nil
}
