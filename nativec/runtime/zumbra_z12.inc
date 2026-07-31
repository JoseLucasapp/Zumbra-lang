#if defined(ZUMBRA_ENABLE_Z12)

/* Z12 data, configuration, sessions and observability runtime. */
#include <sys/stat.h>
#include <limits.h>
#include <unistd.h>
#if defined(ZUMBRA_ENABLE_POSTGRES)
#include <postgresql/libpq-fe.h>
#endif
#if defined(ZUMBRA_ENABLE_REDIS)
#include <hiredis/hiredis.h>
#endif
#if defined(ZUMBRA_ENABLE_SQLITE)
#include <sqlite3.h>
#endif

extern char **environ;

typedef struct {
    char *data;
    size_t len;
    size_t cap;
} ZZ12Buffer;

typedef struct {
    const char *text;
    size_t length;
    size_t offset;
} ZZ12JSONParser;

typedef struct ZSQLRowsNative ZSQLRowsNative;
typedef struct {
    ZValue receiver;
    char *name;
} ZZ12BoundMethod;

struct ZSQLRowsNative {
    pthread_mutex_t mutex;
    bool open;
    void *payload;
    ZValue (*next_fn)(void *payload, bool *ok);
    ZValue (*columns_fn)(void *payload);
    void (*close_fn)(void *payload);
};

static ZValue z_z12_value(ZTag tag, ZKind kind, void *pointer) {
    ZValue value = z_value(tag, kind);
    value.as.p = pointer;
    return value;
}

static int64_t z_z12_now_ms(void) {
    struct timespec value;
    clock_gettime(CLOCK_REALTIME, &value);
    return (int64_t)value.tv_sec * 1000 + (int64_t)value.tv_nsec / 1000000;
}

static double z_z12_monotonic_ms(void) {
    struct timespec value;
    clock_gettime(CLOCK_MONOTONIC, &value);
    return (double)value.tv_sec * 1000.0 + (double)value.tv_nsec / 1000000.0;
}

static void z_z12_buffer_init(ZZ12Buffer *buffer) { memset(buffer, 0, sizeof(*buffer)); }
static void z_z12_buffer_reserve(ZZ12Buffer *buffer, size_t needed) {
    if (needed <= buffer->cap) return;
    size_t capacity = buffer->cap == 0 ? 64 : buffer->cap;
    while (capacity < needed) capacity *= 2;
    char *grown = (char *)realloc(buffer->data, capacity);
    if (grown == NULL) z_fatal("out of memory growing Z12 buffer");
    buffer->data = grown;
    buffer->cap = capacity;
}
static void z_z12_buffer_append_bytes(ZZ12Buffer *buffer, const void *data, size_t length) {
    z_z12_buffer_reserve(buffer, buffer->len + length + 1);
    if (length != 0) memcpy(buffer->data + buffer->len, data, length);
    buffer->len += length;
    buffer->data[buffer->len] = '\0';
}
static void z_z12_buffer_append(ZZ12Buffer *buffer, const char *text) {
    z_z12_buffer_append_bytes(buffer, text, strlen(text));
}
static void z_z12_buffer_printf(ZZ12Buffer *buffer, const char *format, ...) {
    va_list arguments;
    va_start(arguments, format);
    va_list copy;
    va_copy(copy, arguments);
    int needed = vsnprintf(NULL, 0, format, copy);
    va_end(copy);
    if (needed < 0) { va_end(arguments); z_fatal("could not format Z12 buffer"); }
    z_z12_buffer_reserve(buffer, buffer->len + (size_t)needed + 1);
    vsnprintf(buffer->data + buffer->len, (size_t)needed + 1, format, arguments);
    buffer->len += (size_t)needed;
    va_end(arguments);
}
static void z_z12_buffer_free(ZZ12Buffer *buffer) { free(buffer->data); memset(buffer, 0, sizeof(*buffer)); }

static int z_z12_unhex(char value) {
    if (value >= '0' && value <= '9') return value - '0';
    if (value >= 'a' && value <= 'f') return value - 'a' + 10;
    if (value >= 'A' && value <= 'F') return value - 'A' + 10;
    return -1;
}

static void z_z12_json_escape(ZZ12Buffer *output, const char *value) {
    z_z12_buffer_append_bytes(output, "\"", 1);
    for (const unsigned char *p = (const unsigned char *)value; *p != 0; p++) {
        switch (*p) {
        case '"': z_z12_buffer_append(output, "\\\""); break;
        case '\\': z_z12_buffer_append(output, "\\\\"); break;
        case '\b': z_z12_buffer_append(output, "\\b"); break;
        case '\f': z_z12_buffer_append(output, "\\f"); break;
        case '\n': z_z12_buffer_append(output, "\\n"); break;
        case '\r': z_z12_buffer_append(output, "\\r"); break;
        case '\t': z_z12_buffer_append(output, "\\t"); break;
        default:
            if (*p < 0x20) z_z12_buffer_printf(output, "\\u%04x", (unsigned int)*p);
            else z_z12_buffer_append_bytes(output, p, 1);
        }
    }
    z_z12_buffer_append_bytes(output, "\"", 1);
}

static void z_z12_json_stringify_value(ZZ12Buffer *output, ZValue value, int depth) {
    if (depth > 128) z_fatal("JSON nesting exceeds 128 levels");
    switch (value.tag) {
    case ZV_NULL: z_z12_buffer_append(output, "null"); break;
    case ZV_BOOL: z_z12_buffer_append(output, value.as.b ? "true" : "false"); break;
    case ZV_INT: z_z12_buffer_printf(output, "%" PRId64, value.as.i); break;
    case ZV_UINT: z_z12_buffer_printf(output, "%" PRIu64, value.as.u); break;
    case ZV_FLOAT:
        if (!isfinite(value.as.f)) z_fatal("JSON cannot encode non-finite floats");
        z_z12_buffer_printf(output, "%.17g", value.as.f);
        break;
    case ZV_STRING: z_z12_json_escape(output, value.as.s); break;
    case ZV_BUFFER: {
        z_z12_buffer_append(output, "{\"$bytes\":\"");
        const uint8_t *bytes = (const uint8_t *)value.as.buffer->data;
        size_t length = value.as.buffer->len * value.as.buffer->elem_size;
        for (size_t index = 0; index < length; index++) z_z12_buffer_printf(output, "%02x", bytes[index]);
        z_z12_buffer_append(output, "\"}");
        break;
    }
    case ZV_ARRAY:
        z_z12_buffer_append_bytes(output, "[", 1);
        for (size_t index = 0; index < value.as.array->len; index++) {
            if (index != 0) z_z12_buffer_append_bytes(output, ",", 1);
            z_z12_json_stringify_value(output, value.as.array->items[index], depth + 1);
        }
        z_z12_buffer_append_bytes(output, "]", 1);
        break;
    case ZV_DICT:
        z_z12_buffer_append_bytes(output, "{", 1);
        for (size_t index = 0; index < value.as.dict->len; index++) {
            if (index != 0) z_z12_buffer_append_bytes(output, ",", 1);
            ZValue key = value.as.dict->keys[index];
            if (key.tag != ZV_STRING) z_fatal("JSON requires string dictionary keys");
            z_z12_json_escape(output, key.as.s);
            z_z12_buffer_append_bytes(output, ":", 1);
            z_z12_json_stringify_value(output, value.as.dict->values[index], depth + 1);
        }
        z_z12_buffer_append_bytes(output, "}", 1);
        break;
    default: z_fatal("JSON does not support this value");
    }
}

static char *z_z12_json_stringify(ZValue value) {
    ZZ12Buffer output;
    z_z12_buffer_init(&output);
    z_z12_json_stringify_value(&output, value, 0);
    char *result = z_strdup(output.data == NULL ? "null" : output.data);
    z_z12_buffer_free(&output);
    return result;
}

static void z_z12_json_skip(ZZ12JSONParser *parser) {
    while (parser->offset < parser->length && isspace((unsigned char)parser->text[parser->offset])) parser->offset++;
}
static bool z_z12_json_take(ZZ12JSONParser *parser, char expected) {
    z_z12_json_skip(parser);
    if (parser->offset >= parser->length || parser->text[parser->offset] != expected) return false;
    parser->offset++;
    return true;
}
static uint32_t z_z12_json_hex4(ZZ12JSONParser *parser) {
    if (parser->offset + 4 > parser->length) z_fatal("invalid JSON unicode escape");
    uint32_t result = 0;
    for (int index = 0; index < 4; index++) {
        int digit = z_z12_unhex(parser->text[parser->offset++]);
        if (digit < 0) z_fatal("invalid JSON unicode escape");
        result = (result << 4) | (uint32_t)digit;
    }
    return result;
}
static void z_z12_json_utf8(ZZ12Buffer *output, uint32_t codepoint) {
    if (codepoint <= 0x7F) { unsigned char bytes[1] = {(unsigned char)codepoint}; z_z12_buffer_append_bytes(output, bytes, 1); }
    else if (codepoint <= 0x7FF) { unsigned char bytes[2] = {(unsigned char)(0xC0 | (codepoint >> 6)), (unsigned char)(0x80 | (codepoint & 0x3F))}; z_z12_buffer_append_bytes(output, bytes, 2); }
    else if (codepoint <= 0xFFFF) { unsigned char bytes[3] = {(unsigned char)(0xE0 | (codepoint >> 12)), (unsigned char)(0x80 | ((codepoint >> 6) & 0x3F)), (unsigned char)(0x80 | (codepoint & 0x3F))}; z_z12_buffer_append_bytes(output, bytes, 3); }
    else if (codepoint <= 0x10FFFF) { unsigned char bytes[4] = {(unsigned char)(0xF0 | (codepoint >> 18)), (unsigned char)(0x80 | ((codepoint >> 12) & 0x3F)), (unsigned char)(0x80 | ((codepoint >> 6) & 0x3F)), (unsigned char)(0x80 | (codepoint & 0x3F))}; z_z12_buffer_append_bytes(output, bytes, 4); }
    else z_fatal("invalid JSON unicode code point");
}
static ZValue z_z12_json_parse_value(ZZ12JSONParser *parser, int depth);
static ZValue z_z12_json_parse_string(ZZ12JSONParser *parser) {
    if (!z_z12_json_take(parser, '"')) z_fatal("expected JSON string");
    ZZ12Buffer output;
    z_z12_buffer_init(&output);
    while (parser->offset < parser->length) {
        unsigned char current = (unsigned char)parser->text[parser->offset++];
        if (current == '"') {
            ZValue result = z_string(output.data == NULL ? "" : output.data);
            z_z12_buffer_free(&output);
            return result;
        }
        if (current < 0x20) { z_z12_buffer_free(&output); z_fatal("invalid JSON control byte"); }
        if (current != '\\') { z_z12_buffer_append_bytes(&output, &current, 1); continue; }
        if (parser->offset >= parser->length) z_fatal("unterminated JSON escape");
        char escaped = parser->text[parser->offset++];
        switch (escaped) {
        case '"': case '\\': case '/': z_z12_buffer_append_bytes(&output, &escaped, 1); break;
        case 'b': { char value='\b'; z_z12_buffer_append_bytes(&output,&value,1); break; }
        case 'f': { char value='\f'; z_z12_buffer_append_bytes(&output,&value,1); break; }
        case 'n': { char value='\n'; z_z12_buffer_append_bytes(&output,&value,1); break; }
        case 'r': { char value='\r'; z_z12_buffer_append_bytes(&output,&value,1); break; }
        case 't': { char value='\t'; z_z12_buffer_append_bytes(&output,&value,1); break; }
        case 'u': {
            uint32_t codepoint = z_z12_json_hex4(parser);
            if (codepoint >= 0xD800 && codepoint <= 0xDBFF) {
                if (parser->offset + 2 > parser->length || parser->text[parser->offset] != '\\' || parser->text[parser->offset + 1] != 'u') z_fatal("invalid JSON surrogate pair");
                parser->offset += 2;
                uint32_t low = z_z12_json_hex4(parser);
                if (low < 0xDC00 || low > 0xDFFF) z_fatal("invalid JSON surrogate pair");
                codepoint = 0x10000 + ((codepoint - 0xD800) << 10) + (low - 0xDC00);
            }
            z_z12_json_utf8(&output, codepoint);
            break;
        }
        default: z_fatal("invalid JSON escape");
        }
    }
    z_fatal("unterminated JSON string");
    return z_null();
}
static ZValue z_z12_json_parse_number(ZZ12JSONParser *parser) {
    z_z12_json_skip(parser);
    size_t start = parser->offset;
    if (parser->offset < parser->length && parser->text[parser->offset] == '-') parser->offset++;
    while (parser->offset < parser->length && isdigit((unsigned char)parser->text[parser->offset])) parser->offset++;
    bool floating = false;
    if (parser->offset < parser->length && parser->text[parser->offset] == '.') { floating = true; parser->offset++; while (parser->offset < parser->length && isdigit((unsigned char)parser->text[parser->offset])) parser->offset++; }
    if (parser->offset < parser->length && (parser->text[parser->offset] == 'e' || parser->text[parser->offset] == 'E')) { floating = true; parser->offset++; if (parser->offset < parser->length && (parser->text[parser->offset] == '+' || parser->text[parser->offset] == '-')) parser->offset++; while (parser->offset < parser->length && isdigit((unsigned char)parser->text[parser->offset])) parser->offset++; }
    size_t length = parser->offset - start;
    if (length == 0) z_fatal("invalid JSON number");
    char *number = (char *)malloc(length + 1);
    if (number == NULL) z_fatal("out of memory parsing JSON number");
    memcpy(number, parser->text + start, length); number[length] = '\0';
    char *end = NULL;
    errno = 0;
    if (floating) {
        double result = strtod(number, &end);
        bool valid = errno == 0 && end != NULL && *end == '\0' && isfinite(result);
        free(number);
        if (!valid) z_fatal("invalid JSON number");
        return z_float(result);
    }
    int64_t result = strtoll(number, &end, 10);
    bool valid = errno == 0 && end != NULL && *end == '\0';
    free(number);
    if (!valid) z_fatal("JSON integer outside int64 range");
    return z_int(result);
}
static ZValue z_z12_json_parse_array(ZZ12JSONParser *parser, int depth) {
    if (!z_z12_json_take(parser, '[')) z_fatal("expected JSON array");
    ZValue result = z_array_from(NULL, 0);
    z_z12_json_skip(parser);
    if (z_z12_json_take(parser, ']')) return result;
    for (;;) {
        ZArray *array = result.as.array;
        if (array->len == array->cap) {
            size_t capacity = array->cap == 0 ? 4 : array->cap * 2;
            ZValue *items = (ZValue *)z_alloc(sizeof(ZValue) * capacity);
            if (array->len != 0) memcpy(items, array->items, sizeof(ZValue) * array->len);
            array->items = items; array->cap = capacity;
        }
        array->items[array->len++] = z_z12_json_parse_value(parser, depth + 1);
        z_z12_json_skip(parser);
        if (z_z12_json_take(parser, ']')) break;
        if (!z_z12_json_take(parser, ',')) z_fatal("expected comma in JSON array");
    }
    return result;
}
static ZValue z_z12_json_parse_object(ZZ12JSONParser *parser, int depth) {
    if (!z_z12_json_take(parser, '{')) z_fatal("expected JSON object");
    ZValue result = z_dict_from(NULL, 0);
    z_z12_json_skip(parser);
    if (z_z12_json_take(parser, '}')) return result;
    for (;;) {
        ZValue key = z_z12_json_parse_string(parser);
        if (!z_z12_json_take(parser, ':')) z_fatal("expected colon in JSON object");
        ZValue value = z_z12_json_parse_value(parser, depth + 1);
        z_set_index(result, key, value);
        z_z12_json_skip(parser);
        if (z_z12_json_take(parser, '}')) break;
        if (!z_z12_json_take(parser, ',')) z_fatal("expected comma in JSON object");
    }
    if (result.as.dict->len == 1 && result.as.dict->keys[0].tag == ZV_STRING && strcmp(result.as.dict->keys[0].as.s, "$bytes") == 0 && result.as.dict->values[0].tag == ZV_STRING) {
        const char *hex = result.as.dict->values[0].as.s;
        size_t length = strlen(hex);
        if (length % 2 != 0) z_fatal("invalid encoded byte array");
        ZValue bytes = z_bytes(length / 2);
        for (size_t index = 0; index < length; index += 2) {
            int high = z_z12_unhex(hex[index]); int low = z_z12_unhex(hex[index + 1]);
            if (high < 0 || low < 0) z_fatal("invalid encoded byte array");
            ((uint8_t *)bytes.as.buffer->data)[index / 2] = (uint8_t)((high << 4) | low);
        }
        return bytes;
    }
    if (result.as.dict->len == 1 && result.as.dict->keys[0].tag == ZV_STRING && strcmp(result.as.dict->keys[0].as.s, "$uint") == 0 && result.as.dict->values[0].tag == ZV_STRING) {
        const char *raw = result.as.dict->values[0].as.s;
        char *end = NULL;
        errno = 0;
        uint64_t value = strtoull(raw, &end, 10);
        if (errno != 0 || end == NULL || *end != '\0') z_fatal("invalid encoded unsigned integer");
        return z_uint(value, ZK_U64);
    }
    return result;
}
static ZValue z_z12_json_parse_value(ZZ12JSONParser *parser, int depth) {
    if (depth > 128) z_fatal("JSON nesting exceeds 128 levels");
    z_z12_json_skip(parser);
    if (parser->offset >= parser->length) z_fatal("unexpected end of JSON");
    char current = parser->text[parser->offset];
    if (current == '"') return z_z12_json_parse_string(parser);
    if (current == '{') return z_z12_json_parse_object(parser, depth);
    if (current == '[') return z_z12_json_parse_array(parser, depth);
    if (current == '-' || isdigit((unsigned char)current)) return z_z12_json_parse_number(parser);
    if (parser->length - parser->offset >= 4 && strncmp(parser->text + parser->offset, "true", 4) == 0) { parser->offset += 4; return z_bool(true); }
    if (parser->length - parser->offset >= 5 && strncmp(parser->text + parser->offset, "false", 5) == 0) { parser->offset += 5; return z_bool(false); }
    if (parser->length - parser->offset >= 4 && strncmp(parser->text + parser->offset, "null", 4) == 0) { parser->offset += 4; return z_null(); }
    z_fatal("invalid JSON at byte %zu", parser->offset);
    return z_null();
}
static ZValue z_z12_json_parse(const char *text) {
    ZZ12JSONParser parser = {text, strlen(text), 0};
    ZValue result = z_z12_json_parse_value(&parser, 0);
    z_z12_json_skip(&parser);
    if (parser.offset != parser.length) z_fatal("unexpected content after JSON value");
    return result;
}

