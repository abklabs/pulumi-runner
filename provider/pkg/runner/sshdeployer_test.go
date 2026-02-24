package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/plugin"

	"github.com/abklabs/pulumi-runner/pkg/ssh"
)

func floatPtr(f float64) *float64 { return &f }

func minimalArgs() SSHDeployerArgs {
	var conn ssh.Connection
	conn.Host = strPtr("localhost")
	conn.User = strPtr("root")
	conn.Port = floatPtr(22)
	conn.DialErrorLimit = intPtr(10)
	conn.PerDialTimeout = intPtr(15)
	return SSHDeployerArgs{
		Connection: conn,
	}
}

func TestDiff_NoChanges(t *testing.T) {
	args := minimalArgs()
	args.Payload = []FileAsset{
		{Contents: strPtr("hello"), Filename: strPtr("a.txt")},
	}

	olds := SSHDeployerState{
		SSHDeployerArgs: args,
		PayloadHashes: map[string]string{
			"a.txt": sha256Hex("hello"),
		},
	}

	resp, err := SSHDeployer{}.Diff(context.Background(), "test-id", olds, args)
	if err != nil {
		t.Fatal(err)
	}
	if resp.HasChanges {
		t.Errorf("expected no changes, got DetailedDiff: %v", resp.DetailedDiff)
	}
}

func TestDiff_InputPropertyChange(t *testing.T) {
	oldArgs := minimalArgs()
	oldArgs.Environment = map[string]string{"KEY": "old"}

	newArgs := minimalArgs()
	newArgs.Environment = map[string]string{"KEY": "new"}

	olds := SSHDeployerState{
		SSHDeployerArgs: oldArgs,
		PayloadHashes:   map[string]string{},
	}

	resp, err := SSHDeployer{}.Diff(context.Background(), "test-id", olds, newArgs)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasChanges {
		t.Fatal("expected changes from environment update")
	}

	hasEnvDiff := false
	for k := range resp.DetailedDiff {
		if k == "environment" || k == "environment.KEY" {
			hasEnvDiff = true
			break
		}
	}
	if !hasEnvDiff {
		t.Errorf("expected diff entry for environment, got: %v", resp.DetailedDiff)
	}
}

func TestDiff_FileContentChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deploy.sh")

	if err := os.WriteFile(path, []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}

	oldArgs := minimalArgs()
	oldArgs.Payload = []FileAsset{
		{LocalPath: strPtr(path), Filename: strPtr("deploy.sh")},
	}

	newArgs := minimalArgs()
	newArgs.Payload = []FileAsset{
		{LocalPath: strPtr(path), Filename: strPtr("deploy.sh")},
	}

	olds := SSHDeployerState{
		SSHDeployerArgs: oldArgs,
		PayloadHashes: map[string]string{
			"deploy.sh": sha256Hex("v0"),
		},
	}

	resp, err := SSHDeployer{}.Diff(context.Background(), "test-id", olds, newArgs)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasChanges {
		t.Fatal("expected changes from file content change")
	}
	if _, ok := resp.DetailedDiff["payloadHashes.deploy.sh"]; !ok {
		t.Error("expected payloadHashes.deploy.sh in DetailedDiff")
	}
}

func TestDiff_NilOldHashes(t *testing.T) {
	args := minimalArgs()
	args.Payload = []FileAsset{
		{Contents: strPtr("data"), Filename: strPtr("f.txt")},
	}

	olds := SSHDeployerState{
		SSHDeployerArgs: args,
		PayloadHashes:   nil,
	}

	resp, err := SSHDeployer{}.Diff(context.Background(), "test-id", olds, args)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasChanges {
		t.Fatal("expected forced update when old hashes are nil")
	}
}

