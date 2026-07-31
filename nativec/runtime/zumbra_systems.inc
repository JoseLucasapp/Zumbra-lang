#if defined(ZUMBRA_ENABLE_SYSTEMS)

#if defined(_WIN32)
#include <io.h>
#include <processthreadsapi.h>
#include <sysinfoapi.h>
#else
#include <dlfcn.h>
#include <fcntl.h>
#include <sys/mman.h>
#include <sys/stat.h>
#include <sys/syscall.h>
#include <unistd.h>
extern long syscall(long number, ...);
#endif

#include <limits.h>

typedef enum {
    ZMEM_HEAP = 1,
    ZMEM_ARENA,
    ZMEM_EXTERNAL,
    ZMEM_MAPPED,
    ZMEM_SHARED
} ZMemoryAllocator;

typedef struct ZMemoryBlockNative {
    uint64_t id;
    uint8_t *data;
    size_t length;
    size_t elem_size;
    ZKind elem_kind;
    bool freed;
    ZMemoryAllocator allocator;
    size_t borrows;
    bool mutable_borrow;
    pthread_mutex_t mutex;
    struct ZMemoryBlockNative *next;
} ZMemoryBlockNative;

typedef struct {
    ZMemoryBlockNative *block;
    size_t offset;
    size_t length;
    bool owned;
    bool mutable_value;
    bool borrowed;
    bool released;
    bool moved;
    void *raw;
    ZKind raw_kind;
} ZMemoryPointerNative;

typedef struct ZArenaLinkNative {
    ZMemoryBlockNative *block;
    struct ZArenaLinkNative *next;
} ZArenaLinkNative;

typedef struct {
    bool closed;
    size_t bytes;
    size_t allocations;
    ZArenaLinkNative *blocks;
} ZMemoryArenaNative;

typedef struct {
    bool closed;
    char *path;
    char *mode;
    size_t size;
    ZMemoryBlockNative *block;
#if defined(_WIN32)
    HANDLE file;
    HANDLE mapping;
#else
    int fd;
#endif
} ZMappedMemoryNative;

typedef struct {
    bool closed;
    char *name;
    size_t size;
    ZMemoryBlockNative *block;
#if defined(_WIN32)
    HANDLE mapping;
#else
    int fd;
#endif
} ZSharedMemoryNative;

typedef struct {
    bool closed;
    char *path;
    char *last_error;
#if defined(_WIN32)
    HMODULE handle;
#else
    void *handle;
#endif
} ZDynamicLibraryNative;

typedef struct {
    pthread_mutex_t mutex;
    ZMemoryBlockNative *blocks;
    uint64_t next_id;
    uint64_t allocations;
    uint64_t frees;
    uint64_t total_bytes;
    uint64_t active_bytes;
    uint64_t peak_bytes;
    uint64_t invalid_accesses;
    uint64_t double_frees;
} ZMemoryRegistryNative;

static ZMemoryRegistryNative z_memory_registry;

static const char *z_system_kind_name(ZKind kind) {
    switch (kind) {
    case ZK_U8: return "u8";
    case ZK_U16: return "u16";
    case ZK_U32: return "u32";
    case ZK_U64: return "u64";
    case ZK_I8: return "i8";
    case ZK_I16: return "i16";
    case ZK_I32: return "i32";
    case ZK_I64: return "i64";
    case ZK_INT: return "int";
    case ZK_FLOAT: return "float";
    case ZK_BOOL: return "bool";
    case ZK_POINTER: return "ptr";
    default: return "unknown";
    }
}

static bool z_system_type(const char *name, ZKind *kind, size_t *size, size_t *alignment) {
    if (name == NULL) return false;
    if (strcmp(name, "u8") == 0) { *kind = ZK_U8; *size = 1; *alignment = 1; return true; }
    if (strcmp(name, "i8") == 0) { *kind = ZK_I8; *size = 1; *alignment = 1; return true; }
    if (strcmp(name, "u16") == 0) { *kind = ZK_U16; *size = 2; *alignment = 2; return true; }
    if (strcmp(name, "i16") == 0) { *kind = ZK_I16; *size = 2; *alignment = 2; return true; }
    if (strcmp(name, "u32") == 0) { *kind = ZK_U32; *size = 4; *alignment = 4; return true; }
    if (strcmp(name, "i32") == 0) { *kind = ZK_I32; *size = 4; *alignment = 4; return true; }
    if (strcmp(name, "u64") == 0) { *kind = ZK_U64; *size = 8; *alignment = 8; return true; }
    if (strcmp(name, "i64") == 0) { *kind = ZK_I64; *size = 8; *alignment = 8; return true; }
    if (strcmp(name, "int") == 0) { *kind = ZK_INT; *size = sizeof(int64_t); *alignment = _Alignof(int64_t); return true; }
    if (strcmp(name, "float") == 0) { *kind = ZK_FLOAT; *size = sizeof(double); *alignment = _Alignof(double); return true; }
    if (strcmp(name, "bool") == 0) { *kind = ZK_BOOL; *size = sizeof(uint8_t); *alignment = _Alignof(uint8_t); return true; }
    if (strcmp(name, "ptr") == 0) { *kind = ZK_POINTER; *size = sizeof(void *); *alignment = _Alignof(void *); return true; }
    return false;
}

static size_t z_system_align_up(size_t value, size_t alignment) {
    if (alignment <= 1) return value;
    size_t remainder = value % alignment;
    return remainder == 0 ? value : value + alignment - remainder;
}

static void z_system_invalid_access(void) {
    pthread_mutex_lock(&z_memory_registry.mutex);
    z_memory_registry.invalid_accesses++;
    pthread_mutex_unlock(&z_memory_registry.mutex);
}

static ZMemoryBlockNative *z_system_block_new(ZKind kind, size_t elem_size, size_t length, ZMemoryAllocator allocator, uint8_t *data, bool count_stats) {
    ZMemoryBlockNative *block = (ZMemoryBlockNative *)calloc(1, sizeof(*block));
    if (block == NULL) z_fatal("out of memory allocating memory block");
    pthread_mutex_init(&block->mutex, NULL);
    block->elem_kind = kind;
    block->elem_size = elem_size;
    block->length = length;
    block->allocator = allocator;
    if (data == NULL) {
        size_t bytes = length * elem_size;
        block->data = (uint8_t *)calloc(bytes == 0 ? 1 : bytes, 1);
        if (block->data == NULL) { free(block); z_fatal("out of memory allocating %zu bytes", bytes); }
    } else {
        block->data = data;
    }
    pthread_mutex_lock(&z_memory_registry.mutex);
    block->id = ++z_memory_registry.next_id;
    block->next = z_memory_registry.blocks;
    z_memory_registry.blocks = block;
    if (count_stats) {
        uint64_t bytes = (uint64_t)(length * elem_size);
        z_memory_registry.allocations++;
        z_memory_registry.total_bytes += bytes;
        z_memory_registry.active_bytes += bytes;
        if (z_memory_registry.active_bytes > z_memory_registry.peak_bytes) z_memory_registry.peak_bytes = z_memory_registry.active_bytes;
    }
    pthread_mutex_unlock(&z_memory_registry.mutex);
    return block;
}

static ZMemoryPointerNative *z_system_pointer_object(ZMemoryBlockNative *block, size_t offset, size_t length, bool owned, bool mutable_value, bool borrowed) {
    ZMemoryPointerNative *pointer = (ZMemoryPointerNative *)z_alloc(sizeof(*pointer));
    pointer->block = block;
    pointer->offset = offset;
    pointer->length = length;
    pointer->owned = owned;
    pointer->mutable_value = mutable_value;
    pointer->borrowed = borrowed;
    pointer->released = false;
    pointer->moved = false;
    pointer->raw = NULL;
    pointer->raw_kind = block == NULL ? ZK_UNKNOWN : block->elem_kind;
    return pointer;
}

static ZValue z_system_pointer_value(ZMemoryPointerNative *pointer) {
    ZValue result = z_value(ZV_MEMORY_POINTER, ZK_POINTER);
    result.as.p = pointer;
    return result;
}

static ZValue z_system_raw_pointer_view(void *raw, ZKind kind, size_t length, bool mutable_value) {
    ZMemoryPointerNative *pointer = z_system_pointer_object(NULL, 0, length, false, mutable_value, false);
    pointer->raw = raw;
    pointer->raw_kind = kind;
    return z_system_pointer_value(pointer);
}

