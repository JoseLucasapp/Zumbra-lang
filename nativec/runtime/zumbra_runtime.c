#define _XOPEN_SOURCE 700
#include "zumbra_runtime.h"

#include <errno.h>
#include <inttypes.h>
#include <math.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <strings.h>
#include <pthread.h>
#include <setjmp.h>
#include <stdatomic.h>
#include <time.h>
#include <ctype.h>

#if defined(ZUMBRA_ENABLE_NETWORK)
#include <arpa/inet.h>
#include <fcntl.h>
#include <limits.h>
#include <netdb.h>
#include <netinet/in.h>
#include <netinet/tcp.h>
#include <poll.h>
#include <signal.h>
#include <sys/socket.h>
#include <sys/time.h>
#include <sys/types.h>
#include <unistd.h>
#ifndef NI_MAXHOST
#define NI_MAXHOST 1025
#endif
#endif

#if defined(ZUMBRA_ENABLE_TLS)
#include <openssl/err.h>
#include <openssl/ssl.h>
#include <openssl/x509v3.h>
#include <openssl/evp.h>
#include <openssl/hmac.h>
#include <openssl/sha.h>
#include <openssl/rand.h>
#endif

#if defined(ZUMBRA_ENABLE_HTTP)
#include <zlib.h>
#endif

#if defined(_WIN32)
#include <windows.h>
#endif

typedef struct ZAllocation {
    void *pointer;
    struct ZAllocation *next;
} ZAllocation;

static ZAllocation *z_allocations = NULL;
static pthread_mutex_t z_allocation_mutex = PTHREAD_MUTEX_INITIALIZER;
static pthread_mutex_t z_task_count_mutex = PTHREAD_MUTEX_INITIALIZER;
static pthread_cond_t z_task_count_condition = PTHREAD_COND_INITIALIZER;
static size_t z_active_tasks = 0;
static _Thread_local jmp_buf *z_task_error_trap = NULL;
static _Thread_local char z_task_error_message[1024];

uint32_t z_abi_version(void) { return ZUMBRA_NATIVE_ABI_VERSION; }


#if defined(ZUMBRA_ENABLE_Z12)
static void z_z12_runtime_init(void);
static void z_z12_runtime_shutdown(void);
static ZValue z_z12_call_builtin(const char *name, const ZValue *args, size_t argc, bool *handled);
static ZValue z_z12_get_field(ZValue value, const char *name, bool *handled);
static ZValue z_z12_call_method(ZValue callable, const ZValue *args, size_t argc);
#endif

#if defined(ZUMBRA_ENABLE_SQLITE)
static void z_sqlite_runtime_init(void);
static void z_sqlite_runtime_shutdown(void);
static ZValue z_sqlite_call_builtin(const char *name, const ZValue *args, size_t argc, bool *handled);
static ZValue z_sqlite_get_field(ZValue value, const char *name, bool *handled);
static ZValue z_sqlite_call_method(ZValue callable, const ZValue *args, size_t argc);
#endif

#if defined(ZUMBRA_ENABLE_HTTP)
static void z_http_runtime_init(void);
static void z_http_runtime_shutdown(void);
static ZValue z_http_call_builtin(const char *name, const ZValue *args, size_t argc, bool *handled);
static ZValue z_http_get_field(ZValue value, const char *name, bool *handled);
static ZValue z_http_call_native_method(ZValue callable, const ZValue *args, size_t argc);
#endif

void z_runtime_init(void) {
    z_allocations = NULL;
    z_active_tasks = 0;
#if defined(ZUMBRA_ENABLE_NETWORK)
    signal(SIGPIPE, SIG_IGN);
#endif
#if defined(ZUMBRA_ENABLE_HTTP)
    z_http_runtime_init();
#endif
#if defined(ZUMBRA_ENABLE_Z12)
    z_z12_runtime_init();
#endif
#if defined(ZUMBRA_ENABLE_SQLITE)
    z_sqlite_runtime_init();
#endif
}

void z_runtime_shutdown(void) {
#if defined(ZUMBRA_ENABLE_SQLITE)
    z_sqlite_runtime_shutdown();
#endif
#if defined(ZUMBRA_ENABLE_Z12)
    z_z12_runtime_shutdown();
#endif
#if defined(ZUMBRA_ENABLE_HTTP)
    z_http_runtime_shutdown();
#endif
    pthread_mutex_lock(&z_task_count_mutex);
    while (z_active_tasks != 0) pthread_cond_wait(&z_task_count_condition, &z_task_count_mutex);
    pthread_mutex_unlock(&z_task_count_mutex);
    pthread_mutex_lock(&z_allocation_mutex);
    ZAllocation *current = z_allocations;
    while (current != NULL) {
        ZAllocation *next = current->next;
        free(current->pointer);
        free(current);
        current = next;
    }
    z_allocations = NULL;
    pthread_mutex_unlock(&z_allocation_mutex);
}

void z_fatal(const char *format, ...) {
    va_list arguments;
    va_start(arguments, format);
    if (z_task_error_trap != NULL) {
        vsnprintf(z_task_error_message, sizeof(z_task_error_message), format, arguments);
        va_end(arguments);
        longjmp(*z_task_error_trap, 1);
    }
    fprintf(stderr, "zumbra runtime error: ");
    vfprintf(stderr, format, arguments);
    va_end(arguments);
    fputc('\n', stderr);
    /* Fatal exits rely on the operating system for cleanup. Waiting for tasks
       here could deadlock when the failure originated inside a task. */
    exit(1);
}

void *z_alloc(size_t size) {
    if (size == 0) {
        size = 1;
    }
    void *pointer = calloc(1, size);
    if (pointer == NULL) {
        z_fatal("out of memory allocating %zu bytes", size);
    }
    ZAllocation *entry = (ZAllocation *)malloc(sizeof(ZAllocation));
    if (entry == NULL) {
        free(pointer);
        z_fatal("out of memory tracking allocation");
    }
    entry->pointer = pointer;
    pthread_mutex_lock(&z_allocation_mutex);
    entry->next = z_allocations;
    z_allocations = entry;
    pthread_mutex_unlock(&z_allocation_mutex);
    return pointer;
}

char *z_strdup(const char *value) {
    if (value == NULL) {
        value = "";
    }
    size_t length = strlen(value);
    char *copy = (char *)z_alloc(length + 1);
    memcpy(copy, value, length + 1);
    return copy;
}

static ZValue z_value(ZTag tag, ZKind kind) {
    ZValue value;
    memset(&value, 0, sizeof(value));
    value.tag = tag;
    value.kind = kind;
    return value;
}

ZValue z_null(void) { return z_value(ZV_NULL, ZK_NULL); }
ZValue z_int(int64_t value) { ZValue result = z_value(ZV_INT, ZK_INT); result.as.i = value; return result; }
ZValue z_uint(uint64_t value, ZKind kind) { ZValue result = z_value(ZV_UINT, kind); result.as.u = value; return result; }
ZValue z_signed(int64_t value, ZKind kind) { ZValue result = z_value(ZV_INT, kind); result.as.i = value; return result; }
ZValue z_float(double value) { ZValue result = z_value(ZV_FLOAT, ZK_FLOAT); result.as.f = value; return result; }
ZValue z_bool(bool value) { ZValue result = z_value(ZV_BOOL, ZK_BOOL); result.as.b = value; return result; }
ZValue z_string(const char *value) { ZValue result = z_value(ZV_STRING, ZK_STRING); result.as.s = z_strdup(value); return result; }
ZValue z_pointer(void *value) { ZValue result = z_value(ZV_POINTER, ZK_POINTER); result.as.p = value; return result; }
ZValue z_function(int id) { ZValue result = z_value(ZV_FUNCTION, ZK_FUNCTION); result.as.id = id; return result; }
ZValue z_builtin(const char *name) { ZValue result = z_value(ZV_BUILTIN, ZK_FUNCTION); result.as.s = name; return result; }
ZValue z_struct_type(int id) { ZValue result = z_value(ZV_STRUCT_TYPE, ZK_STRUCT); result.as.id = id; return result; }
ZValue z_enum_type(int id) { ZValue result = z_value(ZV_ENUM_TYPE, ZK_ENUM); result.as.id = id; return result; }
ZValue z_enum(int type_id, int ordinal) { ZValue result = z_value(ZV_ENUM, ZK_ENUM); result.as.id = (type_id << 16) | (ordinal & 0xffff); return result; }
ZValue z_bound_method(int function_id, ZStruct *receiver) {
    ZBoundMethod *method = (ZBoundMethod *)z_alloc(sizeof(ZBoundMethod));
    method->function_id = function_id;
    method->receiver = receiver;
    ZValue result = z_value(ZV_BOUND_METHOD, ZK_FUNCTION);
    result.as.method = method;
    return result;
}

static bool z_is_numeric(ZValue value) {
    return value.tag == ZV_INT || value.tag == ZV_UINT || value.tag == ZV_FLOAT || value.tag == ZV_BOOL;
}

int64_t z_as_i64(ZValue value) {
    switch (value.tag) {
    case ZV_INT: return value.as.i;
    case ZV_UINT: return (int64_t)value.as.u;
    case ZV_FLOAT: return (int64_t)value.as.f;
    case ZV_BOOL: return value.as.b ? 1 : 0;
    default: z_fatal("expected numeric value"); return 0;
    }
}

uint64_t z_as_u64(ZValue value) {
    switch (value.tag) {
    case ZV_INT: return (uint64_t)value.as.i;
    case ZV_UINT: return value.as.u;
    case ZV_FLOAT: return (uint64_t)value.as.f;
    case ZV_BOOL: return value.as.b ? 1u : 0u;
    default: z_fatal("expected numeric value"); return 0;
    }
}

double z_as_f64(ZValue value) {
    switch (value.tag) {
    case ZV_INT: return (double)value.as.i;
    case ZV_UINT: return (double)value.as.u;
    case ZV_FLOAT: return value.as.f;
    case ZV_BOOL: return value.as.b ? 1.0 : 0.0;
    default: z_fatal("expected numeric value"); return 0.0;
    }
}

bool z_as_bool(ZValue value) { return z_truthy(value); }

const char *z_as_cstring(ZValue value) {
    if (value.tag != ZV_STRING) z_fatal("expected string for C FFI");
    return value.as.s;
}

void *z_as_pointer(ZValue value) {
    if (value.tag == ZV_NULL) return NULL;
    if (value.tag != ZV_POINTER) z_fatal("expected ptr for C FFI");
    return value.as.p;
}

static unsigned z_kind_bits(ZKind kind) {
    switch (kind) {
    case ZK_U8: case ZK_I8: return 8;
    case ZK_U16: case ZK_I16: return 16;
    case ZK_U32: case ZK_I32: return 32;
    case ZK_U64: case ZK_I64: return 64;
    default: return 64;
    }
}

static bool z_kind_signed(ZKind kind) {
    return kind == ZK_I8 || kind == ZK_I16 || kind == ZK_I32 || kind == ZK_I64 || kind == ZK_INT;
}

static bool z_kind_fixed(ZKind kind) {
    return kind >= ZK_U8 && kind <= ZK_I64;
}

static uint64_t z_mask(unsigned bits) {
    return bits >= 64 ? UINT64_MAX : ((UINT64_C(1) << bits) - UINT64_C(1));
}

static ZValue z_from_raw(uint64_t raw, ZKind kind) {
    if (!z_kind_fixed(kind)) {
        return z_int((int64_t)raw);
    }
    unsigned bits = z_kind_bits(kind);
    raw &= z_mask(bits);
    if (!z_kind_signed(kind)) {
        return z_uint(raw, kind);
    }
    if (bits < 64 && (raw & (UINT64_C(1) << (bits - 1))) != 0) {
        raw |= ~z_mask(bits);
    }
    return z_signed((int64_t)raw, kind);
}

bool z_truthy(ZValue value) {
    switch (value.tag) {
    case ZV_NULL: return false;
    case ZV_BOOL: return value.as.b;
    case ZV_INT: return value.as.i != 0;
    case ZV_UINT: return value.as.u != 0;
    case ZV_FLOAT: return value.as.f != 0.0;
    case ZV_STRING: return value.as.s != NULL && value.as.s[0] != '\0';
    case ZV_ARRAY: return value.as.array != NULL && value.as.array->len != 0;
    case ZV_DICT: return value.as.dict != NULL && value.as.dict->len != 0;
    case ZV_BUFFER: return value.as.buffer != NULL && value.as.buffer->len != 0;
    default: return true;
    }
}

bool z_equal(ZValue left, ZValue right) {
    if (z_is_numeric(left) && z_is_numeric(right)) {
        if (left.tag == ZV_FLOAT || right.tag == ZV_FLOAT) {
            return z_as_f64(left) == z_as_f64(right);
        }
        if (left.tag == ZV_UINT || right.tag == ZV_UINT) {
            return z_as_u64(left) == z_as_u64(right);
        }
        return z_as_i64(left) == z_as_i64(right);
    }
    if (left.tag != right.tag) {
        return false;
    }
    switch (left.tag) {
    case ZV_NULL: return true;
    case ZV_BOOL: return left.as.b == right.as.b;
    case ZV_STRING: return strcmp(left.as.s, right.as.s) == 0;
    case ZV_ENUM: return left.as.id == right.as.id;
    case ZV_STRUCT: return left.as.structure == right.as.structure;
    case ZV_ARRAY: return left.as.array == right.as.array;
    case ZV_DICT: return left.as.dict == right.as.dict;
    case ZV_BUFFER: return left.as.buffer == right.as.buffer;
    case ZV_FUNCTION: return left.as.id == right.as.id;
    default: return false;
    }
}

ZValue z_convert(ZValue value, ZKind target) {
    switch (target) {
    case ZK_INT: return z_int(z_as_i64(value));
    case ZK_FLOAT: return z_float(z_as_f64(value));
    case ZK_BOOL: return z_bool(z_truthy(value));
    case ZK_U8: case ZK_U16: case ZK_U32: case ZK_U64:
    case ZK_I8: case ZK_I16: case ZK_I32: case ZK_I64:
        return z_from_raw(z_as_u64(value), target);
    default: return value;
    }
}

