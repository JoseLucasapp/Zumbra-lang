//go:build !linux || !cgo

package builtins

import (
	"fmt"

	"zumbra/object"
)

func openMappedMemory(path, mode string, requestedSize int64) (*object.MappedMemory, error) {
	return nil, fmt.Errorf("memory mapping is not available on this platform")
}
func flushMappedMemory(mapped *object.MappedMemory) error {
	return fmt.Errorf("memory mapping is not available on this platform")
}
func closeMappedMemory(mapped *object.MappedMemory) error {
	return fmt.Errorf("memory mapping is not available on this platform")
}
func openSharedMemory(name string, size int64, create bool) (*object.SharedMemory, error) {
	return nil, fmt.Errorf("shared memory is not available on this platform")
}
func closeSharedMemory(shared *object.SharedMemory) error {
	return fmt.Errorf("shared memory is not available on this platform")
}
func unlinkSharedMemory(name string) error {
	return fmt.Errorf("shared memory is not available on this platform")
}
func platformMemoryFence(mode string) error { return nil }
func platformAtomicPointer(operation string, pointer *object.Pointer, args ...object.Object) object.Object {
	return NewError("atomic pointer operations are not available on this platform")
}
func protectPointerMemory(pointer *object.Pointer, mode string) error {
	return fmt.Errorf("memory protection is not available on this platform")
}
func lockPointerMemory(pointer *object.Pointer) error {
	return fmt.Errorf("memory locking is not available on this platform")
}
func unlockPointerMemory(pointer *object.Pointer) error {
	return fmt.Errorf("memory unlocking is not available on this platform")
}
func openDynamicLibrary(path string) (*object.DynamicLibrary, error) {
	return nil, fmt.Errorf("dynamic loading is not available on this platform")
}
func lookupDynamicSymbol(library *object.DynamicLibrary, name string) (*object.Pointer, error) {
	return nil, fmt.Errorf("dynamic loading is not available on this platform")
}
func closeDynamicLibrary(library *object.DynamicLibrary) error {
	return fmt.Errorf("dynamic loading is not available on this platform")
}
func platformRawSyscall(number uintptr, args []uintptr) (uintptr, uintptr, error) {
	return 0, 0, fmt.Errorf("raw syscalls are not available on this platform")
}

func callDynamicFunction(address uintptr, args []uintptr) (uintptr, error) {
	return 0, fmt.Errorf("dynamic calls are not available on this platform")
}