static char *z_z12_read_text_file(const char *path, size_t *length_out) {
    FILE *file = fopen(path, "rb");
    if (file == NULL) z_fatal("open file %s: %s", path, strerror(errno));
    if (fseek(file, 0, SEEK_END) != 0) { fclose(file); z_fatal("seek file %s failed", path); }
    long length = ftell(file);
    if (length < 0) { fclose(file); z_fatal("read file length %s failed", path); }
    rewind(file);
    char *data = (char *)malloc((size_t)length + 1);
    if (data == NULL) { fclose(file); z_fatal("out of memory reading %s", path); }
    size_t count = fread(data, 1, (size_t)length, file);
    if (ferror(file)) { free(data); fclose(file); z_fatal("read file %s failed", path); }
    fclose(file);
    data[count] = '\0';
    if (length_out != NULL) *length_out = count;
    return data;
}

static char *z_z12_heap_strdup(const char *value) {
    if (value == NULL) value = "";
    size_t length = strlen(value);
    char *copy = (char *)malloc(length + 1);
    if (copy == NULL) z_fatal("out of memory copying temporary string");
    memcpy(copy, value, length + 1);
    return copy;
}

static void z_z12_ensure_parent_directories(const char *path) {
    char *copy = z_z12_heap_strdup(path);
    for (char *cursor = copy + 1; *cursor != '\0'; cursor++) {
        if (*cursor != '/') continue;
        *cursor = '\0';
        if (*copy != '\0' && mkdir(copy, 0755) != 0 && errno != EEXIST) {
            free(copy);
            z_fatal("create directory for %s: %s", path, strerror(errno));
        }
        *cursor = '/';
    }
    free(copy);
}

static void z_z12_atomic_write(const char *path, const void *data, size_t length, mode_t mode) {
    z_z12_ensure_parent_directories(path);
    size_t path_length = strlen(path);
    char *temporary = (char *)malloc(path_length + 32);
    if (temporary == NULL) z_fatal("out of memory creating temporary filename");
    snprintf(temporary, path_length + 32, "%s.zumbra-XXXXXX", path);
    int descriptor = mkstemp(temporary);
    if (descriptor < 0) { free(temporary); z_fatal("create temporary file for %s: %s", path, strerror(errno)); }
    if (fchmod(descriptor, mode) != 0) { close(descriptor); unlink(temporary); free(temporary); z_fatal("chmod temporary file: %s", strerror(errno)); }
    const uint8_t *bytes = (const uint8_t *)data;
    size_t written = 0;
    while (written < length) {
        ssize_t count = write(descriptor, bytes + written, length - written);
        if (count < 0) { close(descriptor); unlink(temporary); free(temporary); z_fatal("write temporary file: %s", strerror(errno)); }
        written += (size_t)count;
    }
    if (fsync(descriptor) != 0) { close(descriptor); unlink(temporary); free(temporary); z_fatal("sync temporary file: %s", strerror(errno)); }
    if (close(descriptor) != 0) { unlink(temporary); free(temporary); z_fatal("close temporary file: %s", strerror(errno)); }
    if (rename(temporary, path) != 0) { unlink(temporary); free(temporary); z_fatal("replace file %s: %s", path, strerror(errno)); }
    free(temporary);
}

static ZValue z_z12_json_read_file(const char *path) {
    size_t length = 0;
    char *text = z_z12_read_text_file(path, &length);
    (void)length;
    ZValue result = z_z12_json_parse(text);
    free(text);
    return result;
}

static int64_t z_z12_json_write_file(const char *path, ZValue value) {
    char *text = z_z12_json_stringify(value);
    size_t length = strlen(text);
    ZZ12Buffer output;
    z_z12_buffer_init(&output);
    z_z12_buffer_append_bytes(&output, text, length);
    z_z12_buffer_append_bytes(&output, "\n", 1);
    z_z12_atomic_write(path, output.data, output.len, 0644);
    int64_t result = (int64_t)output.len;
    z_z12_buffer_free(&output);
    return result;
}


static ZValue z_z12_operation_result(bool ok, ZValue value, const char *error) {
    ZValue pairs[3] = {
        z_pair(z_string("ok"), z_bool(ok)),
        z_pair(z_string("value"), value),
        z_pair(z_string("error"), z_string(error == NULL ? "" : error))
    };
    return z_dict_from(pairs, 3);
}

#define Z_Z12_RETURN_RESULT(expression) do { \
    jmp_buf z12_trap; \
    jmp_buf *z12_previous_trap = z_task_error_trap; \
    z_task_error_trap = &z12_trap; \
    if (setjmp(z12_trap) == 0) { \
        ZValue z12_value = (expression); \
        z_task_error_trap = z12_previous_trap; \
        return z_z12_operation_result(true, z12_value, ""); \
    } \
    char *z12_error = z_strdup(z_task_error_message[0] == '\0' ? "operation failed" : z_task_error_message); \
    z_task_error_trap = z12_previous_trap; \
    return z_z12_operation_result(false, z_null(), z12_error); \
} while (0)

static bool z_z12_file_exists(const char *path) {
    struct stat information;
    return path != NULL && stat(path, &information) == 0;
}

static void z_z12_csv_push_value(ZValue **items, size_t *count, size_t *capacity, ZValue value) {
    if (*count == *capacity) {
        size_t next = *capacity == 0 ? 8 : *capacity * 2;
        ZValue *grown = (ZValue *)realloc(*items, sizeof(ZValue) * next);
        if (grown == NULL) z_fatal("out of memory parsing CSV");
        *items = grown;
        *capacity = next;
    }
    (*items)[(*count)++] = value;
}

static void z_z12_csv_finish_field(ZZ12Buffer *field, ZValue **cells, size_t *cell_count, size_t *cell_capacity) {
    z_z12_csv_push_value(cells, cell_count, cell_capacity, z_string(field->data == NULL ? "" : field->data));
    field->len = 0;
    if (field->data != NULL) field->data[0] = '\0';
}

static ZValue z_z12_csv_parse(const char *text, size_t length) {
    ZValue *rows = NULL;
    size_t row_count = 0, row_capacity = 0;
    ZValue *cells = NULL;
    size_t cell_count = 0, cell_capacity = 0;
    ZZ12Buffer field;
    z_z12_buffer_init(&field);
    bool quoted = false;
    bool row_started = false;

    for (size_t index = 0; index < length; index++) {
        char current = text[index];
        if (quoted) {
            if (current == '"') {
                if (index + 1 < length && text[index + 1] == '"') {
                    z_z12_buffer_append_bytes(&field, "\"", 1);
                    index++;
                } else {
                    quoted = false;
                }
            } else {
                z_z12_buffer_append_bytes(&field, &current, 1);
            }
            row_started = true;
            continue;
        }
        if (current == '"' && field.len == 0) {
            quoted = true;
            row_started = true;
            continue;
        }
        if (current == ',') {
            z_z12_csv_finish_field(&field, &cells, &cell_count, &cell_capacity);
            row_started = true;
            continue;
        }
        if (current == '\r' || current == '\n') {
            if (current == '\r' && index + 1 < length && text[index + 1] == '\n') index++;
            z_z12_csv_finish_field(&field, &cells, &cell_count, &cell_capacity);
            z_z12_csv_push_value(&rows, &row_count, &row_capacity, z_array_from(cells, cell_count));
            cell_count = 0;
            row_started = false;
            continue;
        }
        z_z12_buffer_append_bytes(&field, &current, 1);
        row_started = true;
    }
    if (quoted) {
        free(rows); free(cells); z_z12_buffer_free(&field);
        z_fatal("csvReadFile: unterminated quoted field");
    }
    if (row_started || field.len != 0 || cell_count != 0) {
        z_z12_csv_finish_field(&field, &cells, &cell_count, &cell_capacity);
        z_z12_csv_push_value(&rows, &row_count, &row_capacity, z_array_from(cells, cell_count));
    }
    ZValue result = z_array_from(rows, row_count);
    free(rows);
    free(cells);
    z_z12_buffer_free(&field);
    return result;
}

static ZValue z_z12_csv_read_file(const char *path) {
    size_t length = 0;
    char *text = z_z12_read_text_file(path, &length);
    ZValue result = z_z12_csv_parse(text, length);
    free(text);
    return result;
}

static void z_z12_csv_append_cell(ZZ12Buffer *output, ZValue value) {
    ZValue converted = z_to_string_value(value);
    const char *text = z_as_cstring(converted);
    bool quoted = strchr(text, ',') != NULL || strchr(text, '"') != NULL || strchr(text, '\n') != NULL || strchr(text, '\r') != NULL;
    if (quoted) z_z12_buffer_append_bytes(output, "\"", 1);
    for (const char *cursor = text; *cursor != '\0'; cursor++) {
        if (*cursor == '"') z_z12_buffer_append(output, "\"\"");
        else z_z12_buffer_append_bytes(output, cursor, 1);
    }
    if (quoted) z_z12_buffer_append_bytes(output, "\"", 1);
}

static int64_t z_z12_csv_write_file(const char *path, ZValue rows) {
    if (rows.tag != ZV_ARRAY) z_fatal("csvWriteFile rows must be an array");
    ZZ12Buffer output;
    z_z12_buffer_init(&output);
    for (size_t row_index = 0; row_index < rows.as.array->len; row_index++) {
        ZValue row = rows.as.array->items[row_index];
        if (row.tag != ZV_ARRAY) {
            z_z12_buffer_free(&output);
            z_fatal("csvWriteFile row %zu must be an array", row_index + 1);
        }
        for (size_t column = 0; column < row.as.array->len; column++) {
            if (column != 0) z_z12_buffer_append_bytes(&output, ",", 1);
            z_z12_csv_append_cell(&output, row.as.array->items[column]);
        }
        z_z12_buffer_append_bytes(&output, "\n", 1);
    }
    z_z12_atomic_write(path, output.data == NULL ? "" : output.data, output.len, 0644);
    int64_t written = (int64_t)output.len;
    z_z12_buffer_free(&output);
    return written;
}

static ZValue z_z12_binary_encode(ZValue value) {
    char *json = z_z12_json_stringify(value);
    static const char envelope[] = "ZB1\n";
    size_t json_length = strlen(json);
    ZValue result = z_bytes(sizeof(envelope) - 1 + json_length);
    memcpy(result.as.buffer->data, envelope, sizeof(envelope) - 1);
    if (json_length != 0) memcpy((uint8_t *)result.as.buffer->data + sizeof(envelope) - 1, json, json_length);
    return result;
}

static ZValue z_z12_binary_decode(ZValue value) {
    if (value.tag != ZV_BUFFER || value.as.buffer->elem_size != 1) z_fatal("binaryDecode expects a byte buffer");
    static const char envelope[] = "ZB1\n";
    size_t length = value.as.buffer->len * value.as.buffer->elem_size;
    if (length < sizeof(envelope) - 1 || memcmp(value.as.buffer->data, envelope, sizeof(envelope) - 1) != 0) z_fatal("binaryDecode unsupported or corrupt envelope");
    size_t payload_length = length - (sizeof(envelope) - 1);
    char *payload = (char *)malloc(payload_length + 1);
    if (payload == NULL) z_fatal("out of memory decoding binary envelope");
    if (payload_length != 0) memcpy(payload, (uint8_t *)value.as.buffer->data + sizeof(envelope) - 1, payload_length);
    payload[payload_length] = '\0';
    ZValue result = z_z12_json_parse(payload);
    free(payload);
    return result;
}

static ZValue z_z12_dict_clone(ZValue source) {
    if (source.tag != ZV_DICT) z_fatal("expected dictionary");
    ZValue result = z_dict_from(NULL, 0);
    for (size_t index = 0; index < source.as.dict->len; index++) z_set_index(result, source.as.dict->keys[index], source.as.dict->values[index]);
    return result;
}
static bool z_z12_dict_find(ZValue dictionary, const char *key, ZValue *result) {
    if (dictionary.tag != ZV_DICT) return false;
    for (size_t index = 0; index < dictionary.as.dict->len; index++) {
        ZValue current = dictionary.as.dict->keys[index];
        if (current.tag == ZV_STRING && strcmp(current.as.s, key) == 0) {
            if (result != NULL) *result = dictionary.as.dict->values[index];
            return true;
        }
    }
    return false;
}
static bool z_z12_sensitive_key(const char *key) {
    char lower[256];
    size_t length = strlen(key);
    if (length >= sizeof(lower)) length = sizeof(lower) - 1;
    for (size_t index = 0; index < length; index++) lower[index] = (char)tolower((unsigned char)key[index]);
    lower[length] = '\0';
    return strstr(lower, "password") != NULL || strstr(lower, "secret") != NULL || strstr(lower, "token") != NULL || strstr(lower, "api_key") != NULL || strstr(lower, "apikey") != NULL || strstr(lower, "authorization") != NULL;
}
static ZValue z_z12_redact_dict(ZValue source) {
    if (source.tag != ZV_DICT) return source;
    ZValue result = z_dict_from(NULL, 0);
    for (size_t index = 0; index < source.as.dict->len; index++) {
        ZValue key = source.as.dict->keys[index];
        ZValue value = source.as.dict->values[index];
        if (key.tag == ZV_STRING && z_z12_sensitive_key(key.as.s)) value = z_string("***");
        else if (value.tag == ZV_DICT) value = z_z12_redact_dict(value);
        z_set_index(result, key, value);
    }
    return result;
}
static double z_z12_number(ZValue value, const char *name) {
    if (value.tag == ZV_FLOAT) return value.as.f;
    if (value.tag == ZV_INT) return (double)value.as.i;
    if (value.tag == ZV_UINT) return (double)value.as.u;
    z_fatal("%s expects numeric value", name);
    return 0;
}

