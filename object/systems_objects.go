package object

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	POINTER_OBJ         ObjectType = "POINTER"
	MEMORY_ARENA_OBJ    ObjectType = "MEMORY_ARENA"
	MAPPED_MEMORY_OBJ   ObjectType = "MAPPED_MEMORY"
	SHARED_MEMORY_OBJ   ObjectType = "SHARED_MEMORY"
	DYNAMIC_LIBRARY_OBJ ObjectType = "DYNAMIC_LIBRARY"
)

// SystemType describes the in-memory ABI representation used by Z17 pointers.
type SystemType struct {
	Name   string
	Size   int
	Align  int
	Kind   FixedIntegerKind
	Signed bool
	Float  bool
	Bool   bool
	Ptr    bool
}

func ParseSystemType(name string) (SystemType, bool) {
	switch name {
	case "u8":
		return SystemType{Name: name, Size: 1, Align: 1, Kind: FixedU8}, true
	case "i8":
		return SystemType{Name: name, Size: 1, Align: 1, Kind: FixedI8, Signed: true}, true
	case "u16":
		return SystemType{Name: name, Size: 2, Align: 2, Kind: FixedU16}, true
	case "i16":
		return SystemType{Name: name, Size: 2, Align: 2, Kind: FixedI16, Signed: true}, true
	case "u32":
		return SystemType{Name: name, Size: 4, Align: 4, Kind: FixedU32}, true
	case "i32":
		return SystemType{Name: name, Size: 4, Align: 4, Kind: FixedI32, Signed: true}, true
	case "u64":
		return SystemType{Name: name, Size: 8, Align: 8, Kind: FixedU64}, true
	case "i64":
		return SystemType{Name: name, Size: 8, Align: 8, Kind: FixedI64, Signed: true}, true
	case "int":
		return SystemType{Name: name, Size: 8, Align: 8, Signed: true}, true
	case "float":
		return SystemType{Name: name, Size: 8, Align: 8, Float: true}, true
	case "bool":
		return SystemType{Name: name, Size: 1, Align: 1, Bool: true}, true
	case "ptr":
		size := int(unsafe.Sizeof(uintptr(0)))
		return SystemType{Name: name, Size: size, Align: size, Ptr: true}, true
	default:
		return SystemType{}, false
	}
}

type MemoryBlock struct {
	Mu            sync.RWMutex
	ID            uint64
	Data          []byte
	Type          SystemType
	Length        int
	Freed         bool
	Allocator     string
	BorrowCount   int
	MutableBorrow bool
	Backing       Object
	Raw           unsafe.Pointer
}

type Pointer struct {
	Block    *MemoryBlock
	Offset   int
	Length   int
	Owned    bool
	Mutable  bool
	Borrowed bool
	Released bool
	Moved    bool
	Raw      unsafe.Pointer
	RawType  SystemType
}

func (p *Pointer) Type() ObjectType { return POINTER_OBJ }
func (p *Pointer) Inspect() string {
	if p == nil || (p.Block == nil && p.Raw == nil) {
		return "Pointer<unknown>(null)"
	}
	typeName := p.ElementType().Name
	if typeName == "" {
		typeName = "unknown"
	}
	return fmt.Sprintf("Pointer<%s>(0x%x)", typeName, p.Address())
}

func (p *Pointer) ElementType() SystemType {
	if p == nil {
		return SystemType{}
	}
	if p.Block != nil {
		return p.Block.Type
	}
	return p.RawType
}

func (p *Pointer) Address() uintptr {
	if p == nil {
		return 0
	}
	if p.Block != nil {
		p.Block.Mu.RLock()
		defer p.Block.Mu.RUnlock()
		if p.Block.Freed || len(p.Block.Data) == 0 {
			return 0
		}
		base := unsafe.Pointer(&p.Block.Data[0])
		return uintptr(base) + uintptr(p.Offset*p.Block.Type.Size)
	}
	return uintptr(p.Raw)
}

func (p *Pointer) IsNull() bool { return p == nil || (p.Block == nil && p.Raw == nil) }

func (p *Pointer) validateState(write bool, index int) error {
	if p == nil || p.IsNull() {
		memoryRegistry.invalidAccess()
		return fmt.Errorf("null pointer dereference")
	}
	if p.Moved {
		memoryRegistry.invalidAccess()
		return fmt.Errorf("pointer was moved")
	}
	if p.Released {
		memoryRegistry.invalidAccess()
		return fmt.Errorf("borrowed pointer was released")
	}
	if write && !p.Mutable {
		memoryRegistry.invalidAccess()
		return fmt.Errorf("cannot write through an immutable pointer")
	}
	if p.Block == nil && p.Raw == nil {
		return fmt.Errorf("opaque pointer is null")
	}
	if index < 0 || index >= p.Length {
		memoryRegistry.invalidAccess()
		return fmt.Errorf("pointer index out of bounds: %d (length %d)", index, p.Length)
	}
	return nil
}

