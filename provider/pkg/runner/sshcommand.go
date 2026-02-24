package runner

import (
	"errors"
	"fmt"
	"os"

	svmkitRunner "github.com/abklabs/svmkit/pkg/runner"
)

// SSHCommand encapsulates a shell command and its execution context for remote
// deployment via SSH. It implements the svmkitRunner.Command interface and
// bundles together the command string, environment variables, and file assets
// needed for remote command execution.
type SSHCommand struct {
	command     string
	environment map[string]string
	payload     []FileAsset
	config      *svmkitRunner.Config
}

// NewSSHCommand creates a new SSHCommand instance
func NewSSHCommand(command string, environment map[string]string, payload []FileAsset, config *svmkitRunner.Config) *SSHCommand {
	return &SSHCommand{
		command:     command,
		environment: environment,
		payload:     payload,
		config:      config,
	}
}

// Check validates the command and payload
func (c *SSHCommand) Check() error {
	// Validate the command
	if c.command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	var errs []error

	for _, asset := range c.payload {
		if err := asset.Validate(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Env returns the environment builder
func (c *SSHCommand) Env() *svmkitRunner.EnvBuilder {
	env := svmkitRunner.NewEnvBuilder()
	env.SetMap(c.environment)
	return env
}

// AddToPayload adds file assets to the payload
func (c *SSHCommand) AddToPayload(p *svmkitRunner.Payload) error {
	var errs []error
	for _, asset := range c.payload {
		rc, err := asset.openContent()
		if err != nil {
			errs = append(errs, err)
			continue
		}

		p.Add(svmkitRunner.PayloadFile{
			Path:   *asset.Filename,
			Reader: rc,
			Mode:   os.FileMode(*asset.Mode),
		})
	}
	p.AddString("steps.sh", c.command)
	return errors.Join(errs...)
}

func (c *SSHCommand) Config() *svmkitRunner.Config {
	if c.config == nil {
		return nil
	}
	return c.config
}
