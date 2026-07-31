package appdist

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func (c *packageContext) signArtifacts() error {
	identity := strings.TrimSpace(c.options.SignIdentity)
	if identity == "" {
		return nil
	}
	switch c.options.Target {
	case "linux":
		tool, err := exec.LookPath("gpg")
		if err != nil {
			return fmt.Errorf("gpg is required to sign Linux artifacts")
		}
		original := append([]Artifact(nil), c.result.Artifacts...)
		for _, artifact := range original {
			if artifact.SHA256 == "" {
				continue
			}
			signature := artifact.Path + ".asc"
			cmd := exec.Command(tool, "--batch", "--yes", "--armor", "--local-user", identity, "--detach-sign", "--output", signature, artifact.Path)
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("gpg signing failed for %s: %w", artifact.Path, err)
			}
			if err := c.addArtifact("signature", signature); err != nil {
				return err
			}
		}
	case "windows":
		tool, err := exec.LookPath("signtool")
		if err != nil {
			return fmt.Errorf("signtool is required to sign Windows artifacts")
		}
		for _, artifact := range c.result.Artifacts {
			if !strings.HasSuffix(strings.ToLower(artifact.Path), ".exe") {
				continue
			}
			cmd := exec.Command(tool, "sign", "/fd", "SHA256", "/n", identity, artifact.Path)
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("Windows signing failed for %s: %w", artifact.Path, err)
			}
		}
	case "macos":
		// macOS application bundles are signed before ZIP creation.
	}
	return nil
}

func (c *packageContext) signMacApp(path string) error {
	identity := strings.TrimSpace(c.options.SignIdentity)
	if identity == "" {
		return nil
	}
	tool, err := exec.LookPath("codesign")
	if err != nil {
		return fmt.Errorf("codesign is required to sign macOS applications")
	}
	cmd := exec.Command(tool, "--force", "--deep", "--options", "runtime", "--sign", identity, path)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("macOS signing failed for %s: %w", path, err)
	}
	return nil
}