func TestDiff_NewFileAdded(t *testing.T) {
	args := minimalArgs()
	args.Payload = []FileAsset{
		{Contents: strPtr("existing"), Filename: strPtr("old.txt")},
		{Contents: strPtr("new"), Filename: strPtr("new.txt")},
	}

	olds := SSHDeployerState{
		SSHDeployerArgs: minimalArgs(),
		PayloadHashes: map[string]string{
			"old.txt": sha256Hex("existing"),
		},
	}

	resp, err := SSHDeployer{}.Diff(context.Background(), "test-id", olds, args)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasChanges {
		t.Fatal("expected changes from new file")
	}

	diff, ok := resp.DetailedDiff["payloadHashes.new.txt"]
	if !ok {
		t.Fatal("expected payloadHashes.new.txt in DetailedDiff")
	}
	if diff.Kind != p.Add {
		t.Errorf("expected Add kind, got %s", diff.Kind)
	}
}

func TestDiff_FileRemoved(t *testing.T) {
	args := minimalArgs()

	olds := SSHDeployerState{
		SSHDeployerArgs: minimalArgs(),
		PayloadHashes: map[string]string{
			"gone.txt": sha256Hex("old data"),
		},
	}

	resp, err := SSHDeployer{}.Diff(context.Background(), "test-id", olds, args)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasChanges {
		t.Fatal("expected changes from removed file")
	}

	diff, ok := resp.DetailedDiff["payloadHashes.gone.txt"]
	if !ok {
		t.Fatal("expected payloadHashes.gone.txt in DetailedDiff")
	}
	if diff.Kind != p.Delete {
		t.Errorf("expected Delete kind, got %s", diff.Kind)
	}
}

func TestAllPayloads_CollectsAll(t *testing.T) {
	args := SSHDeployerArgs{
		Payload: []FileAsset{{Contents: strPtr("g"), Filename: strPtr("global.txt")}},
		Create:  &CommandDefinition{Command: "c", Payload: []FileAsset{{Contents: strPtr("c"), Filename: strPtr("create.txt")}}},
		Update:  &CommandDefinition{Command: "u", Payload: []FileAsset{{Contents: strPtr("u"), Filename: strPtr("update.txt")}}},
		Delete:  &CommandDefinition{Command: "d", Payload: []FileAsset{{Contents: strPtr("d"), Filename: strPtr("delete.txt")}}},
	}

	payloads := args.allPayloads()
	if len(payloads) != 4 {
		t.Fatalf("expected 4 payload slices, got %d", len(payloads))
	}
}

func TestAllPayloads_NilCommandDefinitions(t *testing.T) {
	args := SSHDeployerArgs{
		Payload: []FileAsset{{Contents: strPtr("g"), Filename: strPtr("global.txt")}},
	}

	payloads := args.allPayloads()
	// Global payload only; nil Create/Update/Delete are skipped.
	if len(payloads) != 1 {
		t.Fatalf("expected 1 payload slice, got %d", len(payloads))
	}
}

func TestDiff_HashComputationFailure(t *testing.T) {
	args := minimalArgs()
	args.Payload = []FileAsset{
		{LocalPath: strPtr("/nonexistent/missing.sh"), Filename: strPtr("missing.sh")},
	}

	olds := SSHDeployerState{
		SSHDeployerArgs: minimalArgs(),
		PayloadHashes: map[string]string{
			"missing.sh": sha256Hex("old content"),
		},
	}

	resp, err := SSHDeployer{}.Diff(context.Background(), "test-id", olds, args)
	if err != nil {
		t.Fatal("expected nil error on hash failure fallback, got:", err)
	}
	if !resp.HasChanges {
		t.Fatal("expected HasChanges when hash computation fails")
	}
	diff, ok := resp.DetailedDiff["payloadHashes"]
	if !ok {
		t.Fatal("expected payloadHashes key in DetailedDiff")
	}
	if diff.Kind != p.Update {
		t.Errorf("expected Update kind, got %s", diff.Kind)
	}
}