/* Generic streaming rows shared by SQLite and PostgreSQL. */
static ZSQLRowsNative *z_z12_expect_rows(ZValue value) {
    if (value.tag != ZV_SQL_ROWS || value.as.p == NULL) z_fatal("expected SQLRows");
    return (ZSQLRowsNative *)value.as.p;
}
static ZValue z_z12_rows_new(void *payload, ZValue (*next_fn)(void *, bool *), ZValue (*columns_fn)(void *), void (*close_fn)(void *)) {
    ZSQLRowsNative *rows = (ZSQLRowsNative *)z_alloc(sizeof(ZSQLRowsNative));
    memset(rows, 0, sizeof(*rows));
    pthread_mutex_init(&rows->mutex, NULL);
    rows->open = true;
    rows->payload = payload;
    rows->next_fn = next_fn;
    rows->columns_fn = columns_fn;
    rows->close_fn = close_fn;
    return z_z12_value(ZV_SQL_ROWS, ZK_SQL_ROWS, rows);
}
static ZValue z_z12_rows_next(ZSQLRowsNative *rows) {
    pthread_mutex_lock(&rows->mutex);
    if (!rows->open) { pthread_mutex_unlock(&rows->mutex); z_fatal("SQL rows are closed"); }
    bool ok = false;
    ZValue row = rows->next_fn(rows->payload, &ok);
    if (!ok) {
        if (rows->close_fn != NULL) rows->close_fn(rows->payload);
        rows->open = false;
    }
    pthread_mutex_unlock(&rows->mutex);
    ZValue values[2] = {ok ? row : z_null(), z_bool(ok)};
    return z_array_from(values, 2);
}
static ZValue z_z12_rows_columns(ZSQLRowsNative *rows) {
    pthread_mutex_lock(&rows->mutex);
    ZValue result = rows->columns_fn(rows->payload);
    pthread_mutex_unlock(&rows->mutex);
    return result;
}
static bool z_z12_rows_close(ZSQLRowsNative *rows) {
    pthread_mutex_lock(&rows->mutex);
    if (rows->open && rows->close_fn != NULL) rows->close_fn(rows->payload);
    rows->open = false;
    pthread_mutex_unlock(&rows->mutex);
    return true;
}
static bool z_z12_rows_open(ZSQLRowsNative *rows) {
    pthread_mutex_lock(&rows->mutex);
    bool result = rows->open;
    pthread_mutex_unlock(&rows->mutex);
    return result;
}

typedef struct { ZValue rows; ZValue columns; size_t index; } ZZ12BufferedRows;
static ZValue z_z12_buffered_next(void *raw, bool *ok) {
    ZZ12BufferedRows *rows = (ZZ12BufferedRows *)raw;
    if (rows->index >= rows->rows.as.array->len) { *ok = false; return z_null(); }
    *ok = true;
    return rows->rows.as.array->items[rows->index++];
}
static ZValue z_z12_buffered_columns(void *raw) { return ((ZZ12BufferedRows *)raw)->columns; }
static void z_z12_buffered_close(void *raw) { (void)raw; }
static ZValue z_z12_rows_from_array(ZValue rows) {
    if (rows.tag != ZV_ARRAY) z_fatal("rows must be array");
    ZZ12BufferedRows *buffered = (ZZ12BufferedRows *)z_alloc(sizeof(ZZ12BufferedRows));
    buffered->rows = rows; buffered->index = 0;
    if (rows.as.array->len > 0 && rows.as.array->items[0].tag == ZV_DICT) {
        ZDict *first = rows.as.array->items[0].as.dict;
        ZValue *columns = first->len == 0 ? NULL : (ZValue *)z_alloc(sizeof(ZValue) * first->len);
        for (size_t index = 0; index < first->len; index++) columns[index] = first->keys[index];
        buffered->columns = z_array_from(columns, first->len);
    } else buffered->columns = z_array_from(NULL, 0);
    return z_z12_rows_new(buffered, z_z12_buffered_next, z_z12_buffered_columns, z_z12_buffered_close);
}

/* Typed configuration. */
typedef struct { pthread_mutex_t mutex; ZValue values; ZValue secrets; } ZConfigNative;
static ZConfigNative *z_z12_expect_config(ZValue value) {
    if (value.tag != ZV_CONFIG || value.as.p == NULL) z_fatal("expected Config");
    return (ZConfigNative *)value.as.p;
}
static ZValue z_z12_config_new(ZValue values) {
    if (values.tag != ZV_DICT) z_fatal("configuration values must be dictionary");
    ZConfigNative *config = (ZConfigNative *)z_alloc(sizeof(ZConfigNative));
    pthread_mutex_init(&config->mutex, NULL);
    config->values = z_z12_dict_clone(values);
    config->secrets = z_dict_from(NULL, 0);
    return z_z12_value(ZV_CONFIG, ZK_CONFIG, config);
}
static void z_z12_config_flatten(ZValue value, const char *prefix, ZValue target) {
    if (value.tag != ZV_DICT) {
        z_set_index(target, z_string(prefix), value);
        return;
    }
    for (size_t index = 0; index < value.as.dict->len; index++) {
        ZValue key = value.as.dict->keys[index];
        if (key.tag != ZV_STRING) z_fatal("config JSON keys must be strings");
        ZZ12Buffer full; z_z12_buffer_init(&full);
        if (prefix != NULL && *prefix != '\0') { z_z12_buffer_append(&full, prefix); z_z12_buffer_append(&full, "."); }
        z_z12_buffer_append(&full, key.as.s);
        z_z12_config_flatten(value.as.dict->values[index], full.data, target);
        z_z12_buffer_free(&full);
    }
}
static char *z_z12_trim(char *text) {
    while (*text != '\0' && isspace((unsigned char)*text)) text++;
    char *end = text + strlen(text);
    while (end > text && isspace((unsigned char)end[-1])) end--;
    *end = '\0';
    return text;
}
static ZValue z_z12_config_load(const char *path) {
    FILE *file = fopen(path, "rb");
    if (file == NULL) z_fatal("open config file %s: %s", path, strerror(errno));
    if (fseek(file, 0, SEEK_END) != 0) { fclose(file); z_fatal("seek config file failed"); }
    long length = ftell(file);
    if (length < 0) { fclose(file); z_fatal("read config file length failed"); }
    rewind(file);
    char *text = (char *)malloc((size_t)length + 1);
    if (text == NULL) { fclose(file); z_fatal("out of memory reading config"); }
    size_t count = fread(text, 1, (size_t)length, file);
    fclose(file); text[count] = '\0';
    const char *dot = strrchr(path, '.');
    ZValue values = z_dict_from(NULL, 0);
    if (dot != NULL && strcmp(dot, ".json") == 0) {
        ZValue parsed = z_z12_json_parse(text);
        if (parsed.tag != ZV_DICT) { free(text); z_fatal("config JSON root must be object"); }
        z_z12_config_flatten(parsed, "", values);
    } else {
        char *save = NULL;
        for (char *line = strtok_r(text, "\n", &save); line != NULL; line = strtok_r(NULL, "\n", &save)) {
            char *trimmed = z_z12_trim(line);
            if (*trimmed == '\0' || *trimmed == '#') continue;
            if (strncmp(trimmed, "export ", 7) == 0) trimmed = z_z12_trim(trimmed + 7);
            char *equal = strchr(trimmed, '=');
            if (equal == NULL) continue;
            *equal = '\0';
            char *key = z_z12_trim(trimmed);
            char *raw = z_z12_trim(equal + 1);
            size_t raw_len = strlen(raw);
            if (raw_len >= 2 && ((raw[0] == '"' && raw[raw_len - 1] == '"') || (raw[0] == '\'' && raw[raw_len - 1] == '\''))) { raw[raw_len - 1] = '\0'; raw++; }
            z_set_index(values, z_string(key), z_string(raw));
        }
    }
    free(text);
    return z_z12_config_new(values);
}
static ZValue z_z12_config_env(const char *prefix) {
    ZValue values = z_dict_from(NULL, 0);
    size_t prefix_len = strlen(prefix);
    for (char **entry = environ; entry != NULL && *entry != NULL; entry++) {
        char *equal = strchr(*entry, '=');
        if (equal == NULL) continue;
        size_t key_len = (size_t)(equal - *entry);
        if (prefix_len > 0 && (key_len < prefix_len || strncmp(*entry, prefix, prefix_len) != 0)) continue;
        const char *key_start = *entry + prefix_len;
        size_t output_len = key_len - prefix_len;
        char *key = (char *)malloc(output_len + 1);
        if (key == NULL) z_fatal("out of memory reading environment");
        memcpy(key, key_start, output_len); key[output_len] = '\0';
        z_set_index(values, z_string(key), z_string(equal + 1));
        free(key);
    }
    return z_z12_config_new(values);
}
static bool z_z12_config_value(ZConfigNative *config, const char *key, ZValue *value) {
    pthread_mutex_lock(&config->mutex);
    bool found = z_z12_dict_find(config->values, key, value);
    pthread_mutex_unlock(&config->mutex);
    return found;
}
static ZValue z_z12_config_required(ZConfigNative *config, const char *key, ZValue fallback, bool has_fallback) {
    ZValue value;
    if (z_z12_config_value(config, key, &value)) return value;
    if (has_fallback) return fallback;
    z_fatal("required configuration key %s is missing", key);
    return z_null();
}
static ZValue z_z12_config_string(ZConfigNative *config, const char *key, ZValue fallback, bool has_fallback) {
    ZValue value = z_z12_config_required(config, key, fallback, has_fallback);
    if (value.tag == ZV_STRING) return value;
    ZValue converted = z_to_string_value(value);
    return converted;
}
static ZValue z_z12_config_int(ZConfigNative *config, const char *key, ZValue fallback, bool has_fallback) {
    ZValue value = z_z12_config_required(config, key, fallback, has_fallback);
    if (value.tag == ZV_INT) return value;
    if (value.tag == ZV_UINT) return z_int((int64_t)value.as.u);
    if (value.tag == ZV_FLOAT) return z_int((int64_t)value.as.f);
    if (value.tag == ZV_STRING) {
        char *end = NULL; errno = 0; long long parsed = strtoll(value.as.s, &end, 10);
        if (errno == 0 && end != NULL && *end == '\0') return z_int((int64_t)parsed);
    }
    z_fatal("configuration key %s is not an integer", key);
    return z_null();
}
static ZValue z_z12_config_float(ZConfigNative *config, const char *key, ZValue fallback, bool has_fallback) {
    ZValue value = z_z12_config_required(config, key, fallback, has_fallback);
    if (value.tag == ZV_FLOAT) return value;
    if (value.tag == ZV_INT) return z_float((double)value.as.i);
    if (value.tag == ZV_UINT) return z_float((double)value.as.u);
    if (value.tag == ZV_STRING) {
        char *end = NULL; errno = 0; double parsed = strtod(value.as.s, &end);
        if (errno == 0 && end != NULL && *end == '\0' && isfinite(parsed)) return z_float(parsed);
    }
    z_fatal("configuration key %s is not a float", key);
    return z_null();
}
static ZValue z_z12_config_bool(ZConfigNative *config, const char *key, ZValue fallback, bool has_fallback) {
    ZValue value = z_z12_config_required(config, key, fallback, has_fallback);
    if (value.tag == ZV_BOOL) return value;
    if (value.tag == ZV_INT) return z_bool(value.as.i != 0);
    if (value.tag == ZV_STRING) {
        if (strcasecmp(value.as.s, "true") == 0 || strcmp(value.as.s, "1") == 0 || strcasecmp(value.as.s, "yes") == 0 || strcasecmp(value.as.s, "on") == 0) return z_bool(true);
        if (strcasecmp(value.as.s, "false") == 0 || strcmp(value.as.s, "0") == 0 || strcasecmp(value.as.s, "no") == 0 || strcasecmp(value.as.s, "off") == 0) return z_bool(false);
    }
    z_fatal("configuration key %s is not a boolean", key);
    return z_null();
}

/* Structured logger. */
typedef struct { pthread_mutex_t mutex; char *name; int level; ZValue fields; FILE *file; bool owns_file; bool closed; } ZLoggerNative;
static int z_z12_log_level(const char *level) {
    if (strcasecmp(level, "debug") == 0) return 0;
    if (strcasecmp(level, "info") == 0) return 1;
    if (strcasecmp(level, "warn") == 0 || strcasecmp(level, "warning") == 0) return 2;
    if (strcasecmp(level, "error") == 0) return 3;
    z_fatal("unknown log level %s", level);
    return 1;
}
static ZLoggerNative *z_z12_expect_logger(ZValue value) { if (value.tag != ZV_LOGGER || value.as.p == NULL) z_fatal("expected Logger"); return (ZLoggerNative *)value.as.p; }
static ZValue z_z12_logger_new(const char *name, const char *level, const char *path) {
    ZLoggerNative *logger = (ZLoggerNative *)z_alloc(sizeof(ZLoggerNative));
    memset(logger, 0, sizeof(*logger)); pthread_mutex_init(&logger->mutex, NULL);
    logger->name = z_strdup(name); logger->level = z_z12_log_level(level); logger->fields = z_dict_from(NULL, 0); logger->file = stderr;
    if (path != NULL && *path != '\0') { logger->file = fopen(path, "ab"); if (logger->file == NULL) z_fatal("open log file %s: %s", path, strerror(errno)); logger->owns_file = true; }
    return z_z12_value(ZV_LOGGER, ZK_LOGGER, logger);
}
static ZValue z_z12_logger_with(ZLoggerNative *source, ZValue fields) {
    if (fields.tag != ZV_DICT) z_fatal("logger fields must be dictionary");
    ZLoggerNative *logger = (ZLoggerNative *)z_alloc(sizeof(ZLoggerNative));
    memset(logger, 0, sizeof(*logger)); pthread_mutex_init(&logger->mutex, NULL);
    pthread_mutex_lock(&source->mutex);
    logger->name = z_strdup(source->name); logger->level = source->level; logger->fields = z_z12_dict_clone(source->fields); logger->file = source->file; logger->owns_file = false;
    pthread_mutex_unlock(&source->mutex);
    for (size_t index = 0; index < fields.as.dict->len; index++) z_set_index(logger->fields, fields.as.dict->keys[index], fields.as.dict->values[index]);
    return z_z12_value(ZV_LOGGER, ZK_LOGGER, logger);
}
static bool z_z12_logger_log(ZLoggerNative *logger, const char *level, const char *message, ZValue fields) {
    if (fields.tag != ZV_DICT) z_fatal("logger fields must be dictionary");
    int numeric = z_z12_log_level(level);
    pthread_mutex_lock(&logger->mutex);
    if (logger->closed) { pthread_mutex_unlock(&logger->mutex); z_fatal("logger is closed"); }
    if (numeric < logger->level) { pthread_mutex_unlock(&logger->mutex); return true; }
    ZValue record = z_z12_dict_clone(logger->fields);
    for (size_t index = 0; index < fields.as.dict->len; index++) z_set_index(record, fields.as.dict->keys[index], fields.as.dict->values[index]);
    z_set_index(record, z_string("timestampMs"), z_int(z_z12_now_ms()));
    z_set_index(record, z_string("logger"), z_string(logger->name));
    z_set_index(record, z_string("level"), z_string(level));
    z_set_index(record, z_string("message"), z_string(message));
    record = z_z12_redact_dict(record);
    char *json = z_z12_json_stringify(record);
    fprintf(logger->file, "%s\n", json); fflush(logger->file);
    pthread_mutex_unlock(&logger->mutex);
    return true;
}
static bool z_z12_logger_close(ZLoggerNative *logger) {
    pthread_mutex_lock(&logger->mutex);
    if (!logger->closed && logger->owns_file && logger->file != NULL) fclose(logger->file);
    logger->closed = true; logger->file = NULL;
    pthread_mutex_unlock(&logger->mutex);
    return true;
}