static ZValue z_system_raw_pointer_value(void *raw, ZKind kind) {
    return z_system_raw_pointer_view(raw, kind, 0, false);
}

static ZMemoryPointerNative *z_system_pointer_subview(ZMemoryPointerNative *source, size_t offset, bool borrowed) {
    if (source == NULL || offset > source->length) z_fatal("pointer offset out of bounds");
    ZMemoryPointerNative *result = z_system_pointer_object(
        source->block,
        source->block == NULL ? 0 : source->offset + offset,
        source->length - offset,
        false,
        source->mutable_value,
        borrowed
    );
    if (source->block == NULL) {
        ZKind ignored = ZK_UNKNOWN;
        size_t elem_size = 0, alignment = 0;
        if (!z_system_type(z_system_kind_name(source->raw_kind), &ignored, &elem_size, &alignment)) {
            z_fatal("pointer offset uses unsupported raw pointer type");
        }
        result->raw = source->raw == NULL ? NULL : (void *)((uint8_t *)source->raw + offset * elem_size);
        result->raw_kind = source->raw_kind;
    }
    return result;
}

static ZMemoryPointerNative *z_system_expect_pointer(ZValue value, const char *label) {
    if (value.tag != ZV_MEMORY_POINTER || value.as.p == NULL) z_fatal("%s expects Pointer", label);
    return (ZMemoryPointerNative *)value.as.p;
}

static bool z_system_pointer_null(ZMemoryPointerNative *pointer) {
    return pointer == NULL || (pointer->block == NULL && pointer->raw == NULL);
}

static void z_system_pointer_validate(ZMemoryPointerNative *pointer, size_t index, bool write, const char *label) {
    if (z_system_pointer_null(pointer)) { z_system_invalid_access(); z_fatal("%s: null pointer dereference", label); }
    if (pointer->moved) { z_system_invalid_access(); z_fatal("%s: pointer was moved", label); }
    if (pointer->released) { z_system_invalid_access(); z_fatal("%s: borrowed pointer was released", label); }
    if (write && !pointer->mutable_value) { z_system_invalid_access(); z_fatal("%s: cannot write through immutable pointer", label); }
    if (pointer->block == NULL) {
        if (pointer->raw == NULL) { z_system_invalid_access(); z_fatal("%s: null pointer dereference", label); }
    } else if (pointer->block->freed) { z_system_invalid_access(); z_fatal("%s: use after free", label); }
    if (index >= pointer->length) { z_system_invalid_access(); z_fatal("%s: pointer index %zu out of bounds (length %zu)", label, index, pointer->length); }
}

static ZValue z_system_read_at(ZMemoryPointerNative *pointer, size_t index) {
    z_system_pointer_validate(pointer, index, false, "pointerRead");
    ZKind kind = pointer->block == NULL ? pointer->raw_kind : pointer->block->elem_kind;
    size_t elem_size = 0, alignment = 0; ZKind ignored = ZK_UNKNOWN;
    if (pointer->block != NULL) elem_size = pointer->block->elem_size;
    else if (!z_system_type(z_system_kind_name(kind), &ignored, &elem_size, &alignment)) z_fatal("pointerRead: unsupported raw pointer type");
    uint8_t *address = pointer->block == NULL ? (uint8_t *)pointer->raw + index * elem_size : pointer->block->data + (pointer->offset + index) * elem_size;
    if (pointer->block != NULL) pthread_mutex_lock(&pointer->block->mutex);
    ZValue result = z_null();
    switch (kind) {
    case ZK_FLOAT: { double value = 0; memcpy(&value, address, sizeof(value)); result = z_float(value); break; }
    case ZK_BOOL: result = z_bool(*address != 0); break;
    case ZK_POINTER: { void *raw = NULL; memcpy(&raw, address, sizeof(raw)); result = z_pointer(raw); break; }
    default: { uint64_t raw = 0; memcpy(&raw, address, elem_size); result = z_from_raw(raw, kind); break; }
    }
    if (pointer->block != NULL) pthread_mutex_unlock(&pointer->block->mutex);
    return result;
}

static void z_system_write_at(ZMemoryPointerNative *pointer, size_t index, ZValue value) {
    z_system_pointer_validate(pointer, index, true, "pointerWrite");
    ZKind kind = pointer->block == NULL ? pointer->raw_kind : pointer->block->elem_kind;
    size_t elem_size = 0, alignment = 0; ZKind ignored = ZK_UNKNOWN;
    if (pointer->block != NULL) elem_size = pointer->block->elem_size;
    else if (!z_system_type(z_system_kind_name(kind), &ignored, &elem_size, &alignment)) z_fatal("pointerWrite: unsupported raw pointer type");
    uint8_t *address = pointer->block == NULL ? (uint8_t *)pointer->raw + index * elem_size : pointer->block->data + (pointer->offset + index) * elem_size;
    if (pointer->block != NULL) pthread_mutex_lock(&pointer->block->mutex);
    switch (kind) {
    case ZK_FLOAT: { double converted = z_as_f64(value); memcpy(address, &converted, sizeof(converted)); break; }
    case ZK_BOOL: { uint8_t converted = z_as_bool(value) ? 1u : 0u; memcpy(address, &converted, 1); break; }
    case ZK_POINTER: { void *converted = z_as_pointer(value); memcpy(address, &converted, sizeof(converted)); break; }
    default: { ZValue converted = z_convert(value, kind); uint64_t raw = z_as_u64(converted); memcpy(address, &raw, elem_size); break; }
    }
    if (pointer->block != NULL) pthread_mutex_unlock(&pointer->block->mutex);
}

static void z_system_block_mark_freed(ZMemoryBlockNative *block, bool release_data) {
    if (block == NULL) return;
    pthread_mutex_lock(&block->mutex);
    if (block->freed) { pthread_mutex_unlock(&block->mutex); return; }
    size_t bytes = block->length * block->elem_size;
    if (release_data && block->data != NULL) free(block->data);
    block->data = NULL;
    block->length = 0;
    block->freed = true;
    pthread_mutex_unlock(&block->mutex);
    if (block->allocator == ZMEM_HEAP || block->allocator == ZMEM_ARENA) {
        pthread_mutex_lock(&z_memory_registry.mutex);
        z_memory_registry.frees++;
        if (z_memory_registry.active_bytes >= bytes) z_memory_registry.active_bytes -= bytes;
        pthread_mutex_unlock(&z_memory_registry.mutex);
    }
}

static ZValue z_system_dict(const char **keys, const ZValue *values, size_t count) {
    ZValue *pairs = count == 0 ? NULL : (ZValue *)z_alloc(sizeof(ZValue) * count);
    for (size_t index = 0; index < count; index++) pairs[index] = z_pair(z_string(keys[index]), values[index]);
    return z_dict_from(pairs, count);
}

static ZValue z_system_dict_get(ZValue dictionary, const char *key) {
    if (dictionary.tag != ZV_DICT) return z_null();
    for (size_t index = 0; index < dictionary.as.dict->len; index++) {
        ZValue current = dictionary.as.dict->keys[index];
        if (current.tag == ZV_STRING && strcmp(current.as.s, key) == 0) return dictionary.as.dict->values[index];
    }
    return z_null();
}

static void z_systems_runtime_init(void) {
    memset(&z_memory_registry, 0, sizeof(z_memory_registry));
    pthread_mutex_init(&z_memory_registry.mutex, NULL);
}

static void z_systems_runtime_shutdown(void) {
    pthread_mutex_lock(&z_memory_registry.mutex);
    ZMemoryBlockNative *block = z_memory_registry.blocks;
    z_memory_registry.blocks = NULL;
    pthread_mutex_unlock(&z_memory_registry.mutex);
    while (block != NULL) {
        ZMemoryBlockNative *next = block->next;
        if (!block->freed && block->data != NULL) {
            if (block->allocator == ZMEM_HEAP || block->allocator == ZMEM_ARENA) free(block->data);
#if !defined(_WIN32)
            else if (block->allocator == ZMEM_MAPPED || block->allocator == ZMEM_SHARED) munmap(block->data, block->length * block->elem_size);
#endif
        }
        pthread_mutex_destroy(&block->mutex);
        free(block);
        block = next;
    }
    pthread_mutex_destroy(&z_memory_registry.mutex);
}