func (p *Pointer) Read(index int) (Object, error) {
	if err := p.validateState(false, index); err != nil {
		return nil, err
	}
	if p.Block == nil {
		t := p.RawType
		start := uintptr(index * t.Size)
		data := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(p.Raw)+start)), t.Size)
		return decodeSystemValue(t, data), nil
	}
	p.Block.Mu.RLock()
	defer p.Block.Mu.RUnlock()
	if p.Block.Freed {
		memoryRegistry.invalidAccess()
		return nil, fmt.Errorf("use after free")
	}
	t := p.Block.Type
	start := (p.Offset + index) * t.Size
	if start < 0 || start+t.Size > len(p.Block.Data) {
		memoryRegistry.invalidAccess()
		return nil, fmt.Errorf("pointer range is outside the backing allocation")
	}
	return decodeSystemValue(t, p.Block.Data[start:start+t.Size]), nil
}

func (p *Pointer) Write(index int, value Object) error {
	if err := p.validateState(true, index); err != nil {
		return err
	}
	// Encode before taking the destination lock. This is important for Pointer<ptr>:
	// obtaining the source address may need to read the same allocation.
	t := p.ElementType()
	encoded := make([]byte, t.Size)
	if err := encodeSystemValue(t, encoded, value); err != nil {
		return err
	}
	if p.Block == nil {
		start := uintptr(index * t.Size)
		data := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(p.Raw)+start)), t.Size)
		copy(data, encoded)
		return nil
	}
	p.Block.Mu.Lock()
	defer p.Block.Mu.Unlock()
	if p.Block.Freed {
		memoryRegistry.invalidAccess()
		return fmt.Errorf("use after free")
	}
	start := (p.Offset + index) * t.Size
	if start < 0 || start+t.Size > len(p.Block.Data) {
		memoryRegistry.invalidAccess()
		return fmt.Errorf("pointer range is outside the backing allocation")
	}
	copy(p.Block.Data[start:start+t.Size], encoded)
	return nil
}

func decodeSystemValue(t SystemType, data []byte) Object {
	if t.Bool {
		return &Boolean{Value: len(data) > 0 && data[0] != 0}
	}
	if t.Float {
		return &Float{Value: math.Float64frombits(binary.NativeEndian.Uint64(data))}
	}
	if t.Ptr {
		var raw uint64
		if t.Size == 4 {
			raw = uint64(binary.NativeEndian.Uint32(data))
		} else {
			raw = binary.NativeEndian.Uint64(data)
		}
		return &Pointer{Raw: unsafe.Pointer(uintptr(raw)), RawType: t, Mutable: true}
	}
	var raw uint64
	switch t.Size {
	case 1:
		raw = uint64(data[0])
	case 2:
		raw = uint64(binary.NativeEndian.Uint16(data))
	case 4:
		raw = uint64(binary.NativeEndian.Uint32(data))
	case 8:
		raw = binary.NativeEndian.Uint64(data)
	}
	if t.Kind != "" {
		return NewFixedIntegerRaw(t.Kind, raw)
	}
	return &Integer{Value: int64(raw)}
}

func encodeSystemValue(t SystemType, data []byte, value Object) error {
	if t.Bool {
		v, ok := value.(*Boolean)
		if !ok {
			return fmt.Errorf("expected bool, got %s", value.Type())
		}
		if v.Value {
			data[0] = 1
		} else {
			data[0] = 0
		}
		return nil
	}
	if t.Float {
		var f float64
		switch v := value.(type) {
		case *Float:
			f = v.Value
		case *Integer:
			f = float64(v.Value)
		case *FixedInteger:
			if v.Kind.Signed() {
				f = float64(v.SignedValue())
			} else {
				f = float64(v.UnsignedValue())
			}
		default:
			return fmt.Errorf("expected numeric value, got %s", value.Type())
		}
		binary.NativeEndian.PutUint64(data, math.Float64bits(f))
		return nil
	}
	if t.Ptr {
		v, ok := value.(*Pointer)
		if !ok {
			return fmt.Errorf("expected pointer, got %s", value.Type())
		}
		raw := uint64(v.Address())
		if t.Size == 4 {
			binary.NativeEndian.PutUint32(data, uint32(raw))
		} else {
			binary.NativeEndian.PutUint64(data, raw)
		}
		return nil
	}
	var raw uint64
	switch v := value.(type) {
	case *Integer:
		raw = uint64(v.Value)
	case *FixedInteger:
		raw = v.UnsignedValue()
	default:
		return fmt.Errorf("expected integer value, got %s", value.Type())
	}
	switch t.Size {
	case 1:
		data[0] = byte(raw)
	case 2:
		binary.NativeEndian.PutUint16(data, uint16(raw))
	case 4:
		binary.NativeEndian.PutUint32(data, uint32(raw))
	case 8:
		binary.NativeEndian.PutUint64(data, raw)
	}
	return nil
}