/* Metrics. */
typedef struct { pthread_mutex_t mutex; ZValue counters; ZValue gauges; ZValue histograms; } ZMetricsNative;
typedef struct { const char *key; const char *value; } ZZ12MetricLabel;
static int z_z12_metric_label_compare(const void *left, const void *right) {
    return strcmp(((const ZZ12MetricLabel *)left)->key, ((const ZZ12MetricLabel *)right)->key);
}
static ZMetricsNative *z_z12_expect_metrics(ZValue value) { if (value.tag != ZV_METRICS || value.as.p == NULL) z_fatal("expected Metrics"); return (ZMetricsNative *)value.as.p; }
static char *z_z12_metric_key(const char *name, ZValue labels) {
    if (labels.tag == ZV_NULL) return z_strdup(name);
    if (labels.tag != ZV_DICT) z_fatal("metric labels must be dictionary or null");
    if (labels.as.dict->len == 0) return z_strdup(name);
    ZZ12MetricLabel *ordered = (ZZ12MetricLabel *)z_alloc(sizeof(ZZ12MetricLabel) * labels.as.dict->len);
    for (size_t index = 0; index < labels.as.dict->len; index++) {
        if (labels.as.dict->keys[index].tag != ZV_STRING || labels.as.dict->values[index].tag != ZV_STRING) z_fatal("metric labels must be string pairs");
        ordered[index].key = labels.as.dict->keys[index].as.s;
        ordered[index].value = labels.as.dict->values[index].as.s;
    }
    qsort(ordered, labels.as.dict->len, sizeof(ZZ12MetricLabel), z_z12_metric_label_compare);
    ZZ12Buffer key; z_z12_buffer_init(&key); z_z12_buffer_append(&key, name); z_z12_buffer_append(&key, "{");
    for (size_t index = 0; index < labels.as.dict->len; index++) {
        if (index != 0) z_z12_buffer_append(&key, ",");
        z_z12_buffer_append(&key, ordered[index].key); z_z12_buffer_append(&key, "="); z_z12_buffer_append(&key, ordered[index].value);
    }
    z_z12_buffer_append(&key, "}");
    char *result = z_strdup(key.data); z_z12_buffer_free(&key); return result;
}
static ZValue z_z12_metrics_new(void) {
    ZMetricsNative *metrics = (ZMetricsNative *)z_alloc(sizeof(ZMetricsNative));
    pthread_mutex_init(&metrics->mutex, NULL); metrics->counters = z_dict_from(NULL, 0); metrics->gauges = z_dict_from(NULL, 0); metrics->histograms = z_dict_from(NULL, 0);
    return z_z12_value(ZV_METRICS, ZK_METRICS, metrics);
}
static bool z_z12_metric_update(ZMetricsNative *metrics, const char *kind, const char *name, double value, ZValue labels) {
    char *key = z_z12_metric_key(name, labels);
    pthread_mutex_lock(&metrics->mutex);
    ZValue current;
    if (strcmp(kind, "counter") == 0) {
        double previous = z_z12_dict_find(metrics->counters, key, &current) ? z_z12_number(current, "counter") : 0;
        z_set_index(metrics->counters, z_string(key), z_float(previous + value));
    } else if (strcmp(kind, "gauge") == 0) z_set_index(metrics->gauges, z_string(key), z_float(value));
    else {
        ZValue histogram;
        if (!z_z12_dict_find(metrics->histograms, key, &histogram)) histogram = z_dict_from(NULL, 0);
        ZValue count, sum, min, max;
        double count_value = z_z12_dict_find(histogram, "count", &count) ? z_z12_number(count, "histogram count") : 0;
        double sum_value = z_z12_dict_find(histogram, "sum", &sum) ? z_z12_number(sum, "histogram sum") : 0;
        double min_value = z_z12_dict_find(histogram, "min", &min) ? z_z12_number(min, "histogram min") : value;
        double max_value = z_z12_dict_find(histogram, "max", &max) ? z_z12_number(max, "histogram max") : value;
        if (value < min_value) min_value = value;
        if (value > max_value) max_value = value;
        z_set_index(histogram, z_string("count"), z_int((int64_t)count_value + 1));
        z_set_index(histogram, z_string("sum"), z_float(sum_value + value)); z_set_index(histogram, z_string("min"), z_float(min_value)); z_set_index(histogram, z_string("max"), z_float(max_value));
        z_set_index(metrics->histograms, z_string(key), histogram);
    }
    pthread_mutex_unlock(&metrics->mutex);
    return true;
}
static ZValue z_z12_metrics_snapshot(ZMetricsNative *metrics) {
    pthread_mutex_lock(&metrics->mutex);
    ZValue pairs[3] = {z_pair(z_string("counters"), z_z12_dict_clone(metrics->counters)), z_pair(z_string("gauges"), z_z12_dict_clone(metrics->gauges)), z_pair(z_string("histograms"), z_z12_dict_clone(metrics->histograms))};
    ZValue result = z_dict_from(pairs, 3); pthread_mutex_unlock(&metrics->mutex); return result;
}
static bool z_z12_metrics_reset(ZMetricsNative *metrics) { pthread_mutex_lock(&metrics->mutex); metrics->counters=z_dict_from(NULL,0);metrics->gauges=z_dict_from(NULL,0);metrics->histograms=z_dict_from(NULL,0);pthread_mutex_unlock(&metrics->mutex);return true; }

/* Tracing. */
typedef struct { pthread_mutex_t mutex; char *trace_id; char *span_id; char *parent_id; char *name; double start_ms; double end_ms; char *status; ZValue attributes; ZValue events; bool active; } ZTraceNative;
static pthread_mutex_t z_z12_random_mutex = PTHREAD_MUTEX_INITIALIZER;
static char *z_z12_random_hex(size_t bytes) {
    unsigned char *raw = (unsigned char *)malloc(bytes); if (raw == NULL) z_fatal("out of memory generating random id");
    FILE *source = fopen("/dev/urandom", "rb");
    bool read_ok = source != NULL && fread(raw, 1, bytes, source) == bytes;
    if (source != NULL) fclose(source);
    if (!read_ok) { pthread_mutex_lock(&z_z12_random_mutex); static bool seeded=false; if(!seeded){srand((unsigned int)(time(NULL)^getpid()));seeded=true;} for(size_t i=0;i<bytes;i++)raw[i]=(unsigned char)(rand()&0xff); pthread_mutex_unlock(&z_z12_random_mutex); }
    char *result = (char *)z_alloc(bytes * 2 + 1); static const char digits[]="0123456789abcdef";
    for (size_t index=0; index<bytes; index++){result[index*2]=digits[raw[index]>>4];result[index*2+1]=digits[raw[index]&15];} result[bytes*2]='\0'; free(raw); return result;
}
static ZTraceNative *z_z12_expect_trace(ZValue value) { if (value.tag != ZV_TRACE_SPAN || value.as.p == NULL) z_fatal("expected TraceSpan"); return (ZTraceNative *)value.as.p; }
static ZValue z_z12_trace_new(const char *name, ZValue attributes, const char *trace_id, const char *parent_id) {
    if (attributes.tag != ZV_DICT) z_fatal("trace attributes must be dictionary");
    ZTraceNative *span=(ZTraceNative*)z_alloc(sizeof(ZTraceNative)); memset(span,0,sizeof(*span));pthread_mutex_init(&span->mutex,NULL);
    span->trace_id=z_strdup(trace_id==NULL?z_z12_random_hex(16):trace_id);span->span_id=z_z12_random_hex(8);span->parent_id=z_strdup(parent_id==NULL?"":parent_id);span->name=z_strdup(name);span->start_ms=z_z12_monotonic_ms();span->status=z_strdup("active");span->attributes=z_z12_dict_clone(attributes);span->events=z_array_from(NULL,0);span->active=true;
    return z_z12_value(ZV_TRACE_SPAN,ZK_TRACE_SPAN,span);
}
static ZValue z_z12_trace_finish(ZTraceNative *span,const char *status){pthread_mutex_lock(&span->mutex);if(span->active){span->active=false;span->end_ms=z_z12_monotonic_ms();span->status=z_strdup(status);}ZValue pairs[10]={z_pair(z_string("traceId"),z_string(span->trace_id)),z_pair(z_string("spanId"),z_string(span->span_id)),z_pair(z_string("parentId"),z_string(span->parent_id)),z_pair(z_string("name"),z_string(span->name)),z_pair(z_string("startMs"),z_float(span->start_ms)),z_pair(z_string("endMs"),z_float(span->end_ms)),z_pair(z_string("durationMs"),z_float(span->end_ms-span->start_ms)),z_pair(z_string("status"),z_string(span->status)),z_pair(z_string("attributes"),z_z12_dict_clone(span->attributes)),z_pair(z_string("events"),span->events)};ZValue result=z_dict_from(pairs,10);pthread_mutex_unlock(&span->mutex);return result;}

/* Fixed-window rate limiter. */
typedef struct ZRateBucket { char *key; int64_t count; int64_t started_ms; struct ZRateBucket *next; } ZRateBucket;
typedef struct { pthread_mutex_t mutex; int64_t limit; int64_t window_ms; ZRateBucket *buckets; } ZRateLimiterNative;
static ZRateLimiterNative *z_z12_expect_limiter(ZValue value){if(value.tag!=ZV_RATE_LIMITER||value.as.p==NULL)z_fatal("expected RateLimiter");return(ZRateLimiterNative*)value.as.p;}
static ZValue z_z12_rate_allow(ZRateLimiterNative *limiter,const char *key){int64_t now=z_z12_now_ms();pthread_mutex_lock(&limiter->mutex);ZRateBucket *bucket=limiter->buckets;while(bucket!=NULL&&strcmp(bucket->key,key)!=0)bucket=bucket->next;if(bucket==NULL){bucket=(ZRateBucket*)z_alloc(sizeof(ZRateBucket));bucket->key=z_strdup(key);bucket->count=0;bucket->started_ms=now;bucket->next=limiter->buckets;limiter->buckets=bucket;}if(now-bucket->started_ms>=limiter->window_ms){bucket->count=0;bucket->started_ms=now;}bool allowed=bucket->count<limiter->limit;if(allowed)bucket->count++;int64_t remaining=limiter->limit-bucket->count;if(remaining<0)remaining=0;int64_t retry=allowed?0:limiter->window_ms-(now-bucket->started_ms);ZValue pairs[3]={z_pair(z_string("allowed"),z_bool(allowed)),z_pair(z_string("remaining"),z_int(remaining)),z_pair(z_string("retryAfterMs"),z_int(retry))};ZValue result=z_dict_from(pairs,3);pthread_mutex_unlock(&limiter->mutex);return result;}
static bool z_z12_rate_reset(ZRateLimiterNative *limiter,const char *key){pthread_mutex_lock(&limiter->mutex);ZRateBucket **slot=&limiter->buckets;while(*slot!=NULL){if(strcmp((*slot)->key,key)==0){*slot=(*slot)->next;break;}slot=&(*slot)->next;}pthread_mutex_unlock(&limiter->mutex);return true;}

/* Persistent sessions. SQLite owns its database; Redis stores use an existing client. */
#if defined(ZUMBRA_ENABLE_SQLITE) || defined(ZUMBRA_ENABLE_REDIS)
typedef enum { Z_SESSION_SQLITE = 1, Z_SESSION_REDIS = 2 } ZSessionDriver;
typedef struct ZSessionNative {
    ZSessionDriver driver;
    pthread_mutex_t mutex;
    bool open;
    void *backend;
    char *prefix;
    struct ZSessionNative *next;
} ZSessionNative;
static ZSessionNative *z_z12_sessions = NULL;
static pthread_mutex_t z_z12_sessions_mutex = PTHREAD_MUTEX_INITIALIZER;
static ZSessionNative *z_z12_expect_session(ZValue value) {
    if (value.tag != ZV_SESSION_STORE || value.as.p == NULL) z_fatal("expected SessionStore");
    return (ZSessionNative *)value.as.p;
}
static void z_z12_register_session(ZSessionNative *store) {
    pthread_mutex_lock(&z_z12_sessions_mutex);
    store->next = z_z12_sessions;
    z_z12_sessions = store;
    pthread_mutex_unlock(&z_z12_sessions_mutex);
}
#endif

#if defined(ZUMBRA_ENABLE_SQLITE)
static sqlite3 *z_z12_sqlite_session_database(ZSessionNative *store) {
    if (store->driver != Z_SESSION_SQLITE || store->backend == NULL) z_fatal("session store is not SQLite");
    return (sqlite3 *)store->backend;
}
static void z_z12_sqlite_session_exec(ZSessionNative *store, const char *sql) {
    sqlite3 *database = z_z12_sqlite_session_database(store);
    char *error = NULL;
    int code = sqlite3_exec(database, sql, NULL, NULL, &error);
    if (code != SQLITE_OK) {
        const char *message = error == NULL ? sqlite3_errmsg(database) : error;
        z_fatal("session SQLite: %s", message);
    }
    if (error != NULL) sqlite3_free(error);
}
static ZValue z_z12_sqlite_session_open(const char *path) {
    sqlite3 *database = NULL;
    int code = sqlite3_open_v2(path, &database, SQLITE_OPEN_READWRITE | SQLITE_OPEN_CREATE | SQLITE_OPEN_FULLMUTEX | SQLITE_OPEN_URI, NULL);
    if (code != SQLITE_OK) {
        const char *message = database == NULL ? sqlite3_errstr(code) : sqlite3_errmsg(database);
        if (database != NULL) sqlite3_close_v2(database);
        z_fatal("open session database: %s", message);
    }
    sqlite3_busy_timeout(database, 5000);
    ZSessionNative *store = (ZSessionNative *)z_alloc(sizeof(ZSessionNative));
    memset(store, 0, sizeof(*store));
    store->driver = Z_SESSION_SQLITE;
    pthread_mutex_init(&store->mutex, NULL);
    store->backend = database;
    store->open = true;
    z_z12_sqlite_session_exec(store, "CREATE TABLE IF NOT EXISTS _zumbra_sessions (id TEXT PRIMARY KEY, data TEXT NOT NULL, expires_at INTEGER NOT NULL, updated_at INTEGER NOT NULL)");
    z_z12_register_session(store);
    return z_z12_value(ZV_SESSION_STORE, ZK_SESSION_STORE, store);
}
static void z_z12_sqlite_session_bind_text(sqlite3_stmt *statement, int index, const char *value) {
    if (sqlite3_bind_text(statement, index, value, -1, SQLITE_TRANSIENT) != SQLITE_OK) z_fatal("bind session text failed");
}
static char *z_z12_sqlite_session_create(ZSessionNative *store, ZValue data, int64_t ttl_ms, const char *forced_id) {
    if (data.tag != ZV_DICT) z_fatal("session data must be dictionary");
    if (ttl_ms <= 0) ttl_ms = 86400000;
    sqlite3 *database = z_z12_sqlite_session_database(store);
    char *id = forced_id == NULL ? z_z12_random_hex(32) : z_strdup(forced_id);
    char *json = z_z12_json_stringify(data);
    int64_t now = z_z12_now_ms();
    sqlite3_stmt *statement = NULL;
    const char *sql = "INSERT OR REPLACE INTO _zumbra_sessions(id,data,expires_at,updated_at) VALUES(?,?,?,?)";
    if (sqlite3_prepare_v2(database, sql, -1, &statement, NULL) != SQLITE_OK) z_fatal("prepare session set: %s", sqlite3_errmsg(database));
    z_z12_sqlite_session_bind_text(statement, 1, id);
    z_z12_sqlite_session_bind_text(statement, 2, json);
    sqlite3_bind_int64(statement, 3, now + ttl_ms);
    sqlite3_bind_int64(statement, 4, now);
    if (sqlite3_step(statement) != SQLITE_DONE) z_fatal("set session: %s", sqlite3_errmsg(database));
    sqlite3_finalize(statement);
    return id;
}
static ZValue z_z12_sqlite_session_get(ZSessionNative *store, const char *id) {
    sqlite3 *database = z_z12_sqlite_session_database(store);
    sqlite3_stmt *statement = NULL;
    if (sqlite3_prepare_v2(database, "SELECT data,expires_at FROM _zumbra_sessions WHERE id=?", -1, &statement, NULL) != SQLITE_OK) z_fatal("prepare session get");
    z_z12_sqlite_session_bind_text(statement, 1, id);
    int code = sqlite3_step(statement);
    if (code == SQLITE_DONE) { sqlite3_finalize(statement); return z_null(); }
    if (code != SQLITE_ROW) z_fatal("get session: %s", sqlite3_errmsg(database));
    int64_t expires = sqlite3_column_int64(statement, 1);
    const char *data = (const char *)sqlite3_column_text(statement, 0);
    char *copy = z_strdup(data == NULL ? "{}" : data);
    sqlite3_finalize(statement);
    if (expires <= z_z12_now_ms()) {
        sqlite3_stmt *remove = NULL;
        sqlite3_prepare_v2(database, "DELETE FROM _zumbra_sessions WHERE id=?", -1, &remove, NULL);
        z_z12_sqlite_session_bind_text(remove, 1, id);
        sqlite3_step(remove);
        sqlite3_finalize(remove);
        return z_null();
    }
    return z_z12_json_parse(copy);
}
static bool z_z12_sqlite_session_delete(ZSessionNative *store, const char *id) {
    sqlite3 *database = z_z12_sqlite_session_database(store);
    sqlite3_stmt *statement = NULL;
    sqlite3_prepare_v2(database, "DELETE FROM _zumbra_sessions WHERE id=?", -1, &statement, NULL);
    z_z12_sqlite_session_bind_text(statement, 1, id);
    sqlite3_step(statement);
    sqlite3_finalize(statement);
    return true;
}
static bool z_z12_sqlite_session_touch(ZSessionNative *store, const char *id, int64_t ttl_ms) {
    if (ttl_ms <= 0) ttl_ms = 86400000;
    sqlite3 *database = z_z12_sqlite_session_database(store);
    sqlite3_stmt *statement = NULL;
    sqlite3_prepare_v2(database, "UPDATE _zumbra_sessions SET expires_at=?,updated_at=? WHERE id=?", -1, &statement, NULL);
    int64_t now = z_z12_now_ms();
    sqlite3_bind_int64(statement, 1, now + ttl_ms);
    sqlite3_bind_int64(statement, 2, now);
    z_z12_sqlite_session_bind_text(statement, 3, id);
    sqlite3_step(statement);
    bool changed = sqlite3_changes(database) > 0;
    sqlite3_finalize(statement);
    return changed;
}
static int64_t z_z12_sqlite_session_cleanup(ZSessionNative *store) {
    sqlite3 *database = z_z12_sqlite_session_database(store);
    sqlite3_stmt *statement = NULL;
    sqlite3_prepare_v2(database, "DELETE FROM _zumbra_sessions WHERE expires_at<=?", -1, &statement, NULL);
    sqlite3_bind_int64(statement, 1, z_z12_now_ms());
    sqlite3_step(statement);
    int64_t changed = sqlite3_changes64(database);
    sqlite3_finalize(statement);
    return changed;
}
static bool z_z12_sqlite_session_close(ZSessionNative *store) {
    if (store->open && store->backend != NULL) {
        sqlite3_close_v2((sqlite3 *)store->backend);
        store->backend = NULL;
        store->open = false;
    }
    return true;
}
#endif

