package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func TestValidate_ValidLocalPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	asset := FileAsset{
		LocalPath: strPtr(path),
		Filename:  strPtr("test.txt"),
		Mode:      intPtr(0644),
	}
	if err := asset.Validate(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidate_ValidContents(t *testing.T) {
	asset := FileAsset{
		Contents: strPtr("hello"),
		Filename: strPtr("hello.txt"),
		Mode:     intPtr(0644),
	}
	if err := asset.Validate(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidate_NeitherSet(t *testing.T) {
	asset := FileAsset{
		Filename: strPtr("empty.txt"),
		Mode:     intPtr(0644),
	}
	if err := asset.Validate(); err == nil {
		t.Fatal("expected error when neither LocalPath nor Contents is set")
	}
}

func TestValidate_BothSet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	asset := FileAsset{
		LocalPath: strPtr(path),
		Contents:  strPtr("also data"),
		Filename:  strPtr("test.txt"),
		Mode:      intPtr(0644),
	}
	if err := asset.Validate(); err == nil {
		t.Fatal("expected error when both LocalPath and Contents are set")
	}
}

func TestValidate_MissingFilename(t *testing.T) {
	asset := FileAsset{
		Contents: strPtr("data"),
		Mode:     intPtr(0644),
	}
	if err := asset.Validate(); err == nil {
		t.Fatal("expected error when Filename is missing")
	}
}

func TestValidate_MissingMode(t *testing.T) {
	asset := FileAsset{
		Contents: strPtr("data"),
		Filename: strPtr("test.txt"),
	}
	if err := asset.Validate(); err == nil {
		t.Fatal("expected error when Mode is missing")
	}
}

func TestGetHash_LocalPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	content := "hello world"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	asset := FileAsset{LocalPath: strPtr(path), Filename: strPtr("test.txt")}
	got, err := asset.GetHash()
	if err != nil {
		t.Fatal(err)
	}

	want := sha256Hex(content)
	if got != want {
		t.Errorf("GetHash() = %s, want %s", got, want)
	}
}

func TestGetHash_Contents(t *testing.T) {
	content := "inline content"
	asset := FileAsset{Contents: strPtr(content), Filename: strPtr("inline.txt")}

	got, err := asset.GetHash()
	if err != nil {
		t.Fatal(err)
	}

	want := sha256Hex(content)
	if got != want {
		t.Errorf("GetHash() = %s, want %s", got, want)
	}
}

func TestGetHash_NeitherSet(t *testing.T) {
	asset := FileAsset{Filename: strPtr("empty.txt")}
	_, err := asset.GetHash()
	if err == nil {
		t.Fatal("expected error when neither LocalPath nor Contents is set")
	}
}

func TestGetHash_MissingFile(t *testing.T) {
	asset := FileAsset{LocalPath: strPtr("/nonexistent/path/file.txt"), Filename: strPtr("file.txt")}
	_, err := asset.GetHash()
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestComputePayloadHashes_MultipleSlices(t *testing.T) {
	c1 := "content one"
	c2 := "content two"
	c3 := "content three"

	slice1 := []FileAsset{
		{Contents: strPtr(c1), Filename: strPtr("a.txt")},
		{Contents: strPtr(c2), Filename: strPtr("b.txt")},
	}
	slice2 := []FileAsset{
		{Contents: strPtr(c3), Filename: strPtr("c.txt")},
	}

	hashes, err := ComputePayloadHashes(slice1, slice2)
	if err != nil {
		t.Fatal(err)
	}

	if len(hashes) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(hashes))
	}

	for name, content := range map[string]string{"a.txt": c1, "b.txt": c2, "c.txt": c3} {
		if hashes[name] != sha256Hex(content) {
			t.Errorf("hash mismatch for %s", name)
		}
	}
}

func TestComputePayloadHashes_NilFilenameSkipped(t *testing.T) {
	slice := []FileAsset{
		{Contents: strPtr("data"), Filename: nil},
		{Contents: strPtr("data"), Filename: strPtr("keep.txt")},
	}

	hashes, err := ComputePayloadHashes(slice)
	if err != nil {
		t.Fatal(err)
	}

	if len(hashes) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(hashes))
	}
	if _, ok := hashes["keep.txt"]; !ok {
		t.Error("expected keep.txt in hashes")
	}
}

func TestComputePayloadHashes_Empty(t *testing.T) {
	hashes, err := ComputePayloadHashes()
	if err != nil {
		t.Fatal(err)
	}
	if len(hashes) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(hashes))
	}
}

func TestComputePayloadHashes_PropagatesGetHashError(t *testing.T) {
	slice := []FileAsset{
		{Contents: strPtr("good"), Filename: strPtr("ok.txt")},
		{LocalPath: strPtr("/nonexistent/bad.txt"), Filename: strPtr("bad.txt")},
	}

	_, err := ComputePayloadHashes(slice)
	if err == nil {
		t.Fatal("expected error to propagate from failing GetHash")
	}
}