type MemoryArena struct {
	Mu     sync.Mutex
	Blocks []*MemoryBlock
	Closed bool
	Bytes  int64
}

func (a *MemoryArena) Type() ObjectType { return MEMORY_ARENA_OBJ }
func (a *MemoryArena) Inspect() string {
	if a == nil {
		return "Arena(closed)"
	}
	a.Mu.Lock()
	defer a.Mu.Unlock()
	return fmt.Sprintf("Arena(bytes=%d, allocations=%d, closed=%t)", a.Bytes, len(a.Blocks), a.Closed)
}

type MappedMemory struct {
	Mu      sync.Mutex
	Path    string
	Mode    string
	Data    []byte
	Block   *MemoryBlock
	Closed  bool
	Private interface{}
}

func (m *MappedMemory) Type() ObjectType { return MAPPED_MEMORY_OBJ }
func (m *MappedMemory) Inspect() string {
	if m == nil {
		return "MappedMemory(closed)"
	}
	return fmt.Sprintf("MappedMemory(%s, %d bytes)", m.Path, len(m.Data))
}

type SharedMemory struct {
	Mu      sync.Mutex
	Name    string
	Data    []byte
	Block   *MemoryBlock
	Closed  bool
	Path    string
	Private interface{}
}

func (m *SharedMemory) Type() ObjectType { return SHARED_MEMORY_OBJ }
func (m *SharedMemory) Inspect() string {
	if m == nil {
		return "SharedMemory(closed)"
	}
	return fmt.Sprintf("SharedMemory(%s, %d bytes)", m.Name, len(m.Data))
}

type DynamicLibrary struct {
	Mu      sync.Mutex
	Path    string
	Handle  uintptr
	Closed  bool
	Private interface{}
	LastErr string
}

func (l *DynamicLibrary) Type() ObjectType { return DYNAMIC_LIBRARY_OBJ }
func (l *DynamicLibrary) Inspect() string {
	if l == nil {
		return "DynamicLibrary(closed)"
	}
	return fmt.Sprintf("DynamicLibrary(%s, closed=%t)", l.Path, l.Closed)
}

type memoryRegistryState struct {
	mu              sync.Mutex
	nextID          uint64
	blocks          map[uint64]*MemoryBlock
	totalAllocs     int64
	totalFrees      int64
	totalBytes      int64
	activeBytes     int64
	peakBytes       int64
	invalidAccesses int64
	doubleFree      int64
}

var memoryRegistry = &memoryRegistryState{blocks: map[uint64]*MemoryBlock{}}

func NewMemoryBlock(t SystemType, count int, allocator string, data []byte) *MemoryBlock {
	if data == nil {
		data = make([]byte, count*t.Size)
	}
	id := atomic.AddUint64(&memoryRegistry.nextID, 1)
	block := &MemoryBlock{ID: id, Data: data, Type: t, Length: count, Allocator: allocator}
	memoryRegistry.mu.Lock()
	memoryRegistry.blocks[id] = block
	memoryRegistry.totalAllocs++
	bytes := int64(len(data))
	memoryRegistry.totalBytes += bytes
	memoryRegistry.activeBytes += bytes
	if memoryRegistry.activeBytes > memoryRegistry.peakBytes {
		memoryRegistry.peakBytes = memoryRegistry.activeBytes
	}
	memoryRegistry.mu.Unlock()
	return block
}

func NewManagedPointer(t SystemType, count int, allocator string) *Pointer {
	block := NewMemoryBlock(t, count, allocator, nil)
	return &Pointer{Block: block, Length: count, Owned: allocator == "heap", Mutable: true}
}

func NewNullPointer(t SystemType) *Pointer { return &Pointer{RawType: t, Mutable: true} }

func (r *memoryRegistryState) invalidAccess() {
	r.mu.Lock()
	r.invalidAccesses++
	r.mu.Unlock()
}

func (r *memoryRegistryState) markFreed(block *MemoryBlock) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if block.Freed {
		r.doubleFree++
		return
	}
	r.totalFrees++
	r.activeBytes -= int64(len(block.Data))
	delete(r.blocks, block.ID)
}

func MemoryStatsSnapshot() map[string]int64 {
	memoryRegistry.mu.Lock()
	defer memoryRegistry.mu.Unlock()
	return map[string]int64{
		"allocations":      memoryRegistry.totalAllocs,
		"frees":            memoryRegistry.totalFrees,
		"active_blocks":    int64(len(memoryRegistry.blocks)),
		"total_bytes":      memoryRegistry.totalBytes,
		"active_bytes":     memoryRegistry.activeBytes,
		"peak_bytes":       memoryRegistry.peakBytes,
		"invalid_accesses": memoryRegistry.invalidAccesses,
		"double_frees":     memoryRegistry.doubleFree,
	}
}