static void *z_systems_as_pointer(ZValue value, bool *handled) {
    if (value.tag != ZV_MEMORY_POINTER) { *handled = false; return NULL; }
    *handled = true;
    ZMemoryPointerNative *pointer = (ZMemoryPointerNative *)value.as.p;
    if (z_system_pointer_null(pointer)) return NULL;
    if (pointer->block == NULL) return pointer->raw;
    if (pointer->block->freed) z_fatal("use after free");
    return pointer->block->data + pointer->offset * pointer->block->elem_size;
}

static ZValue z_systems_pointer_get(ZValue value, size_t index, bool *handled) {
    if (value.tag != ZV_MEMORY_POINTER) { *handled = false; return z_null(); }
    *handled = true;
    return z_system_read_at((ZMemoryPointerNative *)value.as.p, index);
}

static void z_systems_pointer_set(ZValue value, size_t index, ZValue item, bool *handled) {
    if (value.tag != ZV_MEMORY_POINTER) { *handled = false; return; }
    *handled = true;
    z_system_write_at((ZMemoryPointerNative *)value.as.p, index, item);
}

static size_t z_systems_pointer_size(ZValue value, bool *handled) {
    if (value.tag != ZV_MEMORY_POINTER) { *handled = false; return 0; }
    *handled = true;
    return ((ZMemoryPointerNative *)value.as.p)->length;
}

static void z_systems_show(ZValue value, bool *handled) {
    *handled = true;
    switch (value.tag) {
    case ZV_MEMORY_POINTER: {
        ZMemoryPointerNative *pointer = (ZMemoryPointerNative *)value.as.p;
        void *address = z_system_pointer_null(pointer) ? NULL : (pointer->block == NULL ? pointer->raw : pointer->block->data + pointer->offset * pointer->block->elem_size);
        fprintf(stdout, "Pointer<%s>(%p)", z_system_kind_name(pointer->block == NULL ? pointer->raw_kind : pointer->block->elem_kind), address);
        return;
    }
    case ZV_MEMORY_ARENA: fputs("MemoryArena", stdout); return;
    case ZV_MAPPED_MEMORY: fputs("MappedMemory", stdout); return;
    case ZV_SHARED_MEMORY: fputs("SharedMemory", stdout); return;
    case ZV_DYNAMIC_LIBRARY: fputs("DynamicLibrary", stdout); return;
    default: *handled = false; return;
    }
}

static ZValue z_systems_to_string(ZValue value, bool *handled) {
    char buffer[192];
    *handled = true;
    switch (value.tag) {
    case ZV_MEMORY_POINTER: {
        ZMemoryPointerNative *pointer = (ZMemoryPointerNative *)value.as.p;
        void *address = z_system_pointer_null(pointer) ? NULL : (pointer->block == NULL ? pointer->raw : pointer->block->data + pointer->offset * pointer->block->elem_size);
        snprintf(buffer, sizeof(buffer), "Pointer<%s>(%p)", z_system_kind_name(pointer->block == NULL ? pointer->raw_kind : pointer->block->elem_kind), address);
        return z_string(buffer);
    }
    case ZV_MEMORY_ARENA: return z_string("MemoryArena");
    case ZV_MAPPED_MEMORY: return z_string("MappedMemory");
    case ZV_SHARED_MEMORY: return z_string("SharedMemory");
    case ZV_DYNAMIC_LIBRARY: return z_string("DynamicLibrary");
    default: *handled = false; return z_null();
    }
}

static ZMemoryArenaNative *z_system_expect_arena(ZValue value, const char *label) {
    if (value.tag != ZV_MEMORY_ARENA || value.as.p == NULL) z_fatal("%s expects MemoryArena", label);
    return (ZMemoryArenaNative *)value.as.p;
}

static ZMappedMemoryNative *z_system_expect_mapping(ZValue value, const char *label) {
    if (value.tag != ZV_MAPPED_MEMORY || value.as.p == NULL) z_fatal("%s expects MappedMemory", label);
    return (ZMappedMemoryNative *)value.as.p;
}

static ZSharedMemoryNative *z_system_expect_shared(ZValue value, const char *label) {
    if (value.tag != ZV_SHARED_MEMORY || value.as.p == NULL) z_fatal("%s expects SharedMemory", label);
    return (ZSharedMemoryNative *)value.as.p;
}

static ZDynamicLibraryNative *z_system_expect_library(ZValue value, const char *label) {
    if (value.tag != ZV_DYNAMIC_LIBRARY || value.as.p == NULL) z_fatal("%s expects DynamicLibrary", label);
    return (ZDynamicLibraryNative *)value.as.p;
}

static ZValue z_system_alloc(const char *type_name, size_t count, ZMemoryAllocator allocator) {
    ZKind kind = ZK_UNKNOWN; size_t elem_size = 0, alignment = 0;
    if (!z_system_type(type_name, &kind, &elem_size, &alignment)) z_fatal("unsupported native type %s", type_name);
    (void)alignment;
    if (elem_size != 0 && count > SIZE_MAX / elem_size) z_fatal("allocation size overflows the native address space");
    ZMemoryBlockNative *block = z_system_block_new(kind, elem_size, count, allocator, NULL, true);
    return z_system_pointer_value(z_system_pointer_object(block, 0, count, allocator == ZMEM_HEAP, true, false));
}

static void z_system_close_mapping(ZMappedMemoryNative *mapping) {
    if (mapping == NULL || mapping->closed) return;
#if defined(_WIN32)
    if (mapping->block != NULL && mapping->block->data != NULL) UnmapViewOfFile(mapping->block->data);
    if (mapping->mapping != NULL) CloseHandle(mapping->mapping);
    if (mapping->file != INVALID_HANDLE_VALUE) CloseHandle(mapping->file);
#else
    if (mapping->block != NULL && mapping->block->data != NULL) {
        if (strcmp(mapping->mode, "read") != 0) msync(mapping->block->data, mapping->size, MS_SYNC);
        munmap(mapping->block->data, mapping->size);
    }
    if (mapping->fd >= 0) close(mapping->fd);
#endif
    if (mapping->block != NULL) { mapping->block->data = NULL; mapping->block->freed = true; mapping->block->length = 0; }
    mapping->closed = true;
}

static void z_system_close_shared(ZSharedMemoryNative *shared) {
    if (shared == NULL || shared->closed) return;
#if defined(_WIN32)
    if (shared->block != NULL && shared->block->data != NULL) UnmapViewOfFile(shared->block->data);
    if (shared->mapping != NULL) CloseHandle(shared->mapping);
#else
    if (shared->block != NULL && shared->block->data != NULL) { msync(shared->block->data, shared->size, MS_SYNC); munmap(shared->block->data, shared->size); }
    if (shared->fd >= 0) close(shared->fd);
#endif
    if (shared->block != NULL) { shared->block->data = NULL; shared->block->freed = true; shared->block->length = 0; }
    shared->closed = true;
}

typedef uintptr_t (*ZDynamicFunction0)(void);
typedef uintptr_t (*ZDynamicFunction1)(uintptr_t);
typedef uintptr_t (*ZDynamicFunction2)(uintptr_t, uintptr_t);
typedef uintptr_t (*ZDynamicFunction3)(uintptr_t, uintptr_t, uintptr_t);
typedef uintptr_t (*ZDynamicFunction4)(uintptr_t, uintptr_t, uintptr_t, uintptr_t);
typedef uintptr_t (*ZDynamicFunction5)(uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t);
typedef uintptr_t (*ZDynamicFunction6)(uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t);

static uintptr_t z_system_dynamic_argument(ZValue value) {
    if (value.tag == ZV_MEMORY_POINTER || value.tag == ZV_POINTER) return (uintptr_t)z_as_pointer(value);
    if (value.tag == ZV_BOOL) return value.as.b ? 1u : 0u;
    if (value.tag == ZV_INT || value.tag == ZV_UINT) return (uintptr_t)z_as_u64(value);
    z_fatal("dynamicCall arguments must be integer, bool or Pointer");
    return 0;
}

