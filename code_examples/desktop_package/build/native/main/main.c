/* Generated from Zumbra MIR. Do not edit manually. */
#include "zumbra_runtime.h"
#include <stdio.h>
#include <string.h>
_Static_assert(ZUMBRA_NATIVE_ABI_VERSION == 6u, "unsupported Zumbra native ABI");

static ZValue zg___zm_3ca93cbc8d3e4342_bytes;
static ZValue zg___zm_3ca93cbc8d3e4342_exists;
static ZValue zg___zm_3ca93cbc8d3e4342_list;
static ZValue zg___zm_3ca93cbc8d3e4342_text;

static ZValue zf_0(const ZValue *args, size_t argc);
static ZValue zf_1(const ZValue *args, size_t argc);
static ZValue zf_2(const ZValue *args, size_t argc);
static ZValue zf_3(const ZValue *args, size_t argc);

const char *z_struct_type_name(int type_id) {
    switch (type_id) {
        default: return "<unknown-struct>";
    }
}

const char *z_enum_type_name(int type_id) {
    switch (type_id) {
        default: return "<unknown-enum>";
    }
}

const char *z_enum_member_name(int type_id, int ordinal) {
    switch (type_id) {
        default: return "<unknown-member>";
    }
}

int z_enum_member_ordinal(int type_id, const char *name) {
    switch (type_id) {
        default: return -1;
    }
}

int z_struct_field_index(int type_id, const char *name) {
    switch (type_id) {
        default: return -1;
    }
}

int z_struct_method_id(int type_id, const char *name) {
    switch (type_id) {
        default: return -1;
    }
}

ZValue z_construct_struct(int type_id, const ZValue *args, size_t argc) {
    switch (type_id) {
        default: z_fatal("unknown struct type id %d", type_id); return z_null();
    }
}

static ZValue zf_0(const ZValue *args, size_t argc) {
    if (argc != 1) z_fatal("__zm_3ca93cbc8d3e4342_exists expects 1 arguments, got %zu", argc);
    ZValue zp_name_0 = args[0];
    ZValue zv_1 = z_builtin("assetExists");
    ZValue zv_2 = zp_name_0;
    ZValue za_1[] = {zv_2};
    ZValue zv_3 = z_call(zv_1, za_1, 1);
    return zv_3;
}

static ZValue zf_1(const ZValue *args, size_t argc) {
    if (argc != 1) z_fatal("__zm_3ca93cbc8d3e4342_text expects 1 arguments, got %zu", argc);
    ZValue zp_name_0 = args[0];
    ZValue zv_5 = z_builtin("assetText");
    ZValue zv_6 = zp_name_0;
    ZValue za_7[] = {zv_6};
    ZValue zv_7 = z_call(zv_5, za_7, 1);
    return zv_7;
}

static ZValue zf_2(const ZValue *args, size_t argc) {
    if (argc != 1) z_fatal("__zm_3ca93cbc8d3e4342_bytes expects 1 arguments, got %zu", argc);
    ZValue zp_name_0 = args[0];
    ZValue zv_9 = z_builtin("assetBytes");
    ZValue zv_10 = zp_name_0;
    ZValue za_13[] = {zv_10};
    ZValue zv_11 = z_call(zv_9, za_13, 1);
    return zv_11;
}

static ZValue zf_3(const ZValue *args, size_t argc) {
    if (argc != 0) z_fatal("__zm_3ca93cbc8d3e4342_list expects 0 arguments, got %zu", argc);
    ZValue zv_13 = z_builtin("assetList");
    ZValue zv_14 = z_call(zv_13, NULL, 0);
    return zv_14;
}

ZValue z_dispatch_function(int function_id, const ZValue *args, size_t argc) {
    switch (function_id) {
        case 0: return zf_0(args, argc);
        case 1: return zf_1(args, argc);
        case 2: return zf_2(args, argc);
        case 3: return zf_3(args, argc);
        default: z_fatal("unknown function id %d", function_id); return z_null();
    }
}

bool z_function_is_async(int function_id) {
    switch (function_id) {
        default: return false;
    }
}

int main(void) {
    z_runtime_init();
    zg___zm_3ca93cbc8d3e4342_bytes = z_null();
    zg___zm_3ca93cbc8d3e4342_exists = z_null();
    zg___zm_3ca93cbc8d3e4342_list = z_null();
    zg___zm_3ca93cbc8d3e4342_text = z_null();
    ZValue zv_4 = z_function(0);
    zg___zm_3ca93cbc8d3e4342_exists = zv_4;
    ZValue zv_8 = z_function(1);
    zg___zm_3ca93cbc8d3e4342_text = zv_8;
    ZValue zv_12 = z_function(2);
    zg___zm_3ca93cbc8d3e4342_bytes = zv_12;
    ZValue zv_15 = z_function(3);
    zg___zm_3ca93cbc8d3e4342_list = zv_15;
    ZValue zv_16 = z_builtin("show");
    ZValue zv_17 = zg___zm_3ca93cbc8d3e4342_text;
    ZValue zv_18 = z_string("assets/message.txt");
    ZValue za_26[] = {zv_18};
    ZValue zv_19 = z_call(zv_17, za_26, 1);
    ZValue za_24[] = {zv_19};
    ZValue zv_20 = z_call(zv_16, za_24, 1);
    ZValue zv_21 = z_builtin("show");
    ZValue zv_22 = zg___zm_3ca93cbc8d3e4342_exists;
    ZValue zv_23 = z_string("assets/message.txt");
    ZValue za_32[] = {zv_23};
    ZValue zv_24 = z_call(zv_22, za_32, 1);
    ZValue za_30[] = {zv_24};
    ZValue zv_25 = z_call(zv_21, za_30, 1);
    ZValue zv_26 = z_builtin("show");
    ZValue zv_27 = z_builtin("sizeOf");
    ZValue zv_28 = zg___zm_3ca93cbc8d3e4342_list;
    ZValue zv_29 = z_call(zv_28, NULL, 0);
    ZValue za_38[] = {zv_29};
    ZValue zv_30 = z_call(zv_27, za_38, 1);
    ZValue za_36[] = {zv_30};
    ZValue zv_31 = z_call(zv_26, za_36, 1);
    z_runtime_shutdown();
    return 0;
}