/* PostgreSQL object API. Uses libpq and keeps transactions/statements pinned to a connection. */
#if defined(ZUMBRA_ENABLE_POSTGRES)
typedef struct ZPostgresNative { pthread_mutex_t mutex; PGconn *connection; bool open; char *dsn; int max_open; int max_idle; int64_t max_lifetime_ms; int64_t max_idle_time_ms; struct ZPostgresNative *next; } ZPostgresNative;
typedef struct { ZPostgresNative *database; char *sql; char *name; bool open; } ZPostgresStatementNative;
typedef struct { ZPostgresNative *database; bool active; } ZPostgresTransactionNative;
typedef struct { PGresult *result; int row; int rows; int columns; ZValue names; } ZPostgresRowsNative;
static ZPostgresNative *z_z12_postgres_databases=NULL;static pthread_mutex_t z_z12_postgres_mutex=PTHREAD_MUTEX_INITIALIZER;static atomic_uint z_z12_statement_counter=1;
static ZPostgresNative *z_z12_expect_postgres(ZValue value){if(value.tag!=ZV_POSTGRES_DATABASE||value.as.p==NULL)z_fatal("expected PostgresDatabase");return(ZPostgresNative*)value.as.p;}
static ZPostgresStatementNative *z_z12_expect_pg_statement(ZValue value){if(value.tag!=ZV_POSTGRES_STATEMENT||value.as.p==NULL)z_fatal("expected PostgresStatement");return(ZPostgresStatementNative*)value.as.p;}
static ZPostgresTransactionNative *z_z12_expect_pg_transaction(ZValue value){if(value.tag!=ZV_POSTGRES_TRANSACTION||value.as.p==NULL)z_fatal("expected PostgresTransaction");return(ZPostgresTransactionNative*)value.as.p;}
static void z_z12_pg_check(ZPostgresNative *database){if(database==NULL||!database->open||database->connection==NULL||PQstatus(database->connection)!=CONNECTION_OK)z_fatal("PostgreSQL database is closed or unavailable");}
static void z_z12_pg_params(ZValue params,const char ***values,int **lengths,int **formats,int *count,char ***owned){if(params.tag!=ZV_ARRAY)z_fatal("PostgreSQL parameters must be array");*count=(int)params.as.array->len;*values=(const char**)calloc((size_t)*count,sizeof(char*));*lengths=(int*)calloc((size_t)*count,sizeof(int));*formats=(int*)calloc((size_t)*count,sizeof(int));*owned=(char**)calloc((size_t)*count,sizeof(char*));if((*count>0)&&(*values==NULL||*lengths==NULL||*formats==NULL||*owned==NULL))z_fatal("out of memory encoding PostgreSQL parameters");for(int i=0;i<*count;i++){ZValue value=params.as.array->items[i];if(value.tag==ZV_NULL){(*values)[i]=NULL;continue;}if(value.tag==ZV_BUFFER){size_t length=value.as.buffer->len*value.as.buffer->elem_size;if(length>INT_MAX)z_fatal("PostgreSQL blob too large");(*values)[i]=(const char*)value.as.buffer->data;(*lengths)[i]=(int)length;(*formats)[i]=1;continue;}ZValue text=z_to_string_value(value);(*owned)[i]=z_z12_heap_strdup(text.as.s);(*values)[i]=(*owned)[i];}}
static void z_z12_pg_free_params(const char **values,int *lengths,int *formats,int count,char **owned){(void)values;(void)count;if(owned!=NULL){for(int i=0;i<count;i++)free(owned[i]);}free((void*)values);free(lengths);free(formats);free(owned);}
static ZValue z_z12_pg_value(PGresult *result,int row,int column){if(PQgetisnull(result,row,column))return z_null();int format=PQfformat(result,column);int length=PQgetlength(result,row,column);const char *value=PQgetvalue(result,row,column);if(format==1){ZValue bytes=z_bytes((size_t)length);if(length>0)memcpy(bytes.as.buffer->data,value,(size_t)length);return bytes;}Oid type=PQftype(result,column);if(type==16)return z_bool(value[0]=='t'||strcmp(value,"1")==0);if(type==20||type==21||type==23){char*end=NULL;errno=0;long long parsed=strtoll(value,&end,10);if(errno==0&&end!=NULL&&*end=='\0')return z_int((int64_t)parsed);}if(type==700||type==701||type==1700){char*end=NULL;errno=0;double parsed=strtod(value,&end);if(errno==0&&end!=NULL&&*end=='\0')return z_float(parsed);}if(type==114||type==3802){return z_z12_json_parse(value);}return z_string(value);}
static ZValue z_z12_pg_result_rows(PGresult *result){int rows=PQntuples(result),columns=PQnfields(result);ZValue *items=rows==0?NULL:(ZValue*)z_alloc(sizeof(ZValue)*(size_t)rows);for(int row=0;row<rows;row++){ZValue *pairs=columns==0?NULL:(ZValue*)z_alloc(sizeof(ZValue)*(size_t)columns);for(int column=0;column<columns;column++)pairs[column]=z_pair(z_string(PQfname(result,column)),z_z12_pg_value(result,row,column));items[row]=z_dict_from(pairs,(size_t)columns);}return z_array_from(items,(size_t)rows);}
static ZValue z_z12_pg_exec(ZPostgresNative *database,const char *sql,ZValue params,bool query){z_z12_pg_check(database);const char **values=NULL;int *lengths=NULL,*formats=NULL,count=0;char **owned=NULL;z_z12_pg_params(params,&values,&lengths,&formats,&count,&owned);pthread_mutex_lock(&database->mutex);PGresult *result=PQexecParams(database->connection,sql,count,NULL,values,lengths,formats,0);ExecStatusType status=PQresultStatus(result);if(query){if(status!=PGRES_TUPLES_OK){const char *message=PQerrorMessage(database->connection);PQclear(result);pthread_mutex_unlock(&database->mutex);z_z12_pg_free_params(values,lengths,formats,count,owned);z_fatal("PostgreSQL query: %s",message);}ZValue rows=z_z12_pg_result_rows(result);PQclear(result);pthread_mutex_unlock(&database->mutex);z_z12_pg_free_params(values,lengths,formats,count,owned);return rows;}if(status!=PGRES_COMMAND_OK){const char *message=PQerrorMessage(database->connection);PQclear(result);pthread_mutex_unlock(&database->mutex);z_z12_pg_free_params(values,lengths,formats,count,owned);z_fatal("PostgreSQL exec: %s",message);}const char *affected=PQcmdTuples(result);int64_t count_value=(affected==NULL||*affected=='\0')?0:strtoll(affected,NULL,10);ZValue pairs[2]={z_pair(z_string("rowsAffected"),z_int(count_value)),z_pair(z_string("lastInsertId"),z_int(0))};ZValue output=z_dict_from(pairs,2);PQclear(result);pthread_mutex_unlock(&database->mutex);z_z12_pg_free_params(values,lengths,formats,count,owned);return output;}
static ZValue z_z12_pg_open(const char *dsn){PGconn *connection=PQconnectdb(dsn);if(connection==NULL||PQstatus(connection)!=CONNECTION_OK){const char *message=connection==NULL?"allocation failed":PQerrorMessage(connection);if(connection!=NULL)PQfinish(connection);z_fatal("open PostgreSQL: %s",message);}ZPostgresNative *database=(ZPostgresNative*)z_alloc(sizeof(ZPostgresNative));memset(database,0,sizeof(*database));pthread_mutex_init(&database->mutex,NULL);database->connection=connection;database->open=true;database->dsn=z_strdup(dsn);database->max_open=4;database->max_idle=2;pthread_mutex_lock(&z_z12_postgres_mutex);database->next=z_z12_postgres_databases;z_z12_postgres_databases=database;pthread_mutex_unlock(&z_z12_postgres_mutex);return z_z12_value(ZV_POSTGRES_DATABASE,ZK_POSTGRES_DATABASE,database);}
static ZValue z_z12_pg_rows_next(void *raw,bool *ok){ZPostgresRowsNative *rows=(ZPostgresRowsNative*)raw;if(rows->row>=rows->rows){*ok=false;return z_null();}ZValue *pairs=rows->columns==0?NULL:(ZValue*)z_alloc(sizeof(ZValue)*(size_t)rows->columns);for(int column=0;column<rows->columns;column++)pairs[column]=z_pair(z_string(PQfname(rows->result,column)),z_z12_pg_value(rows->result,rows->row,column));rows->row++;*ok=true;return z_dict_from(pairs,(size_t)rows->columns);}
static ZValue z_z12_pg_rows_columns(void *raw){return((ZPostgresRowsNative*)raw)->names;}
static void z_z12_pg_rows_close(void *raw){ZPostgresRowsNative *rows=(ZPostgresRowsNative*)raw;if(rows->result!=NULL){PQclear(rows->result);rows->result=NULL;}}
static ZValue z_z12_pg_stream(ZPostgresNative *database,const char *sql,ZValue params){z_z12_pg_check(database);const char **values=NULL;int *lengths=NULL,*formats=NULL,count=0;char **owned=NULL;z_z12_pg_params(params,&values,&lengths,&formats,&count,&owned);pthread_mutex_lock(&database->mutex);PGresult *result=PQexecParams(database->connection,sql,count,NULL,values,lengths,formats,0);pthread_mutex_unlock(&database->mutex);z_z12_pg_free_params(values,lengths,formats,count,owned);if(PQresultStatus(result)!=PGRES_TUPLES_OK){const char *message=PQerrorMessage(database->connection);PQclear(result);z_fatal("PostgreSQL stream: %s",message);}ZPostgresRowsNative *rows=(ZPostgresRowsNative*)z_alloc(sizeof(ZPostgresRowsNative));rows->result=result;rows->row=0;rows->rows=PQntuples(result);rows->columns=PQnfields(result);ZValue *names=rows->columns==0?NULL:(ZValue*)z_alloc(sizeof(ZValue)*(size_t)rows->columns);for(int column=0;column<rows->columns;column++)names[column]=z_string(PQfname(result,column));rows->names=z_array_from(names,(size_t)rows->columns);return z_z12_rows_new(rows,z_z12_pg_rows_next,z_z12_pg_rows_columns,z_z12_pg_rows_close);}
#endif

/* Redis object API using hiredis. */
#if defined(ZUMBRA_ENABLE_REDIS)
typedef struct ZRedisNative { pthread_mutex_t mutex; redisContext *context; bool open; char *host; int port; int database; struct ZRedisNative *next; } ZRedisNative;
static ZRedisNative *z_z12_redis_clients=NULL;static pthread_mutex_t z_z12_redis_mutex=PTHREAD_MUTEX_INITIALIZER;
static ZRedisNative *z_z12_expect_redis(ZValue value){if(value.tag!=ZV_REDIS_CLIENT||value.as.p==NULL)z_fatal("expected RedisClient");return(ZRedisNative*)value.as.p;}
static void z_z12_redis_check(ZRedisNative *client){if(client==NULL||!client->open||client->context==NULL||client->context->err)z_fatal("Redis client is closed or unavailable");}
static redisReply *z_z12_redis_command(ZRedisNative *client,const char *format,...){pthread_mutex_lock(&client->mutex);z_z12_redis_check(client);va_list arguments;va_start(arguments,format);redisReply *reply=(redisReply*)redisvCommand(client->context,format,arguments);va_end(arguments);if(reply==NULL){const char *message=client->context==NULL?"no context":client->context->errstr;pthread_mutex_unlock(&client->mutex);z_fatal("Redis command: %s",message);}if(reply->type==REDIS_REPLY_ERROR){char *message=z_strdup(reply->str==NULL?"Redis error":reply->str);freeReplyObject(reply);pthread_mutex_unlock(&client->mutex);z_fatal("Redis command: %s",message);}pthread_mutex_unlock(&client->mutex);return reply;}
static ZValue z_z12_redis_open(const char *host,int port,const char *password,int database){struct timeval timeout={5,0};redisContext *context=redisConnectWithTimeout(host,port,timeout);if(context==NULL||context->err){const char *message=context==NULL?"allocation failed":context->errstr;if(context!=NULL)redisFree(context);z_fatal("open Redis: %s",message);}if(password!=NULL&&*password!='\0'){redisReply *reply=(redisReply*)redisCommand(context,"AUTH %s",password);if(reply==NULL||reply->type==REDIS_REPLY_ERROR){const char *message=reply!=NULL&&reply->str!=NULL?reply->str:"authentication failed";if(reply!=NULL)freeReplyObject(reply);redisFree(context);z_fatal("Redis AUTH: %s",message);}freeReplyObject(reply);}if(database!=0){redisReply *reply=(redisReply*)redisCommand(context,"SELECT %d",database);if(reply==NULL||reply->type==REDIS_REPLY_ERROR){const char *message=reply!=NULL&&reply->str!=NULL?reply->str:"select failed";if(reply!=NULL)freeReplyObject(reply);redisFree(context);z_fatal("Redis SELECT: %s",message);}freeReplyObject(reply);}ZRedisNative *client=(ZRedisNative*)z_alloc(sizeof(ZRedisNative));memset(client,0,sizeof(*client));pthread_mutex_init(&client->mutex,NULL);client->context=context;client->open=true;client->host=z_strdup(host);client->port=port;client->database=database;pthread_mutex_lock(&z_z12_redis_mutex);client->next=z_z12_redis_clients;z_z12_redis_clients=client;pthread_mutex_unlock(&z_z12_redis_mutex);return z_z12_value(ZV_REDIS_CLIENT,ZK_REDIS_CLIENT,client);}
#endif

#if defined(ZUMBRA_ENABLE_REDIS)
static ZRedisNative *z_z12_redis_session_client(ZSessionNative *store) {
    if (store->driver != Z_SESSION_REDIS || store->backend == NULL) z_fatal("session store is not Redis");
    return (ZRedisNative *)store->backend;
}
static char *z_z12_redis_session_key(ZSessionNative *store, const char *id) {
    const char *prefix = store->prefix == NULL ? "zumbra:session:" : store->prefix;
    size_t length = strlen(prefix) + strlen(id) + 1;
    char *key = (char *)z_alloc(length);
    snprintf(key, length, "%s%s", prefix, id);
    return key;
}
static ZValue z_z12_redis_session_open(ZValue client_value, const char *prefix) {
    ZRedisNative *client = z_z12_expect_redis(client_value);
    z_z12_redis_check(client);
    ZSessionNative *store = (ZSessionNative *)z_alloc(sizeof(ZSessionNative));
    memset(store, 0, sizeof(*store));
    store->driver = Z_SESSION_REDIS;
    pthread_mutex_init(&store->mutex, NULL);
    store->backend = client;
    store->prefix = z_strdup(prefix == NULL || *prefix == '\0' ? "zumbra:session:" : prefix);
    store->open = true;
    z_z12_register_session(store);
    return z_z12_value(ZV_SESSION_STORE, ZK_SESSION_STORE, store);
}
static char *z_z12_redis_session_create(ZSessionNative *store, ZValue data, int64_t ttl_ms, const char *forced_id) {
    if (data.tag != ZV_DICT) z_fatal("session data must be dictionary");
    if (ttl_ms <= 0) ttl_ms = 86400000;
    char *id = forced_id == NULL ? z_z12_random_hex(32) : z_strdup(forced_id);
    char *key = z_z12_redis_session_key(store, id);
    char *json = z_z12_json_stringify(data);
    redisReply *reply = z_z12_redis_command(z_z12_redis_session_client(store), "SET %s %s PX %lld", key, json, (long long)ttl_ms);
    freeReplyObject(reply);
    return id;
}
static ZValue z_z12_redis_session_get(ZSessionNative *store, const char *id) {
    char *key = z_z12_redis_session_key(store, id);
    redisReply *reply = z_z12_redis_command(z_z12_redis_session_client(store), "GET %s", key);
    if (reply->type == REDIS_REPLY_NIL) { freeReplyObject(reply); return z_null(); }
    if (reply->type != REDIS_REPLY_STRING || reply->str == NULL) { freeReplyObject(reply); z_fatal("invalid Redis session value"); }
    char *json = z_strdup(reply->str);
    freeReplyObject(reply);
    return z_z12_json_parse(json);
}
static bool z_z12_redis_session_delete(ZSessionNative *store, const char *id) {
    char *key = z_z12_redis_session_key(store, id);
    redisReply *reply = z_z12_redis_command(z_z12_redis_session_client(store), "DEL %s", key);
    freeReplyObject(reply);
    return true;
}
static bool z_z12_redis_session_touch(ZSessionNative *store, const char *id, int64_t ttl_ms) {
    if (ttl_ms <= 0) ttl_ms = 86400000;
    char *key = z_z12_redis_session_key(store, id);
    redisReply *reply = z_z12_redis_command(z_z12_redis_session_client(store), "PEXPIRE %s %lld", key, (long long)ttl_ms);
    bool changed = reply->integer > 0;
    freeReplyObject(reply);
    return changed;
}
static char *z_z12_redis_session_rotate(ZSessionNative *store, const char *old_id, int64_t ttl_ms) {
    if (ttl_ms <= 0) ttl_ms = 86400000;
    char *new_id = z_z12_random_hex(32);
    char *old_key = z_z12_redis_session_key(store, old_id);
    char *new_key = z_z12_redis_session_key(store, new_id);
    const char *script = "local v=redis.call('GET',KEYS[1]); if not v then return 0 end; redis.call('SET',KEYS[2],v,'PX',ARGV[1]); redis.call('DEL',KEYS[1]); return 1";
    redisReply *reply = z_z12_redis_command(z_z12_redis_session_client(store), "EVAL %s 2 %s %s %lld", script, old_key, new_key, (long long)ttl_ms);
    bool rotated = reply->type == REDIS_REPLY_INTEGER && reply->integer == 1;
    freeReplyObject(reply);
    if (!rotated) z_fatal("session not found");
    return new_id;
}
#endif

