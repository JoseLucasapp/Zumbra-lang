package builtins

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"

	"zumbra/object"
)

func requireNoBuiltinError(t *testing.T, value object.Object) object.Object {
	t.Helper()
	if err, ok := value.(*object.Error); ok {
		t.Fatalf("unexpected builtin error: %s", err.Message)
	}
	return value
}

func integerValueForTest(t *testing.T, value object.Object) int64 {
	t.Helper()
	switch value := value.(type) {
	case *object.Integer:
		return value.Value
	case *object.FixedInteger:
		if value.Kind.Signed() {
			return value.SignedValue()
		}
		return int64(value.UnsignedValue())
	default:
		t.Fatalf("expected integer, got %T", value)
		return 0
	}
}

func TestZ17ExplicitMemoryOwnershipAndReallocation(t *testing.T) {
	requireNoBuiltinError(t, MemoryResetStatsBuiltin().Fn())
	pointer := requireNoBuiltinError(t, AllocBuiltin().Fn(NewString("i32"), NewInteger(4))).(*object.Pointer)
	for index, expected := range []int64{10, 20, 30, 40} {
		requireNoBuiltinError(t, PointerWriteBuiltin().Fn(pointer, NewInteger(int64(index)), &object.FixedInteger{Kind: object.FixedI32, Raw: uint64(expected)}))
	}
	if got := integerValueForTest(t, requireNoBuiltinError(t, PointerReadBuiltin().Fn(pointer, NewInteger(2)))); got != 30 {
		t.Fatalf("pointer read = %d, want 30", got)
	}
	borrowed := requireNoBuiltinError(t, BorrowPointerBuiltin(false).Fn(pointer)).(*object.Pointer)
	if _, ok := FreeBuiltin().Fn(pointer).(*object.Error); !ok {
		t.Fatal("free should reject active borrows")
	}
	if _, ok := MovePointerBuiltin().Fn(pointer).(*object.Error); !ok {
		t.Fatal("move should reject active borrows")
	}
	requireNoBuiltinError(t, ReleaseBorrowBuiltin().Fn(borrowed))
	requireNoBuiltinError(t, ReallocBuiltin().Fn(pointer, NewInteger(8)))
	if pointer.Length != 8 {
		t.Fatalf("reallocated length = %d, want 8", pointer.Length)
	}
	moved := requireNoBuiltinError(t, MovePointerBuiltin().Fn(pointer)).(*object.Pointer)
	if requireNoBuiltinError(t, PointerIsValidBuiltin().Fn(pointer)).(*object.Boolean).Value {
		t.Fatal("moved-from pointer should be invalid")
	}
	requireNoBuiltinError(t, FreeBuiltin().Fn(moved))
	if _, ok := FreeBuiltin().Fn(moved).(*object.Error); !ok {
		t.Fatal("second free should be reported")
	}
	stats := object.MemoryStatsSnapshot()
	if stats["active_blocks"] != 0 || stats["double_frees"] == 0 {
		t.Fatalf("unexpected memory stats: %#v", stats)
	}
}

func TestZ17RawPointerViewAndPointerRelations(t *testing.T) {
	data := []byte{4, 5, 6}
	address := uint64(uintptr(unsafe.Pointer(&data[0])))
	raw := requireNoBuiltinError(t, PointerFromAddressBuiltin().Fn(
		NewString("u8"), &object.FixedInteger{Kind: object.FixedU64, Raw: address}, NewInteger(3), NewBoolean(true),
	)).(*object.Pointer)
	if got := integerValueForTest(t, requireNoBuiltinError(t, PointerReadBuiltin().Fn(raw, NewInteger(1)))); got != 5 {
		t.Fatalf("raw read = %d, want 5", got)
	}
	requireNoBuiltinError(t, PointerWriteBuiltin().Fn(raw, NewInteger(1), &object.FixedInteger{Kind: object.FixedU8, Raw: 9}))
	if data[1] != 9 {
		t.Fatalf("raw write did not reach source slice: %v", data)
	}
	addressView := requireNoBuiltinError(t, AddressOfBuiltin().Fn(raw, NewInteger(1))).(*object.Pointer)
	if got := integerValueForTest(t, requireNoBuiltinError(t, PointerReadBuiltin().Fn(addressView))); got != 9 {
		t.Fatalf("raw addressOf read = %d, want 9", got)
	}
	offset := requireNoBuiltinError(t, PointerOffsetBuiltin().Fn(raw, NewInteger(1))).(*object.Pointer)
	if got := integerValueForTest(t, requireNoBuiltinError(t, PointerByteLengthBuiltin().Fn(offset))); got != 2 {
		t.Fatalf("raw pointer byte length = %d, want 2", got)
	}
	if !requireNoBuiltinError(t, PointerEqualBuiltin().Fn(offset, requireNoBuiltinError(t, PointerFromAddressBuiltin().Fn(
		NewString("u8"), &object.FixedInteger{Kind: object.FixedU64, Raw: address + 1}, NewInteger(2), NewBoolean(false),
	)))).(*object.Boolean).Value {
		t.Fatal("equivalent raw pointer addresses should compare equal")
	}
	if got := integerValueForTest(t, requireNoBuiltinError(t, PointerCompareBuiltin().Fn(raw, offset))); got >= 0 {
		t.Fatalf("pointer comparison = %d, want negative", got)
	}
}