ZValue z_unary(const char *op, ZValue value, ZKind target) {
    if (strcmp(op, "!") == 0 || strcmp(op, "not") == 0) {
        return z_bool(!z_truthy(value));
    }
    if (strcmp(op, "-") == 0) {
        if (target == ZK_FLOAT || value.tag == ZV_FLOAT) {
            return z_float(-z_as_f64(value));
        }
        if (z_kind_fixed(target)) {
            return z_from_raw((uint64_t)(-z_as_i64(value)), target);
        }
        return z_int(-z_as_i64(value));
    }
    if (strcmp(op, "bnot") == 0 || strcmp(op, "~") == 0) {
        ZKind kind = z_kind_fixed(target) ? target : value.kind;
        return z_from_raw(~z_as_u64(value), kind);
    }
    z_fatal("unsupported unary operator %s", op);
    return z_null();
}

static ZValue z_concat(ZValue left, ZValue right) {
    const char *a = left.tag == ZV_STRING ? left.as.s : "";
    const char *b = right.tag == ZV_STRING ? right.as.s : "";
    size_t length = strlen(a) + strlen(b);
    char *text = (char *)z_alloc(length + 1);
    strcpy(text, a);
    strcat(text, b);
    ZValue result = z_value(ZV_STRING, ZK_STRING);
    result.as.s = text;
    return result;
}

ZValue z_binary(const char *op, ZValue left, ZValue right, ZKind target) {
    if (strcmp(op, "&&") == 0 || strcmp(op, "and") == 0) return z_bool(z_truthy(left) && z_truthy(right));
    if (strcmp(op, "||") == 0 || strcmp(op, "or") == 0) return z_bool(z_truthy(left) || z_truthy(right));
    if (strcmp(op, "==") == 0) return z_bool(z_equal(left, right));
    if (strcmp(op, "!=") == 0) return z_bool(!z_equal(left, right));
    if (strcmp(op, "+") == 0 && left.tag == ZV_STRING && right.tag == ZV_STRING) return z_concat(left, right);

    if (!z_is_numeric(left) || !z_is_numeric(right)) {
        z_fatal("operator %s requires numeric operands", op);
    }

    if (strcmp(op, "<") == 0) return z_bool(z_as_f64(left) < z_as_f64(right));
    if (strcmp(op, ">") == 0) return z_bool(z_as_f64(left) > z_as_f64(right));
    if (strcmp(op, "<=") == 0) return z_bool(z_as_f64(left) <= z_as_f64(right));
    if (strcmp(op, ">=") == 0) return z_bool(z_as_f64(left) >= z_as_f64(right));

    if (target == ZK_FLOAT || left.tag == ZV_FLOAT || right.tag == ZV_FLOAT) {
        double a = z_as_f64(left), b = z_as_f64(right);
        if (strcmp(op, "+") == 0) return z_float(a + b);
        if (strcmp(op, "-") == 0) return z_float(a - b);
        if (strcmp(op, "*") == 0) return z_float(a * b);
        if (strcmp(op, "/") == 0) { if (b == 0.0) z_fatal("division by zero"); return z_float(a / b); }
        if (strcmp(op, "%") == 0) { if (b == 0.0) z_fatal("division by zero"); return z_float(fmod(a, b)); }
        if (strcmp(op, "**") == 0) return z_float(pow(a, b));
        z_fatal("unsupported floating-point operator %s", op);
    }

    ZKind numeric_kind = z_kind_fixed(target) ? target : (z_kind_fixed(left.kind) ? left.kind : (z_kind_fixed(right.kind) ? right.kind : ZK_INT));
    uint64_t a = z_as_u64(left), b = z_as_u64(right), raw = 0;
    if (strcmp(op, "+") == 0) raw = a + b;
    else if (strcmp(op, "-") == 0) raw = a - b;
    else if (strcmp(op, "*") == 0) raw = a * b;
    else if (strcmp(op, "/") == 0) {
        if (b == 0) z_fatal("division by zero");
        if (z_kind_signed(numeric_kind)) return z_from_raw((uint64_t)(z_as_i64(left) / z_as_i64(right)), numeric_kind);
        raw = a / b;
    } else if (strcmp(op, "%") == 0) {
        if (b == 0) z_fatal("division by zero");
        if (z_kind_signed(numeric_kind)) return z_from_raw((uint64_t)(z_as_i64(left) % z_as_i64(right)), numeric_kind);
        raw = a % b;
    } else if (strcmp(op, "band") == 0 || strcmp(op, "&") == 0) raw = a & b;
    else if (strcmp(op, "bor") == 0 || strcmp(op, "|") == 0) raw = a | b;
    else if (strcmp(op, "bxor") == 0 || strcmp(op, "^") == 0) raw = a ^ b;
    else if (strcmp(op, "shl") == 0 || strcmp(op, "<<") == 0) {
        unsigned bits = z_kind_fixed(numeric_kind) ? z_kind_bits(numeric_kind) : 64;
        if (b >= bits) z_fatal("shift count must be smaller than %u", bits);
        raw = a << b;
    } else if (strcmp(op, "shr") == 0 || strcmp(op, ">>") == 0) {
        unsigned bits = z_kind_fixed(numeric_kind) ? z_kind_bits(numeric_kind) : 64;
        if (b >= bits) z_fatal("shift count must be smaller than %u", bits);
        if (z_kind_signed(numeric_kind)) return z_from_raw((uint64_t)(z_as_i64(left) >> b), numeric_kind);
        raw = a >> b;
    } else if (strcmp(op, "**") == 0) {
        raw = 1;
        while (b > 0) { if ((b & 1u) != 0) raw *= a; a *= a; b >>= 1u; }
    } else {
        z_fatal("unsupported integer operator %s", op);
    }
    return z_from_raw(raw, numeric_kind);
}

ZValue z_array_from(const ZValue *items, size_t count) {
    ZArray *array = (ZArray *)z_alloc(sizeof(ZArray));
    array->len = count;
    array->cap = count;
    array->items = count == 0 ? NULL : (ZValue *)z_alloc(sizeof(ZValue) * count);
    if (count != 0) memcpy(array->items, items, sizeof(ZValue) * count);
    ZValue result = z_value(ZV_ARRAY, ZK_ARRAY);
    result.as.array = array;
    return result;
}

ZValue z_pair(ZValue key, ZValue value) {
    ZPair *pair = (ZPair *)z_alloc(sizeof(ZPair));
    pair->key = key;
    pair->value = value;
    ZValue result = z_value(ZV_PAIR, ZK_UNKNOWN);
    result.as.pair = pair;
    return result;
}

ZValue z_dict_from(const ZValue *pairs, size_t count) {
    ZDict *dict = (ZDict *)z_alloc(sizeof(ZDict));
    dict->len = count;
    dict->cap = count;
    dict->keys = count == 0 ? NULL : (ZValue *)z_alloc(sizeof(ZValue) * count);
    dict->values = count == 0 ? NULL : (ZValue *)z_alloc(sizeof(ZValue) * count);
    for (size_t i = 0; i < count; i++) {
        if (pairs[i].tag != ZV_PAIR) z_fatal("dictionary entry is not a key/value pair");
        dict->keys[i] = pairs[i].as.pair->key;
        dict->values[i] = pairs[i].as.pair->value;
    }
    ZValue result = z_value(ZV_DICT, ZK_DICT);
    result.as.dict = dict;
    return result;
}

static size_t z_index_value(ZValue index) {
    int64_t value = z_as_i64(index);
    if (value < 0) z_fatal("index cannot be negative: %" PRId64, value);
    return (size_t)value;
}

static size_t z_elem_size(ZKind kind) {
    switch (kind) {
    case ZK_U8: case ZK_I8: return 1;
    case ZK_U16: case ZK_I16: return 2;
    case ZK_U32: case ZK_I32: return 4;
    case ZK_U64: case ZK_I64: return 8;
    default: return 1;
    }
}

static ZValue z_buffer_get(ZBuffer *buffer, size_t index) {
    if (buffer == NULL || index >= buffer->len) z_fatal("buffer index %zu out of range", index);
    uint8_t *address = (uint8_t *)buffer->data + index * buffer->elem_size;
    uint64_t raw = 0;
    memcpy(&raw, address, buffer->elem_size);
    return z_from_raw(raw, buffer->elem_kind);
}

static void z_buffer_set(ZBuffer *buffer, size_t index, ZValue value) {
    if (buffer == NULL || index >= buffer->len) z_fatal("buffer index %zu out of range", index);
    uint64_t raw = z_as_u64(z_convert(value, buffer->elem_kind));
    uint8_t *address = (uint8_t *)buffer->data + index * buffer->elem_size;
    memcpy(address, &raw, buffer->elem_size);
}

ZValue z_index(ZValue collection, ZValue index) {
    if (collection.tag == ZV_DICT) {
        for (size_t i = 0; i < collection.as.dict->len; i++) {
            if (z_equal(collection.as.dict->keys[i], index)) return collection.as.dict->values[i];
        }
        return z_null();
    }
    size_t position = z_index_value(index);
    if (collection.tag == ZV_ARRAY) {
        if (position >= collection.as.array->len) z_fatal("array index %zu out of range", position);
        return collection.as.array->items[position];
    }
    if (collection.tag == ZV_BUFFER) return z_buffer_get(collection.as.buffer, position);
    if (collection.tag == ZV_STRING) {
        size_t length = strlen(collection.as.s);
        if (position >= length) z_fatal("string index %zu out of range", position);
        char text[2] = { collection.as.s[position], '\0' };
        return z_string(text);
    }
    z_fatal("value is not indexable");
    return z_null();
}

void z_set_index(ZValue collection, ZValue index, ZValue value) {
    if (collection.tag == ZV_DICT) {
        ZDict *dict = collection.as.dict;
        for (size_t i = 0; i < dict->len; i++) {
            if (z_equal(dict->keys[i], index)) {
                dict->values[i] = value;
                return;
            }
        }
        if (dict->len == dict->cap) {
            size_t next_cap = dict->cap == 0 ? 4 : dict->cap * 2;
            ZValue *next_keys = (ZValue *)z_alloc(sizeof(ZValue) * next_cap);
            ZValue *next_values = (ZValue *)z_alloc(sizeof(ZValue) * next_cap);
            if (dict->len != 0) {
                memcpy(next_keys, dict->keys, sizeof(ZValue) * dict->len);
                memcpy(next_values, dict->values, sizeof(ZValue) * dict->len);
            }
            dict->keys = next_keys;
            dict->values = next_values;
            dict->cap = next_cap;
        }
        dict->keys[dict->len] = index;
        dict->values[dict->len] = value;
        dict->len++;
        return;
    }
    size_t position = z_index_value(index);
    if (collection.tag == ZV_ARRAY) {
        if (position >= collection.as.array->len) z_fatal("array index %zu out of range", position);
        collection.as.array->items[position] = value;
        return;
    }
    if (collection.tag == ZV_BUFFER) {
        z_buffer_set(collection.as.buffer, position, value);
        return;
    }
    z_fatal("value does not support index assignment");
}

ZValue z_slice(ZValue collection, size_t start, size_t end) {
    if (end < start) z_fatal("slice end %zu is before start %zu", end, start);
    if (collection.tag == ZV_BUFFER) {
        ZBuffer *source = collection.as.buffer;
        if (end > source->len) z_fatal("slice end %zu exceeds buffer length %zu", end, source->len);
        ZBuffer *view = (ZBuffer *)z_alloc(sizeof(ZBuffer));
        *view = *source;
        view->data = (uint8_t *)source->data + start * source->elem_size;
        view->len = end - start;
        ZValue result = z_value(ZV_BUFFER, ZK_SLICE);
        result.as.buffer = view;
        return result;
    }
    if (collection.tag == ZV_ARRAY) {
        ZArray *source = collection.as.array;
        if (end > source->len) z_fatal("slice end %zu exceeds array length %zu", end, source->len);
        ZArray *view = (ZArray *)z_alloc(sizeof(ZArray));
        view->items = source->items + start;
        view->len = end - start;
        view->cap = view->len;
        ZValue result = z_value(ZV_ARRAY, ZK_SLICE);
        result.as.array = view;
        return result;
    }
    z_fatal("slice expects an array or compact buffer");
    return z_null();
}

void z_fill(ZValue collection, ZValue value) {
    if (collection.tag == ZV_ARRAY) {
        for (size_t i = 0; i < collection.as.array->len; i++) collection.as.array->items[i] = value;
        return;
    }
    if (collection.tag == ZV_BUFFER) {
        for (size_t i = 0; i < collection.as.buffer->len; i++) z_buffer_set(collection.as.buffer, i, value);
        return;
    }
    z_fatal("fill expects an array or compact buffer");
}

size_t z_size_of(ZValue value) {
    switch (value.tag) {
    case ZV_STRING: return strlen(value.as.s);
    case ZV_ARRAY: return value.as.array->len;
    case ZV_DICT: return value.as.dict->len;
    case ZV_BUFFER: return value.as.buffer->len;
    case ZV_STRUCT: return value.as.structure->field_count;
    default: z_fatal("sizeOf does not support this value"); return 0;
    }
}

ZValue z_get_field(ZValue object, const char *name) {
#if defined(ZUMBRA_ENABLE_Z12)
    bool z12_handled = false;
    ZValue z12_value = z_z12_get_field(object, name, &z12_handled);
    if (z12_handled) return z12_value;
#endif
#if defined(ZUMBRA_ENABLE_SQLITE)
    bool sqlite_handled = false;
    ZValue sqlite_value = z_sqlite_get_field(object, name, &sqlite_handled);
    if (sqlite_handled) return sqlite_value;
#endif
#if defined(ZUMBRA_ENABLE_HTTP)
    bool http_handled = false;
    ZValue http_value = z_http_get_field(object, name, &http_handled);
    if (http_handled) return http_value;
#endif
    if (object.tag == ZV_STRUCT) {
        int field = z_struct_field_index(object.as.structure->type_id, name);
        if (field >= 0) return object.as.structure->fields[field];
        int method = z_struct_method_id(object.as.structure->type_id, name);
        if (method >= 0) return z_bound_method(method, object.as.structure);
        z_fatal("%s has no field or method named %s", z_struct_type_name(object.as.structure->type_id), name);
    }
    if (object.tag == ZV_ENUM_TYPE) {
        int ordinal = z_enum_member_ordinal(object.as.id, name);
        if (ordinal < 0) z_fatal("%s has no member named %s", z_enum_type_name(object.as.id), name);
        return z_enum(object.as.id, ordinal);
    }
    z_fatal("field access requires a struct or enum type");
    return z_null();
}

