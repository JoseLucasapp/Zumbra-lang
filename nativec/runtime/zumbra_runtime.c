#include "zumbra_runtime.h"

#include <errno.h>
#include <inttypes.h>
#include <math.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#if defined(_WIN32)
#include <windows.h>
#endif

typedef struct ZAllocation {
    void *pointer;
    struct ZAllocation *next;
} ZAllocation;

static ZAllocation *z_allocations = NULL;

void z_runtime_init(void) { z_allocations = NULL; }

void z_runtime_shutdown(void) {
    ZAllocation *current = z_allocations;
    while (current != NULL) {
        ZAllocation *next = current->next;
        free(current->pointer);
        free(current);
        current = next;
    }
    z_allocations = NULL;
}

void z_fatal(const char *format, ...) {
    va_list arguments;
    fprintf(stderr, "zumbra runtime error: ");
    va_start(arguments, format);
    vfprintf(stderr, format, arguments);
    va_end(arguments);
    fputc('\n', stderr);
    z_runtime_shutdown();
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
    entry->next = z_allocations;
    z_allocations = entry;
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

static int64_t z_as_i64(ZValue value) {
    switch (value.tag) {
    case ZV_INT: return value.as.i;
    case ZV_UINT: return (int64_t)value.as.u;
    case ZV_FLOAT: return (int64_t)value.as.f;
    case ZV_BOOL: return value.as.b ? 1 : 0;
    default: z_fatal("expected numeric value"); return 0;
    }
}

static uint64_t z_as_u64(ZValue value) {
    switch (value.tag) {
    case ZV_INT: return (uint64_t)value.as.i;
    case ZV_UINT: return value.as.u;
    case ZV_FLOAT: return (uint64_t)value.as.f;
    case ZV_BOOL: return value.as.b ? 1u : 0u;
    default: z_fatal("expected numeric value"); return 0;
    }
}

static double z_as_f64(ZValue value) {
    switch (value.tag) {
    case ZV_INT: return (double)value.as.i;
    case ZV_UINT: return (double)value.as.u;
    case ZV_FLOAT: return value.as.f;
    case ZV_BOOL: return value.as.b ? 1.0 : 0.0;
    default: z_fatal("expected numeric value"); return 0.0;
    }
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

ZValue z_call(ZValue callable, const ZValue *args, size_t argc) {
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

static void z_show_inner(ZValue value) {
    switch (value.tag) {
    case ZV_NULL: fputs("null", stdout); break;
    case ZV_INT: fprintf(stdout, "%" PRId64, value.as.i); break;
    case ZV_UINT: fprintf(stdout, "%" PRIu64, value.as.u); break;
    case ZV_FLOAT: fprintf(stdout, "%.15g", value.as.f); break;
    case ZV_BOOL: fputs(value.as.b ? "true" : "false", stdout); break;
    case ZV_STRING: fputs(value.as.s, stdout); break;
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

ZValue z_call_builtin(const char *name, const ZValue *args, size_t argc) {
    if (strcmp(name,"show")==0){z_expect_args(name,argc,1);z_show(args[0]);return z_null();}
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