#if defined(ZUMBRA_ENABLE_SQLITE) || defined(ZUMBRA_ENABLE_REDIS)
static char *z_z12_session_create(ZSessionNative *store, ZValue data, int64_t ttl_ms, const char *forced_id) {
    if (!store->open) z_fatal("session store is closed");
#if defined(ZUMBRA_ENABLE_SQLITE)
    if (store->driver == Z_SESSION_SQLITE) return z_z12_sqlite_session_create(store, data, ttl_ms, forced_id);
#endif
#if defined(ZUMBRA_ENABLE_REDIS)
    if (store->driver == Z_SESSION_REDIS) return z_z12_redis_session_create(store, data, ttl_ms, forced_id);
#endif
    z_fatal("unknown session driver");
    return NULL;
}
static ZValue z_z12_session_get(ZSessionNative *store, const char *id) {
    if (!store->open) z_fatal("session store is closed");
#if defined(ZUMBRA_ENABLE_SQLITE)
    if (store->driver == Z_SESSION_SQLITE) return z_z12_sqlite_session_get(store, id);
#endif
#if defined(ZUMBRA_ENABLE_REDIS)
    if (store->driver == Z_SESSION_REDIS) return z_z12_redis_session_get(store, id);
#endif
    z_fatal("unknown session driver");
    return z_null();
}
static bool z_z12_session_delete(ZSessionNative *store, const char *id) {
    if (!store->open) z_fatal("session store is closed");
#if defined(ZUMBRA_ENABLE_SQLITE)
    if (store->driver == Z_SESSION_SQLITE) return z_z12_sqlite_session_delete(store, id);
#endif
#if defined(ZUMBRA_ENABLE_REDIS)
    if (store->driver == Z_SESSION_REDIS) return z_z12_redis_session_delete(store, id);
#endif
    z_fatal("unknown session driver");
    return false;
}
static bool z_z12_session_touch(ZSessionNative *store, const char *id, int64_t ttl_ms) {
    if (!store->open) z_fatal("session store is closed");
#if defined(ZUMBRA_ENABLE_SQLITE)
    if (store->driver == Z_SESSION_SQLITE) return z_z12_sqlite_session_touch(store, id, ttl_ms);
#endif
#if defined(ZUMBRA_ENABLE_REDIS)
    if (store->driver == Z_SESSION_REDIS) return z_z12_redis_session_touch(store, id, ttl_ms);
#endif
    z_fatal("unknown session driver");
    return false;
}
static int64_t z_z12_session_cleanup(ZSessionNative *store) {
    if (!store->open) z_fatal("session store is closed");
#if defined(ZUMBRA_ENABLE_SQLITE)
    if (store->driver == Z_SESSION_SQLITE) return z_z12_sqlite_session_cleanup(store);
#endif
#if defined(ZUMBRA_ENABLE_REDIS)
    if (store->driver == Z_SESSION_REDIS) return 0;
#endif
    z_fatal("unknown session driver");
    return 0;
}
static char *z_z12_session_rotate(ZSessionNative *store, const char *id, int64_t ttl_ms) {
    if (!store->open) z_fatal("session store is closed");
#if defined(ZUMBRA_ENABLE_REDIS)
    if (store->driver == Z_SESSION_REDIS) return z_z12_redis_session_rotate(store, id, ttl_ms);
#endif
    ZValue data = z_z12_session_get(store, id);
    if (data.tag == ZV_NULL) z_fatal("session not found");
    char *new_id = z_z12_session_create(store, data, ttl_ms, NULL);
    z_z12_session_delete(store, id);
    return new_id;
}
static bool z_z12_session_close(ZSessionNative *store) {
    pthread_mutex_lock(&store->mutex);
    if (!store->open) { pthread_mutex_unlock(&store->mutex); return true; }
#if defined(ZUMBRA_ENABLE_SQLITE)
    if (store->driver == Z_SESSION_SQLITE) z_z12_sqlite_session_close(store);
#endif
#if defined(ZUMBRA_ENABLE_REDIS)
    if (store->driver == Z_SESSION_REDIS) { store->backend = NULL; store->open = false; }
#endif
    pthread_mutex_unlock(&store->mutex);
    return true;
}
#endif

static ZValue z_z12_bound_method(ZValue receiver,const char *name){ZZ12BoundMethod *method=(ZZ12BoundMethod*)z_alloc(sizeof(ZZ12BoundMethod));method->receiver=receiver;method->name=z_strdup(name);return z_z12_value(ZV_Z12_METHOD,ZK_FUNCTION,method);}