void z_set_field(ZValue object, const char *name, ZValue value) {
    if (object.tag != ZV_STRUCT) z_fatal("field assignment requires a struct instance");
    int field = z_struct_field_index(object.as.structure->type_id, name);
    if (field < 0) z_fatal("%s has no field named %s", z_struct_type_name(object.as.structure->type_id), name);
    object.as.structure->fields[field] = value;
}


typedef struct {
    pthread_mutex_t mutex;
    pthread_cond_t condition;
    bool done;
    bool cancelled;
    bool failed;
    const char *error_message;
    ZValue result;
    ZValue callable;
    ZValue *args;
    size_t argc;
} ZTaskNative;

typedef struct {
    pthread_mutex_t mutex;
    pthread_cond_t not_empty;
    pthread_cond_t not_full;
    ZValue *items;
    size_t capacity;
    size_t count;
    size_t head;
    size_t tail;
    bool closed;
} ZChannelNative;

typedef struct { pthread_mutex_t mutex; } ZMutexNative;
typedef struct { pthread_rwlock_t mutex; } ZRWMutexNative;
typedef struct { pthread_mutex_t mutex; pthread_cond_t condition; int64_t count; } ZWaitGroupNative;
typedef struct { pthread_mutex_t mutex; pthread_cond_t condition; int64_t available; int64_t capacity; } ZSemaphoreNative;

static ZValue z_call_sync(ZValue callable, const ZValue *args, size_t argc) {
#if defined(ZUMBRA_ENABLE_Z12)
    if (callable.tag == ZV_Z12_METHOD) return z_z12_call_method(callable, args, argc);
#endif
#if defined(ZUMBRA_ENABLE_HTTP)
    if (callable.tag == ZV_NATIVE_METHOD) return z_http_call_native_method(callable, args, argc);
#endif
    if (callable.tag == ZV_FUNCTION) return z_dispatch_function(callable.as.id, args, argc);
    if (callable.tag == ZV_BUILTIN) return z_call_builtin(callable.as.s, args, argc);
    if (callable.tag == ZV_STRUCT_TYPE) return z_construct_struct(callable.as.id, args, argc);
    if (callable.tag == ZV_BOUND_METHOD) {
        ZValue *full = (ZValue *)z_alloc(sizeof(ZValue) * (argc + 1));
        ZValue receiver = z_value(ZV_STRUCT, ZK_STRUCT);
        receiver.as.structure = callable.as.method->receiver;
        full[0] = receiver;
        if (argc != 0) memcpy(full + 1, args, sizeof(ZValue) * argc);
        return z_dispatch_function(callable.as.method->function_id, full, argc + 1);
    }
    z_fatal("value is not callable");
    return z_null();
}

static void *z_task_worker(void *raw) {
    ZTaskNative *task = (ZTaskNative *)raw;
    jmp_buf trap;
    z_task_error_trap = &trap;
    if (setjmp(trap) == 0) {
        ZValue result = z_call_sync(task->callable, task->args, task->argc);
        z_task_error_trap = NULL;
        pthread_mutex_lock(&task->mutex);
        if (!task->done) {
            task->result = result;
            task->failed = false;
            task->error_message = NULL;
            task->done = true;
        }
        pthread_cond_broadcast(&task->condition);
        pthread_mutex_unlock(&task->mutex);
    } else {
        char captured_error[sizeof(z_task_error_message)];
        strncpy(captured_error, z_task_error_message, sizeof(captured_error) - 1);
        captured_error[sizeof(captured_error) - 1] = '\0';
        z_task_error_trap = NULL;
        pthread_mutex_lock(&task->mutex);
        if (!task->done) {
            task->result = z_null();
            task->failed = true;
            task->error_message = z_strdup(captured_error);
            task->done = true;
        }
        pthread_cond_broadcast(&task->condition);
        pthread_mutex_unlock(&task->mutex);
    }
    pthread_mutex_lock(&z_task_count_mutex);
    z_active_tasks--;
    pthread_cond_broadcast(&z_task_count_condition);
    pthread_mutex_unlock(&z_task_count_mutex);
    return NULL;
}

ZValue z_spawn(ZValue callable, const ZValue *args, size_t argc) {
    ZTaskNative *task = (ZTaskNative *)z_alloc(sizeof(ZTaskNative));
    pthread_mutex_init(&task->mutex, NULL);
    pthread_cond_init(&task->condition, NULL);
    task->done = false;
    task->cancelled = false;
    task->failed = false;
    task->error_message = NULL;
    task->result = z_null();
    task->callable = callable;
    task->argc = argc;
    task->args = argc == 0 ? NULL : (ZValue *)z_alloc(sizeof(ZValue) * argc);
    if (argc != 0) memcpy(task->args, args, sizeof(ZValue) * argc);
    pthread_mutex_lock(&z_task_count_mutex);
    z_active_tasks++;
    pthread_mutex_unlock(&z_task_count_mutex);
    pthread_t thread;
    if (pthread_create(&thread, NULL, z_task_worker, task) != 0) z_fatal("could not create native task");
    pthread_detach(thread);
    ZValue value = z_value(ZV_TASK, ZK_TASK);
    value.as.p = task;
    return value;
}

static ZTaskNative *z_expect_task(ZValue value) {
    if (value.tag != ZV_TASK || value.as.p == NULL) z_fatal("expected Task");
    return (ZTaskNative *)value.as.p;
}

ZValue z_task_await(ZValue value) {
    ZTaskNative *task = z_expect_task(value);
    pthread_mutex_lock(&task->mutex);
    while (!task->done) pthread_cond_wait(&task->condition, &task->mutex);
    ZValue result = task->result;
    bool failed = task->failed;
    const char *error_message = task->error_message;
    pthread_mutex_unlock(&task->mutex);
    if (failed) z_fatal("task failed: %s", error_message == NULL ? "unknown task error" : error_message);
    return result;
}

bool z_task_cancel(ZValue value) {
    ZTaskNative *task = z_expect_task(value);
    pthread_mutex_lock(&task->mutex);
    bool changed = !task->done && !task->cancelled;
    if (changed) { task->cancelled = true; task->done = true; task->result = z_null(); pthread_cond_broadcast(&task->condition); }
    pthread_mutex_unlock(&task->mutex);
    return changed;
}
bool z_task_done(ZValue value) { ZTaskNative *task=z_expect_task(value); pthread_mutex_lock(&task->mutex); bool result=task->done; pthread_mutex_unlock(&task->mutex); return result; }
bool z_task_cancelled(ZValue value) { ZTaskNative *task=z_expect_task(value); pthread_mutex_lock(&task->mutex); bool result=task->cancelled; pthread_mutex_unlock(&task->mutex); return result; }

static struct timespec z_deadline_after_ms(int64_t milliseconds) {
    struct timespec deadline;
    clock_gettime(CLOCK_REALTIME, &deadline);
    if (milliseconds < 0) milliseconds = 0;
    deadline.tv_sec += (time_t)(milliseconds / 1000);
    deadline.tv_nsec += (long)((milliseconds % 1000) * 1000000);
    if (deadline.tv_nsec >= 1000000000L) { deadline.tv_sec++; deadline.tv_nsec -= 1000000000L; }
    return deadline;
}

static ZValue z_task_await_timeout(ZValue value, int64_t milliseconds, bool *completed) {
    ZTaskNative *task = z_expect_task(value);
    struct timespec deadline = z_deadline_after_ms(milliseconds);
    pthread_mutex_lock(&task->mutex);
    while (!task->done) {
        int status = pthread_cond_timedwait(&task->condition, &task->mutex, &deadline);
        if (status == ETIMEDOUT) { *completed = false; pthread_mutex_unlock(&task->mutex); return z_null(); }
    }
    *completed = true;
    ZValue result = task->result;
    bool failed = task->failed;
    const char *error_message = task->error_message;
    pthread_mutex_unlock(&task->mutex);
    if (failed) z_fatal("task failed: %s", error_message == NULL ? "unknown task error" : error_message);
    return result;
}

static ZChannelNative *z_expect_channel(ZValue value) { if(value.tag!=ZV_CHANNEL||value.as.p==NULL)z_fatal("expected Channel");return (ZChannelNative *)value.as.p; }
static ZValue z_channel_new(size_t capacity) {
    ZChannelNative *channel=(ZChannelNative *)z_alloc(sizeof(ZChannelNative));
    pthread_mutex_init(&channel->mutex,NULL);pthread_cond_init(&channel->not_empty,NULL);pthread_cond_init(&channel->not_full,NULL);
    channel->capacity=capacity;channel->count=0;channel->head=0;channel->tail=0;channel->closed=false;
    channel->items=(ZValue *)z_alloc(sizeof(ZValue)*(capacity==0?1:capacity));
    ZValue result=z_value(ZV_CHANNEL,ZK_CHANNEL);result.as.p=channel;return result;
}
static void z_channel_send(ZValue channelValue,ZValue value){ZChannelNative *c=z_expect_channel(channelValue);pthread_mutex_lock(&c->mutex);if(c->capacity==0){while(c->count!=0&&!c->closed)pthread_cond_wait(&c->not_full,&c->mutex);}else{while(c->count==c->capacity&&!c->closed)pthread_cond_wait(&c->not_full,&c->mutex);}if(c->closed){pthread_mutex_unlock(&c->mutex);z_fatal("cannot send to closed channel");}c->items[c->tail]=value;c->tail=(c->tail+1)%(c->capacity==0?1:c->capacity);c->count++;pthread_cond_signal(&c->not_empty);if(c->capacity==0){while(c->count!=0&&!c->closed)pthread_cond_wait(&c->not_full,&c->mutex);}pthread_mutex_unlock(&c->mutex);}
static ZValue z_channel_receive(ZValue channelValue,bool *open){ZChannelNative *c=z_expect_channel(channelValue);pthread_mutex_lock(&c->mutex);while(c->count==0&&!c->closed)pthread_cond_wait(&c->not_empty,&c->mutex);if(c->count==0&&c->closed){*open=false;pthread_mutex_unlock(&c->mutex);return z_null();}ZValue value=c->items[c->head];c->head=(c->head+1)%(c->capacity==0?1:c->capacity);c->count--;*open=true;pthread_cond_broadcast(&c->not_full);pthread_mutex_unlock(&c->mutex);return value;}
static ZValue z_channel_receive_timeout(ZValue channelValue,int64_t milliseconds,bool *open,bool *received){
    ZChannelNative *c=z_expect_channel(channelValue);
    struct timespec deadline=z_deadline_after_ms(milliseconds);
    pthread_mutex_lock(&c->mutex);
    while(c->count==0&&!c->closed){int status=pthread_cond_timedwait(&c->not_empty,&c->mutex,&deadline);if(status==ETIMEDOUT){*received=false;*open=true;pthread_mutex_unlock(&c->mutex);return z_null();}}
    if(c->count==0&&c->closed){*received=true;*open=false;pthread_mutex_unlock(&c->mutex);return z_null();}
    ZValue value=c->items[c->head];c->head=(c->head+1)%(c->capacity==0?1:c->capacity);c->count--;*received=true;*open=true;pthread_cond_broadcast(&c->not_full);pthread_mutex_unlock(&c->mutex);return value;
}
static bool z_channel_close(ZValue channelValue){ZChannelNative *c=z_expect_channel(channelValue);pthread_mutex_lock(&c->mutex);bool changed=!c->closed;c->closed=true;pthread_cond_broadcast(&c->not_empty);pthread_cond_broadcast(&c->not_full);pthread_mutex_unlock(&c->mutex);return changed;}
static void z_sleep_ms(int64_t milliseconds){if(milliseconds<0)z_fatal("sleepMs expects non-negative milliseconds");struct timespec request;request.tv_sec=(time_t)(milliseconds/1000);request.tv_nsec=(long)((milliseconds%1000)*1000000);nanosleep(&request,NULL);}

ZValue z_call(ZValue callable, const ZValue *args, size_t argc) {
#if defined(ZUMBRA_ENABLE_SQLITE)
    if (callable.tag == ZV_SQLITE_METHOD) return z_sqlite_call_method(callable, args, argc);
#endif
#if defined(ZUMBRA_ENABLE_Z12)
    if (callable.tag == ZV_Z12_METHOD) return z_z12_call_method(callable, args, argc);
#endif
#if defined(ZUMBRA_ENABLE_HTTP)
    if (callable.tag == ZV_NATIVE_METHOD) return z_http_call_native_method(callable, args, argc);
#endif
    if (callable.tag == ZV_FUNCTION) { if (z_function_is_async(callable.as.id)) return z_spawn(callable,args,argc); return z_dispatch_function(callable.as.id, args, argc); }
    if (callable.tag == ZV_BUILTIN) return z_call_builtin(callable.as.s, args, argc);
    if (callable.tag == ZV_STRUCT_TYPE) return z_construct_struct(callable.as.id, args, argc);
    if (callable.tag == ZV_BOUND_METHOD) {
        if (z_function_is_async(callable.as.method->function_id)) return z_spawn(callable, args, argc);
        ZValue *full = (ZValue *)z_alloc(sizeof(ZValue) * (argc + 1));
        ZValue receiver = z_value(ZV_STRUCT, ZK_STRUCT);
        receiver.as.structure = callable.as.method->receiver;
        full[0] = receiver;
        if (argc != 0) memcpy(full + 1, args, sizeof(ZValue) * argc);
        return z_dispatch_function(callable.as.method->function_id, full, argc + 1);
    }
    z_fatal("value is not callable");
    return z_null();
}

