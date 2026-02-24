package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// FileAsset represents a file that can be uploaded to a remote server
type FileAsset struct {
	// Local file for uploading
	LocalPath *string `pulumi:"localPath,optional"`

	// Specify the contents in a string
	Contents *string `pulumi:"contents,optional"`

	// Filename required when Contents is provided
	Filename *string `pulumi:"filename,optional"`

	// File permissions mode (e.g., 0o0755)
	Mode *int `pulumi:"mode,optional"`
}

// openContent returns a reader for the asset's content. This is the
// single decision point for which content source a FileAsset uses.
func (f *FileAsset) openContent() (io.ReadCloser, error) {
	hasLocalPath := !IsEmptyStr(f.LocalPath)
	hasContents := f.Contents != nil

	switch {
	case hasLocalPath && hasContents:
		return nil, fmt.Errorf("cannot set both LocalPath and Contents")
	case hasLocalPath:
		file, err := os.Open(*f.LocalPath)
		if err != nil {
			return nil, fmt.Errorf("opening %s: %w", *f.LocalPath, err)
		}
		return file, nil
	case hasContents:
		return io.NopCloser(strings.NewReader(*f.Contents)), nil
	default:
		return nil, fmt.Errorf("exactly one of LocalPath or Contents must be set")
	}
}

// Validate ensures the FileAsset is properly configured
func (f *FileAsset) Validate() error {
	var errs []error

	if rc, err := f.openContent(); err != nil {
		errs = append(errs, err)
	} else {
		_ = rc.Close()
	}

	if IsEmptyStr(f.Filename) {
		errs = append(errs, fmt.Errorf("'Filename' must be set"))
	}

	if f.Mode == nil {
		errs = append(errs, fmt.Errorf("'Mode' must be set"))
	}
	return errors.Join(errs...)
}

// IsEmptyStr checks if a string pointer is nil or contains only whitespace
func IsEmptyStr(s *string) bool {
	return s == nil || strings.TrimSpace(*s) == ""
}

// GetHash computes the SHA-256 hash of the file asset's content.
func (f *FileAsset) GetHash() (string, error) {
	rc, err := f.openContent()
	if err != nil {
		return "", err
	}
	defer func() { _ = rc.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, rc); err != nil {
		return "", fmt.Errorf("reading content: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ComputePayloadHashes computes SHA-256 hashes for all file assets across
// multiple payload slices, returning a map from filename to hash.
func ComputePayloadHashes(payloads ...[]FileAsset) (map[string]string, error) {
	hashes := make(map[string]string)

	for _, payload := range payloads {
		for i := range payload {
			asset := &payload[i]
			if asset.Filename == nil {
				continue
			}

			hash, err := asset.GetHash()
			if err != nil {
				return nil, err
			}

			hashes[*asset.Filename] = hash
		}
	}

	return hashes, nil
}