static ZValue z_z12_call_builtin(const char *name,const ZValue *args,size_t argc,bool *handled){
    *handled=true;
    if(strcmp(name,"sqlRowsNext")==0){z_expect_args(name,argc,1);return z_z12_rows_next(z_z12_expect_rows(args[0]));}
    if(strcmp(name,"sqlRowsColumns")==0){z_expect_args(name,argc,1);return z_z12_rows_columns(z_z12_expect_rows(args[0]));}
    if(strcmp(name,"sqlRowsClose")==0){z_expect_args(name,argc,1);return z_bool(z_z12_rows_close(z_z12_expect_rows(args[0])));}
    if(strcmp(name,"sqlRowsOpen")==0){z_expect_args(name,argc,1);return z_bool(z_z12_rows_open(z_z12_expect_rows(args[0])));}

    if(strcmp(name,"configLoad")==0){z_expect_args(name,argc,1);return z_z12_config_load(z_as_cstring(args[0]));}
    if(strcmp(name,"configFrom")==0){z_expect_args(name,argc,1);return z_z12_config_new(args[0]);}
    if(strcmp(name,"configEnv")==0){if(argc>1)z_fatal("configEnv expects optional prefix");return z_z12_config_env(argc==0?"":z_as_cstring(args[0]));}
    if(strcmp(name,"configMerge")==0){z_expect_args(name,argc,2);ZConfigNative *left=z_z12_expect_config(args[0]);ZConfigNative *right=z_z12_expect_config(args[1]);pthread_mutex_lock(&left->mutex);ZValue values=z_z12_dict_clone(left->values);pthread_mutex_unlock(&left->mutex);pthread_mutex_lock(&right->mutex);for(size_t i=0;i<right->values.as.dict->len;i++)z_set_index(values,right->values.as.dict->keys[i],right->values.as.dict->values[i]);pthread_mutex_unlock(&right->mutex);return z_z12_config_new(values);}
    if(strcmp(name,"configRequired")==0||strcmp(name,"configString")==0||strcmp(name,"configInt")==0||strcmp(name,"configFloat")==0||strcmp(name,"configBool")==0){if(argc<2||argc>3)z_fatal("%s expects config, key and optional default",name);ZConfigNative *config=z_z12_expect_config(args[0]);const char *key=z_as_cstring(args[1]);ZValue fallback=argc==3?args[2]:z_null();bool has=argc==3;if(strcmp(name,"configRequired")==0)return z_z12_config_required(config,key,fallback,has);if(strcmp(name,"configString")==0)return z_z12_config_string(config,key,fallback,has);if(strcmp(name,"configInt")==0)return z_z12_config_int(config,key,fallback,has);if(strcmp(name,"configFloat")==0)return z_z12_config_float(config,key,fallback,has);return z_z12_config_bool(config,key,fallback,has);}
    if(strcmp(name,"configSecret")==0){z_expect_args(name,argc,2);ZConfigNative *config=z_z12_expect_config(args[0]);const char *key=z_as_cstring(args[1]);pthread_mutex_lock(&config->mutex);z_set_index(config->secrets,z_string(key),z_bool(true));pthread_mutex_unlock(&config->mutex);return args[0];}
    if(strcmp(name,"configRedacted")==0){z_expect_args(name,argc,1);ZConfigNative *config=z_z12_expect_config(args[0]);pthread_mutex_lock(&config->mutex);ZValue result=z_z12_dict_clone(config->values);for(size_t i=0;i<config->secrets.as.dict->len;i++)if(config->secrets.as.dict->keys[i].tag==ZV_STRING)z_set_index(result,config->secrets.as.dict->keys[i],z_string("[REDACTED]"));pthread_mutex_unlock(&config->mutex);return result;}

    if(strcmp(name,"logger")==0){if(argc<1||argc>3)z_fatal("logger expects name, optional level and optional path");return z_z12_logger_new(z_as_cstring(args[0]),argc>1?z_as_cstring(args[1]):"info",argc>2?z_as_cstring(args[2]):NULL);}
    if(strcmp(name,"loggerWith")==0){z_expect_args(name,argc,2);return z_z12_logger_with(z_z12_expect_logger(args[0]),args[1]);}
    if(strcmp(name,"loggerSetLevel")==0){z_expect_args(name,argc,2);ZLoggerNative *logger=z_z12_expect_logger(args[0]);pthread_mutex_lock(&logger->mutex);logger->level=z_z12_log_level(z_as_cstring(args[1]));pthread_mutex_unlock(&logger->mutex);return args[0];}
    if(strcmp(name,"loggerLog")==0){if(argc<3||argc>4)z_fatal("loggerLog expects logger, level, message and optional fields");return z_bool(z_z12_logger_log(z_z12_expect_logger(args[0]),z_as_cstring(args[1]),z_as_cstring(args[2]),argc==4?args[3]:z_dict_from(NULL,0)));}
    if(strcmp(name,"loggerClose")==0){z_expect_args(name,argc,1);return z_bool(z_z12_logger_close(z_z12_expect_logger(args[0])));}

    if(strcmp(name,"metrics")==0){z_expect_args(name,argc,0);return z_z12_metrics_new();}
    if(strcmp(name,"metricsCounter")==0||strcmp(name,"metricsGauge")==0||strcmp(name,"metricsHistogram")==0){if(argc<3||argc>4)z_fatal("metric update expects registry, name, value and optional labels");const char *kind=strcmp(name,"metricsCounter")==0?"counter":strcmp(name,"metricsGauge")==0?"gauge":"histogram";return z_bool(z_z12_metric_update(z_z12_expect_metrics(args[0]),kind,z_as_cstring(args[1]),z_z12_number(args[2],name),argc==4?args[3]:z_null()));}
    if(strcmp(name,"metricsSnapshot")==0){z_expect_args(name,argc,1);return z_z12_metrics_snapshot(z_z12_expect_metrics(args[0]));}
    if(strcmp(name,"metricsReset")==0){z_expect_args(name,argc,1);return z_bool(z_z12_metrics_reset(z_z12_expect_metrics(args[0])));}

    if(strcmp(name,"traceStart")==0){if(argc<1||argc>2)z_fatal("traceStart expects name and optional attributes");return z_z12_trace_new(z_as_cstring(args[0]),argc==2?args[1]:z_dict_from(NULL,0),NULL,NULL);}
    if(strcmp(name,"traceChild")==0){if(argc<2||argc>3)z_fatal("traceChild expects parent, name and optional attributes");ZTraceNative *parent=z_z12_expect_trace(args[0]);return z_z12_trace_new(z_as_cstring(args[1]),argc==3?args[2]:z_dict_from(NULL,0),parent->trace_id,parent->span_id);}
    if(strcmp(name,"traceSet")==0){z_expect_args(name,argc,3);ZTraceNative *span=z_z12_expect_trace(args[0]);pthread_mutex_lock(&span->mutex);z_set_index(span->attributes,z_string(z_as_cstring(args[1])),args[2]);pthread_mutex_unlock(&span->mutex);return args[0];}
    if(strcmp(name,"traceEvent")==0){if(argc<2||argc>3)z_fatal("traceEvent expects span, name and optional attributes");ZTraceNative *span=z_z12_expect_trace(args[0]);ZValue pairs[3]={z_pair(z_string("name"),z_string(z_as_cstring(args[1]))),z_pair(z_string("timestampMs"),z_int(z_z12_now_ms())),z_pair(z_string("attributes"),argc==3?args[2]:z_dict_from(NULL,0))};ZValue event=z_dict_from(pairs,3);pthread_mutex_lock(&span->mutex);ZArray *events=span->events.as.array;if(events->len==events->cap){size_t cap=events->cap==0?4:events->cap*2;ZValue *items=(ZValue*)z_alloc(sizeof(ZValue)*cap);if(events->len>0)memcpy(items,events->items,sizeof(ZValue)*events->len);events->items=items;events->cap=cap;}events->items[events->len++]=event;pthread_mutex_unlock(&span->mutex);return args[0];}
    if(strcmp(name,"traceFinish")==0){if(argc<1||argc>2)z_fatal("traceFinish expects span and optional status");return z_z12_trace_finish(z_z12_expect_trace(args[0]),argc==2?z_as_cstring(args[1]):"ok");}
    if(strcmp(name,"traceActive")==0){z_expect_args(name,argc,1);ZTraceNative *span=z_z12_expect_trace(args[0]);pthread_mutex_lock(&span->mutex);bool active=span->active;pthread_mutex_unlock(&span->mutex);return z_bool(active);}

    if(strcmp(name,"rateLimiter")==0){z_expect_args(name,argc,2);int64_t limit=z_as_i64(args[0]),window=z_as_i64(args[1]);if(limit<=0||window<=0)z_fatal("rateLimiter values must be positive");ZRateLimiterNative *limiter=(ZRateLimiterNative*)z_alloc(sizeof(ZRateLimiterNative));memset(limiter,0,sizeof(*limiter));pthread_mutex_init(&limiter->mutex,NULL);limiter->limit=limit;limiter->window_ms=window;return z_z12_value(ZV_RATE_LIMITER,ZK_RATE_LIMITER,limiter);}
    if(strcmp(name,"rateAllow")==0){z_expect_args(name,argc,2);return z_z12_rate_allow(z_z12_expect_limiter(args[0]),z_as_cstring(args[1]));}
    if(strcmp(name,"rateReset")==0){z_expect_args(name,argc,2);return z_bool(z_z12_rate_reset(z_z12_expect_limiter(args[0]),z_as_cstring(args[1])));}

    if(strcmp(name,"fileExists")==0){z_expect_args(name,argc,1);return z_bool(z_z12_file_exists(z_as_cstring(args[0])));}
    if(strcmp(name,"jsonReadFile")==0){z_expect_args(name,argc,1);return z_z12_json_read_file(z_as_cstring(args[0]));}
    if(strcmp(name,"jsonWriteFile")==0){if(argc<2||argc>3)z_fatal("jsonWriteFile expects path, value and optional pretty flag");return z_int(z_z12_json_write_file(z_as_cstring(args[0]),args[1]));}
    if(strcmp(name,"jsonReadResult")==0){z_expect_args(name,argc,1);Z_Z12_RETURN_RESULT(z_z12_json_read_file(z_as_cstring(args[0])));}
    if(strcmp(name,"jsonWriteResult")==0){if(argc<2||argc>3)z_fatal("jsonWriteResult expects path, value and optional pretty flag");Z_Z12_RETURN_RESULT(z_int(z_z12_json_write_file(z_as_cstring(args[0]),args[1])));}
    if(strcmp(name,"csvReadFile")==0){z_expect_args(name,argc,1);return z_z12_csv_read_file(z_as_cstring(args[0]));}
    if(strcmp(name,"csvWriteFile")==0){z_expect_args(name,argc,2);return z_int(z_z12_csv_write_file(z_as_cstring(args[0]),args[1]));}
    if(strcmp(name,"csvReadResult")==0){z_expect_args(name,argc,1);Z_Z12_RETURN_RESULT(z_z12_csv_read_file(z_as_cstring(args[0])));}
    if(strcmp(name,"csvWriteResult")==0){z_expect_args(name,argc,2);Z_Z12_RETURN_RESULT(z_int(z_z12_csv_write_file(z_as_cstring(args[0]),args[1])));}
    if(strcmp(name,"binaryEncode")==0){z_expect_args(name,argc,1);return z_z12_binary_encode(args[0]);}
    if(strcmp(name,"binaryDecode")==0){z_expect_args(name,argc,1);return z_z12_binary_decode(args[0]);}
    if(strcmp(name,"binaryWriteFile")==0){z_expect_args(name,argc,2);ZValue encoded=z_z12_binary_encode(args[1]);size_t length=encoded.as.buffer->len*encoded.as.buffer->elem_size;z_z12_atomic_write(z_as_cstring(args[0]),encoded.as.buffer->data,length,0600);return z_int((int64_t)length);}
    if(strcmp(name,"binaryReadFile")==0){z_expect_args(name,argc,1);size_t length=0;char *data=z_z12_read_text_file(z_as_cstring(args[0]),&length);ZValue encoded=z_bytes(length);if(length!=0)memcpy(encoded.as.buffer->data,data,length);free(data);return z_z12_binary_decode(encoded);}

#if defined(ZUMBRA_ENABLE_SQLITE)
    if(strcmp(name,"sessionSQLite")==0){z_expect_args(name,argc,1);return z_z12_sqlite_session_open(z_as_cstring(args[0]));}
#endif
#if defined(ZUMBRA_ENABLE_REDIS)
    if(strcmp(name,"sessionRedis")==0){if(argc<1||argc>2)z_fatal("sessionRedis expects RedisClient and optional prefix");return z_z12_redis_session_open(args[0],argc==2?z_as_cstring(args[1]):"zumbra:session:");}
#endif
#if defined(ZUMBRA_ENABLE_SQLITE) || defined(ZUMBRA_ENABLE_REDIS)
    if(strcmp(name,"sessionCreate")==0){z_expect_args(name,argc,3);ZSessionNative *store=z_z12_expect_session(args[0]);pthread_mutex_lock(&store->mutex);char *id=z_z12_session_create(store,args[1],z_as_i64(args[2]),NULL);pthread_mutex_unlock(&store->mutex);return z_string(id);}
    if(strcmp(name,"sessionGet")==0){z_expect_args(name,argc,2);ZSessionNative *store=z_z12_expect_session(args[0]);pthread_mutex_lock(&store->mutex);ZValue result=z_z12_session_get(store,z_as_cstring(args[1]));pthread_mutex_unlock(&store->mutex);return result;}
    if(strcmp(name,"sessionSet")==0){z_expect_args(name,argc,4);ZSessionNative *store=z_z12_expect_session(args[0]);pthread_mutex_lock(&store->mutex);z_z12_session_create(store,args[2],z_as_i64(args[3]),z_as_cstring(args[1]));pthread_mutex_unlock(&store->mutex);return z_bool(true);}
    if(strcmp(name,"sessionDelete")==0){z_expect_args(name,argc,2);ZSessionNative *store=z_z12_expect_session(args[0]);pthread_mutex_lock(&store->mutex);bool result=z_z12_session_delete(store,z_as_cstring(args[1]));pthread_mutex_unlock(&store->mutex);return z_bool(result);}
    if(strcmp(name,"sessionRotate")==0){z_expect_args(name,argc,3);ZSessionNative *store=z_z12_expect_session(args[0]);pthread_mutex_lock(&store->mutex);char *id=z_z12_session_rotate(store,z_as_cstring(args[1]),z_as_i64(args[2]));pthread_mutex_unlock(&store->mutex);return z_string(id);}
    if(strcmp(name,"sessionTouch")==0){z_expect_args(name,argc,3);ZSessionNative *store=z_z12_expect_session(args[0]);pthread_mutex_lock(&store->mutex);bool result=z_z12_session_touch(store,z_as_cstring(args[1]),z_as_i64(args[2]));pthread_mutex_unlock(&store->mutex);return z_bool(result);}
    if(strcmp(name,"sessionCleanup")==0){z_expect_args(name,argc,1);ZSessionNative *store=z_z12_expect_session(args[0]);pthread_mutex_lock(&store->mutex);int64_t count=z_z12_session_cleanup(store);pthread_mutex_unlock(&store->mutex);return z_int(count);}
    if(strcmp(name,"sessionClose")==0){z_expect_args(name,argc,1);return z_bool(z_z12_session_close(z_z12_expect_session(args[0])));}
#endif

#if defined(ZUMBRA_ENABLE_POSTGRES)
    if(strcmp(name,"postgresOpen")==0){if(argc<1||argc>2)z_fatal("postgresOpen expects DSN and optional options");ZValue result=z_z12_pg_open(z_as_cstring(args[0]));if(argc==2&&args[1].tag==ZV_DICT){ZPostgresNative *db=z_z12_expect_postgres(result);ZValue value;if(z_z12_dict_find(args[1],"maxOpen",&value))db->max_open=(int)z_as_i64(value);if(z_z12_dict_find(args[1],"maxIdle",&value))db->max_idle=(int)z_as_i64(value);}return result;}
    if(strcmp(name,"postgresConfigurePool")==0){z_expect_args(name,argc,5);ZPostgresNative *db=z_z12_expect_postgres(args[0]);db->max_open=(int)z_as_i64(args[1]);db->max_idle=(int)z_as_i64(args[2]);db->max_lifetime_ms=z_as_i64(args[3]);db->max_idle_time_ms=z_as_i64(args[4]);return args[0];}
    if(strcmp(name,"postgresPoolStats")==0){z_expect_args(name,argc,1);ZPostgresNative *db=z_z12_expect_postgres(args[0]);ZValue pairs[7]={z_pair(z_string("maxOpen"),z_int(db->max_open)),z_pair(z_string("open"),z_int(db->open?1:0)),z_pair(z_string("inUse"),z_int(0)),z_pair(z_string("idle"),z_int(db->open?1:0)),z_pair(z_string("waitCount"),z_int(0)),z_pair(z_string("waitDurationMs"),z_int(0)),z_pair(z_string("maxIdleClosed"),z_int(0))};return z_dict_from(pairs,7);}
    if(strcmp(name,"postgresPing")==0){z_expect_args(name,argc,1);ZPostgresNative *db=z_z12_expect_postgres(args[0]);z_z12_pg_check(db);PGresult *result=PQexec(db->connection,"SELECT 1");bool ok=PQresultStatus(result)==PGRES_TUPLES_OK;PQclear(result);return z_bool(ok);}
    if(strcmp(name,"postgresClose")==0){z_expect_args(name,argc,1);ZPostgresNative *db=z_z12_expect_postgres(args[0]);pthread_mutex_lock(&db->mutex);if(db->open){PQfinish(db->connection);db->connection=NULL;db->open=false;}pthread_mutex_unlock(&db->mutex);return z_bool(true);}
    if(strcmp(name,"postgresIsOpen")==0){z_expect_args(name,argc,1);ZPostgresNative *db=z_z12_expect_postgres(args[0]);return z_bool(db->open&&db->connection!=NULL&&PQstatus(db->connection)==CONNECTION_OK);}
    if(strcmp(name,"postgresExecDb")==0){z_expect_args(name,argc,3);return z_z12_pg_exec(z_z12_expect_postgres(args[0]),z_as_cstring(args[1]),args[2],false);}
    if(strcmp(name,"postgresQueryDb")==0){z_expect_args(name,argc,3);return z_z12_pg_exec(z_z12_expect_postgres(args[0]),z_as_cstring(args[1]),args[2],true);}
    if(strcmp(name,"postgresQueryOne")==0){z_expect_args(name,argc,3);ZValue rows=z_z12_pg_exec(z_z12_expect_postgres(args[0]),z_as_cstring(args[1]),args[2],true);return rows.as.array->len==0?z_null():rows.as.array->items[0];}
    if(strcmp(name,"postgresQueryStream")==0){z_expect_args(name,argc,3);return z_z12_pg_stream(z_z12_expect_postgres(args[0]),z_as_cstring(args[1]),args[2]);}
    if(strcmp(name,"postgresPrepare")==0){z_expect_args(name,argc,2);ZPostgresNative *db=z_z12_expect_postgres(args[0]);ZPostgresStatementNative *statement=(ZPostgresStatementNative*)z_alloc(sizeof(ZPostgresStatementNative));statement->database=db;statement->sql=z_strdup(z_as_cstring(args[1]));unsigned id=atomic_fetch_add(&z_z12_statement_counter,1);char generated[64];snprintf(generated,sizeof(generated),"zumbra_%u",id);statement->name=z_strdup(generated);PGresult *result=PQprepare(db->connection,statement->name,statement->sql,0,NULL);if(PQresultStatus(result)!=PGRES_COMMAND_OK){const char *message=PQerrorMessage(db->connection);PQclear(result);z_fatal("prepare PostgreSQL statement: %s",message);}PQclear(result);statement->open=true;return z_z12_value(ZV_POSTGRES_STATEMENT,ZK_POSTGRES_STATEMENT,statement);}
    if(strcmp(name,"postgresBegin")==0){z_expect_args(name,argc,1);ZPostgresNative *db=z_z12_expect_postgres(args[0]);PGresult *result=PQexec(db->connection,"BEGIN");if(PQresultStatus(result)!=PGRES_COMMAND_OK){const char *message=PQerrorMessage(db->connection);PQclear(result);z_fatal("begin PostgreSQL transaction: %s",message);}PQclear(result);ZPostgresTransactionNative *tx=(ZPostgresTransactionNative*)z_alloc(sizeof(ZPostgresTransactionNative));tx->database=db;tx->active=true;return z_z12_value(ZV_POSTGRES_TRANSACTION,ZK_POSTGRES_TRANSACTION,tx);}
    if(strcmp(name,"postgresStatementExec")==0||strcmp(name,"postgresStatementQuery")==0||strcmp(name,"postgresStatementStream")==0){z_expect_args(name,argc,2);ZPostgresStatementNative *statement=z_z12_expect_pg_statement(args[0]);if(!statement->open)z_fatal("PostgreSQL statement is closed");bool query=strcmp(name,"postgresStatementExec")!=0;ZValue result=z_z12_pg_exec(statement->database,statement->sql,args[1],query);return strcmp(name,"postgresStatementStream")==0?z_z12_rows_from_array(result):result;}
    if(strcmp(name,"postgresStatementClose")==0){z_expect_args(name,argc,1);ZPostgresStatementNative *statement=z_z12_expect_pg_statement(args[0]);statement->open=false;return z_bool(true);}
    if(strcmp(name,"postgresStatementOpen")==0){z_expect_args(name,argc,1);return z_bool(z_z12_expect_pg_statement(args[0])->open);}
    if(strcmp(name,"postgresStatementSQL")==0){z_expect_args(name,argc,1);return z_string(z_z12_expect_pg_statement(args[0])->sql);}
    if(strcmp(name,"postgresTransactionExec")==0||strcmp(name,"postgresTransactionQuery")==0||strcmp(name,"postgresTransactionStream")==0){z_expect_args(name,argc,3);ZPostgresTransactionNative *tx=z_z12_expect_pg_transaction(args[0]);if(!tx->active)z_fatal("PostgreSQL transaction is inactive");bool query=strcmp(name,"postgresTransactionExec")!=0;ZValue result=z_z12_pg_exec(tx->database,z_as_cstring(args[1]),args[2],query);return strcmp(name,"postgresTransactionStream")==0?z_z12_rows_from_array(result):result;}
    if(strcmp(name,"postgresTransactionPrepare")==0){z_expect_args(name,argc,2);ZValue forwarded[2]={z_z12_value(ZV_POSTGRES_DATABASE,ZK_POSTGRES_DATABASE,z_z12_expect_pg_transaction(args[0])->database),args[1]};bool nested=false;return z_z12_call_builtin("postgresPrepare",forwarded,2,&nested);}
    if(strcmp(name,"postgresSavepoint")==0||strcmp(name,"postgresRollbackTo")==0||strcmp(name,"postgresRelease")==0){z_expect_args(name,argc,2);ZPostgresTransactionNative *tx=z_z12_expect_pg_transaction(args[0]);const char *point=z_as_cstring(args[1]);for(const char*p=point;*p;p++)if(!(isalnum((unsigned char)*p)||*p=='_'))z_fatal("invalid savepoint name");ZZ12Buffer sql;z_z12_buffer_init(&sql);z_z12_buffer_append(&sql,strcmp(name,"postgresSavepoint")==0?"SAVEPOINT ":strcmp(name,"postgresRollbackTo")==0?"ROLLBACK TO SAVEPOINT ":"RELEASE SAVEPOINT ");z_z12_buffer_append(&sql,point);PGresult *result=PQexec(tx->database->connection,sql.data);bool ok=PQresultStatus(result)==PGRES_COMMAND_OK;if(!ok){const char*message=PQerrorMessage(tx->database->connection);PQclear(result);z_z12_buffer_free(&sql);z_fatal("PostgreSQL savepoint: %s",message);}PQclear(result);z_z12_buffer_free(&sql);return z_bool(true);}
    if(strcmp(name,"postgresCommit")==0||strcmp(name,"postgresRollback")==0){z_expect_args(name,argc,1);ZPostgresTransactionNative *tx=z_z12_expect_pg_transaction(args[0]);if(!tx->active)z_fatal("PostgreSQL transaction is inactive");PGresult *result=PQexec(tx->database->connection,strcmp(name,"postgresCommit")==0?"COMMIT":"ROLLBACK");bool ok=PQresultStatus(result)==PGRES_COMMAND_OK;if(!ok){const char*message=PQerrorMessage(tx->database->connection);PQclear(result);z_fatal("finish PostgreSQL transaction: %s",message);}PQclear(result);tx->active=false;return z_bool(true);}
    if(strcmp(name,"postgresTransactionActive")==0){z_expect_args(name,argc,1);return z_bool(z_z12_expect_pg_transaction(args[0])->active);}
#endif

#if defined(ZUMBRA_ENABLE_REDIS)
    if(strcmp(name,"redisOpen")==0){if(argc<2||argc>5)z_fatal("redisOpen expects host, port, optional password, database and pool size");return z_z12_redis_open(z_as_cstring(args[0]),(int)z_as_i64(args[1]),argc>2?z_as_cstring(args[2]):"",argc>3?(int)z_as_i64(args[3]):0);}
    if(strcmp(name,"redisPing")==0){z_expect_args(name,argc,1);redisReply *reply=z_z12_redis_command(z_z12_expect_redis(args[0]),"PING");bool ok=reply->type==REDIS_REPLY_STATUS&&reply->str!=NULL&&strcasecmp(reply->str,"PONG")==0;freeReplyObject(reply);return z_bool(ok);}
    if(strcmp(name,"redisClose")==0){z_expect_args(name,argc,1);ZRedisNative *client=z_z12_expect_redis(args[0]);pthread_mutex_lock(&client->mutex);if(client->open){redisFree(client->context);client->context=NULL;client->open=false;}pthread_mutex_unlock(&client->mutex);return z_bool(true);}
    if(strcmp(name,"redisIsOpen")==0){z_expect_args(name,argc,1);ZRedisNative *client=z_z12_expect_redis(args[0]);return z_bool(client->open&&client->context!=NULL&&!client->context->err);}
    if(strcmp(name,"redisSetClient")==0){if(argc<3||argc>4)z_fatal("redisSetClient expects client, key, value and optional ttlMs");char *json=z_z12_json_stringify(args[2]);redisReply *reply=argc==4?z_z12_redis_command(z_z12_expect_redis(args[0]),"SET %s %s PX %lld",z_as_cstring(args[1]),json,(long long)z_as_i64(args[3])):z_z12_redis_command(z_z12_expect_redis(args[0]),"SET %s %s",z_as_cstring(args[1]),json);freeReplyObject(reply);return z_bool(true);}
    if(strcmp(name,"redisGetClient")==0){z_expect_args(name,argc,2);redisReply *reply=z_z12_redis_command(z_z12_expect_redis(args[0]),"GET %s",z_as_cstring(args[1]));if(reply->type==REDIS_REPLY_NIL){freeReplyObject(reply);return z_null();}ZValue result=z_z12_json_parse(reply->str==NULL?"null":reply->str);freeReplyObject(reply);return result;}
    if(strcmp(name,"redisDelete")==0){z_expect_args(name,argc,2);redisReply *reply=z_z12_redis_command(z_z12_expect_redis(args[0]),"DEL %s",z_as_cstring(args[1]));int64_t count=reply->integer;freeReplyObject(reply);return z_int(count);}
    if(strcmp(name,"redisExists")==0){z_expect_args(name,argc,2);redisReply *reply=z_z12_redis_command(z_z12_expect_redis(args[0]),"EXISTS %s",z_as_cstring(args[1]));bool exists=reply->integer>0;freeReplyObject(reply);return z_bool(exists);}
    if(strcmp(name,"redisExpire")==0){z_expect_args(name,argc,3);redisReply *reply=z_z12_redis_command(z_z12_expect_redis(args[0]),"PEXPIRE %s %lld",z_as_cstring(args[1]),(long long)z_as_i64(args[2]));bool changed=reply->integer>0;freeReplyObject(reply);return z_bool(changed);}
    if(strcmp(name,"redisTTL")==0){z_expect_args(name,argc,2);redisReply *reply=z_z12_redis_command(z_z12_expect_redis(args[0]),"PTTL %s",z_as_cstring(args[1]));int64_t ttl=reply->integer;freeReplyObject(reply);return z_int(ttl);}
    if(strcmp(name,"redisIncrement")==0){if(argc<2||argc>3)z_fatal("redisIncrement expects client, key and optional delta");redisReply *reply=z_z12_redis_command(z_z12_expect_redis(args[0]),"INCRBY %s %lld",z_as_cstring(args[1]),(long long)(argc==3?z_as_i64(args[2]):1));int64_t value=reply->integer;freeReplyObject(reply);return z_int(value);}
    if(strcmp(name,"redisPipeline")==0){z_expect_args(name,argc,2);if(args[1].tag!=ZV_ARRAY)z_fatal("redisPipeline commands must be array");ZValue *results=args[1].as.array->len==0?NULL:(ZValue*)z_alloc(sizeof(ZValue)*args[1].as.array->len);for(size_t i=0;i<args[1].as.array->len;i++){ZValue command=args[1].as.array->items[i];if(command.tag!=ZV_DICT)z_fatal("redis pipeline command must be dict");ZValue op,key,value,ttl;if(!z_z12_dict_find(command,"op",&op)||op.tag!=ZV_STRING)z_fatal("redis pipeline command requires op");if(!z_z12_dict_find(command,"key",&key)||key.tag!=ZV_STRING)z_fatal("redis pipeline command requires key");if(strcasecmp(op.as.s,"get")==0){ZValue forwarded[2]={args[0],key};bool nested=false;results[i]=z_z12_call_builtin("redisGetClient",forwarded,2,&nested);}else if(strcasecmp(op.as.s,"del")==0){ZValue forwarded[2]={args[0],key};bool nested=false;results[i]=z_z12_call_builtin("redisDelete",forwarded,2,&nested);}else if(strcasecmp(op.as.s,"set")==0){if(!z_z12_dict_find(command,"value",&value))z_fatal("redis set command requires value");bool has_ttl=z_z12_dict_find(command,"ttlMs",&ttl);ZValue forwarded[4]={args[0],key,value,ttl};bool nested=false;results[i]=z_z12_call_builtin("redisSetClient",forwarded,has_ttl?4:3,&nested);}else z_fatal("unsupported redis pipeline op %s",op.as.s);}return z_array_from(results,args[1].as.array->len);}
    if(strcmp(name,"redisPoolStats")==0){z_expect_args(name,argc,1);ZRedisNative *client=z_z12_expect_redis(args[0]);ZValue pairs[5]={z_pair(z_string("hits"),z_int(0)),z_pair(z_string("misses"),z_int(0)),z_pair(z_string("timeouts"),z_int(0)),z_pair(z_string("total"),z_int(client->open?1:0)),z_pair(z_string("idle"),z_int(client->open?1:0))};return z_dict_from(pairs,5);}
#endif

    *handled=false;return z_null();
}

