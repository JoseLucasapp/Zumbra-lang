package appdist

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func (c *packageContext) generateSymbols() error {
	base := filepath.Join(c.outDir, c.baseName())
	switch c.options.Target {
	case "linux", "windows":
		tool := ""
		for _, candidate := range []string{"objcopy", "llvm-objcopy"} {
			if found, err := exec.LookPath(candidate); err == nil {
				tool = found
				break
			}
		}
		if tool == "" {
			return fmt.Errorf("objcopy or llvm-objcopy is required for --symbols")
		}
		output := base + ".debug"
		cmd := exec.Command(tool, "--only-keep-debug", c.options.Binary, output)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("extract debug symbols: %w", err)
		}
		if err := os.Chtimes(output, c.epoch, c.epoch); err != nil {
			return err
		}
		return c.addArtifact("debug-symbols", output)
	case "macos":
		tool, err := exec.LookPath("dsymutil")
		if err != nil {
			return fmt.Errorf("dsymutil is required for macOS --symbols")
		}
		dsym := base + ".dSYM"
		_ = os.RemoveAll(dsym)
		cmd := exec.Command(tool, c.options.Binary, "-o", dsym)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("extract macOS debug symbols: %w", err)
		}
		archive := base + "-dSYM.zip"
		if err := writeZip(dsym, archive, filepath.Base(dsym), c.epoch); err != nil {
			return err
		}
		return c.addArtifact("debug-symbols", archive)
	}
	return nil
}
