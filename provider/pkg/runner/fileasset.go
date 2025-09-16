package runner

import (
	"errors"
	"fmt"
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

// Validate ensures the FileAsset is properly configured
func (f *FileAsset) Validate() error {
	// Either LocalPath or Contents, but not both
	var errs []error

	hasLocalPath := !IsEmptyStr(f.LocalPath)
	hasContents := f.Contents != nil

	if !hasLocalPath && !hasContents {
		errs = append(errs, fmt.Errorf("exactly one of LocalPath or Contents must be set"))
	}

	if hasLocalPath && hasContents {
		errs = append(errs, fmt.Errorf("cannot set both LocalPath and Contents"))
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