static void z_show_inner(ZValue value) {
    switch (value.tag) {
    case ZV_NULL: fputs("null", stdout); break;
    case ZV_INT: fprintf(stdout, "%" PRId64, value.as.i); break;
    case ZV_UINT: fprintf(stdout, "%" PRIu64, value.as.u); break;
    case ZV_FLOAT: fprintf(stdout, "%.15g", value.as.f); break;
    case ZV_BOOL: fputs(value.as.b ? "true" : "false", stdout); break;
    case ZV_STRING: fputs(value.as.s, stdout); break;
    case ZV_POINTER: fprintf(stdout, "%p", value.as.p); break;
    case ZV_TASK: fputs("Task", stdout); break;
    case ZV_CHANNEL: fputs("Channel", stdout); break;
    case ZV_MUTEX: fputs("Mutex", stdout); break;
    case ZV_RW_MUTEX: fputs("RWMutex", stdout); break;
    case ZV_WAIT_GROUP: fputs("WaitGroup", stdout); break;
    case ZV_SEMAPHORE: fputs("Semaphore", stdout); break;
    case ZV_ATOMIC_INT: fprintf(stdout, "AtomicInt<%" PRId64 ">", atomic_load((_Atomic int64_t *)value.as.p)); break;
#if defined(ZUMBRA_ENABLE_SQLITE)
    case ZV_SQLITE_DATABASE: fputs("SQLiteDatabase", stdout); break;
    case ZV_SQLITE_STATEMENT: fputs("SQLiteStatement", stdout); break;
    case ZV_SQLITE_TRANSACTION: fputs("SQLiteTransaction", stdout); break;
    case ZV_SQL_ROWS: fputs("SQLRows", stdout); break;
    case ZV_POSTGRES_DATABASE: fputs("PostgresDatabase", stdout); break;
    case ZV_POSTGRES_STATEMENT: fputs("PostgresStatement", stdout); break;
    case ZV_POSTGRES_TRANSACTION: fputs("PostgresTransaction", stdout); break;
    case ZV_REDIS_CLIENT: fputs("RedisClient", stdout); break;
    case ZV_CONFIG: fputs("Config", stdout); break;
    case ZV_LOGGER: fputs("Logger", stdout); break;
    case ZV_METRICS: fputs("Metrics", stdout); break;
    case ZV_TRACE_SPAN: fputs("TraceSpan", stdout); break;
    case ZV_SESSION_STORE: fputs("SessionStore", stdout); break;
    case ZV_RATE_LIMITER: fputs("RateLimiter", stdout); break;
#endif
#if defined(ZUMBRA_ENABLE_NETWORK)
    case ZV_NET_LISTENER: fputs("NetListener", stdout); break;
    case ZV_NET_STREAM: fputs("NetStream", stdout); break;
    case ZV_UDP_SOCKET: fputs("UDPSocket", stdout); break;
#endif
#if defined(ZUMBRA_ENABLE_HTTP)
    case ZV_HTTP_APP: fputs("HttpApp", stdout); break;
    case ZV_HTTP_SERVER: fputs("HttpServer", stdout); break;
    case ZV_HTTP_REQUEST: fputs("HttpRequest", stdout); break;
    case ZV_HTTP_RESPONSE: fputs("HttpResponse", stdout); break;
    case ZV_HTTP_CLIENT_RESPONSE: fputs("HttpClientResponse", stdout); break;
    case ZV_HTTP_STREAM: fputs("HttpStream", stdout); break;
    case ZV_HTTP_FILE: fputs("HttpFile", stdout); break;
    case ZV_WEB_SOCKET: fputs("WebSocket", stdout); break;
#endif
    case ZV_ENUM: {
        int type_id = value.as.id >> 16;
        int ordinal = value.as.id & 0xffff;
        fprintf(stdout, "%s.%s", z_enum_type_name(type_id), z_enum_member_name(type_id, ordinal));
        break;
    }
    case ZV_STRUCT: fprintf(stdout, "%s{...}", z_struct_type_name(value.as.structure->type_id)); break;
    case ZV_ARRAY:
        fputc('[', stdout);
        for (size_t i = 0; i < value.as.array->len; i++) { if (i != 0) fputs(", ", stdout); z_show_inner(value.as.array->items[i]); }
        fputc(']', stdout);
        break;
    case ZV_BUFFER:
        fputc('[', stdout);
        for (size_t i = 0; i < value.as.buffer->len; i++) { if (i != 0) fputs(", ", stdout); z_show_inner(z_buffer_get(value.as.buffer, i)); }
        fputc(']', stdout);
        break;
    case ZV_DICT:
        fputc('{', stdout);
        for (size_t i = 0; i < value.as.dict->len; i++) {
            if (i != 0) fputs(", ", stdout);
            z_show_inner(value.as.dict->keys[i]); fputs(": ", stdout); z_show_inner(value.as.dict->values[i]);
        }
        fputc('}', stdout);
        break;
    default: fputs("<value>", stdout); break;
    }
}

void z_show(ZValue value) { z_show_inner(value); fputc('\n', stdout); }

static ZValue z_to_string_value(ZValue value) {
    char buffer[160];
    switch (value.tag) {
    case ZV_STRING: return value;
    case ZV_NULL: return z_string("null");
    case ZV_INT: snprintf(buffer, sizeof(buffer), "%" PRId64, value.as.i); return z_string(buffer);
    case ZV_UINT: snprintf(buffer, sizeof(buffer), "%" PRIu64, value.as.u); return z_string(buffer);
    case ZV_FLOAT: snprintf(buffer, sizeof(buffer), "%.15g", value.as.f); return z_string(buffer);
    case ZV_BOOL: return z_string(value.as.b ? "true" : "false");
    case ZV_POINTER: snprintf(buffer, sizeof(buffer), "%p", value.as.p); return z_string(buffer);
    case ZV_ENUM: {
        int type_id = value.as.id >> 16;
        int ordinal = value.as.id & 0xffff;
        snprintf(buffer, sizeof(buffer), "%s.%s", z_enum_type_name(type_id), z_enum_member_name(type_id, ordinal));
        return z_string(buffer);
    }
    case ZV_STRUCT: snprintf(buffer, sizeof(buffer), "%s{...}", z_struct_type_name(value.as.structure->type_id)); return z_string(buffer);
    case ZV_TASK: return z_string("Task");
    case ZV_CHANNEL: return z_string("Channel");
    case ZV_MUTEX: return z_string("Mutex");
    case ZV_RW_MUTEX: return z_string("RWMutex");
    case ZV_WAIT_GROUP: return z_string("WaitGroup");
    case ZV_SEMAPHORE: return z_string("Semaphore");
    case ZV_ATOMIC_INT:
        snprintf(buffer, sizeof(buffer), "AtomicInt<%" PRId64 ">", atomic_load((_Atomic int64_t *)value.as.p));
        return z_string(buffer);
    default:
        return z_string("<value>");
    }
}

ZValue z_bytes(size_t size) {
    ZBuffer *buffer = (ZBuffer *)z_alloc(sizeof(ZBuffer));
    buffer->data = z_alloc(size == 0 ? 1 : size);
    buffer->len = size;
    buffer->elem_size = 1;
    buffer->elem_kind = ZK_U8;
    ZValue result = z_value(ZV_BUFFER, ZK_BYTE_ARRAY);
    result.as.buffer = buffer;
    return result;
}

static ZKind z_kind_from_name(const char *name) {
    if (strcmp(name, "u8") == 0) return ZK_U8;
    if (strcmp(name, "u16") == 0) return ZK_U16;
    if (strcmp(name, "u32") == 0) return ZK_U32;
    if (strcmp(name, "u64") == 0) return ZK_U64;
    if (strcmp(name, "i8") == 0) return ZK_I8;
    if (strcmp(name, "i16") == 0) return ZK_I16;
    if (strcmp(name, "i32") == 0) return ZK_I32;
    if (strcmp(name, "i64") == 0) return ZK_I64;
    z_fatal("unknown compact array element type %s", name);
    return ZK_UNKNOWN;
}

ZValue z_array_of(const char *kind_name, size_t size) {
    ZKind kind = z_kind_from_name(kind_name);
    ZBuffer *buffer = (ZBuffer *)z_alloc(sizeof(ZBuffer));
    buffer->elem_size = z_elem_size(kind);
    buffer->len = size;
    buffer->elem_kind = kind;
    buffer->data = z_alloc((size == 0 ? 1 : size) * buffer->elem_size);
    ZValue result = z_value(ZV_BUFFER, ZK_TYPED_ARRAY);
    result.as.buffer = buffer;
    return result;
}

static ZBuffer *z_expect_byte_buffer(ZValue value) {
    if (value.tag != ZV_BUFFER || value.as.buffer->elem_size != 1) z_fatal("operation expects a byte buffer");
    return value.as.buffer;
}

ZValue z_read_bytes(const char *path) {
    FILE *file = fopen(path, "rb");
    if (file == NULL) z_fatal("cannot open %s: %s", path, strerror(errno));
    if (fseek(file, 0, SEEK_END) != 0) z_fatal("cannot seek %s", path);
    long length = ftell(file);
    if (length < 0) z_fatal("cannot determine size of %s", path);
    rewind(file);
    ZValue result = z_bytes((size_t)length);
    if (length > 0 && fread(result.as.buffer->data, 1, (size_t)length, file) != (size_t)length) z_fatal("cannot read %s", path);
    fclose(file);
    return result;
}

size_t z_write_bytes(const char *path, ZValue value) {
    ZBuffer *buffer = z_expect_byte_buffer(value);
    FILE *file = fopen(path, "wb");
    if (file == NULL) z_fatal("cannot create %s: %s", path, strerror(errno));
    size_t written = fwrite(buffer->data, 1, buffer->len, file);
    if (written != buffer->len) z_fatal("could not write all bytes to %s", path);
    fclose(file);
    return written;
}

ZValue z_read_uint(ZValue value, size_t offset, unsigned bits, bool little_endian) {
    ZBuffer *buffer = z_expect_byte_buffer(value);
    size_t count = bits / 8;
    if (offset + count > buffer->len) z_fatal("binary read exceeds buffer bounds");
    const uint8_t *data = (const uint8_t *)buffer->data + offset;
    uint64_t result = 0;
    for (size_t i = 0; i < count; i++) {
        size_t source = little_endian ? i : (count - 1 - i);
        result |= ((uint64_t)data[source]) << (8 * i);
    }
    ZKind kind = bits == 16 ? ZK_U16 : bits == 32 ? ZK_U32 : ZK_U64;
    return z_uint(result, kind);
}

void z_write_uint(ZValue value, size_t offset, ZValue input, unsigned bits, bool little_endian) {
    ZBuffer *buffer = z_expect_byte_buffer(value);
    size_t count = bits / 8;
    if (offset + count > buffer->len) z_fatal("binary write exceeds buffer bounds");
    uint8_t *data = (uint8_t *)buffer->data + offset;
    uint64_t raw = z_as_u64(input);
    for (size_t i = 0; i < count; i++) {
        size_t destination = little_endian ? i : (count - 1 - i);
        data[destination] = (uint8_t)((raw >> (8 * i)) & 0xffu);
    }
}

void z_copy_bytes(ZValue destination, size_t destination_start, ZValue source, size_t source_start, size_t length) {
    ZBuffer *dst = z_expect_byte_buffer(destination);
    ZBuffer *src = z_expect_byte_buffer(source);
    if (destination_start + length > dst->len || source_start + length > src->len) z_fatal("copyBytes exceeds buffer bounds");
    memmove((uint8_t *)dst->data + destination_start, (uint8_t *)src->data + source_start, length);
}

bool z_bytes_equal(ZValue left, ZValue right) {
    ZBuffer *a = z_expect_byte_buffer(left);
    ZBuffer *b = z_expect_byte_buffer(right);
    return a->len == b->len && memcmp(a->data, b->data, a->len) == 0;
}

/* Compact public-domain-style SHA-256 implementation for ROM identity. */
typedef struct { uint32_t state[8]; uint64_t bit_count; uint8_t block[64]; size_t used; } ZSha256;
static uint32_t z_rotr(uint32_t value, uint32_t bits) { return (value >> bits) | (value << (32u - bits)); }
static const uint32_t z_sha_k[64] = {
    0x428a2f98u,0x71374491u,0xb5c0fbcfu,0xe9b5dba5u,0x3956c25bu,0x59f111f1u,0x923f82a4u,0xab1c5ed5u,
    0xd807aa98u,0x12835b01u,0x243185beu,0x550c7dc3u,0x72be5d74u,0x80deb1feu,0x9bdc06a7u,0xc19bf174u,
    0xe49b69c1u,0xefbe4786u,0x0fc19dc6u,0x240ca1ccu,0x2de92c6fu,0x4a7484aau,0x5cb0a9dcu,0x76f988dau,
    0x983e5152u,0xa831c66du,0xb00327c8u,0xbf597fc7u,0xc6e00bf3u,0xd5a79147u,0x06ca6351u,0x14292967u,
    0x27b70a85u,0x2e1b2138u,0x4d2c6dfcu,0x53380d13u,0x650a7354u,0x766a0abbu,0x81c2c92eu,0x92722c85u,
    0xa2bfe8a1u,0xa81a664bu,0xc24b8b70u,0xc76c51a3u,0xd192e819u,0xd6990624u,0xf40e3585u,0x106aa070u,
    0x19a4c116u,0x1e376c08u,0x2748774cu,0x34b0bcb5u,0x391c0cb3u,0x4ed8aa4au,0x5b9cca4fu,0x682e6ff3u,
    0x748f82eeu,0x78a5636fu,0x84c87814u,0x8cc70208u,0x90befffau,0xa4506cebu,0xbef9a3f7u,0xc67178f2u
};
static void z_sha_transform(ZSha256 *ctx, const uint8_t block[64]) {
    uint32_t w[64];
    for (int i = 0; i < 16; i++) w[i] = ((uint32_t)block[i*4] << 24) | ((uint32_t)block[i*4+1] << 16) | ((uint32_t)block[i*4+2] << 8) | block[i*4+3];
    for (int i = 16; i < 64; i++) { uint32_t s0=z_rotr(w[i-15],7)^z_rotr(w[i-15],18)^(w[i-15]>>3); uint32_t s1=z_rotr(w[i-2],17)^z_rotr(w[i-2],19)^(w[i-2]>>10); w[i]=w[i-16]+s0+w[i-7]+s1; }
    uint32_t a=ctx->state[0],b=ctx->state[1],c=ctx->state[2],d=ctx->state[3],e=ctx->state[4],f=ctx->state[5],g=ctx->state[6],h=ctx->state[7];
    for (int i=0;i<64;i++){ uint32_t s1=z_rotr(e,6)^z_rotr(e,11)^z_rotr(e,25); uint32_t ch=(e&f)^((~e)&g); uint32_t t1=h+s1+ch+z_sha_k[i]+w[i]; uint32_t s0=z_rotr(a,2)^z_rotr(a,13)^z_rotr(a,22); uint32_t maj=(a&b)^(a&c)^(b&c); uint32_t t2=s0+maj; h=g;g=f;f=e;e=d+t1;d=c;c=b;b=a;a=t1+t2; }
    ctx->state[0]+=a;ctx->state[1]+=b;ctx->state[2]+=c;ctx->state[3]+=d;ctx->state[4]+=e;ctx->state[5]+=f;ctx->state[6]+=g;ctx->state[7]+=h;
}
static void z_sha_init(ZSha256 *ctx){ static const uint32_t initial[8]={0x6a09e667u,0xbb67ae85u,0x3c6ef372u,0xa54ff53au,0x510e527fu,0x9b05688cu,0x1f83d9abu,0x5be0cd19u}; memcpy(ctx->state,initial,sizeof(initial));ctx->bit_count=0;ctx->used=0; }
static void z_sha_update(ZSha256 *ctx,const uint8_t *data,size_t length){ ctx->bit_count+=(uint64_t)length*8u; while(length>0){size_t space=64-ctx->used;size_t take=length<space?length:space;memcpy(ctx->block+ctx->used,data,take);ctx->used+=take;data+=take;length-=take;if(ctx->used==64){z_sha_transform(ctx,ctx->block);ctx->used=0;}} }
static void z_sha_final(ZSha256 *ctx,uint8_t digest[32]){ctx->block[ctx->used++]=0x80u;if(ctx->used>56){while(ctx->used<64)ctx->block[ctx->used++]=0;z_sha_transform(ctx,ctx->block);ctx->used=0;}while(ctx->used<56)ctx->block[ctx->used++]=0;for(int i=7;i>=0;i--)ctx->block[ctx->used++]=(uint8_t)(ctx->bit_count>>(i*8));z_sha_transform(ctx,ctx->block);for(int i=0;i<8;i++){digest[i*4]=(uint8_t)(ctx->state[i]>>24);digest[i*4+1]=(uint8_t)(ctx->state[i]>>16);digest[i*4+2]=(uint8_t)(ctx->state[i]>>8);digest[i*4+3]=(uint8_t)ctx->state[i];}}
ZValue z_sha256(ZValue value){ZBuffer *buffer=z_expect_byte_buffer(value);ZSha256 ctx;uint8_t digest[32];static const char hex[]="0123456789abcdef";char *text=(char *)z_alloc(65);z_sha_init(&ctx);z_sha_update(&ctx,(const uint8_t *)buffer->data,buffer->len);z_sha_final(&ctx,digest);for(int i=0;i<32;i++){text[i*2]=hex[digest[i]>>4];text[i*2+1]=hex[digest[i]&15];}text[64]='\0';ZValue result=z_value(ZV_STRING,ZK_STRING);result.as.s=text;return result;}

