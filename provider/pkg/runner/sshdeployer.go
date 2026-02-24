package runner

import (
	"context"
	"fmt"
	"maps"
	"os"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/plugin"

	"github.com/abklabs/pulumi-runner/pkg/ssh"
	"github.com/abklabs/pulumi-runner/pkg/utils"
	svmkitRunner "github.com/abklabs/svmkit/pkg/runner"
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
	PayloadHashes map[string]string `pulumi:"payloadHashes,optional"`
}

func (a SSHDeployerArgs) allPayloads() [][]FileAsset {
	var payloads [][]FileAsset

	payloads = append(payloads, a.Payload)

	for _, def := range []*CommandDefinition{a.Create, a.Update, a.Delete} {
		if def != nil {
			payloads = append(payloads, def.Payload)
		}
	}

	return payloads
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

	if !preview {
		hashes, err := ComputePayloadHashes(input.allPayloads()...)
		if err != nil {
			p.GetLogger(ctx).Warning("failed to compute payload hashes: " + err.Error())
		} else {
			state.PayloadHashes = hashes
		}
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

	if !preview {
		hashes, err := ComputePayloadHashes(newInput.allPayloads()...)
		if err != nil {
			p.GetLogger(ctx).Warning("failed to compute payload hashes: " + err.Error())
		} else {
			state.PayloadHashes = hashes
		}
	}

	return state, nil
}

func (SSHDeployer) Delete(ctx context.Context, name string, state SSHDeployerState) error {
	err := runDeployerCommand(ctx, state.Delete, &state, false)
	return err
}

func convertDiffKind(k plugin.DiffKind) p.DiffKind {
	switch k {
	case plugin.DiffAdd:
		return p.Add
	case plugin.DiffAddReplace:
		return p.AddReplace
	case plugin.DiffDelete:
		return p.Delete
	case plugin.DiffDeleteReplace:
		return p.DeleteReplace
	case plugin.DiffUpdate:
		return p.Update
	case plugin.DiffUpdateReplace:
		return p.UpdateReplace
	default:
		return p.Update
	}
}

func (SSHDeployer) Diff(ctx context.Context, id string, olds SSHDeployerState, news SSHDeployerArgs) (p.DiffResponse, error) {
	resp := p.DiffResponse{
		DetailedDiff: make(map[string]p.PropertyDiff),
	}

	// Structural diff on input properties.
	oldMap := resource.NewPropertyMap(olds.SSHDeployerArgs)
	newMap := resource.NewPropertyMap(news)

	if objDiff := oldMap.Diff(newMap); objDiff != nil {
		for k, v := range plugin.NewDetailedDiffFromObjectDiff(objDiff, true) {
			resp.DetailedDiff[k] = p.PropertyDiff{
				Kind:      convertDiffKind(v.Kind),
				InputDiff: v.InputDiff,
			}
		}
	}

	// Hash-based diff on payload file contents.
	newHashes, err := ComputePayloadHashes(news.allPayloads()...)
	if err != nil {
		p.GetLogger(ctx).Warning("failed to compute payload hashes, assuming changed: " + err.Error())
		resp.DetailedDiff["payloadHashes"] = p.PropertyDiff{Kind: p.Update, InputDiff: true}
		resp.HasChanges = true
		return resp, nil
	}

	oldHashes := olds.PayloadHashes

	// Nil old hashes means state predates this feature; force an update so
	// hashes get persisted.
	if oldHashes == nil {
		resp.DetailedDiff["payloadHashes"] = p.PropertyDiff{Kind: p.Update, InputDiff: true}
	} else {
		for name, newHash := range newHashes {
			if oldHash, ok := oldHashes[name]; !ok {
				resp.DetailedDiff["payloadHashes."+name] = p.PropertyDiff{Kind: p.Add, InputDiff: true}
			} else if oldHash != newHash {
				resp.DetailedDiff["payloadHashes."+name] = p.PropertyDiff{Kind: p.Update, InputDiff: true}
			}
		}
		for name := range oldHashes {
			if _, ok := newHashes[name]; !ok {
				resp.DetailedDiff["payloadHashes."+name] = p.PropertyDiff{Kind: p.Delete, InputDiff: true}
			}
		}
	}

	resp.HasChanges = len(resp.DetailedDiff) > 0

	return resp, nil
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