static ZValue z_z12_call_method(ZValue callable,const ZValue *args,size_t argc){ZZ12BoundMethod *method=(ZZ12BoundMethod*)callable.as.p;if(method==NULL)z_fatal("invalid Z12 method");ZValue *forwarded=(ZValue*)z_alloc(sizeof(ZValue)*(argc+1));forwarded[0]=method->receiver;if(argc>0)memcpy(forwarded+1,args,sizeof(ZValue)*argc);const char *builtin=NULL;switch(method->receiver.tag){case ZV_SQL_ROWS:if(strcmp(method->name,"next")==0)builtin="sqlRowsNext";else if(strcmp(method->name,"columns")==0)builtin="sqlRowsColumns";else if(strcmp(method->name,"close")==0)builtin="sqlRowsClose";else if(strcmp(method->name,"isOpen")==0)builtin="sqlRowsOpen";break;case ZV_CONFIG:if(strcmp(method->name,"merge")==0)builtin="configMerge";else if(strcmp(method->name,"required")==0)builtin="configRequired";else if(strcmp(method->name,"string")==0)builtin="configString";else if(strcmp(method->name,"int")==0)builtin="configInt";else if(strcmp(method->name,"float")==0)builtin="configFloat";else if(strcmp(method->name,"bool")==0)builtin="configBool";else if(strcmp(method->name,"secret")==0)builtin="configSecret";else if(strcmp(method->name,"redacted")==0)builtin="configRedacted";break;case ZV_LOGGER:if(strcmp(method->name,"with")==0)builtin="loggerWith";else if(strcmp(method->name,"setLevel")==0)builtin="loggerSetLevel";else if(strcmp(method->name,"log")==0)builtin="loggerLog";else if(strcmp(method->name,"close")==0)builtin="loggerClose";break;case ZV_METRICS:if(strcmp(method->name,"counter")==0)builtin="metricsCounter";else if(strcmp(method->name,"gauge")==0)builtin="metricsGauge";else if(strcmp(method->name,"histogram")==0)builtin="metricsHistogram";else if(strcmp(method->name,"snapshot")==0)builtin="metricsSnapshot";else if(strcmp(method->name,"reset")==0)builtin="metricsReset";break;case ZV_TRACE_SPAN:if(strcmp(method->name,"child")==0)builtin="traceChild";else if(strcmp(method->name,"set")==0)builtin="traceSet";else if(strcmp(method->name,"event")==0)builtin="traceEvent";else if(strcmp(method->name,"finish")==0)builtin="traceFinish";else if(strcmp(method->name,"active")==0)builtin="traceActive";break;case ZV_SESSION_STORE:if(strcmp(method->name,"create")==0)builtin="sessionCreate";else if(strcmp(method->name,"get")==0)builtin="sessionGet";else if(strcmp(method->name,"set")==0)builtin="sessionSet";else if(strcmp(method->name,"delete")==0)builtin="sessionDelete";else if(strcmp(method->name,"rotate")==0)builtin="sessionRotate";else if(strcmp(method->name,"touch")==0)builtin="sessionTouch";else if(strcmp(method->name,"cleanup")==0)builtin="sessionCleanup";else if(strcmp(method->name,"close")==0)builtin="sessionClose";break;case ZV_RATE_LIMITER:if(strcmp(method->name,"allow")==0)builtin="rateAllow";else if(strcmp(method->name,"reset")==0)builtin="rateReset";break;
#if defined(ZUMBRA_ENABLE_POSTGRES)
case ZV_POSTGRES_DATABASE:if(strcmp(method->name,"configurePool")==0)builtin="postgresConfigurePool";else if(strcmp(method->name,"poolStats")==0)builtin="postgresPoolStats";else if(strcmp(method->name,"ping")==0)builtin="postgresPing";else if(strcmp(method->name,"close")==0)builtin="postgresClose";else if(strcmp(method->name,"isOpen")==0)builtin="postgresIsOpen";else if(strcmp(method->name,"exec")==0)builtin="postgresExecDb";else if(strcmp(method->name,"query")==0)builtin="postgresQueryDb";else if(strcmp(method->name,"queryOne")==0)builtin="postgresQueryOne";else if(strcmp(method->name,"stream")==0)builtin="postgresQueryStream";else if(strcmp(method->name,"prepare")==0)builtin="postgresPrepare";else if(strcmp(method->name,"begin")==0)builtin="postgresBegin";break;case ZV_POSTGRES_STATEMENT:if(strcmp(method->name,"exec")==0)builtin="postgresStatementExec";else if(strcmp(method->name,"query")==0)builtin="postgresStatementQuery";else if(strcmp(method->name,"stream")==0)builtin="postgresStatementStream";else if(strcmp(method->name,"close")==0)builtin="postgresStatementClose";else if(strcmp(method->name,"isOpen")==0)builtin="postgresStatementOpen";else if(strcmp(method->name,"sql")==0)builtin="postgresStatementSQL";break;case ZV_POSTGRES_TRANSACTION:if(strcmp(method->name,"exec")==0)builtin="postgresTransactionExec";else if(strcmp(method->name,"query")==0)builtin="postgresTransactionQuery";else if(strcmp(method->name,"stream")==0)builtin="postgresTransactionStream";else if(strcmp(method->name,"prepare")==0)builtin="postgresTransactionPrepare";else if(strcmp(method->name,"savepoint")==0)builtin="postgresSavepoint";else if(strcmp(method->name,"rollbackTo")==0)builtin="postgresRollbackTo";else if(strcmp(method->name,"release")==0)builtin="postgresRelease";else if(strcmp(method->name,"commit")==0)builtin="postgresCommit";else if(strcmp(method->name,"rollback")==0)builtin="postgresRollback";else if(strcmp(method->name,"active")==0)builtin="postgresTransactionActive";break;
#endif
#if defined(ZUMBRA_ENABLE_REDIS)
case ZV_REDIS_CLIENT:if(strcmp(method->name,"ping")==0)builtin="redisPing";else if(strcmp(method->name,"close")==0)builtin="redisClose";else if(strcmp(method->name,"isOpen")==0)builtin="redisIsOpen";else if(strcmp(method->name,"set")==0)builtin="redisSetClient";else if(strcmp(method->name,"get")==0)builtin="redisGetClient";else if(strcmp(method->name,"delete")==0)builtin="redisDelete";else if(strcmp(method->name,"exists")==0)builtin="redisExists";else if(strcmp(method->name,"expire")==0)builtin="redisExpire";else if(strcmp(method->name,"ttl")==0)builtin="redisTTL";else if(strcmp(method->name,"increment")==0)builtin="redisIncrement";else if(strcmp(method->name,"pipeline")==0)builtin="redisPipeline";else if(strcmp(method->name,"poolStats")==0)builtin="redisPoolStats";break;
#endif
default:break;}if(builtin==NULL)z_fatal("unknown Z12 method %s",method->name);bool handled=false;ZValue result=z_z12_call_builtin(builtin,forwarded,argc+1,&handled);if(!handled)z_fatal("Z12 method %s is not implemented",method->name);return result;}

static ZValue z_z12_get_field(ZValue value,const char *name,bool *handled){bool match=false;switch(value.tag){case ZV_SQL_ROWS:match=strcmp(name,"next")==0||strcmp(name,"columns")==0||strcmp(name,"close")==0||strcmp(name,"isOpen")==0;break;case ZV_CONFIG:match=strcmp(name,"merge")==0||strcmp(name,"required")==0||strcmp(name,"string")==0||strcmp(name,"int")==0||strcmp(name,"float")==0||strcmp(name,"bool")==0||strcmp(name,"secret")==0||strcmp(name,"redacted")==0;break;case ZV_LOGGER:match=strcmp(name,"with")==0||strcmp(name,"setLevel")==0||strcmp(name,"log")==0||strcmp(name,"close")==0;break;case ZV_METRICS:match=strcmp(name,"counter")==0||strcmp(name,"gauge")==0||strcmp(name,"histogram")==0||strcmp(name,"snapshot")==0||strcmp(name,"reset")==0;break;case ZV_TRACE_SPAN:match=strcmp(name,"child")==0||strcmp(name,"set")==0||strcmp(name,"event")==0||strcmp(name,"finish")==0||strcmp(name,"active")==0;break;case ZV_SESSION_STORE:match=strcmp(name,"create")==0||strcmp(name,"get")==0||strcmp(name,"set")==0||strcmp(name,"delete")==0||strcmp(name,"rotate")==0||strcmp(name,"touch")==0||strcmp(name,"cleanup")==0||strcmp(name,"close")==0;break;case ZV_RATE_LIMITER:match=strcmp(name,"allow")==0||strcmp(name,"reset")==0;break;case ZV_POSTGRES_DATABASE:match=strcmp(name,"configurePool")==0||strcmp(name,"poolStats")==0||strcmp(name,"ping")==0||strcmp(name,"close")==0||strcmp(name,"isOpen")==0||strcmp(name,"exec")==0||strcmp(name,"query")==0||strcmp(name,"queryOne")==0||strcmp(name,"stream")==0||strcmp(name,"prepare")==0||strcmp(name,"begin")==0;break;case ZV_POSTGRES_STATEMENT:match=strcmp(name,"exec")==0||strcmp(name,"query")==0||strcmp(name,"stream")==0||strcmp(name,"close")==0||strcmp(name,"isOpen")==0||strcmp(name,"sql")==0;break;case ZV_POSTGRES_TRANSACTION:match=strcmp(name,"exec")==0||strcmp(name,"query")==0||strcmp(name,"stream")==0||strcmp(name,"prepare")==0||strcmp(name,"savepoint")==0||strcmp(name,"rollbackTo")==0||strcmp(name,"release")==0||strcmp(name,"commit")==0||strcmp(name,"rollback")==0||strcmp(name,"active")==0;break;case ZV_REDIS_CLIENT:match=strcmp(name,"ping")==0||strcmp(name,"close")==0||strcmp(name,"isOpen")==0||strcmp(name,"set")==0||strcmp(name,"get")==0||strcmp(name,"delete")==0||strcmp(name,"exists")==0||strcmp(name,"expire")==0||strcmp(name,"ttl")==0||strcmp(name,"increment")==0||strcmp(name,"pipeline")==0||strcmp(name,"poolStats")==0;break;default:*handled=false;return z_null();}*handled=true;if(!match)z_fatal("Z12 object has no method %s",name);return z_z12_bound_method(value,name);}

static void z_z12_runtime_init(void){
    (void)&z_z12_rows_from_array;
#if defined(ZUMBRA_ENABLE_SQLITE) || defined(ZUMBRA_ENABLE_REDIS)
    z_z12_sessions=NULL;
#endif
#if defined(ZUMBRA_ENABLE_POSTGRES)
    z_z12_postgres_databases=NULL;
#endif
#if defined(ZUMBRA_ENABLE_REDIS)
    z_z12_redis_clients=NULL;
#endif
}
static void z_z12_runtime_shutdown(void){
#if defined(ZUMBRA_ENABLE_SQLITE) || defined(ZUMBRA_ENABLE_REDIS)
    pthread_mutex_lock(&z_z12_sessions_mutex);
    for(ZSessionNative *store=z_z12_sessions;store!=NULL;store=store->next){
        if(!store->open)continue;
#if defined(ZUMBRA_ENABLE_SQLITE)
        if(store->driver==Z_SESSION_SQLITE&&store->backend!=NULL)sqlite3_close_v2((sqlite3*)store->backend);
#endif
        store->backend=NULL;store->open=false;
    }
    z_z12_sessions=NULL;
    pthread_mutex_unlock(&z_z12_sessions_mutex);
#endif
#if defined(ZUMBRA_ENABLE_POSTGRES)
    pthread_mutex_lock(&z_z12_postgres_mutex);for(ZPostgresNative *database=z_z12_postgres_databases;database!=NULL;database=database->next)if(database->open){PQfinish(database->connection);database->connection=NULL;database->open=false;}z_z12_postgres_databases=NULL;pthread_mutex_unlock(&z_z12_postgres_mutex);
#endif
#if defined(ZUMBRA_ENABLE_REDIS)
    pthread_mutex_lock(&z_z12_redis_mutex);for(ZRedisNative *client=z_z12_redis_clients;client!=NULL;client=client->next)if(client->open){redisFree(client->context);client->context=NULL;client->open=false;}z_z12_redis_clients=NULL;pthread_mutex_unlock(&z_z12_redis_mutex);
#endif
}

#endif