func TestDiff_StructuralAndHashChangeCombined(t *testing.T) {
	oldArgs := minimalArgs()
	oldArgs.Environment = map[string]string{"KEY": "old"}
	oldArgs.Payload = []FileAsset{
		{Contents: strPtr("v1"), Filename: strPtr("app.txt")},
	}

	newArgs := minimalArgs()
	newArgs.Environment = map[string]string{"KEY": "new"}
	newArgs.Payload = []FileAsset{
		{Contents: strPtr("v2"), Filename: strPtr("app.txt")},
	}

	olds := SSHDeployerState{
		SSHDeployerArgs: oldArgs,
		PayloadHashes: map[string]string{
			"app.txt": sha256Hex("v1"),
		},
	}

	resp, err := SSHDeployer{}.Diff(context.Background(), "test-id", olds, newArgs)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasChanges {
		t.Fatal("expected changes")
	}

	// Structural diff should detect the environment change.
	hasStructural := false
	for k := range resp.DetailedDiff {
		if k == "environment" || k == "environment.KEY" {
			hasStructural = true
			break
		}
	}
	if !hasStructural {
		t.Errorf("expected structural diff entry for environment, got: %v", resp.DetailedDiff)
	}

	// Hash diff should detect the payload content change.
	if _, ok := resp.DetailedDiff["payloadHashes.app.txt"]; !ok {
		t.Errorf("expected hash diff entry for app.txt, got: %v", resp.DetailedDiff)
	}
}

func TestDiff_EmptyNonNilHashesNoPayloads(t *testing.T) {
	args := minimalArgs()

	olds := SSHDeployerState{
		SSHDeployerArgs: args,
		PayloadHashes:   map[string]string{},
	}

	resp, err := SSHDeployer{}.Diff(context.Background(), "test-id", olds, args)
	if err != nil {
		t.Fatal(err)
	}
	if resp.HasChanges {
		t.Errorf("expected no changes with empty hashes and no payloads, got DetailedDiff: %v", resp.DetailedDiff)
	}
}

func TestAllPayloads_PartialCommandDefinitions(t *testing.T) {
	args := SSHDeployerArgs{
		Payload: []FileAsset{{Contents: strPtr("g"), Filename: strPtr("global.txt")}},
		Create:  &CommandDefinition{Command: "c", Payload: []FileAsset{{Contents: strPtr("c"), Filename: strPtr("create.txt")}}},
		Delete:  &CommandDefinition{Command: "d", Payload: []FileAsset{{Contents: strPtr("d"), Filename: strPtr("delete.txt")}}},
	}

	payloads := args.allPayloads()
	if len(payloads) != 3 {
		t.Fatalf("expected 3 payload slices (global + create + delete), got %d", len(payloads))
	}
}

func TestConvertDiffKind(t *testing.T) {
	cases := []struct {
		input plugin.DiffKind
		want  p.DiffKind
	}{
		{plugin.DiffAdd, p.Add},
		{plugin.DiffAddReplace, p.AddReplace},
		{plugin.DiffDelete, p.Delete},
		{plugin.DiffDeleteReplace, p.DeleteReplace},
		{plugin.DiffUpdate, p.Update},
		{plugin.DiffUpdateReplace, p.UpdateReplace},
	}

	for _, tc := range cases {
		got := convertDiffKind(tc.input)
		if got != tc.want {
			t.Errorf("convertDiffKind(%d) = %s, want %s", tc.input, got, tc.want)
		}
	}
}

func TestConvertDiffKind_UnknownDefault(t *testing.T) {
	got := convertDiffKind(plugin.DiffKind(99))
	if got != p.Update {
		t.Errorf("convertDiffKind(99) = %s, want %s", got, p.Update)
	}
}

func TestComputePayloadHashes_DuplicateFilenameLastWriterWins(t *testing.T) {
	global := []FileAsset{
		{Contents: strPtr("global version"), Filename: strPtr("deploy.sh")},
	}
	cmdPayload := []FileAsset{
		{Contents: strPtr("command version"), Filename: strPtr("deploy.sh")},
	}

	hashes, err := ComputePayloadHashes(global, cmdPayload)
	if err != nil {
		t.Fatal(err)
	}

	if len(hashes) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(hashes))
	}

	// The command-level payload is iterated after global, so it wins.
	want := sha256Hex("command version")
	if hashes["deploy.sh"] != want {
		t.Errorf("expected command version hash, got hash of something else")
	}
}
