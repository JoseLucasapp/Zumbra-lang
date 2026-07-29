package appdist

import (
	"bytes"
	"debug/elf"
	"fmt"
	"os"
	"path/filepath"
)

// cacheAppImageRuntime extracts the ELF runtime prefix from a generated type-2
// AppImage. This turns the first successful online build into an offline,
// pinned input for later reproducible builds.
func cacheAppImageRuntime(appImage, arch string) (string, error) {
	data, err := os.ReadFile(appImage)
	if err != nil {
		return "", err
	}
	file, err := elf.Open(appImage)
	if err != nil {
		return "", fmt.Errorf("inspect generated AppImage runtime: %w", err)
	}
	defer file.Close()
	minimum := uint64(0)
	for _, program := range file.Progs {
		if end := program.Off + program.Filesz; end > minimum {
			minimum = end
		}
	}
	for _, section := range file.Sections {
		if end := section.Offset + section.Size; end > minimum {
			minimum = end
		}
	}
	if minimum >= uint64(len(data)) {
		return "", fmt.Errorf("generated AppImage does not contain an appended filesystem")
	}
	index := bytes.Index(data[minimum:], []byte{'h', 's', 'q', 's'})
	if index < 0 {
		return "", fmt.Errorf("generated AppImage does not contain a SquashFS payload")
	}
	runtimeEnd := int(minimum) + index
	if runtimeEnd <= 0 {
		return "", fmt.Errorf("generated AppImage runtime is empty")
	}
	path, err := AppImageRuntimeCachePath(arch)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data[:runtimeEnd], 0o755); err != nil {
		return "", err
	}
	return path, nil
}