static void z_expect_args(const char *name, size_t argc, size_t expected) {
    if (argc != expected) z_fatal("%s expects %zu arguments, got %zu", name, expected, argc);
}

static ZValue z_checked_arithmetic(const char *name, ZValue a, ZValue b, char op, bool saturating) {
    ZKind kind = z_kind_fixed(a.kind) ? a.kind : b.kind;
    if (!z_kind_fixed(kind)) z_fatal("%s requires a fixed integer", name);
    unsigned bits = z_kind_bits(kind);
    if (z_kind_signed(kind)) {
        __int128 x=z_as_i64(a), y=z_as_i64(b), r=op=='+'?x+y:op=='-'?x-y:x*y;
        __int128 min=bits==64?((__int128)INT64_MIN):-((__int128)1<<(bits-1));
        __int128 max=bits==64?((__int128)INT64_MAX):(((__int128)1<<(bits-1))-1);
        if (r<min || r>max) { if (!saturating) z_fatal("%s overflow", name); r=r<min?min:max; }
        return z_signed((int64_t)r,kind);
    }
    unsigned __int128 x=z_as_u64(a), y=z_as_u64(b), r=op=='+'?x+y:op=='-'?(x>=y?x-y:(unsigned __int128)-1):x*y;
    unsigned __int128 max=bits==64?(unsigned __int128)UINT64_MAX:(((unsigned __int128)1<<bits)-1);
    if ((op=='-' && x<y) || r>max) { if (!saturating) z_fatal("%s overflow",name); r=op=='-'?0:max; }
    return z_uint((uint64_t)r,kind);
}


#if defined(ZUMBRA_ENABLE_NETWORK)

typedef struct {
    int fd;
    bool closed;
    bool tls;
#if defined(ZUMBRA_ENABLE_TLS)
    SSL_CTX *tls_context;
#endif
} ZNetListenerNative;

typedef struct {
    int fd;
    bool closed;
    bool tls;
#if defined(ZUMBRA_ENABLE_TLS)
    SSL *tls_session;
    SSL_CTX *owned_context;
#endif
} ZNetStreamNative;

typedef struct {
    int fd;
    bool closed;
} ZUDPSocketNative;

static void z_network_require_port(int64_t port, const char *name) {
    if (port < 0 || port > 65535) z_fatal("%s port must be between 0 and 65535", name);
}

static void z_network_require_size(int64_t size, const char *name) {
    if (size <= 0) z_fatal("%s size must be greater than zero", name);
    if ((uint64_t)size > 64u * 1024u * 1024u) z_fatal("%s size exceeds the 64 MiB safety limit", name);
}

static int z_network_wait_fd(int fd, short events, int64_t milliseconds) {
    struct pollfd descriptor;
    memset(&descriptor, 0, sizeof(descriptor));
    descriptor.fd = fd;
    descriptor.events = events;
    int timeout = milliseconds < 0 ? -1 : (milliseconds > INT_MAX ? INT_MAX : (int)milliseconds);
    for (;;) {
        int result = poll(&descriptor, 1, timeout);
        if (result < 0 && errno == EINTR) continue;
        return result;
    }
}

static void z_network_set_timeout(int fd, int option, int64_t milliseconds) {
    struct timeval timeout;
    memset(&timeout, 0, sizeof(timeout));
    if (milliseconds >= 0) {
        timeout.tv_sec = (time_t)(milliseconds / 1000);
        timeout.tv_usec = (suseconds_t)((milliseconds % 1000) * 1000);
    }
    if (setsockopt(fd, SOL_SOCKET, option, &timeout, sizeof(timeout)) != 0) {
        z_fatal("could not configure socket timeout: %s", strerror(errno));
    }
}

static struct addrinfo *z_network_resolve(const char *host, int64_t port, int socktype, bool passive, const char *name) {
    z_network_require_port(port, name);
    char service[16];
    snprintf(service, sizeof(service), "%" PRId64, port);
    struct addrinfo hints;
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = socktype;
    hints.ai_flags = passive ? AI_PASSIVE : 0;
    struct addrinfo *result = NULL;
    const char *node = host != NULL && host[0] != '\0' ? host : NULL;
    int status = getaddrinfo(node, service, &hints, &result);
    if (status != 0) z_fatal("%s could not resolve %s:%" PRId64 ": %s", name, node == NULL ? "*" : node, port, gai_strerror(status));
    return result;
}

static int z_network_bound_socket(const char *host, int64_t port, int socktype, bool listening, const char *name) {
    struct addrinfo *addresses = z_network_resolve(host, port, socktype, true, name);
    int fd = -1;
    int last_error = 0;
    for (struct addrinfo *current = addresses; current != NULL; current = current->ai_next) {
        fd = socket(current->ai_family, current->ai_socktype, current->ai_protocol);
        if (fd < 0) { last_error = errno; continue; }
        int enabled = 1;
        (void)setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &enabled, sizeof(enabled));
        if (bind(fd, current->ai_addr, current->ai_addrlen) != 0) {
            last_error = errno;
            close(fd);
            fd = -1;
            continue;
        }
        if (listening && listen(fd, SOMAXCONN) != 0) {
            last_error = errno;
            close(fd);
            fd = -1;
            continue;
        }
        break;
    }
    freeaddrinfo(addresses);
    if (fd < 0) z_fatal("%s could not bind: %s", name, strerror(last_error));
    return fd;
}

static int z_network_connect_socket(const char *host, int64_t port, int64_t timeout_ms, const char *name) {
    struct addrinfo *addresses = z_network_resolve(host, port, SOCK_STREAM, false, name);
    int fd = -1;
    int last_error = 0;
    for (struct addrinfo *current = addresses; current != NULL; current = current->ai_next) {
        fd = socket(current->ai_family, current->ai_socktype, current->ai_protocol);
        if (fd < 0) { last_error = errno; continue; }
        int flags = fcntl(fd, F_GETFL, 0);
        if (timeout_ms >= 0 && flags >= 0) (void)fcntl(fd, F_SETFL, flags | O_NONBLOCK);
        int status = connect(fd, current->ai_addr, current->ai_addrlen);
        if (status != 0 && timeout_ms >= 0 && errno == EINPROGRESS) {
            int ready = z_network_wait_fd(fd, POLLOUT, timeout_ms);
            if (ready > 0) {
                socklen_t length = sizeof(last_error);
                if (getsockopt(fd, SOL_SOCKET, SO_ERROR, &last_error, &length) != 0) last_error = errno;
                status = last_error == 0 ? 0 : -1;
            } else {
                last_error = ready == 0 ? ETIMEDOUT : errno;
                status = -1;
            }
        } else if (status != 0) {
            last_error = errno;
        }
        if (timeout_ms >= 0 && flags >= 0) (void)fcntl(fd, F_SETFL, flags);
        if (status == 0) break;
        close(fd);
        fd = -1;
    }
    freeaddrinfo(addresses);
    if (fd < 0) z_fatal("%s could not connect to %s:%" PRId64 ": %s", name, host, port, strerror(last_error));
    return fd;
}

static void z_network_address(int fd, bool peer, char *host, size_t host_size, int64_t *port) {
    struct sockaddr_storage address;
    socklen_t length = sizeof(address);
    int status = peer ? getpeername(fd, (struct sockaddr *)&address, &length) : getsockname(fd, (struct sockaddr *)&address, &length);
    if (status != 0) z_fatal("could not inspect socket address: %s", strerror(errno));
    char service[32];
    status = getnameinfo((struct sockaddr *)&address, length, host, (socklen_t)host_size, service, sizeof(service), NI_NUMERICHOST | NI_NUMERICSERV);
    if (status != 0) z_fatal("could not format socket address: %s", gai_strerror(status));
    *port = strtoll(service, NULL, 10);
}

static ZNetListenerNative *z_expect_net_listener(ZValue value) {
    if (value.tag != ZV_NET_LISTENER || value.as.p == NULL) z_fatal("operation expects NetListener");
    return (ZNetListenerNative *)value.as.p;
}

static ZNetStreamNative *z_expect_net_stream(ZValue value) {
    if (value.tag != ZV_NET_STREAM || value.as.p == NULL) z_fatal("operation expects NetStream");
    return (ZNetStreamNative *)value.as.p;
}

static ZUDPSocketNative *z_expect_udp_socket(ZValue value) {
    if (value.tag != ZV_UDP_SOCKET || value.as.p == NULL) z_fatal("operation expects UDPSocket");
    return (ZUDPSocketNative *)value.as.p;
}

static ZValue z_net_listener_value(ZNetListenerNative *listener) {
    ZValue value = z_value(ZV_NET_LISTENER, ZK_NET_LISTENER);
    value.as.p = listener;
    return value;
}

static ZValue z_net_stream_value(ZNetStreamNative *stream) {
    ZValue value = z_value(ZV_NET_STREAM, ZK_NET_STREAM);
    value.as.p = stream;
    return value;
}

static ZValue z_udp_socket_value(ZUDPSocketNative *socket_value) {
    ZValue value = z_value(ZV_UDP_SOCKET, ZK_UDP_SOCKET);
    value.as.p = socket_value;
    return value;
}

static ZValue z_tcp_listen_native(const char *host, int64_t port) {
    ZNetListenerNative *listener = (ZNetListenerNative *)z_alloc(sizeof(ZNetListenerNative));
    listener->fd = z_network_bound_socket(host, port, SOCK_STREAM, true, "tcpListen");
    return z_net_listener_value(listener);
}

static ZValue z_tcp_connect_native(const char *host, int64_t port, int64_t timeout_ms) {
    ZNetStreamNative *stream = (ZNetStreamNative *)z_alloc(sizeof(ZNetStreamNative));
    stream->fd = z_network_connect_socket(host, port, timeout_ms, timeout_ms < 0 ? "tcpConnect" : "tcpConnectTimeout");
    return z_net_stream_value(stream);
}

#if defined(ZUMBRA_ENABLE_TLS)
static pthread_once_t z_tls_once = PTHREAD_ONCE_INIT;
static void z_tls_global_initialize(void) { (void)OPENSSL_init_ssl(0, NULL); }
static void z_tls_init_once(void) { pthread_once(&z_tls_once, z_tls_global_initialize); }
#endif

static ZValue z_tls_listen_native(const char *host, int64_t port, const char *certificate, const char *key) {
#if defined(ZUMBRA_ENABLE_TLS)
    z_tls_init_once();
    SSL_CTX *context = SSL_CTX_new(TLS_server_method());
    if (context == NULL) z_fatal("tlsListen could not create TLS context");
    SSL_CTX_set_min_proto_version(context, TLS1_2_VERSION);
    if (SSL_CTX_use_certificate_chain_file(context, certificate) != 1) {
        SSL_CTX_free(context);
        z_fatal("tlsListen could not load certificate %s", certificate);
    }
    if (SSL_CTX_use_PrivateKey_file(context, key, SSL_FILETYPE_PEM) != 1 || SSL_CTX_check_private_key(context) != 1) {
        SSL_CTX_free(context);
        z_fatal("tlsListen could not load private key %s", key);
    }
    ZNetListenerNative *listener = (ZNetListenerNative *)z_alloc(sizeof(ZNetListenerNative));
    listener->fd = z_network_bound_socket(host, port, SOCK_STREAM, true, "tlsListen");
    listener->tls = true;
    listener->tls_context = context;
    return z_net_listener_value(listener);
#else
    (void)host; (void)port; (void)certificate; (void)key;
    z_fatal("TLS support was not linked into this native build");
    return z_null();
#endif
}

