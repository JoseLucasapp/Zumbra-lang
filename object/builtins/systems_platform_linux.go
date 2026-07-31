//go:build linux && cgo

package builtins

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
#include <stdint.h>

static void *z17_dlopen(const char *path) { return dlopen(path, RTLD_NOW | RTLD_LOCAL); }
static void *z17_dlsym(void *handle, const char *name) { dlerror(); return dlsym(handle, name); }
static int z17_dlclose(void *handle) { return dlclose(handle); }
static const char *z17_dlerror(void) { const char *value = dlerror(); return value == NULL ? "" : value; }

typedef uintptr_t (*z17_fn0)(void);
typedef uintptr_t (*z17_fn1)(uintptr_t);
typedef uintptr_t (*z17_fn2)(uintptr_t, uintptr_t);
typedef uintptr_t (*z17_fn3)(uintptr_t, uintptr_t, uintptr_t);
typedef uintptr_t (*z17_fn4)(uintptr_t, uintptr_t, uintptr_t, uintptr_t);
typedef uintptr_t (*z17_fn5)(uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t);
typedef uintptr_t (*z17_fn6)(uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t);

static uintptr_t z17_dynamic_call(void *symbol, int argc, uintptr_t *args) {
    switch (argc) {
    case 0: return ((z17_fn0)symbol)();
    case 1: return ((z17_fn1)symbol)(args[0]);
    case 2: return ((z17_fn2)symbol)(args[0], args[1]);
    case 3: return ((z17_fn3)symbol)(args[0], args[1], args[2]);
    case 4: return ((z17_fn4)symbol)(args[0], args[1], args[2], args[3]);
    case 5: return ((z17_fn5)symbol)(args[0], args[1], args[2], args[3], args[4]);
    case 6: return ((z17_fn6)symbol)(args[0], args[1], args[2], args[3], args[4], args[5]);
    default: return 0;
    }
}
*/
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"unsafe"

	"zumbra/collections"
	"zumbra/object"
)

func msyncBytes(data []byte, flags int) error {
	if len(data) == 0 {
		return nil
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_MSYNC,
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
		uintptr(flags),
	)
	if errno != 0 {
		return errno
	}
	return nil
}

func openMappedMemory(path, mode string, requestedSize int64) (*object.MappedMemory, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	flags := os.O_RDONLY
	protection := syscall.PROT_READ
	mapFlags := syscall.MAP_SHARED
	switch mode {
	case "read", "r":
		mode = "read"
	case "write", "readwrite", "rw":
		mode = "readwrite"
		flags = os.O_RDWR | os.O_CREATE
		protection |= syscall.PROT_WRITE
	case "private":
		mode = "private"
		flags = os.O_RDWR | os.O_CREATE
		protection |= syscall.PROT_WRITE
		mapFlags = syscall.MAP_PRIVATE
	default:
		return nil, fmt.Errorf("mode must be read, readwrite or private")
	}
	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	size := info.Size()
	if requestedSize > size && mode != "read" {
		if err := file.Truncate(requestedSize); err != nil {
			file.Close()
			return nil, err
		}
		size = requestedSize
	}
	if requestedSize > 0 && requestedSize < size {
		size = requestedSize
	}
	if size <= 0 {
		file.Close()
		return nil, fmt.Errorf("mapping size must be positive")
	}
	data, err := syscall.Mmap(int(file.Fd()), 0, int(size), protection, mapFlags)
	if err != nil {
		file.Close()
		return nil, err
	}
	t, _ := object.ParseSystemType("u8")
	block := &object.MemoryBlock{Data: data, Type: t, Length: len(data), Allocator: "mapped"}
	mapped := &object.MappedMemory{Path: path, Mode: mode, Data: data, Block: block, Private: file}
	block.Backing = mapped
	return mapped, nil
}

func flushMappedMemory(mapped *object.MappedMemory) error {
	mapped.Mu.Lock()
	defer mapped.Mu.Unlock()
	if mapped.Closed {
		return fmt.Errorf("mapping is closed")
	}
	return msyncBytes(mapped.Data, syscall.MS_SYNC)
}

func closeMappedMemory(mapped *object.MappedMemory) error {
	mapped.Mu.Lock()
	defer mapped.Mu.Unlock()
	if mapped.Closed {
		return nil
	}
	var first error
	if mapped.Mode != "read" {
		if err := msyncBytes(mapped.Data, syscall.MS_SYNC); err != nil {
			first = err
		}
	}
	if err := syscall.Munmap(mapped.Data); err != nil && first == nil {
		first = err
	}
	if file, ok := mapped.Private.(*os.File); ok {
		if err := file.Close(); err != nil && first == nil {
			first = err
		}
	}
	mapped.Closed = true
	mapped.Data = nil
	if mapped.Block != nil {
		mapped.Block.Mu.Lock()
		mapped.Block.Freed = true
		mapped.Block.Data = nil
		mapped.Block.Length = 0
		mapped.Block.Mu.Unlock()
	}
	return first
}

