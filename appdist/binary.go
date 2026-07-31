package appdist

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// BinaryInfo describes the operating system and architecture encoded in a
// native executable header. It intentionally avoids shelling out to file(1),
// keeping package validation deterministic and available on every host.
type BinaryInfo struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

func InspectBinary(path string) (BinaryInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return BinaryInfo{}, fmt.Errorf("open binary: %w", err)
	}
	defer file.Close()

	header := make([]byte, 4096)
	n, readErr := io.ReadFull(file, header)
	if readErr != nil && readErr != io.ErrUnexpectedEOF {
		return BinaryInfo{}, fmt.Errorf("read binary header: %w", readErr)
	}
	header = header[:n]

	if info, ok := inspectELF(header); ok {
		return info, nil
	}
	if info, ok := inspectPE(header); ok {
		return info, nil
	}
	if info, ok := inspectMachO(header); ok {
		return info, nil
	}
	return BinaryInfo{}, fmt.Errorf("unrecognized native binary format; expected ELF, PE or Mach-O")
}

func inspectELF(data []byte) (BinaryInfo, bool) {
	if len(data) < 20 || data[0] != 0x7f || string(data[1:4]) != "ELF" {
		return BinaryInfo{}, false
	}
	var order binary.ByteOrder = binary.LittleEndian
	if data[5] == 2 {
		order = binary.BigEndian
	}
	machine := order.Uint16(data[18:20])
	arch := ""
	switch machine {
	case 0x3e:
		arch = "amd64"
	case 0xb7:
		arch = "arm64"
	case 0x03:
		arch = "386"
	case 0x28:
		arch = "arm"
	}
	return BinaryInfo{OS: "linux", Arch: arch}, true
}

func inspectPE(data []byte) (BinaryInfo, bool) {
	if len(data) < 0x40 || data[0] != 'M' || data[1] != 'Z' {
		return BinaryInfo{}, false
	}
	offset := int(binary.LittleEndian.Uint32(data[0x3c:0x40]))
	if offset < 0 || offset+6 > len(data) || string(data[offset:offset+4]) != "PE\x00\x00" {
		return BinaryInfo{}, false
	}
	machine := binary.LittleEndian.Uint16(data[offset+4 : offset+6])
	arch := ""
	switch machine {
	case 0x8664:
		arch = "amd64"
	case 0xaa64:
		arch = "arm64"
	case 0x014c:
		arch = "386"
	case 0x01c4:
		arch = "arm"
	}
	return BinaryInfo{OS: "windows", Arch: arch}, true
}

func inspectMachO(data []byte) (BinaryInfo, bool) {
	if len(data) < 8 {
		return BinaryInfo{}, false
	}
	magic := binary.BigEndian.Uint32(data[:4])
	var order binary.ByteOrder = binary.BigEndian
	fat := false
	switch magic {
	case 0xfeedface, 0xfeedfacf:
		order = binary.BigEndian
	case 0xcefaedfe, 0xcffaedfe:
		order = binary.LittleEndian
	case 0xcafebabe, 0xcafebabf:
		order = binary.BigEndian
		fat = true
	case 0xbebafeca, 0xbfbafeca:
		order = binary.LittleEndian
		fat = true
	default:
		return BinaryInfo{}, false
	}
	if fat {
		// Universal binaries can satisfy either supported architecture. The
		// package validator treats this explicit marker as compatible.
		return BinaryInfo{OS: "macos", Arch: "universal"}, true
	}
	cpu := order.Uint32(data[4:8])
	arch := ""
	switch cpu {
	case 0x01000007:
		arch = "amd64"
	case 0x0100000c:
		arch = "arm64"
	case 7:
		arch = "386"
	case 12:
		arch = "arm"
	}
	return BinaryInfo{OS: "macos", Arch: arch}, true
}

func validateBinaryForTarget(path, target, arch string) (BinaryInfo, error) {
	info, err := InspectBinary(path)
	if err != nil {
		return BinaryInfo{}, fmt.Errorf("validate package binary %s: %w", path, err)
	}
	if info.OS != target {
		return info, fmt.Errorf("binary target mismatch: executable is %s/%s but package target is %s/%s", info.OS, displayArch(info.Arch), target, arch)
	}
	if info.Arch != "" && info.Arch != "universal" && info.Arch != arch {
		return info, fmt.Errorf("binary architecture mismatch: executable is %s/%s but package target is %s/%s", info.OS, info.Arch, target, arch)
	}
	if info.Arch == "" {
		return info, fmt.Errorf("binary architecture is unsupported or could not be determined for %s", path)
	}
	return info, nil
}

func displayArch(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}
