package runner

import (
	"context"
	"fmt"
	"maps"
	"os"

	"github.com/abklabs/pulumi-runner/pkg/ssh"
	"github.com/abklabs/pulumi-runner/pkg/utils"
)

type SSHDeployer struct{}

type CommandDefinition struct {
	Command     string            `pulumi:"command"`
	Environment map[string]string `pulumi:"environment,optional"`
	Payload     []FileAsset       `pulumi:"payload,optional"`
}

type SSHDeployerArgs struct {
	Connection  ssh.Connection     `pulumi:"connection"`
	Environment map[string]string  `pulumi:"environment,optional"`
	Payload     []FileAsset        `pulumi:"payload,optional"`
	Create      *CommandDefinition `pulumi:"create,optional"`
	Update      *CommandDefinition `pulumi:"update,optional"`
	Delete      *CommandDefinition `pulumi:"delete,optional"`
}

// SSHDeployerState represents the state of an SSHDeployer resource
type SSHDeployerState struct {
	SSHDeployerArgs
}

// runDeployerCommand executes a deployment command
func runDeployerCommand(ctx context.Context, def *CommandDefinition, state *SSHDeployerState, preview bool) (err error) {

	// Command not defined so this is just null op.
	if def == nil {
		return
	}

	if def.Command == "" {
		return fmt.Errorf("command is empty")
	}

	payload := append([]FileAsset{}, state.Payload...)
	payload = append(payload, def.Payload...)

	environment := make(map[string]string)
	maps.Copy(environment, state.Environment)
	maps.Copy(environment, def.Environment)

	cmd := NewSSHCommand(def.Command, environment, payload)

	if preview {
		return
	}

	err = utils.RunnerHelper(ctx, utils.RunnerArgs{Connection: state.Connection}, cmd)

	return
}

func (SSHDeployer) Create(ctx context.Context, name string, input SSHDeployerArgs, preview bool) (string, SSHDeployerState, error) {
	var def *CommandDefinition
	if input.Create != nil {
		def = input.Create
	} else if input.Update != nil {
		def = input.Update
	}

	state := SSHDeployerState{
		SSHDeployerArgs: input,
	}

	err := runDeployerCommand(ctx, def, &state, preview)
	if err != nil {
		return "", SSHDeployerState{}, err
	}

	return name, state, nil
}

func (SSHDeployer) Update(ctx context.Context, name string, state SSHDeployerState, newInput SSHDeployerArgs, preview bool) (SSHDeployerState, error) {
	var def *CommandDefinition
	if newInput.Update != nil {
		def = newInput.Update
	} else if newInput.Create != nil {
		def = newInput.Create
	}

	state = SSHDeployerState{
		SSHDeployerArgs: newInput,
	}

	err := runDeployerCommand(ctx, def, &state, preview)
	if err != nil {
		return SSHDeployerState{}, err
	}

	return state, nil
}

func (SSHDeployer) Delete(ctx context.Context, name string, state SSHDeployerState) error {
	err := runDeployerCommand(ctx, state.Delete, &state, false)
	return err
}

type LocalFile struct{}

func (LocalFile) Call(ctx context.Context, input FileAsset) (asset FileAsset, err error) {
	if IsEmptyStr(input.LocalPath) {
		err = fmt.Errorf("'LocalPath' must be set for LocalFile")
		return
	}

	fname := input.Filename
	if IsEmptyStr(fname) {
		fname = input.LocalPath
	}

	asset = FileAsset{
		Filename:  fname,
		LocalPath: input.LocalPath,
		Mode:      input.Mode,
	}

	if asset.Mode == nil {
		var f os.FileInfo
		f, err = os.Stat(*asset.LocalPath)
		if err != nil {
			return
		}
		m := int(f.Mode())
		asset.Mode = &m
	}
	return
}

type StringFile struct{}

func (StringFile) Call(ctx context.Context, input FileAsset) (asset FileAsset, err error) {
	asset = FileAsset{
		Filename: input.Filename,
		Contents: input.Contents,
		Mode:     input.Mode,
	}
	return
}