func sharedMemoryPath(name string) (string, error) {
	clean := strings.TrimSpace(name)
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" || strings.Contains(clean, "..") || strings.ContainsAny(clean, `/\\`) {
		return "", fmt.Errorf("invalid shared memory name")
	}
	return filepath.Join("/dev/shm", "zumbra-"+clean), nil
}

func openSharedMemory(name string, size int64, create bool) (*object.SharedMemory, error) {
	if size <= 0 {
		return nil, fmt.Errorf("size must be positive")
	}
	path, err := sharedMemoryPath(name)
	if err != nil {
		return nil, err
	}
	flags := os.O_RDWR
	if create {
		flags |= os.O_CREATE
	}
	file, err := os.OpenFile(path, flags, 0o600)
	if err != nil {
		return nil, err
	}
	if create {
		if err := file.Truncate(size); err != nil {
			file.Close()
			return nil, err
		}
	} else if info, statErr := file.Stat(); statErr == nil {
		size = info.Size()
	}
	data, err := syscall.Mmap(int(file.Fd()), 0, int(size), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		file.Close()
		return nil, err
	}
	t, _ := object.ParseSystemType("u8")
	block := &object.MemoryBlock{Data: data, Type: t, Length: len(data), Allocator: "shared"}
	shared := &object.SharedMemory{Name: name, Path: path, Data: data, Block: block, Private: file}
	block.Backing = shared
	return shared, nil
}

func closeSharedMemory(shared *object.SharedMemory) error {
	shared.Mu.Lock()
	defer shared.Mu.Unlock()
	if shared.Closed {
		return nil
	}
	var first error
	if err := msyncBytes(shared.Data, syscall.MS_SYNC); err != nil {
		first = err
	}
	if err := syscall.Munmap(shared.Data); err != nil && first == nil {
		first = err
	}
	if file, ok := shared.Private.(*os.File); ok {
		if err := file.Close(); err != nil && first == nil {
			first = err
		}
	}
	shared.Closed = true
	shared.Data = nil
	if shared.Block != nil {
		shared.Block.Mu.Lock()
		shared.Block.Freed = true
		shared.Block.Data = nil
		shared.Block.Length = 0
		shared.Block.Mu.Unlock()
	}
	return first
}

func unlinkSharedMemory(name string) error {
	path, err := sharedMemoryPath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

var fenceValue uint32

func platformMemoryFence(mode string) error {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "acquire", "release", "acq_rel", "seq_cst", "sequential":
		atomic.StoreUint32(&fenceValue, atomic.LoadUint32(&fenceValue)+1)
		return nil
	default:
		return fmt.Errorf("mode must be acquire, release, acq_rel or seq_cst")
	}
}

func platformAtomicPointer(operation string, pointer *object.Pointer, args ...object.Object) object.Object {
	if pointer.Block == nil || pointer.Length == 0 {
		return NewError("atomic pointer operation expects managed memory")
	}
	if pointer.ElementType().Float || pointer.ElementType().Bool || pointer.ElementType().Ptr {
		return NewError("atomic pointer operations require an integer element type")
	}
	pointer.Block.Mu.Lock()
	defer pointer.Block.Mu.Unlock()
	if pointer.Block.Freed {
		return NewError("atomic pointer operation: use after free")
	}
	start := pointer.Offset * pointer.ElementType().Size
	current := decodeAtomic(pointer.ElementType(), pointer.Block.Data[start:start+pointer.ElementType().Size])
	switch operation {
	case "load":
		return current
	case "store":
		if len(args) != 1 {
			return NewError("atomicPointerStore expects 2 arguments")
		}
		if err := encodeAtomic(pointer.ElementType(), pointer.Block.Data[start:start+pointer.ElementType().Size], args[0]); err != nil {
			return NewError("atomicPointerStore: %s", err)
		}
		return &object.Null{}
	case "swap":
		if len(args) != 1 {
			return NewError("atomicPointerSwap expects 2 arguments")
		}
		if err := encodeAtomic(pointer.ElementType(), pointer.Block.Data[start:start+pointer.ElementType().Size], args[0]); err != nil {
			return NewError("atomicPointerSwap: %s", err)
		}
		return current
	case "add":
		if len(args) != 1 {
			return NewError("atomicPointerAdd expects 2 arguments")
		}
		left, _ := collections.IntegerValue(current)
		right, err := collections.IntegerValue(args[0])
		if err != nil {
			return NewError("atomicPointerAdd: %s", err)
		}
		value := &object.Integer{Value: left + right}
		if err := encodeAtomic(pointer.ElementType(), pointer.Block.Data[start:start+pointer.ElementType().Size], value); err != nil {
			return NewError("atomicPointerAdd: %s", err)
		}
		return decodeAtomic(pointer.ElementType(), pointer.Block.Data[start:start+pointer.ElementType().Size])
	case "compare_swap":
		if len(args) != 2 {
			return NewError("atomicPointerCompareSwap expects 3 arguments")
		}
		expected, err := collections.IntegerValue(args[0])
		if err != nil {
			return NewError("atomicPointerCompareSwap: %s", err)
		}
		actual, _ := collections.IntegerValue(current)
		if actual != expected {
			return NewBoolean(false)
		}
		if err := encodeAtomic(pointer.ElementType(), pointer.Block.Data[start:start+pointer.ElementType().Size], args[1]); err != nil {
			return NewError("atomicPointerCompareSwap: %s", err)
		}
		return NewBoolean(true)
	default:
		return NewError("unknown atomic pointer operation")
	}
}

