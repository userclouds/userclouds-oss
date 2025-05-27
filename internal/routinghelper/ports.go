package routinghelper

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"

	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/yamlconfig"
)

// ServiceMountEndpointOnlyConfig is a dummy config struct used to load the service config files
type ServiceMountEndpointOnlyConfig struct {
	MountPoint service.Endpoint `yaml:"svc_listener" json:"svc_listener"`
}

// Validate implements Validateable
func (o ServiceMountEndpointOnlyConfig) Validate() error {
	return nil
}

func getPortsForServicesMap(ctx context.Context, uv universe.Universe, services []serviceNamespace.Service) (map[serviceNamespace.Service]int, error) {
	ports := make(map[serviceNamespace.Service]int)
	baseDirs := yamlconfig.GetBaseDirs()
	if len(baseDirs) != 1 {
		return nil, ucerr.Errorf("expected exactly one base dir, got %d", len(baseDirs))
	}
	baseDir := baseDirs[0]
	for _, svc := range services {
		var cfg ServiceMountEndpointOnlyConfig
		path := filepath.Join(baseDir, string(svc), fmt.Sprintf("%v.yaml", uv))
		uclog.Debugf(ctx, "Loading %v config from: %v", svc, path)
		if err := yamlconfig.LoadAndDecodeFromPath(path, &cfg, true); err != nil {
			return nil, ucerr.Wrap(err)
		}
		port, err := strconv.Atoi(cfg.MountPoint.Port)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		ports[svc] = port
	}
	return ports, nil
}

func getPortsForServices(ctx context.Context, uv universe.Universe, services []serviceNamespace.Service) ([]int, error) {
	portMap, err := getPortsForServicesMap(ctx, uv, services)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	ports := make([]int, 0, len(portMap))
	for _, port := range portMap {
		ports = append(ports, port)
	}
	slices.Sort(ports)
	return ports, nil
}
