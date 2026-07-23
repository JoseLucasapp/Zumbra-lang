#ifndef ZUMBRA_RUNTIME_H
#define ZUMBRA_RUNTIME_H

#define ZUMBRA_NATIVE_ABI_VERSION 1u

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

typedef enum {
    ZK_UNKNOWN = 0,
    ZK_INT,
    ZK_U8,
    ZK_U16,
    ZK_U32,
    ZK_U64,
    ZK_I8,
    ZK_I16,
    ZK_I32,
    ZK_I64,
    ZK_FLOAT,
    ZK_BOOL,
    ZK_STRING,
    ZK_NULL,
    ZK_ARRAY,
    ZK_BYTE_ARRAY,
    ZK_TYPED_ARRAY,
    ZK_SLICE,
    ZK_DICT,
    ZK_STRUCT,
    ZK_ENUM,
    ZK_FUNCTION,
    ZK_POINTER
} ZKind;

typedef enum {
    ZV_NULL = 0,
    ZV_INT,
    ZV_UINT,
    ZV_FLOAT,
    ZV_BOOL,
    ZV_STRING,
    ZV_ARRAY,
    ZV_DICT,
    ZV_PAIR,
    ZV_BUFFER,
    ZV_STRUCT,
    ZV_ENUM,
    ZV_FUNCTION,
    ZV_BOUND_METHOD,
    ZV_BUILTIN,
    ZV_STRUCT_TYPE,
    ZV_ENUM_TYPE,
    ZV_POINTER
} ZTag;

typedef struct ZValue ZValue;
typedef struct ZArray ZArray;
typedef struct ZDict ZDict;
typedef struct ZPair ZPair;
typedef struct ZBuffer ZBuffer;
typedef struct ZStruct ZStruct;
typedef struct ZBoundMethod ZBoundMethod;

struct ZValue {
    ZTag tag;
    ZKind kind;
    union {
        int64_t i;
        uint64_t u;
        double f;
        bool b;
        const char *s;
        ZArray *array;
        ZDict *dict;
        ZPair *pair;
        ZBuffer *buffer;
        ZStruct *structure;
        ZBoundMethod *method;
        void *p;
        int id;
    } as;
};

struct ZArray {
    size_t len;
    size_t cap;
    ZValue *items;
};

struct ZDict {
    size_t len;
    size_t cap;
    ZValue *keys;
    ZValue *values;
};

struct ZPair {
    ZValue key;
    ZValue value;
};

struct ZBuffer {
    void *data;
    size_t len;
    size_t elem_size;
    ZKind elem_kind;
};

struct ZStruct {
    int type_id;
    size_t field_count;
    ZValue *fields;
};

struct ZBoundMethod {
    int function_id;
    ZStruct *receiver;
};

uint32_t z_abi_version(void);
void z_runtime_init(void);
void z_runtime_shutdown(void);
void z_fatal(const char *format, ...);
void *z_alloc(size_t size);
char *z_strdup(const char *value);

ZValue z_null(void);
ZValue z_int(int64_t value);
ZValue z_uint(uint64_t value, ZKind kind);
ZValue z_signed(int64_t value, ZKind kind);
ZValue z_float(double value);
ZValue z_bool(bool value);
ZValue z_string(const char *value);
ZValue z_pointer(void *value);
ZValue z_function(int id);
ZValue z_builtin(const char *name);
ZValue z_struct_type(int id);
ZValue z_enum_type(int id);
ZValue z_enum(int type_id, int ordinal);
ZValue z_bound_method(int function_id, ZStruct *receiver);

bool z_truthy(ZValue value);
bool z_equal(ZValue left, ZValue right);
ZValue z_unary(const char *op, ZValue value, ZKind target);
ZValue z_binary(const char *op, ZValue left, ZValue right, ZKind target);
ZValue z_convert(ZValue value, ZKind target);
int64_t z_as_i64(ZValue value);
uint64_t z_as_u64(ZValue value);
double z_as_f64(ZValue value);
bool z_as_bool(ZValue value);
const char *z_as_cstring(ZValue value);
void *z_as_pointer(ZValue value);

ZValue z_array_from(const ZValue *items, size_t count);
ZValue z_pair(ZValue key, ZValue value);
ZValue z_dict_from(const ZValue *pairs, size_t count);
ZValue z_index(ZValue collection, ZValue index);
void z_set_index(ZValue collection, ZValue index, ZValue value);
ZValue z_slice(ZValue collection, size_t start, size_t end);
void z_fill(ZValue collection, ZValue value);
size_t z_size_of(ZValue value);

ZValue z_get_field(ZValue object, const char *name);
void z_set_field(ZValue object, const char *name, ZValue value);

ZValue z_call(ZValue callable, const ZValue *args, size_t argc);
ZValue z_call_builtin(const char *name, const ZValue *args, size_t argc);
void z_show(ZValue value);

ZValue z_bytes(size_t size);
ZValue z_array_of(const char *kind, size_t size);
ZValue z_read_bytes(const char *path);
size_t z_write_bytes(const char *path, ZValue buffer);
ZValue z_read_uint(ZValue buffer, size_t offset, unsigned bits, bool little_endian);
void z_write_uint(ZValue buffer, size_t offset, ZValue value, unsigned bits, bool little_endian);
void z_copy_bytes(ZValue destination, size_t destination_start, ZValue source, size_t source_start, size_t length);
bool z_bytes_equal(ZValue left, ZValue right);
ZValue z_sha256(ZValue buffer);

/* Hooks emitted by the MIR native backend. */
ZValue z_dispatch_function(int function_id, const ZValue *args, size_t argc);
ZValue z_construct_struct(int type_id, const ZValue *args, size_t argc);
int z_struct_field_index(int type_id, const char *name);
int z_struct_method_id(int type_id, const char *name);
int z_enum_member_ordinal(int type_id, const char *name);
const char *z_struct_type_name(int type_id);
const char *z_enum_type_name(int type_id);
const char *z_enum_member_name(int type_id, int ordinal);

#endif