func decodeAtomic(t object.SystemType, data []byte) object.Object {
	block := &object.MemoryBlock{Data: data, Type: t, Length: 1}
	pointer := &object.Pointer{Block: block, Length: 1, Mutable: true}
	value, _ := pointer.Read(0)
	return value
}

func encodeAtomic(t object.SystemType, data []byte, value object.Object) error {
	block := &object.MemoryBlock{Data: data, Type: t, Length: 1}
	pointer := &object.Pointer{Block: block, Length: 1, Mutable: true}
	return pointer.Write(0, value)
}

func pointerBytes(pointer *object.Pointer) ([]byte, error) {
	if pointer == nil || pointer.Block == nil || pointer.Block.Freed {
		return nil, fmt.Errorf("pointer does not reference active managed memory")
	}
	start := pointer.Offset * pointer.ElementType().Size
	end := start + pointer.Length*pointer.ElementType().Size
	if start < 0 || end > len(pointer.Block.Data) {
		return nil, fmt.Errorf("pointer range is invalid")
	}
	return pointer.Block.Data[start:end], nil
}

func protectPointerMemory(pointer *object.Pointer, mode string) error {
	data, err := pointerBytes(pointer)
	if err != nil {
		return err
	}
	protection := 0
	for _, part := range strings.FieldsFunc(strings.ToLower(mode), func(r rune) bool { return r == '|' || r == ',' || r == '+' }) {
		switch strings.TrimSpace(part) {
		case "none":
		case "read", "r":
			protection |= syscall.PROT_READ
		case "write", "w":
			protection |= syscall.PROT_WRITE
		case "exec", "execute", "x":
			protection |= syscall.PROT_EXEC
		default:
			return fmt.Errorf("unknown protection %q", part)
		}
	}
	return syscall.Mprotect(data, protection)
}

func lockPointerMemory(pointer *object.Pointer) error {
	data, err := pointerBytes(pointer)
	if err != nil {
		return err
	}
	return syscall.Mlock(data)
}

func unlockPointerMemory(pointer *object.Pointer) error {
	data, err := pointerBytes(pointer)
	if err != nil {
		return err
	}
	return syscall.Munlock(data)
}

func openDynamicLibrary(path string) (*object.DynamicLibrary, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	handle := C.z17_dlopen(cPath)
	if handle == nil {
		return nil, fmt.Errorf("%s", C.GoString(C.z17_dlerror()))
	}
	return &object.DynamicLibrary{Path: path, Handle: uintptr(handle), Private: handle}, nil
}

func lookupDynamicSymbol(library *object.DynamicLibrary, name string) (*object.Pointer, error) {
	library.Mu.Lock()
	defer library.Mu.Unlock()
	if library.Closed || library.Private == nil {
		return nil, fmt.Errorf("library is closed")
	}
	handle, ok := library.Private.(unsafe.Pointer)
	if !ok || handle == nil {
		return nil, fmt.Errorf("library handle is invalid")
	}
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	// Call through a dedicated helper after clearing dlerror.
	symbol := C.z17_dlsym(handle, cName)
	if symbol == nil {
		message := C.GoString(C.z17_dlerror())
		if message == "" {
			message = "symbol not found"
		}
		return nil, fmt.Errorf("%s", message)
	}
	t, _ := object.ParseSystemType("ptr")
	return &object.Pointer{Raw: unsafe.Pointer(symbol), RawType: t, Mutable: false}, nil
}

func closeDynamicLibrary(library *object.DynamicLibrary) error {
	library.Mu.Lock()
	defer library.Mu.Unlock()
	if library.Closed {
		return nil
	}
	if library.Private != nil {
		handle, ok := library.Private.(unsafe.Pointer)
		if !ok || handle == nil {
			return fmt.Errorf("library handle is invalid")
		}
		if C.z17_dlclose(handle) != 0 {
			return fmt.Errorf("%s", C.GoString(C.z17_dlerror()))
		}
	}
	library.Closed = true
	library.Private = nil
	library.Handle = 0
	return nil
}

func platformRawSyscall(number uintptr, args []uintptr) (uintptr, uintptr, error) {
	result, _, errno := syscall.Syscall6(number, args[0], args[1], args[2], args[3], args[4], args[5])
	if errno != 0 {
		return result, uintptr(errno), errno
	}
	return result, 0, nil
}

func callDynamicFunction(address uintptr, args []uintptr) (uintptr, error) {
	if address == 0 {
		return 0, fmt.Errorf("dynamic function pointer is null")
	}
	if len(args) > 6 {
		return 0, fmt.Errorf("dynamic calls support at most 6 arguments")
	}
	values := make([]C.uintptr_t, 6)
	for index, value := range args {
		values[index] = C.uintptr_t(value)
	}
	return uintptr(C.z17_dynamic_call(unsafe.Pointer(address), C.int(len(args)), &values[0])), nil
}