static ZValue z_tls_connect_native(const char *host, int64_t port, const char *server_name, bool insecure, int64_t timeout_ms) {
#if defined(ZUMBRA_ENABLE_TLS)
    z_tls_init_once();
    int fd = z_network_connect_socket(host, port, timeout_ms, timeout_ms < 0 ? "tlsConnect" : "tlsConnectTimeout");
    SSL_CTX *context = SSL_CTX_new(TLS_client_method());
    if (context == NULL) { close(fd); z_fatal("tlsConnect could not create TLS context"); }
    SSL_CTX_set_min_proto_version(context, TLS1_2_VERSION);
    if (insecure) {
        SSL_CTX_set_verify(context, SSL_VERIFY_NONE, NULL);
    } else {
        SSL_CTX_set_verify(context, SSL_VERIFY_PEER, NULL);
        if (SSL_CTX_set_default_verify_paths(context) != 1) { SSL_CTX_free(context); close(fd); z_fatal("tlsConnect could not load system trust store"); }
    }
    SSL *session = SSL_new(context);
    if (session == NULL) { SSL_CTX_free(context); close(fd); z_fatal("tlsConnect could not create TLS session"); }
    const char *identity = server_name != NULL && server_name[0] != '\0' ? server_name : host;
    if (identity != NULL && identity[0] != '\0') {
        (void)SSL_set_tlsext_host_name(session, identity);
        if (!insecure && SSL_set1_host(session, identity) != 1) { SSL_free(session); SSL_CTX_free(context); close(fd); z_fatal("tlsConnect could not set hostname verification"); }
    }
    SSL_set_fd(session, fd);
    if (timeout_ms >= 0) { z_network_set_timeout(fd, SO_RCVTIMEO, timeout_ms); z_network_set_timeout(fd, SO_SNDTIMEO, timeout_ms); }
    if (SSL_connect(session) != 1) { SSL_free(session); SSL_CTX_free(context); close(fd); z_fatal("tlsConnect handshake failed"); }
    if (timeout_ms >= 0) { z_network_set_timeout(fd, SO_RCVTIMEO, -1); z_network_set_timeout(fd, SO_SNDTIMEO, -1); }
    ZNetStreamNative *stream = (ZNetStreamNative *)z_alloc(sizeof(ZNetStreamNative));
    stream->fd = fd;
    stream->tls = true;
    stream->tls_session = session;
    stream->owned_context = context;
    return z_net_stream_value(stream);
#else
    (void)host; (void)port; (void)server_name; (void)insecure; (void)timeout_ms;
    z_fatal("TLS support was not linked into this native build");
    return z_null();
#endif
}

static ZValue z_listener_accept_native(ZValue listener_value, int64_t timeout_ms, bool *accepted) {
    ZNetListenerNative *listener = z_expect_net_listener(listener_value);
    if (listener->closed) z_fatal("cannot accept on closed listener");
    if (timeout_ms >= 0) {
        int ready = z_network_wait_fd(listener->fd, POLLIN, timeout_ms);
        if (ready == 0) { *accepted = false; return z_null(); }
        if (ready < 0) z_fatal("listenerAcceptTimeout failed: %s", strerror(errno));
    }
    int fd;
    for (;;) {
        fd = accept(listener->fd, NULL, NULL);
        if (fd < 0 && errno == EINTR) continue;
        break;
    }
    if (fd < 0) z_fatal("listenerAccept failed: %s", strerror(errno));
    ZNetStreamNative *stream = (ZNetStreamNative *)z_alloc(sizeof(ZNetStreamNative));
    stream->fd = fd;
#if defined(ZUMBRA_ENABLE_TLS)
    if (listener->tls) {
        SSL *session = SSL_new(listener->tls_context);
        if (session == NULL) { close(fd); z_fatal("TLS accept could not create session"); }
        SSL_set_fd(session, fd);
        if (timeout_ms >= 0) { z_network_set_timeout(fd, SO_RCVTIMEO, timeout_ms); z_network_set_timeout(fd, SO_SNDTIMEO, timeout_ms); }
        if (SSL_accept(session) != 1) { SSL_free(session); close(fd); z_fatal("TLS server handshake failed"); }
        if (timeout_ms >= 0) { z_network_set_timeout(fd, SO_RCVTIMEO, -1); z_network_set_timeout(fd, SO_SNDTIMEO, -1); }
        stream->tls = true;
        stream->tls_session = session;
    }
#endif
    *accepted = true;
    return z_net_stream_value(stream);
}

static bool z_listener_close_native(ZValue value) {
    ZNetListenerNative *listener = z_expect_net_listener(value);
    if (listener->closed) return false;
    listener->closed = true;
    close(listener->fd);
#if defined(ZUMBRA_ENABLE_TLS)
    if (listener->tls_context != NULL) { SSL_CTX_free(listener->tls_context); listener->tls_context = NULL; }
#endif
    return true;
}

static ssize_t z_stream_read_once(ZNetStreamNative *stream, void *buffer, size_t length, bool *eof) {
    *eof = false;
#if defined(ZUMBRA_ENABLE_TLS)
    if (stream->tls) {
        int requested = length > INT_MAX ? INT_MAX : (int)length;
        int count = SSL_read(stream->tls_session, buffer, requested);
        if (count > 0) return count;
        int error = SSL_get_error(stream->tls_session, count);
        if (error == SSL_ERROR_ZERO_RETURN) { *eof = true; return 0; }
        if (error == SSL_ERROR_WANT_READ || error == SSL_ERROR_WANT_WRITE || (error == SSL_ERROR_SYSCALL && (errno == EAGAIN || errno == EWOULDBLOCK))) return -2;
        z_fatal("TLS stream read failed");
    }
#endif
    ssize_t count;
    do { count = recv(stream->fd, buffer, length, 0); } while (count < 0 && errno == EINTR);
    if (count == 0) *eof = true;
    if (count < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) return -2;
    if (count < 0) z_fatal("stream read failed: %s", strerror(errno));
    return count;
}

static ssize_t z_stream_write_once(ZNetStreamNative *stream, const void *buffer, size_t length) {
#if defined(ZUMBRA_ENABLE_TLS)
    if (stream->tls) {
        int requested = length > INT_MAX ? INT_MAX : (int)length;
        int count = SSL_write(stream->tls_session, buffer, requested);
        if (count > 0) return count;
        int error = SSL_get_error(stream->tls_session, count);
        if (error == SSL_ERROR_WANT_READ || error == SSL_ERROR_WANT_WRITE || (error == SSL_ERROR_SYSCALL && (errno == EAGAIN || errno == EWOULDBLOCK))) return -2;
        z_fatal("TLS stream write failed");
    }
#endif
    ssize_t count;
    do { count = send(stream->fd, buffer, length, 0); } while (count < 0 && errno == EINTR);
    if (count < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) return -2;
    if (count < 0) z_fatal("stream write failed: %s", strerror(errno));
    return count;
}

static ZValue z_stream_read_native(ZValue stream_value, int64_t requested, bool exact, int64_t timeout_ms, bool timeout_mode) {
    z_network_require_size(requested, exact ? "streamReadExact" : timeout_mode ? "streamReadTimeout" : "streamRead");
    ZNetStreamNative *stream = z_expect_net_stream(stream_value);
    if (stream->closed) z_fatal("cannot read a closed stream");
    if (timeout_mode) z_network_set_timeout(stream->fd, SO_RCVTIMEO, timeout_ms);
    ZValue result = z_bytes((size_t)requested);
    size_t total = 0;
    bool eof = false;
    bool timed_out = false;
    do {
        ssize_t count = z_stream_read_once(stream, (uint8_t *)result.as.buffer->data + total, (size_t)requested - total, &eof);
        if (count == -2) { timed_out = true; break; }
        total += (size_t)count;
        if (!exact || eof || count == 0) break;
    } while (total < (size_t)requested);
    if (timeout_mode) z_network_set_timeout(stream->fd, SO_RCVTIMEO, -1);
    result.as.buffer->len = total;
    if (exact && total != (size_t)requested) z_fatal("streamReadExact expected %" PRId64 " bytes, received %zu", requested, total);
    if (timeout_mode) {
        ZValue values[3] = {result, z_bool(!timed_out && total > 0), z_bool(eof)};
        return z_array_from(values, 3);
    }
    return result;
}

static void z_network_data(ZValue value, const uint8_t **data, size_t *length) {
    if (value.tag == ZV_STRING) { *data = (const uint8_t *)value.as.s; *length = strlen(value.as.s); return; }
    ZBuffer *buffer = z_expect_byte_buffer(value);
    *data = (const uint8_t *)buffer->data;
    *length = buffer->len;
}

static ZValue z_stream_write_native(ZValue stream_value, ZValue data_value, bool all) {
    ZNetStreamNative *stream = z_expect_net_stream(stream_value);
    if (stream->closed) z_fatal("cannot write a closed stream");
    const uint8_t *data = NULL;
    size_t length = 0;
    z_network_data(data_value, &data, &length);
    size_t total = 0;
    while (total < length) {
        ssize_t count = z_stream_write_once(stream, data + total, length - total);
        if (count == -2) z_fatal("stream write timed out");
        if (count == 0) z_fatal("stream write made no progress");
        total += (size_t)count;
        if (!all) break;
    }
    return z_int((int64_t)total);
}

static bool z_stream_close_native(ZValue value) {
    ZNetStreamNative *stream = z_expect_net_stream(value);
    if (stream->closed) return false;
    stream->closed = true;
#if defined(ZUMBRA_ENABLE_TLS)
    if (stream->tls_session != NULL) { (void)SSL_shutdown(stream->tls_session); SSL_free(stream->tls_session); stream->tls_session = NULL; }
    if (stream->owned_context != NULL) { SSL_CTX_free(stream->owned_context); stream->owned_context = NULL; }
#endif
    close(stream->fd);
    return true;
}

static ZValue z_dns_lookup_sync(const char *host) {
    struct addrinfo hints;
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    struct addrinfo *addresses = NULL;
    int status = getaddrinfo(host, NULL, &hints, &addresses);
    if (status != 0) z_fatal("dnsLookup %s: %s", host, gai_strerror(status));
    ZValue *items = NULL;
    size_t count = 0;
    size_t capacity = 0;
    for (struct addrinfo *current = addresses; current != NULL; current = current->ai_next) {
        char text[NI_MAXHOST];
        status = getnameinfo(current->ai_addr, current->ai_addrlen, text, sizeof(text), NULL, 0, NI_NUMERICHOST);
        if (status != 0) continue;
        bool duplicate = false;
        for (size_t index = 0; index < count; index++) if (strcmp(items[index].as.s, text) == 0) { duplicate = true; break; }
        if (duplicate) continue;
        if (count == capacity) {
            size_t next = capacity == 0 ? 4 : capacity * 2;
            ZValue *grown = (ZValue *)realloc(items, next * sizeof(ZValue));
            if (grown == NULL) { free(items); freeaddrinfo(addresses); z_fatal("out of memory resolving DNS"); }
            items = grown;
            capacity = next;
        }
        items[count++] = z_string(text);
    }
    freeaddrinfo(addresses);
    ZValue result = z_array_from(items, count);
    free(items);
    return result;
}

typedef struct {
    pthread_mutex_t mutex;
    pthread_cond_t condition;
    char *host;
    bool done;
    bool abandoned;
    int status;
    struct addrinfo *addresses;
} ZDNSRequest;

static void *z_dns_worker(void *raw) {
    ZDNSRequest *request = (ZDNSRequest *)raw;
    struct addrinfo hints;
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    struct addrinfo *addresses = NULL;
    int status = getaddrinfo(request->host, NULL, &hints, &addresses);
    pthread_mutex_lock(&request->mutex);
    if (request->abandoned) {
        pthread_mutex_unlock(&request->mutex);
        if (addresses != NULL) freeaddrinfo(addresses);
        free(request->host);
        pthread_cond_destroy(&request->condition);
        pthread_mutex_destroy(&request->mutex);
        free(request);
    } else {
        request->status = status;
        request->addresses = addresses;
        request->done = true;
        pthread_cond_signal(&request->condition);
        pthread_mutex_unlock(&request->mutex);
    }
    pthread_mutex_lock(&z_task_count_mutex);
    z_active_tasks--;
    pthread_cond_broadcast(&z_task_count_condition);
    pthread_mutex_unlock(&z_task_count_mutex);
    return NULL;
}

static ZValue z_dns_lookup_timeout_native(const char *host, int64_t timeout_ms) {
    if (timeout_ms < 0) z_fatal("dnsLookupTimeout timeout must be non-negative");
    ZDNSRequest *request = (ZDNSRequest *)calloc(1, sizeof(ZDNSRequest));
    if (request == NULL) z_fatal("out of memory starting DNS lookup");
    pthread_mutex_init(&request->mutex, NULL);
    pthread_cond_init(&request->condition, NULL);
    request->host = strdup(host);
    if (request->host == NULL) z_fatal("out of memory copying DNS host");
    pthread_t thread;
    pthread_mutex_lock(&z_task_count_mutex);
    z_active_tasks++;
    pthread_mutex_unlock(&z_task_count_mutex);
    if (pthread_create(&thread, NULL, z_dns_worker, request) != 0) z_fatal("could not create DNS worker");
    pthread_detach(thread);
    struct timespec deadline = z_deadline_after_ms(timeout_ms);
    pthread_mutex_lock(&request->mutex);
    int status = 0;
    while (!request->done && status != ETIMEDOUT) status = pthread_cond_timedwait(&request->condition, &request->mutex, &deadline);
    if (!request->done) {
        request->abandoned = true;
        pthread_mutex_unlock(&request->mutex);
        ZValue values[2] = {z_array_from(NULL, 0), z_bool(false)};
        return z_array_from(values, 2);
    }
    int lookup_status = request->status;
    struct addrinfo *addresses = request->addresses;
    pthread_mutex_unlock(&request->mutex);
    free(request->host);
    pthread_cond_destroy(&request->condition);
    pthread_mutex_destroy(&request->mutex);
    free(request);
    if (lookup_status != 0) { if (addresses != NULL) freeaddrinfo(addresses); z_fatal("dnsLookupTimeout %s: %s", host, gai_strerror(lookup_status)); }
    ZValue *items = NULL;
    size_t count = 0, capacity = 0;
    for (struct addrinfo *current = addresses; current != NULL; current = current->ai_next) {
        char text[NI_MAXHOST];
        if (getnameinfo(current->ai_addr, current->ai_addrlen, text, sizeof(text), NULL, 0, NI_NUMERICHOST) != 0) continue;
        bool duplicate = false;
        for (size_t index = 0; index < count; index++) if (strcmp(items[index].as.s, text) == 0) { duplicate = true; break; }
        if (duplicate) continue;
        if (count == capacity) {
            size_t next = capacity == 0 ? 4 : capacity * 2;
            ZValue *grown = (ZValue *)realloc(items, next * sizeof(ZValue));
            if (grown == NULL) { free(items); freeaddrinfo(addresses); z_fatal("out of memory resolving DNS"); }
            items = grown;
            capacity = next;
        }
        items[count++] = z_string(text);
    }
    freeaddrinfo(addresses);
    ZValue resolved = z_array_from(items, count);
    free(items);
    ZValue values[2] = {resolved, z_bool(true)};
    return z_array_from(values, 2);
}