static uintptr_t z_system_dynamic_call(void *symbol, const uintptr_t *args, size_t argc) {
    switch (argc) {
    case 0: return ((ZDynamicFunction0)symbol)();
    case 1: return ((ZDynamicFunction1)symbol)(args[0]);
    case 2: return ((ZDynamicFunction2)symbol)(args[0], args[1]);
    case 3: return ((ZDynamicFunction3)symbol)(args[0], args[1], args[2]);
    case 4: return ((ZDynamicFunction4)symbol)(args[0], args[1], args[2], args[3]);
    case 5: return ((ZDynamicFunction5)symbol)(args[0], args[1], args[2], args[3], args[4]);
    case 6: return ((ZDynamicFunction6)symbol)(args[0], args[1], args[2], args[3], args[4], args[5]);
    default: z_fatal("dynamicCall supports at most 6 arguments");
    }
    return 0;
}

static uint64_t z_system_now_ns(void) {
#if defined(_WIN32)
    LARGE_INTEGER frequency, counter;
    QueryPerformanceFrequency(&frequency); QueryPerformanceCounter(&counter);
    return (uint64_t)((counter.QuadPart * 1000000000ull) / frequency.QuadPart);
#else
    struct timespec now; clock_gettime(CLOCK_MONOTONIC, &now);
    return (uint64_t)now.tv_sec * 1000000000ull + (uint64_t)now.tv_nsec;
#endif
}

