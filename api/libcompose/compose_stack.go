package libcompose

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/portainer/libcompose/config"
	"github.com/portainer/libcompose/docker"
	"github.com/portainer/libcompose/docker/client"
	"github.com/portainer/libcompose/docker/ctx"
	"github.com/portainer/libcompose/lookup"
	"github.com/portainer/libcompose/project"
	"github.com/portainer/libcompose/project/options"
	"github.com/portainer/portainer/api"
)

const (
	dockerClientVersion = "1.24"
)

// ComposeStackManager represents a service for managing compose stacks.
type ComposeStackManager struct {
	dataPath             string
	reverseTunnelService portainer.ReverseTunnelService
}

// NewComposeStackManager initializes a new ComposeStackManager service.
func NewComposeStackManager(dataPath string, reverseTunnelService portainer.ReverseTunnelService) *ComposeStackManager {
	return &ComposeStackManager{
		dataPath:             dataPath,
		reverseTunnelService: reverseTunnelService,
	}
}

func (manager *ComposeStackManager) createClient(endpoint *portainer.Endpoint) (client.Factory, error) {

	endpointURL := endpoint.URL
	if endpoint.Type == portainer.EdgeAgentEnvironment {
		tunnel := manager.reverseTunnelService.GetTunnelDetails(endpoint.ID)
		endpointURL = fmt.Sprintf("tcp://127.0.0.1:%d", tunnel.Port)
	}

	clientOpts := client.Options{
		Host:       endpointURL,
		APIVersion: dockerClientVersion,
	}

	if endpoint.TLSConfig.TLS {
		clientOpts.TLS = endpoint.TLSConfig.TLS
		clientOpts.TLSVerify = !endpoint.TLSConfig.TLSSkipVerify
		clientOpts.TLSCAFile = endpoint.TLSConfig.TLSCACertPath
		clientOpts.TLSCertFile = endpoint.TLSConfig.TLSCertPath
		clientOpts.TLSKeyFile = endpoint.TLSConfig.TLSKeyPath
	}

	return client.NewDefaultFactory(clientOpts)
}

// Up will deploy a compose stack (equivalent of docker-compose up)
func (manager *ComposeStackManager) Up(stack *portainer.Stack, endpoint *portainer.Endpoint) error {

	clientFactory, err := manager.createClient(endpoint)
	if err != nil {
		return err
	}

	env := make(map[string]string)
	for _, envvar := range stack.Env {
		env[envvar.Name] = envvar.Value
	}

	composeFilePath := path.Join(stack.ProjectPath, stack.EntryPoint)
	proj, err := docker.NewProject(&ctx.Context{
		ConfigDir: manager.dataPath,
		Context: project.Context{
			ComposeFiles: []string{composeFilePath},
			EnvironmentLookup: &lookup.ComposableEnvLookup{
				Lookups: []config.EnvironmentLookup{
					&lookup.EnvfileLookup{
						Path: filepath.Join(stack.ProjectPath, ".env"),
					},
					&lookup.MapLookup{
						Vars: env,
					},
				},
			},
			ProjectName: stack.Name,
		},
		ClientFactory: clientFactory,
	}, nil)
	if err != nil {
		return err
	}

	return proj.Up(context.Background(), options.Up{})
}

// Down will shutdown a compose stack (equivalent of docker-compose down)
func (manager *ComposeStackManager) Down(stack *portainer.Stack, endpoint *portainer.Endpoint) error {
	clientFactory, err := manager.createClient(endpoint)
	if err != nil {
		return err
	}

	composeFilePath := path.Join(stack.ProjectPath, stack.EntryPoint)
	proj, err := docker.NewProject(&ctx.Context{
		Context: project.Context{
			ComposeFiles: []string{composeFilePath},
			ProjectName:  stack.Name,
		},
		ClientFactory: clientFactory,
	}, nil)
	if err != nil {
		return err
	}

	return proj.Down(context.Background(), options.Down{RemoveVolume: false, RemoveOrphans: true})
}