static ZValue z_udp_bind_native(const char *host, int64_t port) {
    ZUDPSocketNative *socket_value = (ZUDPSocketNative *)z_alloc(sizeof(ZUDPSocketNative));
    socket_value->fd = z_network_bound_socket(host, port, SOCK_DGRAM, false, "udpBind");
    return z_udp_socket_value(socket_value);
}

static ZValue z_udp_send_to_native(ZValue socket_value, const char *host, int64_t port, ZValue data_value) {
    ZUDPSocketNative *socket_object = z_expect_udp_socket(socket_value);
    if (socket_object->closed) z_fatal("cannot send from a closed UDP socket");
    struct addrinfo *addresses = z_network_resolve(host, port, SOCK_DGRAM, false, "udpSendTo");
    const uint8_t *data = NULL;
    size_t length = 0;
    z_network_data(data_value, &data, &length);
    ssize_t written = sendto(socket_object->fd, data, length, 0, addresses->ai_addr, addresses->ai_addrlen);
    int error = errno;
    freeaddrinfo(addresses);
    if (written < 0) z_fatal("udpSendTo failed: %s", strerror(error));
    return z_int((int64_t)written);
}

static ZValue z_udp_receive_from_native(ZValue socket_value, int64_t requested, int64_t timeout_ms, bool timeout_mode) {
    z_network_require_size(requested, timeout_mode ? "udpReceiveFromTimeout" : "udpReceiveFrom");
    ZUDPSocketNative *socket_object = z_expect_udp_socket(socket_value);
    if (socket_object->closed) z_fatal("cannot receive from a closed UDP socket");
    if (timeout_mode) z_network_set_timeout(socket_object->fd, SO_RCVTIMEO, timeout_ms);
    ZValue bytes = z_bytes((size_t)requested);
    struct sockaddr_storage source;
    socklen_t source_length = sizeof(source);
    ssize_t count;
    do { count = recvfrom(socket_object->fd, bytes.as.buffer->data, (size_t)requested, 0, (struct sockaddr *)&source, &source_length); } while (count < 0 && errno == EINTR);
    if (timeout_mode) z_network_set_timeout(socket_object->fd, SO_RCVTIMEO, -1);
    if (count < 0 && timeout_mode && (errno == EAGAIN || errno == EWOULDBLOCK)) {
        ZValue values[4] = {z_bytes(0), z_string(""), z_int(0), z_bool(false)};
        return z_array_from(values, 4);
    }
    if (count < 0) z_fatal("udpReceiveFrom failed: %s", strerror(errno));
    bytes.as.buffer->len = (size_t)count;
    char host[NI_MAXHOST];
    char service[32];
    int status = getnameinfo((struct sockaddr *)&source, source_length, host, sizeof(host), service, sizeof(service), NI_NUMERICHOST | NI_NUMERICSERV);
    if (status != 0) z_fatal("udpReceiveFrom could not format source: %s", gai_strerror(status));
    ZValue values[4] = {bytes, z_string(host), z_int(strtoll(service, NULL, 10)), z_bool(true)};
    return z_array_from(values, timeout_mode ? 4 : 3);
}

static bool z_udp_close_native(ZValue value) {
    ZUDPSocketNative *socket_value = z_expect_udp_socket(value);
    if (socket_value->closed) return false;
    socket_value->closed = true;
    close(socket_value->fd);
    return true;
}

#endif

