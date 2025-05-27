package genschemas

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/uclog"
)

// StartContainer starts a test database container with the given image
func StartContainer(ctx context.Context,
	namePrefix string,
	portPrefix int,
	portToMap int,
	envVarPassword string,
	image string) (*exec.Cmd, string, int) {

	name := fmt.Sprintf("%s-%s", namePrefix, crypto.MustRandomHex(4))
	port := portPrefix*100 + rand.Intn(100)

	tmpDir, err := os.MkdirTemp(os.TempDir(), "ucgs")
	if err != nil {
		uclog.Fatalf(ctx, "failed to mkdirtmp: %v", err)
	}

	pg := exec.Command("docker",
		"run",
		"--rm",
		"-d",
		"-e",
		fmt.Sprintf("%s=mysecretpassword", envVarPassword),
		"--name",
		name,
		"--tmpfs",
		fmt.Sprintf("%s:size=1G", tmpDir),
		"-p",
		fmt.Sprintf("%d:%d", port, portToMap),
		image,
	)

	output, err := pg.Output()
	if err != nil {
		insp, err := exec.Command("docker", "inspect", string(output)).Output()
		if err != nil {
			uclog.Errorf(ctx, "Error inspecting %s DB container: %v", image, err)
		}
		logs, err := exec.Command("docker", "logs", string(output)).Output()
		if err != nil {
			uclog.Errorf(ctx, "Error getting logs for %s DB container: %v", image, err)
		}
		uclog.Fatalf(ctx, "Error starting %s DB container: %v (%v) (inspect: %v) (logs: %v)", image, err, string(output), string(insp), string(logs))
	}
	uclog.Infof(ctx, "%s container started: %s", image, string(output))

	return pg, name, port
}
