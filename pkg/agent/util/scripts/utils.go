// Copyright (c) OpenMMLab. All rights reserved.

package scripts

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"text/template"
)

// Generate and execute script
func executeScript(ctx context.Context, tmpl *template.Template) (output []byte, err error) {
	var scriptBuf bytes.Buffer
	if err := tmpl.Execute(&scriptBuf, nil); err != nil {
		return nil, err
	}

	// Create temporary file and execute
	return executeShellScript(scriptBuf.String())
}

// Execute shell script
func executeShellScript(scriptContent string) ([]byte, error) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "script_*.sh")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	// Write script content
	if _, err := tmpFile.WriteString(scriptContent); err != nil {
		return nil, err
	}
	if err := tmpFile.Close(); err != nil {
		return nil, err
	}

	// Set execution permissions
	if err := os.Chmod(tmpFile.Name(), 0700); err != nil {
		return nil, err
	}

	// Execute script
	cmd := exec.Command("bash", "-c", tmpFile.Name())

	output, err := cmd.CombinedOutput()
	if err != nil {
		return []byte{}, fmt.Errorf("%v: %s", err, string(output))
	}

	return output, nil
}