ZValue z_call_builtin(const char *name, const ZValue *args, size_t argc) {
#if defined(ZUMBRA_ENABLE_Z12)
    bool z12_handled = false;
    ZValue z12_result = z_z12_call_builtin(name, args, argc, &z12_handled);
    if (z12_handled) return z12_result;
#endif
#if defined(ZUMBRA_ENABLE_SQLITE)
    bool sqlite_handled = false;
    ZValue sqlite_result = z_sqlite_call_builtin(name, args, argc, &sqlite_handled);
    if (sqlite_handled) return sqlite_result;
#endif
#if defined(ZUMBRA_ENABLE_HTTP)
    bool http_handled = false;
    ZValue http_result = z_http_call_builtin(name, args, argc, &http_handled);
    if (http_handled) return http_result;
#endif
#if defined(ZUMBRA_ENABLE_NETWORK)
    if(strcmp(name,"tcpListen")==0){z_expect_args(name,argc,2);return z_tcp_listen_native(z_as_cstring(args[0]),z_as_i64(args[1]));}
    if(strcmp(name,"tcpConnect")==0){z_expect_args(name,argc,2);return z_tcp_connect_native(z_as_cstring(args[0]),z_as_i64(args[1]),-1);}
    if(strcmp(name,"tcpConnectTimeout")==0){z_expect_args(name,argc,3);return z_tcp_connect_native(z_as_cstring(args[0]),z_as_i64(args[1]),z_as_i64(args[2]));}
    if(strcmp(name,"tlsListen")==0){z_expect_args(name,argc,4);return z_tls_listen_native(z_as_cstring(args[0]),z_as_i64(args[1]),z_as_cstring(args[2]),z_as_cstring(args[3]));}
    if(strcmp(name,"tlsConnect")==0){z_expect_args(name,argc,4);return z_tls_connect_native(z_as_cstring(args[0]),z_as_i64(args[1]),z_as_cstring(args[2]),z_as_bool(args[3]),-1);}
    if(strcmp(name,"tlsConnectTimeout")==0){z_expect_args(name,argc,5);return z_tls_connect_native(z_as_cstring(args[0]),z_as_i64(args[1]),z_as_cstring(args[2]),z_as_bool(args[3]),z_as_i64(args[4]));}
    if(strcmp(name,"listenerAccept")==0){z_expect_args(name,argc,1);bool accepted=false;return z_listener_accept_native(args[0],-1,&accepted);}
    if(strcmp(name,"listenerAcceptTimeout")==0){z_expect_args(name,argc,2);bool accepted=false;ZValue stream=z_listener_accept_native(args[0],z_as_i64(args[1]),&accepted);ZValue values[2]={stream,z_bool(accepted)};return z_array_from(values,2);}
    if(strcmp(name,"listenerClose")==0){z_expect_args(name,argc,1);return z_bool(z_listener_close_native(args[0]));}
    if(strcmp(name,"listenerClosed")==0){z_expect_args(name,argc,1);return z_bool(z_expect_net_listener(args[0])->closed);}
    if(strcmp(name,"listenerAddress")==0||strcmp(name,"listenerPort")==0){z_expect_args(name,argc,1);ZNetListenerNative*l=z_expect_net_listener(args[0]);char host[NI_MAXHOST];int64_t port=0;z_network_address(l->fd,false,host,sizeof(host),&port);return strcmp(name,"listenerPort")==0?z_int(port):z_string(host);}
    if(strcmp(name,"streamRead")==0){z_expect_args(name,argc,2);return z_stream_read_native(args[0],z_as_i64(args[1]),false,-1,false);}
    if(strcmp(name,"streamReadExact")==0){z_expect_args(name,argc,2);return z_stream_read_native(args[0],z_as_i64(args[1]),true,-1,false);}
    if(strcmp(name,"streamReadTimeout")==0){z_expect_args(name,argc,3);return z_stream_read_native(args[0],z_as_i64(args[1]),false,z_as_i64(args[2]),true);}
    if(strcmp(name,"streamWrite")==0){z_expect_args(name,argc,2);return z_stream_write_native(args[0],args[1],false);}
    if(strcmp(name,"streamWriteAll")==0){z_expect_args(name,argc,2);return z_stream_write_native(args[0],args[1],true);}
    if(strcmp(name,"streamClose")==0){z_expect_args(name,argc,1);return z_bool(z_stream_close_native(args[0]));}
    if(strcmp(name,"streamClosed")==0){z_expect_args(name,argc,1);return z_bool(z_expect_net_stream(args[0])->closed);}
    if(strcmp(name,"streamShutdownRead")==0||strcmp(name,"streamShutdownWrite")==0){z_expect_args(name,argc,1);ZNetStreamNative*s=z_expect_net_stream(args[0]);int how=strcmp(name,"streamShutdownWrite")==0?SHUT_WR:SHUT_RD;if(shutdown(s->fd,how)!=0)z_fatal("%s failed: %s",name,strerror(errno));return z_bool(true);}
    if(strcmp(name,"streamLocalAddress")==0||strcmp(name,"streamLocalPort")==0||strcmp(name,"streamRemoteAddress")==0||strcmp(name,"streamRemotePort")==0){z_expect_args(name,argc,1);ZNetStreamNative*s=z_expect_net_stream(args[0]);bool remote=strstr(name,"Remote")!=NULL;bool want_port=strstr(name,"Port")!=NULL;char host[NI_MAXHOST];int64_t port=0;z_network_address(s->fd,remote,host,sizeof(host),&port);return want_port?z_int(port):z_string(host);}
    if(strcmp(name,"streamSetReadTimeout")==0||strcmp(name,"streamSetWriteTimeout")==0){z_expect_args(name,argc,2);ZNetStreamNative*s=z_expect_net_stream(args[0]);z_network_set_timeout(s->fd,strcmp(name,"streamSetReadTimeout")==0?SO_RCVTIMEO:SO_SNDTIMEO,z_as_i64(args[1]));return z_null();}
    if(strcmp(name,"tcpSetKeepAlive")==0){z_expect_args(name,argc,3);ZNetStreamNative*s=z_expect_net_stream(args[0]);int enabled=z_as_bool(args[1])?1:0;if(setsockopt(s->fd,SOL_SOCKET,SO_KEEPALIVE,&enabled,sizeof(enabled))!=0)z_fatal("tcpSetKeepAlive failed: %s",strerror(errno));
#ifdef TCP_KEEPIDLE
        if(enabled){int seconds=(int)((z_as_i64(args[2])+999)/1000);if(seconds<1)seconds=1;if(setsockopt(s->fd,IPPROTO_TCP,TCP_KEEPIDLE,&seconds,sizeof(seconds))!=0)z_fatal("tcpSetKeepAlive idle failed: %s",strerror(errno));}
#endif
        return z_null();}
    if(strcmp(name,"dnsLookup")==0){z_expect_args(name,argc,1);return z_dns_lookup_sync(z_as_cstring(args[0]));}
    if(strcmp(name,"dnsLookupTimeout")==0){z_expect_args(name,argc,2);return z_dns_lookup_timeout_native(z_as_cstring(args[0]),z_as_i64(args[1]));}
    if(strcmp(name,"udpBind")==0){z_expect_args(name,argc,2);return z_udp_bind_native(z_as_cstring(args[0]),z_as_i64(args[1]));}
    if(strcmp(name,"udpSendTo")==0){z_expect_args(name,argc,4);return z_udp_send_to_native(args[0],z_as_cstring(args[1]),z_as_i64(args[2]),args[3]);}
    if(strcmp(name,"udpReceiveFrom")==0){z_expect_args(name,argc,2);return z_udp_receive_from_native(args[0],z_as_i64(args[1]),-1,false);}
    if(strcmp(name,"udpReceiveFromTimeout")==0){z_expect_args(name,argc,3);return z_udp_receive_from_native(args[0],z_as_i64(args[1]),z_as_i64(args[2]),true);}
    if(strcmp(name,"udpClose")==0){z_expect_args(name,argc,1);return z_bool(z_udp_close_native(args[0]));}
    if(strcmp(name,"udpClosed")==0){z_expect_args(name,argc,1);return z_bool(z_expect_udp_socket(args[0])->closed);}
    if(strcmp(name,"udpAddress")==0||strcmp(name,"udpPort")==0){z_expect_args(name,argc,1);ZUDPSocketNative*u=z_expect_udp_socket(args[0]);char host[NI_MAXHOST];int64_t port=0;z_network_address(u->fd,false,host,sizeof(host),&port);return strcmp(name,"udpPort")==0?z_int(port):z_string(host);}
#endif
    if(strcmp(name,"join")==0){z_expect_args(name,argc,1);return z_task_await(args[0]);}
    if(strcmp(name,"cancel")==0){z_expect_args(name,argc,1);return z_bool(z_task_cancel(args[0]));}
    if(strcmp(name,"taskDone")==0){z_expect_args(name,argc,1);return z_bool(z_task_done(args[0]));}
    if(strcmp(name,"taskCancelled")==0){z_expect_args(name,argc,1);return z_bool(z_task_cancelled(args[0]));}
    if(strcmp(name,"joinTimeout")==0){z_expect_args(name,argc,2);bool completed=false;ZValue value=z_task_await_timeout(args[0],z_as_i64(args[1]),&completed);ZValue pair[2]={value,z_bool(completed)};return z_array_from(pair,2);}
    if(strcmp(name,"sleepMs")==0){z_expect_args(name,argc,1);z_sleep_ms(z_as_i64(args[0]));return z_null();}
    if(strcmp(name,"channel")==0){if(argc>1)z_fatal("channel expects zero or one argument");return z_channel_new(argc==0?0:(size_t)z_as_u64(args[0]));}
    if(strcmp(name,"send")==0){z_expect_args(name,argc,2);z_channel_send(args[0],args[1]);return z_null();}
    if(strcmp(name,"receive")==0){z_expect_args(name,argc,1);bool open=false;return z_channel_receive(args[0],&open);}
    if(strcmp(name,"receiveOk")==0){z_expect_args(name,argc,1);bool open=false;ZValue value=z_channel_receive(args[0],&open);ZValue pair[2]={value,z_bool(open)};return z_array_from(pair,2);}
    if(strcmp(name,"receiveTimeout")==0){z_expect_args(name,argc,2);bool open=false,received=false;ZValue value=z_channel_receive_timeout(args[0],z_as_i64(args[1]),&open,&received);ZValue triple[3]={value,z_bool(open),z_bool(received)};return z_array_from(triple,3);}
    if(strcmp(name,"closeChannel")==0){z_expect_args(name,argc,1);return z_bool(z_channel_close(args[0]));}
    if(strcmp(name,"channelClosed")==0){z_expect_args(name,argc,1);ZChannelNative*c=z_expect_channel(args[0]);pthread_mutex_lock(&c->mutex);bool closed=c->closed;pthread_mutex_unlock(&c->mutex);return z_bool(closed);}
    if(strcmp(name,"channelLen")==0){z_expect_args(name,argc,1);ZChannelNative*c=z_expect_channel(args[0]);pthread_mutex_lock(&c->mutex);size_t count=c->count;pthread_mutex_unlock(&c->mutex);return z_int((int64_t)count);}
    if(strcmp(name,"channelCap")==0){z_expect_args(name,argc,1);return z_int((int64_t)z_expect_channel(args[0])->capacity);}
    if(strcmp(name,"mutex")==0){z_expect_args(name,argc,0);ZMutexNative*m=(ZMutexNative*)z_alloc(sizeof(ZMutexNative));pthread_mutex_init(&m->mutex,NULL);ZValue v=z_value(ZV_MUTEX,ZK_MUTEX);v.as.p=m;return v;}
    if(strcmp(name,"lock")==0){z_expect_args(name,argc,1);if(args[0].tag==ZV_MUTEX)pthread_mutex_lock(&((ZMutexNative*)args[0].as.p)->mutex);else if(args[0].tag==ZV_RW_MUTEX)pthread_rwlock_wrlock(&((ZRWMutexNative*)args[0].as.p)->mutex);else z_fatal("lock expects Mutex or RWMutex");return z_null();}
    if(strcmp(name,"unlock")==0){z_expect_args(name,argc,1);if(args[0].tag==ZV_MUTEX)pthread_mutex_unlock(&((ZMutexNative*)args[0].as.p)->mutex);else if(args[0].tag==ZV_RW_MUTEX)pthread_rwlock_unlock(&((ZRWMutexNative*)args[0].as.p)->mutex);else z_fatal("unlock expects Mutex or RWMutex");return z_null();}
    if(strcmp(name,"rwMutex")==0){z_expect_args(name,argc,0);ZRWMutexNative*m=(ZRWMutexNative*)z_alloc(sizeof(ZRWMutexNative));pthread_rwlock_init(&m->mutex,NULL);ZValue v=z_value(ZV_RW_MUTEX,ZK_RW_MUTEX);v.as.p=m;return v;}
    if(strcmp(name,"rLock")==0){z_expect_args(name,argc,1);if(args[0].tag!=ZV_RW_MUTEX)z_fatal("rLock expects RWMutex");pthread_rwlock_rdlock(&((ZRWMutexNative*)args[0].as.p)->mutex);return z_null();}
    if(strcmp(name,"rUnlock")==0){z_expect_args(name,argc,1);if(args[0].tag!=ZV_RW_MUTEX)z_fatal("rUnlock expects RWMutex");pthread_rwlock_unlock(&((ZRWMutexNative*)args[0].as.p)->mutex);return z_null();}
    if(strcmp(name,"waitGroup")==0){z_expect_args(name,argc,0);ZWaitGroupNative*w=(ZWaitGroupNative*)z_alloc(sizeof(ZWaitGroupNative));pthread_mutex_init(&w->mutex,NULL);pthread_cond_init(&w->condition,NULL);w->count=0;ZValue v=z_value(ZV_WAIT_GROUP,ZK_WAIT_GROUP);v.as.p=w;return v;}
    if(strcmp(name,"wgAdd")==0){z_expect_args(name,argc,2);if(args[0].tag!=ZV_WAIT_GROUP)z_fatal("wgAdd expects WaitGroup");ZWaitGroupNative*w=(ZWaitGroupNative*)args[0].as.p;pthread_mutex_lock(&w->mutex);w->count+=z_as_i64(args[1]);if(w->count<0){pthread_mutex_unlock(&w->mutex);z_fatal("negative WaitGroup counter");}if(w->count==0)pthread_cond_broadcast(&w->condition);pthread_mutex_unlock(&w->mutex);return z_null();}
    if(strcmp(name,"wgDone")==0){ZValue call[2]={args[0],z_int(-1)};z_expect_args(name,argc,1);return z_call_builtin("wgAdd",call,2);}
    if(strcmp(name,"wgWait")==0){z_expect_args(name,argc,1);if(args[0].tag!=ZV_WAIT_GROUP)z_fatal("wgWait expects WaitGroup");ZWaitGroupNative*w=(ZWaitGroupNative*)args[0].as.p;pthread_mutex_lock(&w->mutex);while(w->count!=0)pthread_cond_wait(&w->condition,&w->mutex);pthread_mutex_unlock(&w->mutex);return z_null();}
    if(strcmp(name,"semaphore")==0){z_expect_args(name,argc,1);int64_t n=z_as_i64(args[0]);if(n<1)z_fatal("semaphore capacity must be positive");ZSemaphoreNative*q=(ZSemaphoreNative*)z_alloc(sizeof(ZSemaphoreNative));pthread_mutex_init(&q->mutex,NULL);pthread_cond_init(&q->condition,NULL);q->available=n;q->capacity=n;ZValue v=z_value(ZV_SEMAPHORE,ZK_SEMAPHORE);v.as.p=q;return v;}
    if(strcmp(name,"acquire")==0){z_expect_args(name,argc,1);if(args[0].tag!=ZV_SEMAPHORE)z_fatal("acquire expects Semaphore");ZSemaphoreNative*q=(ZSemaphoreNative*)args[0].as.p;pthread_mutex_lock(&q->mutex);while(q->available==0)pthread_cond_wait(&q->condition,&q->mutex);q->available--;pthread_mutex_unlock(&q->mutex);return z_null();}
    if(strcmp(name,"release")==0){z_expect_args(name,argc,1);if(args[0].tag!=ZV_SEMAPHORE)z_fatal("release expects Semaphore");ZSemaphoreNative*q=(ZSemaphoreNative*)args[0].as.p;pthread_mutex_lock(&q->mutex);if(q->available==q->capacity){pthread_mutex_unlock(&q->mutex);z_fatal("release called without a matching acquire");}q->available++;pthread_cond_signal(&q->condition);pthread_mutex_unlock(&q->mutex);return z_null();}
    if(strcmp(name,"atomicInt")==0){if(argc>1)z_fatal("atomicInt expects zero or one argument");_Atomic int64_t*a=(_Atomic int64_t*)z_alloc(sizeof(_Atomic int64_t));atomic_init(a,argc==0?0:z_as_i64(args[0]));ZValue v=z_value(ZV_ATOMIC_INT,ZK_ATOMIC_INT);v.as.p=a;return v;}
    if(strcmp(name,"atomicLoad")==0){z_expect_args(name,argc,1);if(args[0].tag!=ZV_ATOMIC_INT)z_fatal("atomicLoad expects AtomicInt");return z_int(atomic_load((_Atomic int64_t*)args[0].as.p));}
    if(strcmp(name,"atomicStore")==0){z_expect_args(name,argc,2);if(args[0].tag!=ZV_ATOMIC_INT)z_fatal("atomicStore expects AtomicInt");atomic_store((_Atomic int64_t*)args[0].as.p,z_as_i64(args[1]));return z_null();}
    if(strcmp(name,"atomicAdd")==0){z_expect_args(name,argc,2);if(args[0].tag!=ZV_ATOMIC_INT)z_fatal("atomicAdd expects AtomicInt");return z_int(atomic_fetch_add((_Atomic int64_t*)args[0].as.p,z_as_i64(args[1]))+z_as_i64(args[1]));}
    if(strcmp(name,"atomicSwap")==0){z_expect_args(name,argc,2);if(args[0].tag!=ZV_ATOMIC_INT)z_fatal("atomicSwap expects AtomicInt");return z_int(atomic_exchange((_Atomic int64_t*)args[0].as.p,z_as_i64(args[1])));}
    if(strcmp(name,"atomicCompareSwap")==0){z_expect_args(name,argc,3);if(args[0].tag!=ZV_ATOMIC_INT)z_fatal("atomicCompareSwap expects AtomicInt");int64_t expected=z_as_i64(args[1]);return z_bool(atomic_compare_exchange_strong((_Atomic int64_t*)args[0].as.p,&expected,z_as_i64(args[2])));}
    if (strcmp(name,"show")==0){z_expect_args(name,argc,1);z_show(args[0]);return z_null();}
    if (strcmp(name,"toString")==0){z_expect_args(name,argc,1);return z_to_string_value(args[0]);}
    if (strcmp(name,"sizeOf")==0){z_expect_args(name,argc,1);return z_int((int64_t)z_size_of(args[0]));}
    if (strcmp(name,"bytes")==0){z_expect_args(name,argc,1);return z_bytes((size_t)z_as_u64(args[0]));}
    if (strcmp(name,"arrayOf")==0){z_expect_args(name,argc,2);if(args[0].tag!=ZV_STRING)z_fatal("arrayOf type must be a string");return z_array_of(args[0].as.s,(size_t)z_as_u64(args[1]));}
    if (strcmp(name,"slice")==0){z_expect_args(name,argc,3);return z_slice(args[0],(size_t)z_as_u64(args[1]),(size_t)z_as_u64(args[2]));}
    if (strcmp(name,"fill")==0){z_expect_args(name,argc,2);z_fill(args[0],args[1]);return args[0];}
    if (strcmp(name,"readBytes")==0){z_expect_args(name,argc,1);if(args[0].tag!=ZV_STRING)z_fatal("readBytes path must be a string");return z_read_bytes(args[0].as.s);}
    if (strcmp(name,"writeBytes")==0){z_expect_args(name,argc,2);if(args[0].tag!=ZV_STRING)z_fatal("writeBytes path must be a string");return z_int((int64_t)z_write_bytes(args[0].as.s,args[1]));}
    if (strcmp(name,"copyBytes")==0){z_expect_args(name,argc,5);z_copy_bytes(args[0],(size_t)z_as_u64(args[1]),args[2],(size_t)z_as_u64(args[3]),(size_t)z_as_u64(args[4]));return args[0];}
    if (strcmp(name,"bytesEqual")==0){z_expect_args(name,argc,2);return z_bool(z_bytes_equal(args[0],args[1]));}
    if (strcmp(name,"sha256")==0){z_expect_args(name,argc,1);return z_sha256(args[0]);}
    if (strncmp(name,"readU",5)==0){z_expect_args(name,argc,2);unsigned bits=(unsigned)strtoul(name+5,NULL,10);bool le=strstr(name,"LE")!=NULL;return z_read_uint(args[0],(size_t)z_as_u64(args[1]),bits,le);}
    if (strncmp(name,"writeU",6)==0){z_expect_args(name,argc,3);unsigned bits=(unsigned)strtoul(name+6,NULL,10);bool le=strstr(name,"LE")!=NULL;z_write_uint(args[0],(size_t)z_as_u64(args[1]),args[2],bits,le);return args[0];}
    if (strcmp(name,"u8")==0){z_expect_args(name,argc,1);return z_convert(args[0],ZK_U8);} if(strcmp(name,"u16")==0){z_expect_args(name,argc,1);return z_convert(args[0],ZK_U16);} if(strcmp(name,"u32")==0){z_expect_args(name,argc,1);return z_convert(args[0],ZK_U32);} if(strcmp(name,"u64")==0){z_expect_args(name,argc,1);return z_convert(args[0],ZK_U64);}
    if (strcmp(name,"i8")==0){z_expect_args(name,argc,1);return z_convert(args[0],ZK_I8);} if(strcmp(name,"i16")==0){z_expect_args(name,argc,1);return z_convert(args[0],ZK_I16);} if(strcmp(name,"i32")==0){z_expect_args(name,argc,1);return z_convert(args[0],ZK_I32);} if(strcmp(name,"i64")==0){z_expect_args(name,argc,1);return z_convert(args[0],ZK_I64);}
    if(strcmp(name,"toInt")==0){z_expect_args(name,argc,1);return z_convert(args[0],ZK_INT);} if(strcmp(name,"toFloat")==0){z_expect_args(name,argc,1);return z_convert(args[0],ZK_FLOAT);} if(strcmp(name,"toBool")==0){z_expect_args(name,argc,1);return z_convert(args[0],ZK_BOOL);}
    if(strcmp(name,"wrapAdd")==0){z_expect_args(name,argc,2);return z_binary("+",args[0],args[1],args[0].kind);} if(strcmp(name,"wrapSub")==0){z_expect_args(name,argc,2);return z_binary("-",args[0],args[1],args[0].kind);} if(strcmp(name,"wrapMul")==0){z_expect_args(name,argc,2);return z_binary("*",args[0],args[1],args[0].kind);}
    if(strcmp(name,"checkedAdd")==0){z_expect_args(name,argc,2);return z_checked_arithmetic(name,args[0],args[1],'+',false);} if(strcmp(name,"checkedSub")==0){z_expect_args(name,argc,2);return z_checked_arithmetic(name,args[0],args[1],'-',false);} if(strcmp(name,"checkedMul")==0){z_expect_args(name,argc,2);return z_checked_arithmetic(name,args[0],args[1],'*',false);}
    if(strcmp(name,"satAdd")==0){z_expect_args(name,argc,2);return z_checked_arithmetic(name,args[0],args[1],'+',true);} if(strcmp(name,"satSub")==0){z_expect_args(name,argc,2);return z_checked_arithmetic(name,args[0],args[1],'-',true);} if(strcmp(name,"satMul")==0){z_expect_args(name,argc,2);return z_checked_arithmetic(name,args[0],args[1],'*',true);}
    z_fatal("builtin %s is not available in the native runtime yet",name);
    return z_null();
}
