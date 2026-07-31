package appdist

import (
	"fmt"
	"os"
	"path/filepath"
)

func (c *packageContext) runtimeFiles(target string) ([]string, error) {
	files, err := c.options.Manifest.RuntimeFilesForTarget(target)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (c *packageContext) copyRuntimeFiles(target, destination string) error {
	files, err := c.runtimeFiles(target)
	if err != nil {
		return err
	}
	for _, source := range files {
		info, err := os.Lstat(source)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("runtime file %s must be a regular non-symlink file", source)
		}
		if err := copyFile(source, filepath.Join(destination, filepath.Base(source)), 0o755, c.epoch); err != nil {
			return err
		}
	}
	return nil
}