static ZValue z_systems_call_builtin(const char *name, const ZValue *args, size_t argc, bool *handled) {
    *handled = true;
    if (strcmp(name, "alloc") == 0 || strcmp(name, "calloc") == 0) {
        z_expect_args(name, argc, 2); return z_system_alloc(z_as_cstring(args[0]), (size_t)z_as_u64(args[1]), ZMEM_HEAP);
    }
    if (strcmp(name, "nullPointer") == 0) {
        z_expect_args(name, argc, 1); ZKind kind; size_t size, alignment;
        if (!z_system_type(z_as_cstring(args[0]), &kind, &size, &alignment)) z_fatal("nullPointer uses unsupported native type");
        return z_system_raw_pointer_value(NULL, kind);
    }
    if (strcmp(name, "pointerFromAddress") == 0) {
        if (argc < 3 || argc > 4) z_fatal("pointerFromAddress expects 3 or 4 arguments");
        ZKind kind; size_t size, alignment; if (!z_system_type(z_as_cstring(args[0]), &kind, &size, &alignment)) z_fatal("pointerFromAddress uses unsupported native type");
        uintptr_t address = (uintptr_t)z_as_u64(args[1]); size_t length = (size_t)z_as_u64(args[2]); bool mutable_value = argc == 4 ? z_as_bool(args[3]) : false;
        return z_system_raw_pointer_view((void *)address, kind, length, mutable_value);
    }
    if (strcmp(name, "realloc") == 0) {
        z_expect_args(name, argc, 2); ZMemoryPointerNative *pointer = z_system_expect_pointer(args[0], name); size_t count = (size_t)z_as_u64(args[1]);
        if (!pointer->owned || pointer->block == NULL || pointer->block->allocator != ZMEM_HEAP || pointer->offset != 0) z_fatal("realloc expects owning heap pointer");
        ZMemoryBlockNative *block = pointer->block; pthread_mutex_lock(&block->mutex);
        if (block->freed) { pthread_mutex_unlock(&block->mutex); z_fatal("realloc: use after free"); }
        if (block->borrows != 0 || block->mutable_borrow) { pthread_mutex_unlock(&block->mutex); z_fatal("realloc: active borrows"); }
        if (block->elem_size != 0 && count > SIZE_MAX / block->elem_size) { pthread_mutex_unlock(&block->mutex); z_fatal("realloc size overflows the native address space"); }
        size_t old_bytes = block->length * block->elem_size, new_bytes = count * block->elem_size;
        uint8_t *data = (uint8_t *)realloc(block->data, new_bytes == 0 ? 1 : new_bytes); if (data == NULL) { pthread_mutex_unlock(&block->mutex); z_fatal("realloc failed"); }
        if (new_bytes > old_bytes) memset(data + old_bytes, 0, new_bytes - old_bytes);
        block->data = data; block->length = count; pointer->length = count; pthread_mutex_unlock(&block->mutex);
        pthread_mutex_lock(&z_memory_registry.mutex); if (new_bytes > old_bytes) z_memory_registry.total_bytes += new_bytes - old_bytes;
        if (new_bytes >= old_bytes) z_memory_registry.active_bytes += new_bytes - old_bytes; else z_memory_registry.active_bytes -= old_bytes - new_bytes;
        if (z_memory_registry.active_bytes > z_memory_registry.peak_bytes) {
            z_memory_registry.peak_bytes = z_memory_registry.active_bytes;
        }
        pthread_mutex_unlock(&z_memory_registry.mutex);
        return args[0];
    }
    if (strcmp(name, "free") == 0) {
        z_expect_args(name, argc, 1); ZMemoryPointerNative *pointer = z_system_expect_pointer(args[0], name);
        if (z_system_pointer_null(pointer)) z_fatal("free: null pointer");
        if (pointer->borrowed) z_fatal("free: borrowed pointer cannot be freed");
        if (!pointer->owned || pointer->block == NULL || pointer->block->allocator != ZMEM_HEAP || pointer->offset != 0) z_fatal("free: pointer does not own heap memory");
        ZMemoryBlockNative *block = pointer->block; pthread_mutex_lock(&block->mutex);
        if (block->freed) { pthread_mutex_unlock(&block->mutex); pthread_mutex_lock(&z_memory_registry.mutex); z_memory_registry.double_frees++; pthread_mutex_unlock(&z_memory_registry.mutex); z_fatal("free: double free detected"); }
        if (block->borrows != 0 || block->mutable_borrow) { pthread_mutex_unlock(&block->mutex); z_fatal("free: active borrows"); }
        size_t bytes = block->length * block->elem_size; free(block->data); block->data = NULL; block->length = 0; block->freed = true; pthread_mutex_unlock(&block->mutex);
        pthread_mutex_lock(&z_memory_registry.mutex); z_memory_registry.frees++; if (z_memory_registry.active_bytes >= bytes) z_memory_registry.active_bytes -= bytes; pthread_mutex_unlock(&z_memory_registry.mutex);
        pointer->released = true; pointer->length = 0; return z_bool(true);
    }
    if (strcmp(name, "addressOf") == 0) {
        if (argc < 1 || argc > 2) {
            z_fatal("addressOf expects 1 or 2 arguments");
        }
        size_t index = argc == 2 ? (size_t)z_as_u64(args[1]) : 0;
        if (args[0].tag == ZV_MEMORY_POINTER) {
            ZMemoryPointerNative *source = z_system_expect_pointer(args[0], name); if (index > source->length) z_fatal("addressOf index out of bounds");
            return z_system_pointer_value(z_system_pointer_subview(source, index, true));
        }
        if (args[0].tag == ZV_BUFFER) {
            ZBuffer *buffer = args[0].as.buffer; if (index > buffer->len) z_fatal("addressOf index out of bounds");
            ZMemoryBlockNative *block = z_system_block_new(buffer->elem_kind, buffer->elem_size, buffer->len, ZMEM_EXTERNAL, (uint8_t *)buffer->data, false);
            return z_system_pointer_value(z_system_pointer_object(block, index, buffer->len - index, false, true, true));
        }
        z_fatal("addressOf expects compact buffer or Pointer");
    }
    if (strcmp(name, "dereference") == 0 || strcmp(name, "pointerRead") == 0 || strcmp(name, "volatileRead") == 0) {
        if (argc < 1 || argc > 2) {
            z_fatal("%s expects 1 or 2 arguments", name);
        }
        return z_system_read_at(z_system_expect_pointer(args[0], name), argc == 2 ? (size_t)z_as_u64(args[1]) : 0);
    }
    if (strcmp(name, "pointerWrite") == 0 || strcmp(name, "volatileWrite") == 0) {
        if (argc != 2 && argc != 3) {
            z_fatal("%s expects 2 or 3 arguments", name);
        }
        size_t index = argc == 3 ? (size_t)z_as_u64(args[1]) : 0;
        ZValue item = args[argc == 3 ? 2 : 1];
        z_system_write_at(z_system_expect_pointer(args[0], name), index, item); return args[0];
    }
    if (strcmp(name, "pointerOffset") == 0) {
        z_expect_args(name, argc, 2); ZMemoryPointerNative *source = z_system_expect_pointer(args[0], name); size_t offset = (size_t)z_as_u64(args[1]); if (offset > source->length) z_fatal("pointerOffset out of bounds");
        return z_system_pointer_value(z_system_pointer_subview(source, offset, true));
    }
    if (strcmp(name, "pointerLength") == 0 || strcmp(name, "pointerByteLength") == 0) {
        z_expect_args(name, argc, 1); ZMemoryPointerNative *pointer = z_system_expect_pointer(args[0], name); size_t value = pointer->length;
        if (strcmp(name, "pointerByteLength") == 0) {
            if (pointer->block != NULL) value *= pointer->block->elem_size;
            else { ZKind ignored = ZK_UNKNOWN; size_t elem_size = 0, alignment = 0; if (!z_system_type(z_system_kind_name(pointer->raw_kind), &ignored, &elem_size, &alignment)) z_fatal("pointerByteLength uses unsupported raw pointer type"); value *= elem_size; }
        }
        return z_int((int64_t)value);
    }
    if (strcmp(name, "pointerType") == 0) { z_expect_args(name, argc, 1); ZMemoryPointerNative *p = z_system_expect_pointer(args[0], name); return z_string(z_system_kind_name(p->block == NULL ? p->raw_kind : p->block->elem_kind)); }
    if (strcmp(name, "pointerAddress") == 0) { z_expect_args(name, argc, 1); bool h = false; return z_uint((uint64_t)(uintptr_t)z_systems_as_pointer(args[0], &h), ZK_U64); }
    if (strcmp(name, "pointerEqual") == 0 || strcmp(name, "pointerCompare") == 0) { z_expect_args(name, argc, 2); bool h1=false,h2=false; uintptr_t left=(uintptr_t)z_systems_as_pointer(args[0],&h1),right=(uintptr_t)z_systems_as_pointer(args[1],&h2); if(!h1||!h2)z_fatal("%s expects pointers",name); if(strcmp(name,"pointerEqual")==0)return z_bool(left==right); return z_int(left<right?-1:(left>right?1:0)); }
    if (strcmp(name, "pointerIsAligned") == 0) { z_expect_args(name,argc,2);bool h=false;uintptr_t address=(uintptr_t)z_systems_as_pointer(args[0],&h);uint64_t alignment=z_as_u64(args[1]);if(!h||alignment==0||(alignment&(alignment-1))!=0)z_fatal("pointerIsAligned requires pointer and power-of-two alignment");return z_bool(address%alignment==0); }
    if (strcmp(name, "pointerIsNull") == 0) { z_expect_args(name, argc, 1); return z_bool(z_system_pointer_null(z_system_expect_pointer(args[0], name))); }
    if (strcmp(name, "pointerIsValid") == 0) { z_expect_args(name, argc, 1); ZMemoryPointerNative *p = z_system_expect_pointer(args[0], name); return z_bool(!z_system_pointer_null(p) && !p->moved && !p->released && (p->block == NULL || !p->block->freed)); }
    if (strcmp(name, "pointerOwned") == 0 || strcmp(name, "pointerBorrowed") == 0 || strcmp(name, "pointerMutable") == 0) {
        z_expect_args(name, argc, 1); ZMemoryPointerNative *p = z_system_expect_pointer(args[0], name);
        if (strcmp(name, "pointerOwned") == 0) return z_bool(p->owned);
        if (strcmp(name, "pointerBorrowed") == 0) return z_bool(p->borrowed);
        return z_bool(p->mutable_value);
    }
    if (strcmp(name, "borrowPointer") == 0 || strcmp(name, "borrowPointerMut") == 0) {
        z_expect_args(name, argc, 1); ZMemoryPointerNative *source = z_system_expect_pointer(args[0], name); bool mutable_value = strcmp(name, "borrowPointerMut") == 0;
        if (source->block == NULL || source->block->freed) {
            z_fatal("%s expects active managed pointer", name);
        }
        pthread_mutex_lock(&source->block->mutex);
        if (mutable_value) { if (!source->mutable_value || source->block->mutable_borrow || source->block->borrows != 0) { pthread_mutex_unlock(&source->block->mutex); z_fatal("borrowPointerMut requires exclusive access"); } source->block->mutable_borrow = true; }
        else { if (source->block->mutable_borrow) { pthread_mutex_unlock(&source->block->mutex); z_fatal("borrowPointer cannot overlap mutable borrow"); } source->block->borrows++; }
        pthread_mutex_unlock(&source->block->mutex); return z_system_pointer_value(z_system_pointer_object(source->block, source->offset, source->length, false, mutable_value, true));
    }
    if (strcmp(name, "releaseBorrow") == 0) {
        z_expect_args(name, argc, 1); ZMemoryPointerNative *p = z_system_expect_pointer(args[0], name); if (!p->borrowed || p->released || p->block == NULL) z_fatal("releaseBorrow expects active borrow");
        pthread_mutex_lock(&p->block->mutex); if (p->mutable_value) p->block->mutable_borrow = false; else if (p->block->borrows > 0) p->block->borrows--; pthread_mutex_unlock(&p->block->mutex); p->released = true; return z_bool(true);
    }
    if (strcmp(name, "movePointer") == 0) {
        z_expect_args(name, argc, 1); ZMemoryPointerNative *p = z_system_expect_pointer(args[0], name); if (!p->owned || p->moved || p->released || p->block == NULL) z_fatal("movePointer expects owning pointer");
        pthread_mutex_lock(&p->block->mutex);
        if (p->block->borrows != 0 || p->block->mutable_borrow) { pthread_mutex_unlock(&p->block->mutex); z_fatal("movePointer: active borrows"); }
        pthread_mutex_unlock(&p->block->mutex);
        ZMemoryPointerNative *result = z_system_pointer_object(p->block, p->offset, p->length, true, p->mutable_value, false); p->moved = true; p->owned = false; p->length = 0; return z_system_pointer_value(result);
    }
    if (strcmp(name, "pointerCopy") == 0) {
        z_expect_args(name, argc, 5); ZMemoryPointerNative *dst = z_system_expect_pointer(args[0], name), *src = z_system_expect_pointer(args[2], name); size_t ds=(size_t)z_as_u64(args[1]), ss=(size_t)z_as_u64(args[3]), count=(size_t)z_as_u64(args[4]);
        if (ds + count > dst->length || ss + count > src->length) {
            z_fatal("pointerCopy range out of bounds");
        }
        ZValue *items = count ? malloc(sizeof(ZValue) * count) : NULL;
        if (count && items == NULL) z_fatal("out of memory");
        for (size_t i = 0; i < count; i++) items[i] = z_system_read_at(src, ss + i);
        for (size_t i = 0; i < count; i++) z_system_write_at(dst, ds + i, items[i]);
        free(items);
        return args[0];
    }
    if (strcmp(name, "pointerFill") == 0) { z_expect_args(name, argc, 2); ZMemoryPointerNative *p=z_system_expect_pointer(args[0],name); for(size_t i=0;i<p->length;i++)z_system_write_at(p,i,args[1]); return args[0]; }
    if (strcmp(name, "sizeOfType") == 0 || strcmp(name, "alignOfType") == 0) { z_expect_args(name,argc,1); ZKind kind; size_t size,align; if(!z_system_type(z_as_cstring(args[0]),&kind,&size,&align))z_fatal("unsupported native type"); return z_int((int64_t)(strcmp(name,"sizeOfType")==0?size:align)); }
    if (strcmp(name, "byteSizeOf") == 0) { z_expect_args(name,argc,1); if(args[0].tag==ZV_MEMORY_POINTER){ZMemoryPointerNative*p=z_system_expect_pointer(args[0],name);return z_int((int64_t)(p->length*(p->block?p->block->elem_size:sizeof(void*))));} if(args[0].tag==ZV_BUFFER)return z_int((int64_t)(args[0].as.buffer->len*args[0].as.buffer->elem_size)); z_fatal("byteSizeOf expects Pointer or compact buffer"); }
    if (strcmp(name, "nativeStructLayout") == 0) {
        z_expect_args(name,argc,1); if(args[0].tag!=ZV_ARRAY)z_fatal("nativeStructLayout expects array"); size_t offset=0,max_align=1,count=args[0].as.array->len; ZValue *fields=count?z_alloc(sizeof(ZValue)*count):NULL;
        for(size_t i=0;i<count;i++){ZValue desc=args[0].as.array->items[i];ZValue n=z_system_dict_get(desc,"name"),tval=z_system_dict_get(desc,"type");if(n.tag!=ZV_STRING||tval.tag!=ZV_STRING)z_fatal("field requires name and type");ZKind kind;size_t size,align;if(!z_system_type(tval.as.s,&kind,&size,&align))z_fatal("unsupported native field type");offset=z_system_align_up(offset,align);const char*keys[]={"name","type","offset","size","align"};ZValue vals[]={n,z_string(tval.as.s),z_int((int64_t)offset),z_int((int64_t)size),z_int((int64_t)align)};fields[i]=z_system_dict(keys,vals,5);offset+=size;if(align>max_align)max_align=align;}
        offset=z_system_align_up(offset,max_align);const char*keys[]={"size","align","fields"};ZValue vals[]={z_int((int64_t)offset),z_int((int64_t)max_align),z_array_from(fields,count)};return z_system_dict(keys,vals,3);
    }
    if (strcmp(name, "arenaCreate") == 0) { z_expect_args(name,argc,0); ZMemoryArenaNative*a=(ZMemoryArenaNative*)z_alloc(sizeof(*a)); ZValue v=z_value(ZV_MEMORY_ARENA,ZK_MEMORY_ARENA);v.as.p=a;return v; }
    if (strcmp(name, "arenaAlloc") == 0) { z_expect_args(name,argc,3);ZMemoryArenaNative*a=z_system_expect_arena(args[0],name);if(a->closed)z_fatal("arena is closed");ZValue p=z_system_alloc(z_as_cstring(args[1]),(size_t)z_as_u64(args[2]),ZMEM_ARENA);ZMemoryPointerNative*mp=(ZMemoryPointerNative*)p.as.p;mp->owned=false;ZArenaLinkNative*link=(ZArenaLinkNative*)z_alloc(sizeof(*link));link->block=mp->block;link->next=a->blocks;a->blocks=link;a->bytes+=mp->block->length*mp->block->elem_size;a->allocations++;return p; }
    if (strcmp(name, "arenaReset") == 0 || strcmp(name, "arenaFree") == 0) { z_expect_args(name,argc,1);ZMemoryArenaNative*a=z_system_expect_arena(args[0],name);if(a->closed)z_fatal("arena is closed");for(ZArenaLinkNative*l=a->blocks;l;l=l->next)z_system_block_mark_freed(l->block,true);a->blocks=NULL;a->bytes=0;a->allocations=0;if(strcmp(name,"arenaFree")==0)a->closed=true;return z_bool(true); }
    if (strcmp(name, "arenaStats") == 0) { z_expect_args(name,argc,1);ZMemoryArenaNative*a=z_system_expect_arena(args[0],name);const char*keys[]={"bytes","allocations","open"};ZValue vals[]={z_int((int64_t)a->bytes),z_int((int64_t)a->allocations),z_bool(!a->closed)};return z_system_dict(keys,vals,3); }
    if (strcmp(name, "memoryStats") == 0) { z_expect_args(name,argc,0);pthread_mutex_lock(&z_memory_registry.mutex);const char*keys[]={"allocations","frees","active_blocks","total_bytes","active_bytes","peak_bytes","invalid_accesses","double_frees"};size_t active=0;for(ZMemoryBlockNative*b=z_memory_registry.blocks;b;b=b->next)if(!b->freed&&(b->allocator==ZMEM_HEAP||b->allocator==ZMEM_ARENA))active++;ZValue vals[]={z_int((int64_t)z_memory_registry.allocations),z_int((int64_t)z_memory_registry.frees),z_int((int64_t)active),z_int((int64_t)z_memory_registry.total_bytes),z_int((int64_t)z_memory_registry.active_bytes),z_int((int64_t)z_memory_registry.peak_bytes),z_int((int64_t)z_memory_registry.invalid_accesses),z_int((int64_t)z_memory_registry.double_frees)};pthread_mutex_unlock(&z_memory_registry.mutex);return z_system_dict(keys,vals,8); }
    if (strcmp(name, "memoryLeaks") == 0) { z_expect_args(name,argc,0);pthread_mutex_lock(&z_memory_registry.mutex);size_t count=0;for(ZMemoryBlockNative*b=z_memory_registry.blocks;b;b=b->next)if(!b->freed&&b->allocator==ZMEM_HEAP)count++;ZValue*items=count?z_alloc(sizeof(ZValue)*count):NULL;size_t pos=0;for(ZMemoryBlockNative*b=z_memory_registry.blocks;b;b=b->next)if(!b->freed&&b->allocator==ZMEM_HEAP){const char*keys[]={"id","type","elements","bytes","allocator"};ZValue vals[]={z_int((int64_t)b->id),z_string(z_system_kind_name(b->elem_kind)),z_int((int64_t)b->length),z_int((int64_t)(b->length*b->elem_size)),z_string("heap")};items[pos++]=z_system_dict(keys,vals,5);}pthread_mutex_unlock(&z_memory_registry.mutex);return z_array_from(items,count); }
    if (strcmp(name, "memoryValidate") == 0) { z_expect_args(name,argc,0);pthread_mutex_lock(&z_memory_registry.mutex);const char*keys[]={"ok","invalid_accesses","double_frees"};ZValue vals[]={z_bool(z_memory_registry.invalid_accesses==0&&z_memory_registry.double_frees==0),z_int((int64_t)z_memory_registry.invalid_accesses),z_int((int64_t)z_memory_registry.double_frees)};pthread_mutex_unlock(&z_memory_registry.mutex);return z_system_dict(keys,vals,3); }
    if (strcmp(name, "memoryResetStats") == 0) { z_expect_args(name,argc,0);pthread_mutex_lock(&z_memory_registry.mutex);z_memory_registry.allocations=z_memory_registry.frees=z_memory_registry.total_bytes=z_memory_registry.active_bytes=z_memory_registry.peak_bytes=z_memory_registry.invalid_accesses=z_memory_registry.double_frees=0;for(ZMemoryBlockNative*b=z_memory_registry.blocks;b;b=b->next)if(!b->freed&&(b->allocator==ZMEM_HEAP||b->allocator==ZMEM_ARENA)){uint64_t bytes=b->length*b->elem_size;z_memory_registry.allocations++;z_memory_registry.total_bytes+=bytes;z_memory_registry.active_bytes+=bytes;}z_memory_registry.peak_bytes=z_memory_registry.active_bytes;pthread_mutex_unlock(&z_memory_registry.mutex);return z_bool(true); }
#if defined(_WIN32)
    if (strcmp(name,"mmapOpen")==0||strcmp(name,"sharedMemoryOpen")==0||strcmp(name,"sharedMemoryUnlink")==0||strcmp(name,"rawSyscall")==0) z_fatal("%s is not implemented on Windows yet",name);
#else
    if (strcmp(name, "mmapOpen") == 0) { if(argc<2||argc>3)z_fatal("mmapOpen expects 2 or 3 arguments");const char*path=z_as_cstring(args[0]),*mode=z_as_cstring(args[1]);size_t requested=argc==3?(size_t)z_as_u64(args[2]):0;int flags=O_RDONLY,prot=PROT_READ,map_flags=MAP_SHARED;if(strcmp(mode,"readwrite")==0||strcmp(mode,"write")==0||strcmp(mode,"rw")==0){flags=O_RDWR|O_CREAT;prot|=PROT_WRITE;}else if(strcmp(mode,"private")==0){flags=O_RDWR|O_CREAT;prot|=PROT_WRITE;map_flags=MAP_PRIVATE;}else if(strcmp(mode,"read")!=0&&strcmp(mode,"r")!=0)z_fatal("mmapOpen mode must be read, readwrite or private");int fd=open(path,flags,0644);if(fd<0)z_fatal("mmapOpen: %s",strerror(errno));struct stat st;if(fstat(fd,&st)!=0){close(fd);z_fatal("mmapOpen: %s",strerror(errno));}size_t size=(size_t)st.st_size;if(requested>size&&flags!=O_RDONLY){if(ftruncate(fd,(off_t)requested)!=0){close(fd);z_fatal("mmapOpen: %s",strerror(errno));}size=requested;}if(requested>0&&requested<size)size=requested;if(size==0){close(fd);z_fatal("mmapOpen size must be positive");}void*data=mmap(NULL,size,prot,map_flags,fd,0);if(data==MAP_FAILED){close(fd);z_fatal("mmapOpen: %s",strerror(errno));}ZMappedMemoryNative*m=(ZMappedMemoryNative*)z_alloc(sizeof(*m));m->fd=fd;m->path=z_strdup(path);m->mode=z_strdup(mode);m->size=size;m->block=z_system_block_new(ZK_U8,1,size,ZMEM_MAPPED,(uint8_t*)data,false);ZValue v=z_value(ZV_MAPPED_MEMORY,ZK_MAPPED_MEMORY);v.as.p=m;return v; }
    if (strcmp(name, "sharedMemoryOpen") == 0) { if(argc<2||argc>3)z_fatal("sharedMemoryOpen expects 2 or 3 arguments");const char*raw=z_as_cstring(args[0]);size_t size=(size_t)z_as_u64(args[1]);bool create=argc<3||z_as_bool(args[2]);char normalized[NAME_MAX];snprintf(normalized,sizeof(normalized),"/%s",raw[0]=='/'?raw+1:raw);int flags=O_RDWR|(create?O_CREAT:0);int fd=shm_open(normalized,flags,0600);if(fd<0)z_fatal("sharedMemoryOpen: %s",strerror(errno));if(create&&ftruncate(fd,(off_t)size)!=0){close(fd);z_fatal("sharedMemoryOpen: %s",strerror(errno));}if(!create){struct stat st;if(fstat(fd,&st)!=0){close(fd);z_fatal("sharedMemoryOpen: %s",strerror(errno));}size=(size_t)st.st_size;}void*data=mmap(NULL,size,PROT_READ|PROT_WRITE,MAP_SHARED,fd,0);if(data==MAP_FAILED){close(fd);z_fatal("sharedMemoryOpen: %s",strerror(errno));}ZSharedMemoryNative*m=(ZSharedMemoryNative*)z_alloc(sizeof(*m));m->fd=fd;m->name=z_strdup(normalized);m->size=size;m->block=z_system_block_new(ZK_U8,1,size,ZMEM_SHARED,(uint8_t*)data,false);ZValue v=z_value(ZV_SHARED_MEMORY,ZK_SHARED_MEMORY);v.as.p=m;return v; }
    if (strcmp(name, "sharedMemoryUnlink") == 0) { z_expect_args(name,argc,1);const char*raw=z_as_cstring(args[0]);char normalized[NAME_MAX];snprintf(normalized,sizeof(normalized),"/%s",raw[0]=='/'?raw+1:raw);if(shm_unlink(normalized)!=0&&errno!=ENOENT)z_fatal("sharedMemoryUnlink: %s",strerror(errno));return z_bool(true); }
    if (strcmp(name, "rawSyscall") == 0) { z_expect_args(name,argc,2);const char*enabled=getenv("ZUMBRA_ALLOW_RAW_SYSCALLS");if(enabled==NULL||strcmp(enabled,"1")!=0)z_fatal("rawSyscall is disabled; set ZUMBRA_ALLOW_RAW_SYSCALLS=1");if(args[1].tag!=ZV_ARRAY||args[1].as.array->len>6)z_fatal("rawSyscall expects up to 6 arguments");long values[6]={0};for(size_t i=0;i<args[1].as.array->len;i++)values[i]=(long)z_as_i64(args[1].as.array->items[i]);errno=0;long result=syscall((long)z_as_i64(args[0]),values[0],values[1],values[2],values[3],values[4],values[5]);int error=errno;const char*keys[]={"ok","value","errno","error"};ZValue vals[]={z_bool(error==0),z_uint((uint64_t)result,ZK_U64),z_int(error),z_string(error==0?"":strerror(error))};return z_system_dict(keys,vals,4); }
#endif
    if (strcmp(name, "mmapPointer") == 0) { z_expect_args(name,argc,1);ZMappedMemoryNative*m=z_system_expect_mapping(args[0],name);if(m->closed)z_fatal("mapping is closed");return z_system_pointer_value(z_system_pointer_object(m->block,0,m->size,false,strcmp(m->mode,"read")!=0,true)); }
    if (strcmp(name, "mmapFlush") == 0) { z_expect_args(name,argc,1);ZMappedMemoryNative*m=z_system_expect_mapping(args[0],name);if(m->closed)z_fatal("mapping is closed");
#if defined(_WIN32)
        if(!FlushViewOfFile(m->block->data,m->size))z_fatal("mmapFlush failed");
#else
        if(msync(m->block->data,m->size,MS_SYNC)!=0)z_fatal("mmapFlush: %s",strerror(errno));
#endif
        return z_bool(true); }
    if (strcmp(name, "mmapClose") == 0) { z_expect_args(name,argc,1);z_system_close_mapping(z_system_expect_mapping(args[0],name));return z_bool(true); }
    if (strcmp(name, "mmapSize") == 0) { z_expect_args(name,argc,1);return z_int((int64_t)z_system_expect_mapping(args[0],name)->size); }
    if (strcmp(name, "sharedMemoryPointer") == 0) { z_expect_args(name,argc,1);ZSharedMemoryNative*m=z_system_expect_shared(args[0],name);if(m->closed)z_fatal("shared memory is closed");return z_system_pointer_value(z_system_pointer_object(m->block,0,m->size,false,true,true)); }
    if (strcmp(name, "sharedMemoryClose") == 0) { z_expect_args(name,argc,1);z_system_close_shared(z_system_expect_shared(args[0],name));return z_bool(true); }
    if (strcmp(name, "memoryFence") == 0) { if(argc>1)z_fatal("memoryFence expects zero or one argument");atomic_thread_fence(memory_order_seq_cst);return z_null(); }
    if (strcmp(name, "atomicPointerLoad") == 0 || strcmp(name, "atomicPointerStore") == 0 || strcmp(name, "atomicPointerAdd") == 0 || strcmp(name, "atomicPointerSwap") == 0 || strcmp(name, "atomicPointerCompareSwap") == 0) {
        ZMemoryPointerNative*p=z_system_expect_pointer(args[0],name);z_system_pointer_validate(p,0,strcmp(name,"atomicPointerLoad")!=0,name);ZMemoryBlockNative*b=p->block;pthread_mutex_lock(&b->mutex);uint8_t*address=b->data+p->offset*b->elem_size;uint64_t raw=0;memcpy(&raw,address,b->elem_size);ZValue previous=z_from_raw(raw,b->elem_kind);
        if(strcmp(name,"atomicPointerLoad")==0){pthread_mutex_unlock(&b->mutex);return previous;}if(strcmp(name,"atomicPointerStore")==0){z_expect_args(name,argc,2);ZValue converted=z_convert(args[1],b->elem_kind);raw=z_as_u64(converted);memcpy(address,&raw,b->elem_size);pthread_mutex_unlock(&b->mutex);return z_null();}
        if(strcmp(name,"atomicPointerSwap")==0){z_expect_args(name,argc,2);ZValue converted=z_convert(args[1],b->elem_kind);raw=z_as_u64(converted);memcpy(address,&raw,b->elem_size);pthread_mutex_unlock(&b->mutex);return previous;}
        if(strcmp(name,"atomicPointerAdd")==0){z_expect_args(name,argc,2);uint64_t next=raw+z_as_u64(args[1]);memcpy(address,&next,b->elem_size);pthread_mutex_unlock(&b->mutex);return z_from_raw(next,b->elem_kind);}
        z_expect_args(name,argc,3);bool equal=raw==z_as_u64(z_convert(args[1],b->elem_kind));if(equal){uint64_t next=z_as_u64(z_convert(args[2],b->elem_kind));memcpy(address,&next,b->elem_size);}pthread_mutex_unlock(&b->mutex);return z_bool(equal);
    }
    if (strcmp(name, "memoryProtect") == 0 || strcmp(name, "memoryLock") == 0 || strcmp(name, "memoryUnlock") == 0) {
        if (argc < 1 || argc > 2) {
            z_fatal("%s invalid arguments", name);
        }
        ZMemoryPointerNative *p = z_system_expect_pointer(args[0], name);
        z_system_pointer_validate(p, 0, false, name);
        uint8_t *data = p->block->data + p->offset * p->block->elem_size;
        size_t length = p->length * p->block->elem_size;
#if defined(_WIN32)
        if(strcmp(name,"memoryProtect")==0){DWORD protection=PAGE_NOACCESS;const char*mode=z_as_cstring(args[1]);if(strstr(mode,"write"))protection=PAGE_READWRITE;else if(strstr(mode,"read"))protection=PAGE_READONLY;DWORD old=0;if(!VirtualProtect(data,length,protection,&old))z_fatal("memoryProtect failed");}
        else if(strcmp(name,"memoryLock")==0){if(!VirtualLock(data,length))z_fatal("memoryLock failed");}else if(!VirtualUnlock(data,length))z_fatal("memoryUnlock failed");
#else
        long page=sysconf(_SC_PAGESIZE);uintptr_t start=(uintptr_t)data&~((uintptr_t)page-1);uintptr_t end=((uintptr_t)data+length+(uintptr_t)page-1)&~((uintptr_t)page-1);if(strcmp(name,"memoryProtect")==0){int prot=0;const char*mode=z_as_cstring(args[1]);if(strstr(mode,"read"))prot|=PROT_READ;if(strstr(mode,"write"))prot|=PROT_WRITE;if(strstr(mode,"exec"))prot|=PROT_EXEC;if(mprotect((void*)start,end-start,prot)!=0)z_fatal("memoryProtect: %s",strerror(errno));}else if(strcmp(name,"memoryLock")==0){if(mlock((void*)start,end-start)!=0)z_fatal("memoryLock: %s",strerror(errno));}else if(munlock((void*)start,end-start)!=0)z_fatal("memoryUnlock: %s",strerror(errno));
#endif
        return z_bool(true);
    }
    if (strcmp(name, "dynamicOpen") == 0) { z_expect_args(name,argc,1);const char*path=z_as_cstring(args[0]);ZDynamicLibraryNative*l=(ZDynamicLibraryNative*)z_alloc(sizeof(*l));l->path=z_strdup(path);l->last_error=z_strdup("");
#if defined(_WIN32)
        l->handle=LoadLibraryA(path);if(l->handle==NULL)z_fatal("dynamicOpen failed");
#else
        l->handle=dlopen(path,RTLD_NOW|RTLD_LOCAL);if(l->handle==NULL)z_fatal("dynamicOpen: %s",dlerror());
#endif
        ZValue v=z_value(ZV_DYNAMIC_LIBRARY,ZK_DYNAMIC_LIBRARY);v.as.p=l;return v; }
    if (strcmp(name, "dynamicSymbol") == 0) { z_expect_args(name,argc,2);ZDynamicLibraryNative*l=z_system_expect_library(args[0],name);if(l->closed)z_fatal("library is closed");const char*symbol=z_as_cstring(args[1]);void*raw=NULL;
#if defined(_WIN32)
        raw=(void*)GetProcAddress(l->handle,symbol);
#else
        dlerror();raw=dlsym(l->handle,symbol);const char*error=dlerror();if(error!=NULL){l->last_error=z_strdup(error);z_fatal("dynamicSymbol: %s",error);}
#endif
        if (raw == NULL) z_fatal("dynamicSymbol not found");
        return z_system_raw_pointer_value(raw, ZK_POINTER);
    }
    if (strcmp(name, "dynamicCall") == 0) {
        z_expect_args(name, argc, 3);
        ZMemoryPointerNative *symbol = z_system_expect_pointer(args[0], name);
        bool pointer_handled = false;
        void *address = z_systems_as_pointer(args[0], &pointer_handled);
        if (!pointer_handled || address == NULL) z_fatal("dynamicCall symbol is null");
        const char *return_type = z_as_cstring(args[1]);
        if (args[2].tag != ZV_ARRAY || args[2].as.array->len > 6) z_fatal("dynamicCall arguments must be an array with at most 6 items");
        uintptr_t native_args[6] = {0};
        for (size_t index = 0; index < args[2].as.array->len; index++) native_args[index] = z_system_dynamic_argument(args[2].as.array->items[index]);
        uintptr_t result = z_system_dynamic_call(address, native_args, args[2].as.array->len);
        (void)symbol;
        if (strcmp(return_type, "void") == 0 || strcmp(return_type, "null") == 0) return z_null();
        ZKind kind = ZK_UNKNOWN; size_t size = 0, alignment = 0;
        if (!z_system_type(return_type, &kind, &size, &alignment)) z_fatal("dynamicCall uses unsupported return type %s", return_type);
        if (kind == ZK_FLOAT) z_fatal("dynamicCall does not support floating-point ABI returns");
        if (kind == ZK_BOOL) return z_bool(result != 0);
        if (kind == ZK_POINTER) return z_system_raw_pointer_value((void *)result, ZK_POINTER);
        return z_from_raw((uint64_t)result, kind);
    }
    if (strcmp(name, "dynamicClose") == 0) { z_expect_args(name,argc,1);ZDynamicLibraryNative*l=z_system_expect_library(args[0],name);if(!l->closed){
#if defined(_WIN32)
        if(!FreeLibrary(l->handle))z_fatal("dynamicClose failed");
#else
        if(dlclose(l->handle)!=0)z_fatal("dynamicClose: %s",dlerror());
#endif
        l->closed=true;}return z_bool(true); }
    if (strcmp(name, "dynamicIsOpen") == 0) { z_expect_args(name,argc,1);return z_bool(!z_system_expect_library(args[0],name)->closed); }
    if (strcmp(name, "dynamicError") == 0) { z_expect_args(name,argc,1);return z_string(z_system_expect_library(args[0],name)->last_error); }
    if (strcmp(name, "systemInfo") == 0) { z_expect_args(name,argc,0);const char*os_name=
#if defined(_WIN32)
        "windows";
#elif defined(__APPLE__)
        "macos";
#else
        "linux";
#endif
        const char*arch=sizeof(void*)==8?"amd64":"386";uint16_t endian=1;const char*keys[]={"os","arch","cpu_count","page_size","pointer_bits","endianness"};
#if defined(_WIN32)
        SYSTEM_INFO info;GetSystemInfo(&info);ZValue vals[]={z_string(os_name),z_string(arch),z_int((int64_t)info.dwNumberOfProcessors),z_int((int64_t)info.dwPageSize),z_int((int64_t)(sizeof(void*)*8)),z_string(*(uint8_t*)&endian?"little":"big")};
#else
        ZValue vals[]={z_string(os_name),z_string(arch),z_int((int64_t)sysconf(_SC_NPROCESSORS_ONLN)),z_int((int64_t)sysconf(_SC_PAGESIZE)),z_int((int64_t)(sizeof(void*)*8)),z_string(*(uint8_t*)&endian?"little":"big")};
#endif
        return z_system_dict(keys,vals,6); }
    if (strcmp(name, "pageSize") == 0) { z_expect_args(name,argc,0);
#if defined(_WIN32)
        SYSTEM_INFO info;GetSystemInfo(&info);return z_int((int64_t)info.dwPageSize);
#else
        return z_int((int64_t)sysconf(_SC_PAGESIZE));
#endif
    }
    if (strcmp(name, "cpuCount") == 0) { z_expect_args(name,argc,0);
#if defined(_WIN32)
        SYSTEM_INFO info;GetSystemInfo(&info);return z_int((int64_t)info.dwNumberOfProcessors);
#else
        return z_int((int64_t)sysconf(_SC_NPROCESSORS_ONLN));
#endif
    }
    if (strcmp(name, "profileNowNs") == 0) { z_expect_args(name,argc,0);return z_uint(z_system_now_ns(),ZK_U64); }
    if (strcmp(name, "profileElapsedNs") == 0) { z_expect_args(name,argc,1);uint64_t start=z_as_u64(args[0]),now=z_system_now_ns();return z_uint(now>=start?now-start:0,ZK_U64); }
    *handled = false;
    return z_null();
}

#endif
