package runner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

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
}

// NewSSHCommand creates a new SSHCommand instance
func NewSSHCommand(command string, environment map[string]string, payload []FileAsset) *SSHCommand {
	return &SSHCommand{
		command:     command,
		environment: environment,
		payload:     payload,
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
// Asset validation is done in FileAsset.Validate()
func (c *SSHCommand) AddToPayload(p *svmkitRunner.Payload) error {
	var errs []error
	for _, asset := range c.payload {
		var (
			content io.Reader
			err     error
		)
		if !IsEmptyStr(asset.LocalPath) {
			if content, err = os.Open(*asset.LocalPath); err != nil {
				errs = append(errs,
					fmt.Errorf("failed to read local file %s: %w",
						*asset.LocalPath,
						err))
				continue
			}
		}

		// Maybe we want empty file, so do not use IsEmptyStr()
		if asset.Contents != nil {
			content = strings.NewReader(*asset.Contents)
		}
		p.Add(svmkitRunner.PayloadFile{
			Path:   *asset.Filename,
			Reader: content,
			Mode:   os.FileMode(*asset.Mode),
		})

	}
	p.AddString("steps.sh", c.command)
	return errors.Join(errs...)
}

// Config returns the command configuration
func (c *SSHCommand) Config() *svmkitRunner.Config {
	return &svmkitRunner.Config{}
}
