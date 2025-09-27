package runner

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"strconv"

	"github.com/abklabs/pulumi-runner/pkg/ssh"
	"github.com/abklabs/pulumi-runner/pkg/utils"
	svmkitRunner "github.com/abklabs/svmkit/pkg/runner"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/plugin"
)

type SSHDeployer struct{}

type CommandDefinition struct {
	Command     string            `pulumi:"command"`
	Environment map[string]string `pulumi:"environment,optional"`
	Payload     []FileAsset       `pulumi:"payload,optional"`
}

type SSHDeployerArgs struct {
	Connection  ssh.Connection       `pulumi:"connection"`
	Environment map[string]string    `pulumi:"environment,optional"`
	Payload     []FileAsset          `pulumi:"payload,optional"`
	Create      *CommandDefinition   `pulumi:"create,optional"`
	Update      *CommandDefinition   `pulumi:"update,optional"`
	Delete      *CommandDefinition   `pulumi:"delete,optional"`
	Config      *svmkitRunner.Config `pulumi:"config,optional"`
}

// SSHDeployerState represents the state of an SSHDeployer resource
type SSHDeployerState struct {
	SSHDeployerArgs
	// The pulumi tag is required for Pulumi to serialize/deserialize
	// this field in state
	PayloadHashes map[string]string `pulumi:"payloadHashes,optional"`
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

	cmd := NewSSHCommand(def.Command, environment, payload, state.Config)

	if preview {
		return
	}

	err = utils.RunnerHelper(ctx, utils.RunnerArgs{Connection: state.Connection}, cmd)

	return
}

func (SSHDeployer) Create(ctx context.Context, name string, input SSHDeployerArgs, preview bool) (string, SSHDeployerState, error) {
	var err error
	var def *CommandDefinition

	state := SSHDeployerState{SSHDeployerArgs: input}

	def = input.Create
	if def == nil {
		def = input.Update
	}
	if def == nil {
		return name, state, nil
	}

	// Calculate Hashes for all payload files
	if state.PayloadHashes, err = getPayloadHashes(input.Payload, def.Payload); err != nil {
		return "", SSHDeployerState{}, fmt.Errorf("failed to calculate payload hashes: %w", err)
	}

	if err = runDeployerCommand(ctx, def, &state, preview); err != nil {
		return "", SSHDeployerState{}, err
	}

	return name, state, nil
}

// defaultDiff - emulate the default pulumi Diff() behavior
func defaultDiff(_ context.Context, state SSHDeployerState, newInput SSHDeployerArgs) (p.DiffResponse, error) {
	// Convert structs to PropertyMap for default diff
	oldProps := resource.NewPropertyMap(state.SSHDeployerArgs)
	newProps := resource.NewPropertyMap(newInput)

	delete(oldProps, "command")

	objDiff := oldProps.Diff(newProps)
	pluginDiff := plugin.NewDetailedDiffFromObjectDiff(objDiff, false)

	diff := map[string]p.PropertyDiff{}
	for k, v := range pluginDiff {
		diff[k] = p.PropertyDiff{
			Kind:      p.Update,
			InputDiff: v.InputDiff,
		}
	}
	return p.DiffResponse{
		HasChanges:   objDiff.AnyChanges(),
		DetailedDiff: diff,
	}, nil
}

func (SSHDeployer) Diff(ctx context.Context, name string, state SSHDeployerState, newInput SSHDeployerArgs) (p.DiffResponse, error) {
	// Start with the default response
	response, err := defaultDiff(ctx, state, newInput)
	if err != nil {
		return p.DiffResponse{}, err
	}

	// Determine which command definition to compare (update vs create)
	newRoot := "update"
	newCmdDef := newInput.Update

	if newCmdDef == nil && newInput.Create != nil {
		newRoot = "create"
		newCmdDef = newInput.Create
	}

	// If there is no command, return early
	if newCmdDef == nil {
		return response, nil
	}

	inputDiff := p.PropertyDiff{
		Kind:      p.Update,
		InputDiff: true,
	}

	// Check top-level payload for hash changes
	if hashChanges, err := payloadHashChanges(newInput.Payload, state.PayloadHashes); err != nil {
		// If we can't compare hashes, assume payload changed
		response.HasChanges = true
		response.DetailedDiff["payload"] = inputDiff
	} else if len(hashChanges) > 0 {
		response.HasChanges = true
		for _, change := range hashChanges {
			response.DetailedDiff["payload["+strconv.Itoa(change.Index)+"]"] = inputDiff
		}
	}

	// Check command-specific payload for hash changes
	if hashChanges, err := payloadHashChanges(newCmdDef.Payload, state.PayloadHashes); err != nil {
		response.HasChanges = true
		response.DetailedDiff[newRoot+".payload"] = inputDiff
	} else if len(hashChanges) > 0 {
		response.HasChanges = true
		for _, change := range hashChanges {
			response.DetailedDiff[newRoot+".payload["+strconv.Itoa(change.Index)+"]"] = inputDiff
		}
	}

	return response, nil
}

func (SSHDeployer) Update(ctx context.Context, name string, state SSHDeployerState, newInput SSHDeployerArgs, preview bool) (SSHDeployerState, error) {
	var err error

	newState := SSHDeployerState{SSHDeployerArgs: newInput}

	def := newInput.Update
	if def == nil {
		def = newInput.Create
	}
	if def == nil {
		return newState, nil
	}

	if newState.PayloadHashes, err = getPayloadHashes(newInput.Payload, def.Payload); err != nil {
		return SSHDeployerState{}, fmt.Errorf("failed to calculate payload hashes: %w", err)
	}

	if err = runDeployerCommand(ctx, def, &newState, preview); err != nil {
		return SSHDeployerState{}, err
	}

	return newState, nil
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

// Find Payload differences where the the has of the contents has
// changed
type FileChange struct {
	Index    int
	Filename string
	Hash     string
}

// payLoadHashChanges compares FileAssets to a previous map<filename,hash> and returns a
// map<filename,FileChange> for changed or missing entries
func payloadHashChanges(payloads []FileAsset, stateHashes map[string]string) (map[string]FileChange, error) {
	changedFiles := make(map[string]FileChange)
	var errRet error

	for index, file := range payloads {
		if file.Filename == nil {
			errRet = errors.Join(errRet, fmt.Errorf("file at index %d has nil filename", index))
			continue
		}

		filename := *file.Filename

		currentHash, err := file.GetHash()
		if err != nil {
			errRet = errors.Join(errRet, fmt.Errorf("failed to calculate hash for file %s: %w", filename, err))
		}

		if oldHash, exists := stateHashes[filename]; !exists || oldHash != currentHash {
			changedFiles[filename] = FileChange{Index: index, Filename: filename, Hash: currentHash}
		}
	}
	return changedFiles, errRet
}

func getPayloadHashes(payloads ...[]FileAsset) (map[string]string, error) {
	hashMap := make(map[string]string)

	for _, payload := range payloads {
		for _, file := range payload {
			hashChecksum, err := file.GetHash()
			if err != nil {
				return nil, fmt.Errorf("failed to calculate hash for file %v: %w", file.Filename, err)
			}
			// Use filename as key (assuming filenames are unique)
			if file.Filename != nil {
				hashMap[*file.Filename] = hashChecksum
			}
		}
	}

	return hashMap, nil
}
