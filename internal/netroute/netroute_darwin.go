//go:build darwin

package netroute

import (
	"context"
	"os/exec"
	"time"
)

func defaultGateway() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "route", "-n", "get", "default").Output()
	if err != nil {
		return ""
	}
	return parseRouteGet(string(out))
}