func TestZ17ArenaInvalidatesPointersAndUpdatesDiagnostics(t *testing.T) {
	requireNoBuiltinError(t, MemoryResetStatsBuiltin().Fn())
	arena := requireNoBuiltinError(t, ArenaCreateBuiltin().Fn()).(*object.MemoryArena)
	pointer := requireNoBuiltinError(t, ArenaAllocBuiltin().Fn(arena, NewString("u16"), NewInteger(3))).(*object.Pointer)
	requireNoBuiltinError(t, PointerWriteBuiltin().Fn(pointer, NewInteger(0), &object.FixedInteger{Kind: object.FixedU16, Raw: 11}))
	requireNoBuiltinError(t, ArenaResetBuiltin().Fn(arena))
	if _, ok := PointerReadBuiltin().Fn(pointer).(*object.Error); !ok {
		t.Fatal("arena pointer should be invalid after reset")
	}
	stats := object.MemoryStatsSnapshot()
	if stats["active_blocks"] != 0 || stats["frees"] == 0 {
		t.Fatalf("arena reset did not update diagnostics: %#v", stats)
	}
	requireNoBuiltinError(t, ArenaFreeBuiltin().Fn(arena))
}

func TestZ17NativeLayoutAndAtomicPointers(t *testing.T) {
	fields := &object.Array{Elements: []object.Object{
		dictFromObjects(map[string]object.Object{"name": NewString("tag"), "type": NewString("u8")}),
		dictFromObjects(map[string]object.Object{"name": NewString("value"), "type": NewString("u64")}),
	}}
	layout := requireNoBuiltinError(t, NativeStructLayoutBuiltin().Fn(fields)).(*object.Dict)
	if got := integerValueForTest(t, dictGet(layout, "size")); got != 16 {
		t.Fatalf("layout size = %d, want 16", got)
	}
	pointer := requireNoBuiltinError(t, AllocBuiltin().Fn(NewString("u64"), NewInteger(1))).(*object.Pointer)
	requireNoBuiltinError(t, AtomicPointerStoreBuiltin().Fn(pointer, &object.FixedInteger{Kind: object.FixedU64, Raw: 4}))
	if got := integerValueForTest(t, requireNoBuiltinError(t, AtomicPointerAddBuiltin().Fn(pointer, NewInteger(3)))); got != 7 {
		t.Fatalf("atomic add = %d, want 7", got)
	}
	if !requireNoBuiltinError(t, AtomicPointerCompareSwapBuiltin().Fn(pointer, NewInteger(7), NewInteger(9))).(*object.Boolean).Value {
		t.Fatal("atomic compare/swap should succeed")
	}
	if got := integerValueForTest(t, requireNoBuiltinError(t, AtomicPointerLoadBuiltin().Fn(pointer))); got != 9 {
		t.Fatalf("atomic load = %d, want 9", got)
	}
	requireNoBuiltinError(t, FreeBuiltin().Fn(pointer))
}

func TestZ17MappedAndSharedMemory(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-specific systems integration")
	}
	path := filepath.Join(t.TempDir(), "mapped.bin")
	mapped := requireNoBuiltinError(t, MmapOpenBuiltin().Fn(NewString(path), NewString("readwrite"), NewInteger(8))).(*object.MappedMemory)
	mappedPointer := requireNoBuiltinError(t, MmapPointerBuiltin().Fn(mapped)).(*object.Pointer)
	requireNoBuiltinError(t, PointerWriteBuiltin().Fn(mappedPointer, NewInteger(0), &object.FixedInteger{Kind: object.FixedU8, Raw: 42}))
	requireNoBuiltinError(t, MmapFlushBuiltin().Fn(mapped))
	requireNoBuiltinError(t, MmapCloseBuiltin().Fn(mapped))
	content, err := os.ReadFile(path)
	if err != nil || len(content) != 8 || content[0] != 42 {
		t.Fatalf("mapped file content = %v, err=%v", content, err)
	}

	name := fmt.Sprintf("zumbra-z17-%d", os.Getpid())
	shared := requireNoBuiltinError(t, SharedMemoryOpenBuiltin().Fn(NewString(name), NewInteger(8), NewBoolean(true))).(*object.SharedMemory)
	sharedPointer := requireNoBuiltinError(t, SharedMemoryPointerBuiltin().Fn(shared)).(*object.Pointer)
	requireNoBuiltinError(t, PointerWriteBuiltin().Fn(sharedPointer, NewInteger(1), &object.FixedInteger{Kind: object.FixedU8, Raw: 77}))
	requireNoBuiltinError(t, SharedMemoryCloseBuiltin().Fn(shared))
	requireNoBuiltinError(t, SharedMemoryUnlinkBuiltin().Fn(NewString(name)))
}

func TestZ17SystemInformationAndProfiling(t *testing.T) {
	info := requireNoBuiltinError(t, SystemInfoBuiltin().Fn()).(*object.Dict)
	if dictGet(info, "os") == nil || integerValueForTest(t, dictGet(info, "pointer_bits")) == 0 {
		t.Fatalf("incomplete system info: %s", info.Inspect())
	}
	start := requireNoBuiltinError(t, ProfileNowBuiltin().Fn())
	elapsed := requireNoBuiltinError(t, ProfileElapsedBuiltin().Fn(start))
	if integerValueForTest(t, elapsed) < 0 {
		t.Fatal("elapsed time cannot be negative")
	}
}

func TestZ17DynamicLibraryCall(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-specific dynamic loading integration")
	}
	library := requireNoBuiltinError(t, DynamicOpenBuiltin().Fn(NewString("libc.so.6"))).(*object.DynamicLibrary)
	symbol := requireNoBuiltinError(t, DynamicSymbolBuiltin().Fn(library, NewString("getpid"))).(*object.Pointer)
	result := requireNoBuiltinError(t, DynamicCallBuiltin().Fn(symbol, NewString("i32"), &object.Array{}))
	if integerValueForTest(t, result) <= 0 {
		t.Fatalf("getpid returned %s", result.Inspect())
	}
	requireNoBuiltinError(t, DynamicCloseBuiltin().Fn(library))
}