func MemoryLeakBlocks() []*MemoryBlock {
	memoryRegistry.mu.Lock()
	defer memoryRegistry.mu.Unlock()
	result := make([]*MemoryBlock, 0, len(memoryRegistry.blocks))
	for _, block := range memoryRegistry.blocks {
		if block.Allocator == "heap" && !block.Freed {
			result = append(result, block)
		}
	}
	return result
}

func ResetMemoryStats() {
	memoryRegistry.mu.Lock()
	defer memoryRegistry.mu.Unlock()
	memoryRegistry.totalAllocs = 0
	memoryRegistry.totalFrees = 0
	memoryRegistry.totalBytes = 0
	memoryRegistry.activeBytes = 0
	memoryRegistry.peakBytes = 0
	memoryRegistry.invalidAccesses = 0
	memoryRegistry.doubleFree = 0
	for _, block := range memoryRegistry.blocks {
		if !block.Freed {
			bytes := int64(len(block.Data))
			memoryRegistry.activeBytes += bytes
			memoryRegistry.totalBytes += bytes
			memoryRegistry.totalAllocs++
		}
	}
	memoryRegistry.peakBytes = memoryRegistry.activeBytes
}

// ReleaseMemoryBlock invalidates a registry-backed non-heap allocation, such
// as an arena allocation. It is idempotent and updates memory diagnostics.
func ReleaseMemoryBlock(block *MemoryBlock) bool {
	if block == nil {
		return false
	}
	block.Mu.Lock()
	defer block.Mu.Unlock()
	if block.Freed {
		return false
	}
	memoryRegistry.markFreed(block)
	block.Freed = true
	block.Data = nil
	block.Length = 0
	return true
}

func FreeMemoryBlock(pointer *Pointer) error {
	if pointer == nil || pointer.IsNull() {
		return fmt.Errorf("cannot free a null pointer")
	}
	if pointer.Moved {
		return fmt.Errorf("pointer was moved")
	}
	if pointer.Borrowed {
		return fmt.Errorf("borrowed pointers cannot be freed")
	}
	if !pointer.Owned || pointer.Block == nil || pointer.Block.Allocator != "heap" || pointer.Offset != 0 {
		return fmt.Errorf("pointer does not own a heap allocation")
	}
	block := pointer.Block
	block.Mu.Lock()
	defer block.Mu.Unlock()
	if block.Freed {
		memoryRegistry.mu.Lock()
		memoryRegistry.doubleFree++
		memoryRegistry.mu.Unlock()
		return fmt.Errorf("double free detected")
	}
	if block.BorrowCount != 0 || block.MutableBorrow {
		return fmt.Errorf("cannot free memory while borrows are active")
	}
	memoryRegistry.markFreed(block)
	block.Freed = true
	block.Data = nil
	block.Length = 0
	pointer.Released = true
	pointer.Length = 0
	return nil
}

func ReallocateMemoryBlock(pointer *Pointer, count int) error {
	if pointer == nil || pointer.IsNull() {
		return fmt.Errorf("cannot realloc a null pointer")
	}
	if count < 0 {
		return fmt.Errorf("allocation count must be non-negative")
	}
	if pointer.Moved || pointer.Released {
		return fmt.Errorf("pointer is no longer valid")
	}
	if !pointer.Owned || pointer.Block == nil || pointer.Block.Allocator != "heap" || pointer.Offset != 0 {
		return fmt.Errorf("realloc expects an owning heap pointer")
	}
	block := pointer.Block
	block.Mu.Lock()
	defer block.Mu.Unlock()
	if block.Freed {
		memoryRegistry.invalidAccess()
		return fmt.Errorf("use after free")
	}
	if block.BorrowCount != 0 || block.MutableBorrow {
		return fmt.Errorf("cannot realloc memory while borrows are active")
	}
	oldBytes := len(block.Data)
	newBytes := count * block.Type.Size
	data := make([]byte, newBytes)
	copy(data, block.Data)
	block.Data = data
	block.Length = count
	pointer.Length = count
	memoryRegistry.mu.Lock()
	delta := int64(newBytes - oldBytes)
	memoryRegistry.totalBytes += maxInt64(delta, 0)
	memoryRegistry.activeBytes += delta
	if memoryRegistry.activeBytes > memoryRegistry.peakBytes {
		memoryRegistry.peakBytes = memoryRegistry.activeBytes
	}
	memoryRegistry.mu.Unlock()
	return nil
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
