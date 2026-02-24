package runner

import (
	"os"
	"path/filepath"
	"testing"
)

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
