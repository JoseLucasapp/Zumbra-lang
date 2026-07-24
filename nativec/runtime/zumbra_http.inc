#if defined(ZUMBRA_ENABLE_HTTP)

/* Z11 HTTP/1.1, JSON, JWT, SSE and WebSocket runtime. This file is appended
   to zumbra_runtime.c by the native backend and therefore intentionally uses
   internal runtime helpers from the same translation unit. */

#define Z_HTTP_DEFAULT_BODY_LIMIT (8u * 1024u * 1024u)
#define Z_HTTP_MAX_BODY_LIMIT (64u * 1024u * 1024u)
#define Z_HTTP_MAX_HEADER_BYTES (64u * 1024u)
#define Z_HTTP_MAX_WS_FRAME (16u * 1024u * 1024u)

typedef struct ZHTTPRouteNative {
    char *method;
    char *pattern;
    ZValue handler;
    struct ZHTTPRouteNative *next;
} ZHTTPRouteNative;

typedef struct ZHTTPMiddlewareNative {
    ZValue handler;
    struct ZHTTPMiddlewareNative *next;
} ZHTTPMiddlewareNative;

typedef struct ZHTTPStaticNative {
    char *prefix;
    char *directory;
    struct ZHTTPStaticNative *next;
} ZHTTPStaticNative;

typedef struct {
    pthread_mutex_t mutex;
    ZHTTPRouteNative *routes;
    ZHTTPMiddlewareNative *middlewares;
    ZHTTPStaticNative *static_routes;
    size_t max_body_bytes;
    bool compression;
    bool cors_enabled;
    ZValue cors_origins;
    ZValue cors_methods;
    ZValue cors_headers;
    bool cors_credentials;
    int64_t cors_max_age;
} ZHTTPAppNative;

typedef struct ZHTTPServerNative {
    pthread_mutex_t mutex;
    pthread_cond_t connections_done;
    size_t active_connections;
    ZHTTPAppNative *app;
    int fd;
    bool tls;
    bool running;
    bool stopping;
    char *address;
    int64_t port;
    pthread_t thread;
    bool thread_started;
#if defined(ZUMBRA_ENABLE_TLS)
    SSL_CTX *tls_context;
#endif
    struct ZHTTPServerNative *next;
} ZHTTPServerNative;

typedef struct {
    char *name;
    char *filename;
    char *content_type;
    ZValue data;
} ZHTTPFileNative;

typedef struct {
    char *method;
    char *scheme;
    char *host;
    char *path;
    char *remote_address;
    int64_t remote_port;
    ZValue params;
    ZValue query;
    ZValue headers;
    ZValue cookies;
    ZValue form;
    ZValue files;
    ZValue body;
    ZValue raw_body;
    ZNetStreamNative *stream;
    bool hijacked;
} ZHTTPRequestNative;

typedef struct {
    int status;
    ZValue headers;
    ZValue body;
    ZValue raw_body;
    char *content_type;
    ZValue stream;
    bool written;
} ZHTTPResponseNative;

typedef struct {
    int status;
    char *status_text;
    char *url;
    ZValue headers;
    ZValue cookies;
    ZValue body;
} ZHTTPClientResponseNative;

typedef struct {
    ZValue channel;
    char *content_type;
    bool sse;
} ZHTTPStreamNative;

typedef struct {
    pthread_mutex_t read_mutex;
    pthread_mutex_t write_mutex;
    ZNetStreamNative *stream;
    bool client_side;
    bool closed;
    bool close_sent;
} ZWebSocketNative;

typedef struct {
    char *data;
    size_t len;
    size_t cap;
} ZHTTPBuffer;

typedef struct {
    const char *text;
    size_t length;
    size_t offset;
} ZJSONParser;

typedef struct {
    ZHTTPServerNative *server;
    ZNetStreamNative *stream;
} ZHTTPConnectionWork;

static pthread_mutex_t z_http_servers_mutex = PTHREAD_MUTEX_INITIALIZER;
static pthread_mutex_t z_http_handler_mutex = PTHREAD_MUTEX_INITIALIZER;
static ZHTTPServerNative *z_http_servers = NULL;

static void z_http_buffer_init(ZHTTPBuffer *buffer) { memset(buffer, 0, sizeof(*buffer)); }

static void z_http_buffer_reserve(ZHTTPBuffer *buffer, size_t additional) {
    if (additional > SIZE_MAX - buffer->len - 1) z_fatal("HTTP buffer size overflow");
    size_t needed = buffer->len + additional + 1;
    if (needed <= buffer->cap) return;
    size_t capacity = buffer->cap == 0 ? 256 : buffer->cap;
    while (capacity < needed) {
        if (capacity > SIZE_MAX / 2) { capacity = needed; break; }
        capacity *= 2;
    }
    char *grown = (char *)realloc(buffer->data, capacity);
    if (grown == NULL) { free(buffer->data); z_fatal("out of memory growing HTTP buffer"); }
    buffer->data = grown;
    buffer->cap = capacity;
}

static void z_http_buffer_append_bytes(ZHTTPBuffer *buffer, const void *data, size_t length) {
    if (length == 0) return;
    z_http_buffer_reserve(buffer, length);
    memcpy(buffer->data + buffer->len, data, length);
    buffer->len += length;
    buffer->data[buffer->len] = '\0';
}

static void z_http_buffer_append(ZHTTPBuffer *buffer, const char *text) {
    z_http_buffer_append_bytes(buffer, text, strlen(text));
}

static void z_http_buffer_printf(ZHTTPBuffer *buffer, const char *format, ...) {
    va_list arguments;
    va_start(arguments, format);
    va_list copy;
    va_copy(copy, arguments);
    int required = vsnprintf(NULL, 0, format, copy);
    va_end(copy);
    if (required < 0) { va_end(arguments); z_fatal("could not format HTTP data"); }
    z_http_buffer_reserve(buffer, (size_t)required);
    vsnprintf(buffer->data + buffer->len, buffer->cap - buffer->len, format, arguments);
    va_end(arguments);
    buffer->len += (size_t)required;
}

static void z_http_buffer_free(ZHTTPBuffer *buffer) {
    free(buffer->data);
    memset(buffer, 0, sizeof(*buffer));
}

static char *z_http_lower_copy(const char *value) {
    char *result = z_strdup(value == NULL ? "" : value);
    for (char *p = result; *p != '\0'; p++) *p = (char)tolower((unsigned char)*p);
    return result;
}

static bool z_http_string_equal_ci(const char *left, const char *right) {
    if (left == NULL || right == NULL) return left == right;
    while (*left != '\0' && *right != '\0') {
        if (tolower((unsigned char)*left) != tolower((unsigned char)*right)) return false;
        left++; right++;
    }
    return *left == *right;
}

static bool z_http_starts_with(const char *text, const char *prefix) {
    return strncmp(text, prefix, strlen(prefix)) == 0;
}

static bool z_http_contains_ci(const char *text, const char *needle) {
    if (text == NULL || needle == NULL) return false;
    size_t length = strlen(needle);
    if (length == 0) return true;
    for (const char *cursor = text; *cursor != '\0'; cursor++) {
        if (strncasecmp(cursor, needle, length) == 0) return true;
    }
    return false;
}

static ZValue z_http_dict_new(void) { return z_dict_from(NULL, 0); }

static ZValue z_http_dict_get(ZValue dictionary, const char *key) {
    if (dictionary.tag != ZV_DICT) return z_null();
    for (size_t index = 0; index < dictionary.as.dict->len; index++) {
        ZValue current = dictionary.as.dict->keys[index];
        if (current.tag == ZV_STRING && z_http_string_equal_ci(current.as.s, key)) return dictionary.as.dict->values[index];
    }
    return z_null();
}

static const char *z_http_dict_string(ZValue dictionary, const char *key, const char *fallback) {
    ZValue value = z_http_dict_get(dictionary, key);
    return value.tag == ZV_STRING ? value.as.s : fallback;
}

static int64_t z_http_dict_int(ZValue dictionary, const char *key, int64_t fallback) {
    ZValue value = z_http_dict_get(dictionary, key);
    return z_is_numeric(value) ? z_as_i64(value) : fallback;
}

static bool z_http_dict_bool(ZValue dictionary, const char *key, bool fallback) {
    ZValue value = z_http_dict_get(dictionary, key);
    return value.tag == ZV_BOOL ? value.as.b : fallback;
}

static void z_http_dict_set(ZValue dictionary, const char *key, ZValue value) {
    if (dictionary.tag != ZV_DICT) z_fatal("HTTP dictionary operation expects a dict");
    z_set_index(dictionary, z_string(key), value);
}

static int z_http_unhex(char value) {
    if (value >= '0' && value <= '9') return value - '0';
    if (value >= 'a' && value <= 'f') return value - 'a' + 10;
    if (value >= 'A' && value <= 'F') return value - 'A' + 10;
    return -1;
}

static char *z_http_url_decode(const char *value, bool plus_as_space) {
    ZHTTPBuffer output; z_http_buffer_init(&output);
    for (size_t index = 0; value[index] != '\0'; index++) {
        if (value[index] == '%' && value[index + 1] != '\0' && value[index + 2] != '\0') {
            int high = z_http_unhex(value[index + 1]);
            int low = z_http_unhex(value[index + 2]);
            if (high >= 0 && low >= 0) {
                unsigned char byte = (unsigned char)((high << 4) | low);
                z_http_buffer_append_bytes(&output, &byte, 1);
                index += 2;
                continue;
            }
        }
        char byte = plus_as_space && value[index] == '+' ? ' ' : value[index];
        z_http_buffer_append_bytes(&output, &byte, 1);
    }
    char *result = z_strdup(output.data == NULL ? "" : output.data);
    z_http_buffer_free(&output);
    return result;
}

static void z_http_parse_pairs(const char *text, char separator, ZValue dictionary, bool decode) {
    if (text == NULL || *text == '\0') return;
    char *copy = strdup(text);
    if (copy == NULL) z_fatal("out of memory parsing HTTP pairs");
    char delimiters[2] = {separator, '\0'};
    char *save = NULL;
    for (char *part = strtok_r(copy, delimiters, &save); part != NULL; part = strtok_r(NULL, delimiters, &save)) {
        while (isspace((unsigned char)*part)) part++;
        char *equals = strchr(part, '=');
        if (equals != NULL) *equals++ = '\0';
        char *key = decode ? z_http_url_decode(part, true) : z_strdup(part);
        char *value = decode ? z_http_url_decode(equals == NULL ? "" : equals, true) : z_strdup(equals == NULL ? "" : equals);
        ZValue existing = z_http_dict_get(dictionary, key);
        if (existing.tag == ZV_NULL) {
            z_http_dict_set(dictionary, key, z_string(value));
        } else if (existing.tag == ZV_ARRAY) {
            ZArray *array = existing.as.array;
            if (array->len == array->cap) {
                size_t next = array->cap == 0 ? 4 : array->cap * 2;
                ZValue *grown = (ZValue *)z_alloc(sizeof(ZValue) * next);
                if (array->len != 0) memcpy(grown, array->items, sizeof(ZValue) * array->len);
                array->items = grown; array->cap = next;
            }
            array->items[array->len++] = z_string(value);
        } else {
            ZValue pair[2] = {existing, z_string(value)};
            z_http_dict_set(dictionary, key, z_array_from(pair, 2));
        }
    }
    free(copy);
}

static const char *z_http_reason_phrase(int status) {
    switch (status) {
    case 100: return "Continue"; case 101: return "Switching Protocols";
    case 200: return "OK"; case 201: return "Created"; case 202: return "Accepted"; case 204: return "No Content";
    case 206: return "Partial Content"; case 301: return "Moved Permanently"; case 302: return "Found"; case 303: return "See Other";
    case 304: return "Not Modified"; case 307: return "Temporary Redirect"; case 308: return "Permanent Redirect";
    case 400: return "Bad Request"; case 401: return "Unauthorized"; case 403: return "Forbidden"; case 404: return "Not Found";
    case 405: return "Method Not Allowed"; case 408: return "Request Timeout"; case 409: return "Conflict"; case 413: return "Payload Too Large";
    case 415: return "Unsupported Media Type"; case 422: return "Unprocessable Content"; case 426: return "Upgrade Required";
    case 429: return "Too Many Requests"; case 500: return "Internal Server Error"; case 501: return "Not Implemented";
    case 502: return "Bad Gateway"; case 503: return "Service Unavailable"; case 504: return "Gateway Timeout";
    default: return "Status";
    }
}

static const char *z_http_mime_type(const char *path) {
    const char *dot = strrchr(path, '.');
    if (dot == NULL) return "application/octet-stream";
    if (z_http_string_equal_ci(dot, ".html") || z_http_string_equal_ci(dot, ".htm")) return "text/html; charset=utf-8";
    if (z_http_string_equal_ci(dot, ".css")) return "text/css; charset=utf-8";
    if (z_http_string_equal_ci(dot, ".js")) return "application/javascript; charset=utf-8";
    if (z_http_string_equal_ci(dot, ".json")) return "application/json";
    if (z_http_string_equal_ci(dot, ".png")) return "image/png";
    if (z_http_string_equal_ci(dot, ".jpg") || z_http_string_equal_ci(dot, ".jpeg")) return "image/jpeg";
    if (z_http_string_equal_ci(dot, ".gif")) return "image/gif";
    if (z_http_string_equal_ci(dot, ".svg")) return "image/svg+xml";
    if (z_http_string_equal_ci(dot, ".ico")) return "image/x-icon";
    if (z_http_string_equal_ci(dot, ".txt")) return "text/plain; charset=utf-8";
    if (z_http_string_equal_ci(dot, ".wasm")) return "application/wasm";
    return "application/octet-stream";
}

static ZValue z_http_value(ZTag tag, ZKind kind, void *pointer) {
    ZValue value = z_value(tag, kind);
    value.as.p = pointer;
    return value;
}

static ZHTTPAppNative *z_http_expect_app(ZValue value) {
    if (value.tag != ZV_HTTP_APP || value.as.p == NULL) z_fatal("expected HttpApp");
    return (ZHTTPAppNative *)value.as.p;
}
static ZHTTPServerNative *z_http_expect_server(ZValue value) {
    if (value.tag != ZV_HTTP_SERVER || value.as.p == NULL) z_fatal("expected HttpServer");
    return (ZHTTPServerNative *)value.as.p;
}
static ZHTTPRequestNative *z_http_expect_request(ZValue value) {
    if (value.tag != ZV_HTTP_REQUEST || value.as.p == NULL) z_fatal("expected HttpRequest");
    return (ZHTTPRequestNative *)value.as.p;
}
static ZHTTPResponseNative *z_http_expect_response(ZValue value) {
    if (value.tag != ZV_HTTP_RESPONSE || value.as.p == NULL) z_fatal("expected HttpResponse");
    return (ZHTTPResponseNative *)value.as.p;
}
static ZHTTPClientResponseNative *z_http_expect_client_response(ZValue value) {
    if (value.tag != ZV_HTTP_CLIENT_RESPONSE || value.as.p == NULL) z_fatal("expected HttpClientResponse");
    return (ZHTTPClientResponseNative *)value.as.p;
}
static ZWebSocketNative *z_http_expect_websocket(ZValue value) {
    if (value.tag != ZV_WEB_SOCKET || value.as.p == NULL) z_fatal("expected WebSocket");
    return (ZWebSocketNative *)value.as.p;
}

static ZValue z_http_response_new(void) {
    ZHTTPResponseNative *response = (ZHTTPResponseNative *)z_alloc(sizeof(ZHTTPResponseNative));
    response->status = 200;
    response->headers = z_http_dict_new();
    response->body = z_null();
    response->raw_body = z_null();
    response->content_type = z_strdup("text/plain; charset=utf-8");
    response->stream = z_null();
    response->written = false;
    return z_http_value(ZV_HTTP_RESPONSE, ZK_HTTP_RESPONSE, response);
}

/* ---------------- JSON ---------------- */

static void z_json_escape_string(ZHTTPBuffer *output, const char *value) {
    z_http_buffer_append_bytes(output, "\"", 1);
    for (const unsigned char *p = (const unsigned char *)value; *p != 0; p++) {
        switch (*p) {
        case '"': z_http_buffer_append(output, "\\\""); break;
        case '\\': z_http_buffer_append(output, "\\\\"); break;
        case '\b': z_http_buffer_append(output, "\\b"); break;
        case '\f': z_http_buffer_append(output, "\\f"); break;
        case '\n': z_http_buffer_append(output, "\\n"); break;
        case '\r': z_http_buffer_append(output, "\\r"); break;
        case '\t': z_http_buffer_append(output, "\\t"); break;
        default:
            if (*p < 0x20) z_http_buffer_printf(output, "\\u%04x", (unsigned int)*p);
            else z_http_buffer_append_bytes(output, p, 1);
        }
    }
    z_http_buffer_append_bytes(output, "\"", 1);
}

static void z_json_stringify_value(ZHTTPBuffer *output, ZValue value, int depth) {
    if (depth > 128) z_fatal("jsonStringify nesting exceeds 128 levels");
    switch (value.tag) {
    case ZV_NULL:
        z_http_buffer_append(output, "null");
        break;
    case ZV_BOOL:
        z_http_buffer_append(output, value.as.b ? "true" : "false");
        break;
    case ZV_INT:
        z_http_buffer_printf(output, "%" PRId64, value.as.i);
        break;
    case ZV_UINT:
        z_http_buffer_printf(output, "%" PRIu64, value.as.u);
        break;
    case ZV_FLOAT:
        if (!isfinite(value.as.f)) z_fatal("jsonStringify cannot encode non-finite floats");
        z_http_buffer_printf(output, "%.17g", value.as.f);
        break;
    case ZV_STRING:
        z_json_escape_string(output, value.as.s);
        break;
    case ZV_ARRAY:
        z_http_buffer_append_bytes(output, "[", 1);
        for (size_t index = 0; index < value.as.array->len; index++) {
            if (index != 0) z_http_buffer_append_bytes(output, ",", 1);
            z_json_stringify_value(output, value.as.array->items[index], depth + 1);
        }
        z_http_buffer_append_bytes(output, "]", 1);
        break;
    case ZV_DICT:
        z_http_buffer_append_bytes(output, "{", 1);
        for (size_t index = 0; index < value.as.dict->len; index++) {
            if (index != 0) z_http_buffer_append_bytes(output, ",", 1);
            ZValue key = value.as.dict->keys[index];
            if (key.tag != ZV_STRING) z_fatal("jsonStringify requires string dictionary keys");
            z_json_escape_string(output, key.as.s);
            z_http_buffer_append_bytes(output, ":", 1);
            z_json_stringify_value(output, value.as.dict->values[index], depth + 1);
        }
        z_http_buffer_append_bytes(output, "}", 1);
        break;
    default:
        z_fatal("jsonStringify does not support this value");
    }
}

static char *z_json_stringify(ZValue value) {
    ZHTTPBuffer output; z_http_buffer_init(&output);
    z_json_stringify_value(&output, value, 0);
    char *result = z_strdup(output.data == NULL ? "null" : output.data);
    z_http_buffer_free(&output);
    return result;
}

static void z_json_skip_space(ZJSONParser *parser) {
    while (parser->offset < parser->length && isspace((unsigned char)parser->text[parser->offset])) parser->offset++;
}

static bool z_json_take(ZJSONParser *parser, char expected) {
    z_json_skip_space(parser);
    if (parser->offset >= parser->length || parser->text[parser->offset] != expected) return false;
    parser->offset++;
    return true;
}

static uint32_t z_json_hex4(ZJSONParser *parser) {
    if (parser->offset + 4 > parser->length) z_fatal("invalid JSON unicode escape");
    uint32_t value = 0;
    for (int index = 0; index < 4; index++) {
        int digit = z_http_unhex(parser->text[parser->offset++]);
        if (digit < 0) z_fatal("invalid JSON unicode escape");
        value = (value << 4) | (uint32_t)digit;
    }
    return value;
}

static void z_json_append_utf8(ZHTTPBuffer *output, uint32_t codepoint) {
    if (codepoint <= 0x7F) {
        unsigned char bytes[1] = {(unsigned char)codepoint}; z_http_buffer_append_bytes(output, bytes, 1);
    } else if (codepoint <= 0x7FF) {
        unsigned char bytes[2] = {(unsigned char)(0xC0 | (codepoint >> 6)), (unsigned char)(0x80 | (codepoint & 0x3F))};
        z_http_buffer_append_bytes(output, bytes, 2);
    } else if (codepoint <= 0xFFFF) {
        unsigned char bytes[3] = {(unsigned char)(0xE0 | (codepoint >> 12)), (unsigned char)(0x80 | ((codepoint >> 6) & 0x3F)), (unsigned char)(0x80 | (codepoint & 0x3F))};
        z_http_buffer_append_bytes(output, bytes, 3);
    } else if (codepoint <= 0x10FFFF) {
        unsigned char bytes[4] = {(unsigned char)(0xF0 | (codepoint >> 18)), (unsigned char)(0x80 | ((codepoint >> 12) & 0x3F)), (unsigned char)(0x80 | ((codepoint >> 6) & 0x3F)), (unsigned char)(0x80 | (codepoint & 0x3F))};
        z_http_buffer_append_bytes(output, bytes, 4);
    } else z_fatal("invalid JSON unicode code point");
}

static ZValue z_json_parse_value(ZJSONParser *parser, int depth);

static ZValue z_json_parse_string(ZJSONParser *parser) {
    if (!z_json_take(parser, '"')) z_fatal("expected JSON string");
    ZHTTPBuffer output; z_http_buffer_init(&output);
    while (parser->offset < parser->length) {
        unsigned char current = (unsigned char)parser->text[parser->offset++];
        if (current == '"') {
            ZValue result = z_string(output.data == NULL ? "" : output.data);
            z_http_buffer_free(&output);
            return result;
        }
        if (current < 0x20) { z_http_buffer_free(&output); z_fatal("invalid control byte in JSON string"); }
        if (current != '\\') { z_http_buffer_append_bytes(&output, &current, 1); continue; }
        if (parser->offset >= parser->length) { z_http_buffer_free(&output); z_fatal("unterminated JSON escape"); }
        char escaped = parser->text[parser->offset++];
        switch (escaped) {
        case '"': case '\\': case '/': z_http_buffer_append_bytes(&output, &escaped, 1); break;
        case 'b': { char byte='\b'; z_http_buffer_append_bytes(&output,&byte,1); break; }
        case 'f': { char byte='\f'; z_http_buffer_append_bytes(&output,&byte,1); break; }
        case 'n': { char byte='\n'; z_http_buffer_append_bytes(&output,&byte,1); break; }
        case 'r': { char byte='\r'; z_http_buffer_append_bytes(&output,&byte,1); break; }
        case 't': { char byte='\t'; z_http_buffer_append_bytes(&output,&byte,1); break; }
        case 'u': {
            uint32_t codepoint = z_json_hex4(parser);
            if (codepoint >= 0xD800 && codepoint <= 0xDBFF) {
                if (parser->offset + 2 > parser->length || parser->text[parser->offset] != '\\' || parser->text[parser->offset + 1] != 'u') z_fatal("invalid JSON surrogate pair");
                parser->offset += 2;
                uint32_t low = z_json_hex4(parser);
                if (low < 0xDC00 || low > 0xDFFF) z_fatal("invalid JSON surrogate pair");
                codepoint = 0x10000 + ((codepoint - 0xD800) << 10) + (low - 0xDC00);
            }
            z_json_append_utf8(&output, codepoint);
            break;
        }
        default: z_http_buffer_free(&output); z_fatal("invalid JSON escape");
        }
    }
    z_http_buffer_free(&output);
    z_fatal("unterminated JSON string");
    return z_null();
}

static ZValue z_json_parse_number(ZJSONParser *parser) {
    z_json_skip_space(parser);
    size_t start = parser->offset;
    if (parser->offset < parser->length && parser->text[parser->offset] == '-') parser->offset++;
    if (parser->offset >= parser->length) z_fatal("invalid JSON number");
    if (parser->text[parser->offset] == '0') parser->offset++;
    else {
        if (!isdigit((unsigned char)parser->text[parser->offset])) z_fatal("invalid JSON number");
        while (parser->offset < parser->length && isdigit((unsigned char)parser->text[parser->offset])) parser->offset++;
    }
    bool floating = false;
    if (parser->offset < parser->length && parser->text[parser->offset] == '.') {
        floating = true; parser->offset++;
        if (parser->offset >= parser->length || !isdigit((unsigned char)parser->text[parser->offset])) z_fatal("invalid JSON number");
        while (parser->offset < parser->length && isdigit((unsigned char)parser->text[parser->offset])) parser->offset++;
    }
    if (parser->offset < parser->length && (parser->text[parser->offset] == 'e' || parser->text[parser->offset] == 'E')) {
        floating = true; parser->offset++;
        if (parser->offset < parser->length && (parser->text[parser->offset] == '+' || parser->text[parser->offset] == '-')) parser->offset++;
        if (parser->offset >= parser->length || !isdigit((unsigned char)parser->text[parser->offset])) z_fatal("invalid JSON exponent");
        while (parser->offset < parser->length && isdigit((unsigned char)parser->text[parser->offset])) parser->offset++;
    }
    size_t length = parser->offset - start;
    char *number = (char *)malloc(length + 1);
    if (number == NULL) z_fatal("out of memory parsing JSON number");
    memcpy(number, parser->text + start, length); number[length] = '\0';
    if (floating) {
        char *end = NULL; errno = 0; double value = strtod(number, &end);
        bool valid = errno == 0 && end != NULL && *end == '\0' && isfinite(value);
        free(number);
        if (!valid) z_fatal("invalid JSON number");
        return z_float(value);
    }
    char *end = NULL; errno = 0; int64_t value = strtoll(number, &end, 10);
    bool valid = errno == 0 && end != NULL && *end == '\0';
    free(number);
    if (!valid) z_fatal("JSON integer is outside int64 range");
    return z_int(value);
}

static ZValue z_json_parse_array(ZJSONParser *parser, int depth) {
    if (!z_json_take(parser, '[')) z_fatal("expected JSON array");
    ZValue *items = NULL; size_t count = 0, capacity = 0;
    z_json_skip_space(parser);
    if (z_json_take(parser, ']')) return z_array_from(NULL, 0);
    for (;;) {
        if (count == capacity) {
            size_t next = capacity == 0 ? 4 : capacity * 2;
            ZValue *grown = (ZValue *)realloc(items, sizeof(ZValue) * next);
            if (grown == NULL) { free(items); z_fatal("out of memory parsing JSON array"); }
            items = grown; capacity = next;
        }
        items[count++] = z_json_parse_value(parser, depth + 1);
        z_json_skip_space(parser);
        if (z_json_take(parser, ']')) break;
        if (!z_json_take(parser, ',')) { free(items); z_fatal("expected comma in JSON array"); }
    }
    ZValue result = z_array_from(items, count); free(items); return result;
}

static ZValue z_json_parse_object(ZJSONParser *parser, int depth) {
    if (!z_json_take(parser, '{')) z_fatal("expected JSON object");
    ZValue dictionary = z_http_dict_new();
    z_json_skip_space(parser);
    if (z_json_take(parser, '}')) return dictionary;
    for (;;) {
        ZValue key = z_json_parse_string(parser);
        if (!z_json_take(parser, ':')) z_fatal("expected colon in JSON object");
        ZValue value = z_json_parse_value(parser, depth + 1);
        z_set_index(dictionary, key, value);
        z_json_skip_space(parser);
        if (z_json_take(parser, '}')) break;
        if (!z_json_take(parser, ',')) z_fatal("expected comma in JSON object");
    }
    return dictionary;
}

static ZValue z_json_parse_value(ZJSONParser *parser, int depth) {
    if (depth > 128) z_fatal("jsonParse nesting exceeds 128 levels");
    z_json_skip_space(parser);
    if (parser->offset >= parser->length) z_fatal("unexpected end of JSON");
    char current = parser->text[parser->offset];
    if (current == '"') return z_json_parse_string(parser);
    if (current == '{') return z_json_parse_object(parser, depth);
    if (current == '[') return z_json_parse_array(parser, depth);
    if (current == '-' || isdigit((unsigned char)current)) return z_json_parse_number(parser);
    if (parser->length - parser->offset >= 4 && strncmp(parser->text + parser->offset, "true", 4) == 0) { parser->offset += 4; return z_bool(true); }
    if (parser->length - parser->offset >= 5 && strncmp(parser->text + parser->offset, "false", 5) == 0) { parser->offset += 5; return z_bool(false); }
    if (parser->length - parser->offset >= 4 && strncmp(parser->text + parser->offset, "null", 4) == 0) { parser->offset += 4; return z_null(); }
    z_fatal("invalid JSON value at byte %zu", parser->offset);
    return z_null();
}

static ZValue z_json_parse(const char *text) {
    ZJSONParser parser = {text, strlen(text), 0};
    ZValue result = z_json_parse_value(&parser, 0);
    z_json_skip_space(&parser);
    if (parser.offset != parser.length) z_fatal("unexpected content after JSON value");
    return result;
}

/* ---------------- Base64/JWT ---------------- */

static char *z_base64_encode(const unsigned char *data, size_t length, bool url_safe, bool padding) {
    size_t encoded_length = 4 * ((length + 2) / 3);
    unsigned char *encoded = (unsigned char *)malloc(encoded_length + 1);
    if (encoded == NULL) z_fatal("out of memory encoding base64");
    int actual = EVP_EncodeBlock(encoded, data, (int)length);
    if (actual < 0) { free(encoded); z_fatal("base64 encoding failed"); }
    size_t final_length = (size_t)actual;
    if (url_safe) {
        for (size_t index = 0; index < final_length; index++) {
            if (encoded[index] == '+') encoded[index] = '-';
            else if (encoded[index] == '/') encoded[index] = '_';
        }
    }
    if (!padding) while (final_length > 0 && encoded[final_length - 1] == '=') final_length--;
    encoded[final_length] = '\0';
    char *result = z_strdup((char *)encoded); free(encoded); return result;
}

static unsigned char *z_base64_decode(const char *text, bool url_safe, size_t *output_length) {
    size_t length = strlen(text);
    size_t padded = (length + 3) & ~(size_t)3;
    char *copy = (char *)malloc(padded + 1);
    if (copy == NULL) z_fatal("out of memory decoding base64");
    memcpy(copy, text, length);
    for (size_t index = 0; index < length; index++) {
        if (url_safe && copy[index] == '-') copy[index] = '+';
        else if (url_safe && copy[index] == '_') copy[index] = '/';
    }
    while (length < padded) copy[length++] = '=';
    copy[padded] = '\0';
    unsigned char *decoded = (unsigned char *)malloc((padded / 4) * 3 + 1);
    if (decoded == NULL) { free(copy); z_fatal("out of memory decoding base64"); }
    int actual = EVP_DecodeBlock(decoded, (const unsigned char *)copy, (int)padded);
    if (actual < 0) { free(copy); free(decoded); return NULL; }
    size_t result_length = (size_t)actual;
    while (padded > 0 && copy[padded - 1] == '=') { result_length--; padded--; }
    decoded[result_length] = 0;
    free(copy); *output_length = result_length; return decoded;
}

static char *z_jwt_sign_hs256(ZValue claims, const char *secret, int64_t expires_seconds) {
    if (claims.tag != ZV_DICT) z_fatal("jwtSignHS256 claims must be a dict");
    if (expires_seconds > 0) z_http_dict_set(claims, "exp", z_int((int64_t)time(NULL) + expires_seconds));
    const char *header = "{\"alg\":\"HS256\",\"typ\":\"JWT\"}";
    char *payload = z_json_stringify(claims);
    char *header64 = z_base64_encode((const unsigned char *)header, strlen(header), true, false);
    char *payload64 = z_base64_encode((const unsigned char *)payload, strlen(payload), true, false);
    ZHTTPBuffer signing; z_http_buffer_init(&signing);
    z_http_buffer_append(&signing, header64); z_http_buffer_append_bytes(&signing, ".", 1); z_http_buffer_append(&signing, payload64);
    unsigned char digest[EVP_MAX_MD_SIZE]; unsigned int digest_length = 0;
    if (HMAC(EVP_sha256(), secret, (int)strlen(secret), (const unsigned char *)signing.data, signing.len, digest, &digest_length) == NULL) z_fatal("JWT signing failed");
    char *signature = z_base64_encode(digest, digest_length, true, false);
    z_http_buffer_append_bytes(&signing, ".", 1); z_http_buffer_append(&signing, signature);
    char *token = z_strdup(signing.data); z_http_buffer_free(&signing); return token;
}

static ZValue z_jwt_verify_hs256(const char *token, const char *secret) {
    char *copy = strdup(token);
    if (copy == NULL) z_fatal("out of memory verifying JWT");
    char *first = strchr(copy, '.');
    if (first == NULL) { free(copy); ZValue out[2]={z_bool(false),z_null()}; return z_array_from(out,2); }
    *first++ = '\0';
    char *second = strchr(first, '.');
    if (second == NULL) { free(copy); ZValue out[2]={z_bool(false),z_null()}; return z_array_from(out,2); }
    *second++ = '\0';
    ZHTTPBuffer signing; z_http_buffer_init(&signing); z_http_buffer_append(&signing, copy); z_http_buffer_append_bytes(&signing, ".", 1); z_http_buffer_append(&signing, first);
    unsigned char digest[EVP_MAX_MD_SIZE]; unsigned int digest_length = 0;
    HMAC(EVP_sha256(), secret, (int)strlen(secret), (const unsigned char *)signing.data, signing.len, digest, &digest_length);
    size_t signature_length = 0; unsigned char *signature = z_base64_decode(second, true, &signature_length);
    bool valid = signature != NULL && signature_length == digest_length && CRYPTO_memcmp(signature, digest, digest_length) == 0;
    free(signature); z_http_buffer_free(&signing);
    if (!valid) { free(copy); ZValue out[2]={z_bool(false),z_null()}; return z_array_from(out,2); }
    size_t payload_length = 0; unsigned char *payload = z_base64_decode(first, true, &payload_length);
    if (payload == NULL) { free(copy); ZValue out[2]={z_bool(false),z_null()}; return z_array_from(out,2); }
    ZValue claims = z_json_parse((char *)payload); free(payload); free(copy);
    if (claims.tag == ZV_DICT) {
        ZValue expiration = z_http_dict_get(claims, "exp");
        if (z_is_numeric(expiration) && (int64_t)time(NULL) >= z_as_i64(expiration)) {
            ZValue out[2]={z_bool(false),z_null()}; return z_array_from(out,2);
        }
    }
    ZValue out[2]={z_bool(true),claims}; return z_array_from(out,2);
}

/* ---------------- HTTP wire helpers ---------------- */

typedef struct {
    bool tls;
    char *scheme;
    char *host;
    int64_t port;
    char *path;
} ZHTTPURL;

static ZHTTPURL z_http_parse_url(const char *url) {
    ZHTTPURL parsed; memset(&parsed, 0, sizeof(parsed));
    const char *separator = strstr(url, "://");
    if (separator == NULL) z_fatal("HTTP URL must include http:// or https://");
    size_t scheme_length = (size_t)(separator - url);
    char scheme[16];
    if (scheme_length == 0 || scheme_length >= sizeof(scheme)) z_fatal("invalid HTTP URL scheme");
    memcpy(scheme, url, scheme_length); scheme[scheme_length] = '\0';
    parsed.tls = z_http_string_equal_ci(scheme, "https") || z_http_string_equal_ci(scheme, "wss");
    if (!parsed.tls && !z_http_string_equal_ci(scheme, "http") && !z_http_string_equal_ci(scheme, "ws")) z_fatal("unsupported URL scheme %s", scheme);
    parsed.scheme = z_strdup(scheme);
    const char *authority = separator + 3;
    const char *path = strchr(authority, '/');
    const char *authority_end = path == NULL ? url + strlen(url) : path;
    if (authority_end == authority) z_fatal("HTTP URL is missing a host");
    const char *host_start = authority;
    const char *host_end = authority_end;
    int64_t port = parsed.tls ? 443 : 80;
    if (*host_start == '[') {
        const char *close = memchr(host_start, ']', (size_t)(authority_end - host_start));
        if (close == NULL) z_fatal("invalid IPv6 URL host");
        host_start++;
        host_end = close;
        if (close + 1 < authority_end) {
            if (close[1] != ':') z_fatal("invalid URL authority");
            port = strtoll(close + 2, NULL, 10);
        }
    } else {
        const char *colon = NULL;
        for (const char *cursor = authority; cursor < authority_end; cursor++) if (*cursor == ':') colon = cursor;
        if (colon != NULL) { host_end = colon; port = strtoll(colon + 1, NULL, 10); }
    }
    if (host_end <= host_start) z_fatal("HTTP URL is missing a host");
    size_t host_length = (size_t)(host_end - host_start);
    char *host = (char *)z_alloc(host_length + 1);
    memcpy(host, host_start, host_length); host[host_length] = '\0';
    z_network_require_port(port, "HTTP URL");
    parsed.host = host;
    parsed.port = port;
    parsed.path = z_strdup(path == NULL || *path == '\0' ? "/" : path);
    return parsed;
}

static ssize_t z_http_stream_read_once(ZNetStreamNative *stream, void *data, size_t length) {
    bool eof = false;
    ssize_t result = z_stream_read_once(stream, data, length, &eof);
    if (result == -2) z_fatal("HTTP stream read timed out");
    return result;
}

static void z_http_stream_write_all(ZNetStreamNative *stream, const void *data, size_t length) {
    size_t offset = 0;
    while (offset < length) {
        ssize_t written = z_stream_write_once(stream, (const uint8_t *)data + offset, length - offset);
        if (written == -2) z_fatal("HTTP stream write timed out");
        if (written <= 0) z_fatal("HTTP stream closed during write");
        offset += (size_t)written;
    }
}

static char *z_http_find_header_end(const char *data, size_t length, size_t *delimiter_length) {
    for (size_t index = 0; index + 3 < length; index++) {
        if (data[index] == '\r' && data[index + 1] == '\n' && data[index + 2] == '\r' && data[index + 3] == '\n') {
            *delimiter_length = 4; return (char *)(data + index);
        }
    }
    for (size_t index = 0; index + 1 < length; index++) {
        if (data[index] == '\n' && data[index + 1] == '\n') { *delimiter_length = 2; return (char *)(data + index); }
    }
    return NULL;
}

static bool z_http_read_headers(ZNetStreamNative *stream, ZHTTPBuffer *buffer, size_t *header_bytes) {
    char chunk[4096];
    for (;;) {
        size_t delimiter_length = 0;
        char *end = z_http_find_header_end(buffer->data == NULL ? "" : buffer->data, buffer->len, &delimiter_length);
        if (end != NULL) {
            *header_bytes = (size_t)(end - buffer->data) + delimiter_length;
            return true;
        }
        if (buffer->len >= Z_HTTP_MAX_HEADER_BYTES) z_fatal("HTTP headers exceed 64 KiB");
        ssize_t count = z_http_stream_read_once(stream, chunk, sizeof(chunk));
        if (count == 0) return false;
        z_http_buffer_append_bytes(buffer, chunk, (size_t)count);
    }
}

static ZValue z_http_parse_headers_block(char *headers_text, char **start_line) {
    ZValue headers = z_http_dict_new();
    char *save = NULL;
    char *line = strtok_r(headers_text, "\n", &save);
    if (line == NULL) z_fatal("empty HTTP header block");
    size_t first_length = strlen(line);
    if (first_length > 0 && line[first_length - 1] == '\r') line[first_length - 1] = '\0';
    *start_line = line;
    while ((line = strtok_r(NULL, "\n", &save)) != NULL) {
        size_t length = strlen(line);
        if (length > 0 && line[length - 1] == '\r') line[length - 1] = '\0';
        if (*line == '\0') continue;
        char *colon = strchr(line, ':');
        if (colon == NULL) z_fatal("malformed HTTP header line");
        *colon++ = '\0';
        while (isspace((unsigned char)*colon)) colon++;
        char *name = z_http_lower_copy(line);
        ZValue existing = z_http_dict_get(headers, name);
        if (existing.tag == ZV_STRING && existing.as.s[0] != '\0') {
            ZHTTPBuffer combined; z_http_buffer_init(&combined);
            z_http_buffer_append(&combined, existing.as.s); z_http_buffer_append(&combined, ","); z_http_buffer_append(&combined, colon);
            z_http_dict_set(headers, name, z_string(combined.data)); z_http_buffer_free(&combined);
        } else z_http_dict_set(headers, name, z_string(colon));
    }
    return headers;
}

static size_t z_http_parse_hex_size(const char *line) {
    errno = 0; char *end = NULL; unsigned long long value = strtoull(line, &end, 16);
    if (errno != 0 || end == line) z_fatal("invalid chunked HTTP length");
    return (size_t)value;
}

static ZValue z_http_read_chunked_body(ZNetStreamNative *stream, ZHTTPBuffer *buffer, size_t offset, size_t limit) {
    ZHTTPBuffer body; z_http_buffer_init(&body);
    size_t cursor = offset;
    for (;;) {
        char *line_end = NULL;
        for (;;) {
            for (size_t index = cursor; index + 1 < buffer->len; index++) {
                if (buffer->data[index] == '\r' && buffer->data[index + 1] == '\n') { line_end = buffer->data + index; break; }
            }
            if (line_end != NULL) break;
            char chunk[4096]; ssize_t count = z_http_stream_read_once(stream, chunk, sizeof(chunk));
            if (count <= 0) z_fatal("unexpected EOF in chunked HTTP body");
            z_http_buffer_append_bytes(buffer, chunk, (size_t)count);
        }
        *line_end = '\0';
        size_t chunk_size = z_http_parse_hex_size(buffer->data + cursor);
        cursor = (size_t)(line_end - buffer->data) + 2;
        if (chunk_size == 0) break;
        if (chunk_size > limit - body.len) { z_http_buffer_free(&body); z_fatal("HTTP body exceeds configured limit"); }
        while (buffer->len - cursor < chunk_size + 2) {
            char chunk[4096]; ssize_t count = z_http_stream_read_once(stream, chunk, sizeof(chunk));
            if (count <= 0) { z_http_buffer_free(&body); z_fatal("unexpected EOF in chunked HTTP body"); }
            z_http_buffer_append_bytes(buffer, chunk, (size_t)count);
        }
        z_http_buffer_append_bytes(&body, buffer->data + cursor, chunk_size);
        cursor += chunk_size;
        if (buffer->data[cursor] != '\r' || buffer->data[cursor + 1] != '\n') { z_http_buffer_free(&body); z_fatal("invalid chunk terminator"); }
        cursor += 2;
    }
    ZValue result = z_bytes(body.len);
    if (body.len != 0) memcpy(result.as.buffer->data, body.data, body.len);
    z_http_buffer_free(&body); return result;
}

static ZValue z_http_read_fixed_body(ZNetStreamNative *stream, ZHTTPBuffer *buffer, size_t offset, size_t length, size_t limit) {
    if (length > limit) z_fatal("HTTP body exceeds configured limit");
    ZValue result = z_bytes(length);
    size_t available = buffer->len > offset ? buffer->len - offset : 0;
    size_t copied = available > length ? length : available;
    if (copied != 0) memcpy(result.as.buffer->data, buffer->data + offset, copied);
    while (copied < length) {
        ssize_t count = z_http_stream_read_once(stream, (uint8_t *)result.as.buffer->data + copied, length - copied);
        if (count <= 0) z_fatal("unexpected EOF in HTTP body");
        copied += (size_t)count;
    }
    return result;
}

static ZValue z_http_read_until_close(ZNetStreamNative *stream, ZHTTPBuffer *buffer, size_t offset, size_t limit) {
    ZHTTPBuffer body; z_http_buffer_init(&body);
    if (buffer->len > offset) z_http_buffer_append_bytes(&body, buffer->data + offset, buffer->len - offset);
    char chunk[8192];
    for (;;) {
        ssize_t count = z_http_stream_read_once(stream, chunk, sizeof(chunk));
        if (count == 0) break;
        if (body.len + (size_t)count > limit) { z_http_buffer_free(&body); z_fatal("HTTP response exceeds 64 MiB safety limit"); }
        z_http_buffer_append_bytes(&body, chunk, (size_t)count);
    }
    ZValue result = z_bytes(body.len);
    if (body.len != 0) memcpy(result.as.buffer->data, body.data, body.len);
    z_http_buffer_free(&body); return result;
}

static ZValue z_http_gunzip(ZValue input) {
    ZBuffer *source = z_expect_byte_buffer(input);
    z_stream stream; memset(&stream, 0, sizeof(stream));
    stream.next_in = (Bytef *)source->data; stream.avail_in = (uInt)source->len;
    if (inflateInit2(&stream, 16 + MAX_WBITS) != Z_OK) z_fatal("could not initialize gzip decoder");
    ZHTTPBuffer output; z_http_buffer_init(&output);
    unsigned char chunk[8192];
    int status;
    do {
        stream.next_out = chunk; stream.avail_out = sizeof(chunk);
        status = inflate(&stream, Z_NO_FLUSH);
        if (status != Z_OK && status != Z_STREAM_END) { inflateEnd(&stream); z_http_buffer_free(&output); z_fatal("invalid gzip HTTP body"); }
        size_t produced = sizeof(chunk) - stream.avail_out;
        if (output.len + produced > Z_HTTP_MAX_BODY_LIMIT) { inflateEnd(&stream); z_http_buffer_free(&output); z_fatal("decompressed HTTP body exceeds 64 MiB"); }
        z_http_buffer_append_bytes(&output, chunk, produced);
    } while (status != Z_STREAM_END);
    inflateEnd(&stream);
    ZValue result = z_bytes(output.len); if (output.len != 0) memcpy(result.as.buffer->data, output.data, output.len);
    z_http_buffer_free(&output); return result;
}

static ZValue z_http_gzip(const void *data, size_t length) {
    z_stream stream; memset(&stream, 0, sizeof(stream));
    if (deflateInit2(&stream, Z_DEFAULT_COMPRESSION, Z_DEFLATED, 16 + MAX_WBITS, 8, Z_DEFAULT_STRATEGY) != Z_OK) z_fatal("could not initialize gzip encoder");
    stream.next_in = (Bytef *)data; stream.avail_in = (uInt)length;
    ZHTTPBuffer output; z_http_buffer_init(&output);
    unsigned char chunk[8192]; int status;
    do {
        stream.next_out = chunk; stream.avail_out = sizeof(chunk);
        status = deflate(&stream, stream.avail_in == 0 ? Z_FINISH : Z_NO_FLUSH);
        if (status != Z_OK && status != Z_STREAM_END) { deflateEnd(&stream); z_http_buffer_free(&output); z_fatal("gzip encoding failed"); }
        z_http_buffer_append_bytes(&output, chunk, sizeof(chunk) - stream.avail_out);
    } while (status != Z_STREAM_END);
    deflateEnd(&stream);
    ZValue result = z_bytes(output.len); if (output.len != 0) memcpy(result.as.buffer->data, output.data, output.len);
    z_http_buffer_free(&output); return result;
}

/* ---------------- Request decoding and routing ---------------- */

static const uint8_t *z_http_memmem(const uint8_t *haystack, size_t haystack_len, const uint8_t *needle, size_t needle_len) {
    if (needle_len == 0) return haystack;
    if (haystack_len < needle_len) return NULL;
    for (size_t index = 0; index <= haystack_len - needle_len; index++) {
        if (haystack[index] == needle[0] && memcmp(haystack + index, needle, needle_len) == 0) return haystack + index;
    }
    return NULL;
}

static char *z_http_parameter(const char *header, const char *parameter) {
    if (header == NULL) return NULL;
    size_t parameter_length = strlen(parameter);
    const char *cursor = header;
    while ((cursor = strchr(cursor, ';')) != NULL) {
        cursor++;
        while (isspace((unsigned char)*cursor)) cursor++;
        if (strncasecmp(cursor, parameter, parameter_length) != 0 || cursor[parameter_length] != '=') continue;
        cursor += parameter_length + 1;
        if (*cursor == '"') {
            cursor++; const char *end = strchr(cursor, '"');
            if (end == NULL) return NULL;
            size_t length = (size_t)(end - cursor); char *value = (char *)z_alloc(length + 1); memcpy(value, cursor, length); value[length] = '\0'; return value;
        }
        const char *end = cursor; while (*end != '\0' && *end != ';' && !isspace((unsigned char)*end)) end++;
        size_t length = (size_t)(end - cursor); char *value = (char *)z_alloc(length + 1); memcpy(value, cursor, length); value[length] = '\0'; return value;
    }
    return NULL;
}

static void z_http_parse_multipart(ZValue raw_body, const char *boundary, ZValue form, ZValue files) {
    ZBuffer *body = z_expect_byte_buffer(raw_body);
    ZHTTPBuffer marker; z_http_buffer_init(&marker); z_http_buffer_append(&marker, "--"); z_http_buffer_append(&marker, boundary);
    const uint8_t *cursor = (const uint8_t *)body->data;
    const uint8_t *end = cursor + body->len;
    const uint8_t *part = z_http_memmem(cursor, body->len, (const uint8_t *)marker.data, marker.len);
    while (part != NULL) {
        part += marker.len;
        if (part + 2 <= end && part[0] == '-' && part[1] == '-') break;
        if (part + 2 <= end && part[0] == '\r' && part[1] == '\n') part += 2;
        const uint8_t separator[] = {'\r','\n','\r','\n'};
        const uint8_t *header_end = z_http_memmem(part, (size_t)(end - part), separator, sizeof(separator));
        if (header_end == NULL) break;
        size_t headers_len = (size_t)(header_end - part);
        char *headers_copy = (char *)malloc(headers_len + 7);
        if (headers_copy == NULL) z_fatal("out of memory parsing multipart body");
        memcpy(headers_copy, "PART\r\n", 6);
        memcpy(headers_copy + 6, part, headers_len); headers_copy[headers_len + 6] = '\0';
        char *start_line = NULL; ZValue headers = z_http_parse_headers_block(headers_copy, &start_line);
        (void)start_line;
        const uint8_t *data_start = header_end + sizeof(separator);
        ZHTTPBuffer next_marker; z_http_buffer_init(&next_marker); z_http_buffer_append(&next_marker, "\r\n--"); z_http_buffer_append(&next_marker, boundary);
        const uint8_t *next = z_http_memmem(data_start, (size_t)(end - data_start), (const uint8_t *)next_marker.data, next_marker.len);
        if (next == NULL) { free(headers_copy); z_http_buffer_free(&next_marker); break; }
        size_t data_length = (size_t)(next - data_start);
        const char *disposition = z_http_dict_string(headers, "content-disposition", "");
        char *name = z_http_parameter(disposition, "name");
        char *filename = z_http_parameter(disposition, "filename");
        if (name != NULL) {
            if (filename != NULL) {
                ZValue bytes = z_bytes(data_length); if (data_length != 0) memcpy(bytes.as.buffer->data, data_start, data_length);
                ZHTTPFileNative *file = (ZHTTPFileNative *)z_alloc(sizeof(ZHTTPFileNative));
                file->name = name; file->filename = filename;
                file->content_type = z_strdup(z_http_dict_string(headers, "content-type", "application/octet-stream"));
                file->data = bytes;
                z_http_dict_set(files, name, z_http_value(ZV_HTTP_FILE, ZK_HTTP_FILE, file));
            } else {
                char *text = (char *)z_alloc(data_length + 1); memcpy(text, data_start, data_length); text[data_length] = '\0';
                z_http_dict_set(form, name, z_string(text));
            }
        }
        free(headers_copy); z_http_buffer_free(&next_marker);
        part = next + 2;
    }
    z_http_buffer_free(&marker);
}

static ZValue z_http_body_object(ZValue raw_body, ZValue headers, ZValue form, ZValue files) {
    ZBuffer *buffer = z_expect_byte_buffer(raw_body);
    const char *content_type = z_http_dict_string(headers, "content-type", "");
    if (z_http_starts_with(content_type, "application/json")) {
        if (buffer->len == 0) return z_null();
        char *text = (char *)malloc(buffer->len + 1); if (text == NULL) z_fatal("out of memory parsing JSON body");
        memcpy(text, buffer->data, buffer->len); text[buffer->len] = '\0';
        ZValue parsed = z_json_parse(text); free(text); return parsed;
    }
    if (z_http_starts_with(content_type, "application/x-www-form-urlencoded")) {
        char *text = (char *)malloc(buffer->len + 1); if (text == NULL) z_fatal("out of memory parsing form body");
        memcpy(text, buffer->data, buffer->len); text[buffer->len] = '\0'; z_http_parse_pairs(text, '&', form, true); free(text); return form;
    }
    if (z_http_starts_with(content_type, "multipart/form-data")) {
        char *boundary = z_http_parameter(content_type, "boundary");
        if (boundary == NULL || *boundary == '\0') z_fatal("multipart request is missing a boundary");
        z_http_parse_multipart(raw_body, boundary, form, files); return form;
    }
    if (buffer->len == 0) return z_null();
    char *text = (char *)z_alloc(buffer->len + 1); memcpy(text, buffer->data, buffer->len); text[buffer->len] = '\0'; return z_string(text);
}

static bool z_http_route_match(const char *pattern, const char *path, ZValue params) {
    char *pattern_copy = strdup(pattern); char *path_copy = strdup(path);
    if (pattern_copy == NULL || path_copy == NULL) { free(pattern_copy); free(path_copy); z_fatal("out of memory matching HTTP route"); }
    char *query = strchr(path_copy, '?'); if (query != NULL) *query = '\0';
    char *pattern_cursor = pattern_copy; char *path_cursor = path_copy;
    while (*pattern_cursor == '/') pattern_cursor++;
    while (*path_cursor == '/') path_cursor++;
    char *pattern_save = NULL, *path_save = NULL;
    char *pattern_segment = strtok_r(pattern_cursor, "/", &pattern_save);
    char *path_segment = strtok_r(path_cursor, "/", &path_save);
    while (pattern_segment != NULL) {
        if (pattern_segment[0] == '*') {
            ZHTTPBuffer rest; z_http_buffer_init(&rest);
            while (path_segment != NULL) {
                if (rest.len != 0) z_http_buffer_append_bytes(&rest, "/", 1);
                z_http_buffer_append(&rest, path_segment);
                path_segment = strtok_r(NULL, "/", &path_save);
            }
            z_http_dict_set(params, pattern_segment + 1, z_string(rest.data == NULL ? "" : rest.data)); z_http_buffer_free(&rest);
            pattern_segment = strtok_r(NULL, "/", &pattern_save);
            break;
        }
        if (path_segment == NULL) { free(pattern_copy); free(path_copy); return false; }
        if (pattern_segment[0] == ':') z_http_dict_set(params, pattern_segment + 1, z_string(z_http_url_decode(path_segment, false)));
        else if (strcmp(pattern_segment, path_segment) != 0) { free(pattern_copy); free(path_copy); return false; }
        pattern_segment = strtok_r(NULL, "/", &pattern_save);
        path_segment = strtok_r(NULL, "/", &path_save);
    }
    bool matched = pattern_segment == NULL && path_segment == NULL;
    free(pattern_copy); free(path_copy); return matched;
}

static ZHTTPRouteNative *z_http_find_route(ZHTTPAppNative *app, const char *method, const char *path, ZValue *params) {
    for (ZHTTPRouteNative *route = app->routes; route != NULL; route = route->next) {
        if (strcmp(route->method, "*") != 0 && !z_http_string_equal_ci(route->method, method)) continue;
        ZValue candidate = z_http_dict_new();
        if (z_http_route_match(route->pattern, path, candidate)) { *params = candidate; return route; }
    }
    return NULL;
}

static bool z_http_safe_static_path(const char *path) {
    if (strstr(path, "..") != NULL || strchr(path, '\\') != NULL || strchr(path, '\0') != path + strlen(path)) return false;
    return true;
}

static bool z_http_static_response(ZHTTPAppNative *app, const char *path, ZValue *response_value) {
    for (ZHTTPStaticNative *route = app->static_routes; route != NULL; route = route->next) {
        if (!z_http_starts_with(path, route->prefix)) continue;
        const char *relative = path + strlen(route->prefix); while (*relative == '/') relative++;
        if (!z_http_safe_static_path(relative)) return false;
        ZHTTPBuffer filename; z_http_buffer_init(&filename); z_http_buffer_append(&filename, route->directory);
        if (filename.len != 0 && filename.data[filename.len - 1] != '/') z_http_buffer_append_bytes(&filename, "/", 1);
        z_http_buffer_append(&filename, *relative == '\0' ? "index.html" : relative);
        FILE *file = fopen(filename.data, "rb");
        if (file == NULL) { z_http_buffer_free(&filename); return false; }
        if (fseek(file, 0, SEEK_END) != 0) { fclose(file); z_http_buffer_free(&filename); return false; }
        long size = ftell(file); if (size < 0 || (uint64_t)size > Z_HTTP_MAX_BODY_LIMIT) { fclose(file); z_http_buffer_free(&filename); return false; }
        rewind(file); ZValue bytes = z_bytes((size_t)size);
        if (size > 0 && fread(bytes.as.buffer->data, 1, (size_t)size, file) != (size_t)size) { fclose(file); z_http_buffer_free(&filename); return false; }
        fclose(file);
        ZValue response = z_http_response_new(); ZHTTPResponseNative *native = z_http_expect_response(response);
        native->status = 200; native->raw_body = bytes; native->content_type = z_strdup(z_http_mime_type(filename.data)); native->written = true;
        *response_value = response; z_http_buffer_free(&filename); return true;
    }
    return false;
}

static ZValue z_http_read_request(ZNetStreamNative *stream, ZHTTPAppNative *app, bool tls) {
    ZHTTPBuffer incoming; z_http_buffer_init(&incoming); size_t header_bytes = 0;
    if (!z_http_read_headers(stream, &incoming, &header_bytes)) { z_http_buffer_free(&incoming); return z_null(); }
    size_t header_text_length = header_bytes;
    while (header_text_length > 0 && (incoming.data[header_text_length - 1] == '\r' || incoming.data[header_text_length - 1] == '\n')) header_text_length--;
    char *header_copy = (char *)malloc(header_text_length + 1); if (header_copy == NULL) z_fatal("out of memory parsing HTTP request");
    memcpy(header_copy, incoming.data, header_text_length); header_copy[header_text_length] = '\0';
    char *request_line = NULL; ZValue headers = z_http_parse_headers_block(header_copy, &request_line);
    char *request_save = NULL;
    char *method = strtok_r(request_line, " ", &request_save); char *target = strtok_r(NULL, " ", &request_save); char *version = strtok_r(NULL, " ", &request_save);
    if (method == NULL || target == NULL || version == NULL || strncmp(version, "HTTP/1.", 7) != 0) { free(header_copy); z_http_buffer_free(&incoming); z_fatal("malformed HTTP request line"); }
    size_t content_length = 0; const char *length_text = z_http_dict_string(headers, "content-length", "");
    if (*length_text != '\0') { errno = 0; unsigned long long parsed = strtoull(length_text, NULL, 10); if (errno != 0) z_fatal("invalid Content-Length"); content_length = (size_t)parsed; }
    bool chunked = strstr(z_http_dict_string(headers, "transfer-encoding", ""), "chunked") != NULL;
    ZValue raw = chunked ? z_http_read_chunked_body(stream, &incoming, header_bytes, app->max_body_bytes) : z_http_read_fixed_body(stream, &incoming, header_bytes, content_length, app->max_body_bytes);
    ZHTTPRequestNative *request = (ZHTTPRequestNative *)z_alloc(sizeof(ZHTTPRequestNative));
    request->method = z_strdup(method); request->scheme = z_strdup(tls ? "https" : "http"); request->host = z_strdup(z_http_dict_string(headers, "host", ""));
    char *query_separator = strchr(target, '?');
    if (query_separator != NULL) { *query_separator++ = '\0'; request->query = z_http_dict_new(); z_http_parse_pairs(query_separator, '&', request->query, true); }
    else request->query = z_http_dict_new();
    request->path = z_http_url_decode(target, false); request->params = z_http_dict_new(); request->headers = headers; request->cookies = z_http_dict_new();
    z_http_parse_pairs(z_http_dict_string(headers, "cookie", ""), ';', request->cookies, false);
    request->form = z_http_dict_new(); request->files = z_http_dict_new(); request->raw_body = raw;
    request->body = z_http_body_object(raw, headers, request->form, request->files);
    char remote[NI_MAXHOST]; int64_t remote_port = 0; z_network_address(stream->fd, true, remote, sizeof(remote), &remote_port);
    request->remote_address = z_strdup(remote); request->remote_port = remote_port; request->stream = stream; request->hijacked = false;
    free(header_copy); z_http_buffer_free(&incoming);
    return z_http_value(ZV_HTTP_REQUEST, ZK_HTTP_REQUEST, request);
}

/* ---------------- HTTP responses and server ---------------- */

static void z_http_response_data(ZHTTPResponseNative *response, const uint8_t **data, size_t *length, ZHTTPBuffer *owned) {
    *data = NULL; *length = 0;
    if (response->raw_body.tag == ZV_BUFFER) {
        *data = (const uint8_t *)response->raw_body.as.buffer->data; *length = response->raw_body.as.buffer->len; return;
    }
    if (response->body.tag == ZV_NULL) return;
    if (response->body.tag == ZV_STRING) { *data = (const uint8_t *)response->body.as.s; *length = strlen(response->body.as.s); return; }
    if (response->body.tag == ZV_BUFFER) { *data = (const uint8_t *)response->body.as.buffer->data; *length = response->body.as.buffer->len; return; }
    char *serialized = z_json_stringify(response->body);
    z_http_buffer_append(owned, serialized); *data = (const uint8_t *)owned->data; *length = owned->len;
    if (response->content_type == NULL || z_http_starts_with(response->content_type, "text/plain")) response->content_type = z_strdup("application/json");
}

static void z_http_append_header_lines(ZHTTPBuffer *head, ZValue headers) {
    if (headers.tag != ZV_DICT) return;
    for (size_t index = 0; index < headers.as.dict->len; index++) {
        ZValue key = headers.as.dict->keys[index], value = headers.as.dict->values[index];
        if (key.tag != ZV_STRING) continue;
        if (value.tag == ZV_ARRAY) {
            for (size_t item = 0; item < value.as.array->len; item++) {
                ZValue element = value.as.array->items[item]; if (element.tag == ZV_STRING) z_http_buffer_printf(head, "%s: %s\r\n", key.as.s, element.as.s);
            }
        } else if (value.tag == ZV_STRING) z_http_buffer_printf(head, "%s: %s\r\n", key.as.s, value.as.s);
        else if (z_is_numeric(value)) z_http_buffer_printf(head, "%s: %" PRId64 "\r\n", key.as.s, z_as_i64(value));
    }
}

static bool z_http_array_contains(ZValue array, const char *value) {
    if (array.tag != ZV_ARRAY) return false;
    for (size_t index = 0; index < array.as.array->len; index++) {
        ZValue item = array.as.array->items[index];
        if (item.tag == ZV_STRING && (strcmp(item.as.s, "*") == 0 || z_http_string_equal_ci(item.as.s, value))) return true;
    }
    return false;
}

static char *z_http_join_string_array(ZValue array) {
    ZHTTPBuffer output; z_http_buffer_init(&output);
    if (array.tag == ZV_ARRAY) {
        for (size_t index = 0; index < array.as.array->len; index++) {
            ZValue item = array.as.array->items[index];
            if (item.tag != ZV_STRING) {
                continue;
            }
            if (output.len != 0) {
                z_http_buffer_append(&output, ", ");
            }
            z_http_buffer_append(&output, item.as.s);
        }
    }
    char *result = z_strdup(output.data == NULL ? "" : output.data); z_http_buffer_free(&output); return result;
}

static void z_http_apply_cors(ZHTTPAppNative *app, ZHTTPRequestNative *request, ZHTTPResponseNative *response) {
    if (!app->cors_enabled) return;
    const char *origin = z_http_dict_string(request->headers, "origin", "");
    if (*origin != '\0' && z_http_array_contains(app->cors_origins, origin)) {
        z_http_dict_set(response->headers, "Access-Control-Allow-Origin", z_string(origin));
        z_http_dict_set(response->headers, "Vary", z_string("Origin"));
    } else if (z_http_array_contains(app->cors_origins, "*")) z_http_dict_set(response->headers, "Access-Control-Allow-Origin", z_string("*"));
    char *methods = z_http_join_string_array(app->cors_methods); char *headers = z_http_join_string_array(app->cors_headers);
    if (*methods != '\0') z_http_dict_set(response->headers, "Access-Control-Allow-Methods", z_string(methods));
    if (*headers != '\0') z_http_dict_set(response->headers, "Access-Control-Allow-Headers", z_string(headers));
    if (app->cors_credentials) z_http_dict_set(response->headers, "Access-Control-Allow-Credentials", z_string("true"));
    if (app->cors_max_age > 0) { char text[32]; snprintf(text, sizeof(text), "%" PRId64, app->cors_max_age); z_http_dict_set(response->headers, "Access-Control-Max-Age", z_string(text)); }
}

static void z_http_write_chunk(ZNetStreamNative *stream, const void *data, size_t length) {
    char size_line[64]; int count = snprintf(size_line, sizeof(size_line), "%zx\r\n", length);
    z_http_stream_write_all(stream, size_line, (size_t)count);
    if (length != 0) z_http_stream_write_all(stream, data, length);
    z_http_stream_write_all(stream, "\r\n", 2);
}

static void z_http_write_response(ZNetStreamNative *stream, ZHTTPRequestNative *request, ZHTTPResponseNative *response, ZHTTPAppNative *app) {
    if (response->status < 100 || response->status > 599) response->status = 500;
    z_http_apply_cors(app, request, response);
    if (response->stream.tag == ZV_HTTP_STREAM) {
        ZHTTPStreamNative *streaming = (ZHTTPStreamNative *)response->stream.as.p;
        ZHTTPBuffer head; z_http_buffer_init(&head);
        z_http_buffer_printf(&head, "HTTP/1.1 %d %s\r\n", response->status, z_http_reason_phrase(response->status));
        z_http_buffer_printf(&head, "Content-Type: %s\r\nTransfer-Encoding: chunked\r\nConnection: close\r\n", streaming->content_type);
        z_http_append_header_lines(&head, response->headers); z_http_buffer_append(&head, "\r\n");
        z_http_stream_write_all(stream, head.data, head.len); z_http_buffer_free(&head);
        bool open = false;
        for (;;) {
            ZValue value = z_channel_receive(streaming->channel, &open);
            if (!open) break;
            if (value.tag == ZV_STRING) z_http_write_chunk(stream, value.as.s, strlen(value.as.s));
            else if (value.tag == ZV_BUFFER) z_http_write_chunk(stream, value.as.buffer->data, value.as.buffer->len);
            else { char *text = z_json_stringify(value); z_http_write_chunk(stream, text, strlen(text)); }
        }
        z_http_write_chunk(stream, "", 0); return;
    }
    ZHTTPBuffer owned; z_http_buffer_init(&owned); const uint8_t *data = NULL; size_t length = 0;
    z_http_response_data(response, &data, &length, &owned);
    ZValue compressed = z_null();
    const char *accept_encoding = z_http_dict_string(request->headers, "accept-encoding", "");
    bool gzip = app->compression && length >= 512 && strstr(accept_encoding, "gzip") != NULL;
    if (gzip) { compressed = z_http_gzip(data, length); data = (const uint8_t *)compressed.as.buffer->data; length = compressed.as.buffer->len; z_http_dict_set(response->headers, "Content-Encoding", z_string("gzip")); z_http_dict_set(response->headers, "Vary", z_string("Accept-Encoding")); }
    ZHTTPBuffer head; z_http_buffer_init(&head);
    z_http_buffer_printf(&head, "HTTP/1.1 %d %s\r\n", response->status, z_http_reason_phrase(response->status));
    z_http_buffer_printf(&head, "Content-Type: %s\r\nContent-Length: %zu\r\nConnection: close\r\n", response->content_type == NULL ? "application/octet-stream" : response->content_type, length);
    z_http_append_header_lines(&head, response->headers); z_http_buffer_append(&head, "\r\n");
    z_http_stream_write_all(stream, head.data, head.len);
    if (length != 0 && strcmp(request->method, "HEAD") != 0) z_http_stream_write_all(stream, data, length);
    z_http_buffer_free(&head); z_http_buffer_free(&owned);
}

static void z_http_merge_response_headers(ZHTTPResponseNative *base, ZHTTPResponseNative *returned) {
    if (base == NULL || returned == NULL || base == returned || base->headers.tag != ZV_DICT) return;
    ZDict *source = base->headers.as.dict;
    for (size_t index = 0; index < source->len; index++) {
        ZValue key = source->keys[index];
        if (key.tag != ZV_STRING) continue;
        if (z_http_dict_get(returned->headers, key.as.s).tag == ZV_NULL) {
            z_http_dict_set(returned->headers, key.as.s, source->values[index]);
        }
    }
}

static ZValue z_http_invoke_handler(ZValue handler, ZValue request, ZValue response) {
    ZValue args[2] = {request, response};
    pthread_mutex_lock(&z_http_handler_mutex);
    ZValue result = z_call_sync(handler, args, 2);
    pthread_mutex_unlock(&z_http_handler_mutex);
    return result;
}

static void z_http_handle_connection(ZHTTPServerNative *server, ZNetStreamNative *stream) {
    ZValue request_value = z_http_read_request(stream, server->app, server->tls);
    if (request_value.tag == ZV_NULL) return;
    ZHTTPRequestNative *request = z_http_expect_request(request_value);
    ZValue response_value = z_http_response_new();
    ZHTTPResponseNative *response = z_http_expect_response(response_value);
    if (server->app->cors_enabled && z_http_string_equal_ci(request->method, "OPTIONS")) {
        response->status = 204; response->written = true; z_http_write_response(stream, request, response, server->app); return;
    }
    if (z_http_static_response(server->app, request->path, &response_value)) {
        response = z_http_expect_response(response_value); z_http_write_response(stream, request, response, server->app); return;
    }
    ZValue params = z_http_dict_new();
    ZHTTPRouteNative *route = z_http_find_route(server->app, request->method, request->path, &params);
    if (route == NULL) {
        response->status = 404; response->body = z_string("Not Found"); response->written = true; z_http_write_response(stream, request, response, server->app); return;
    }
    request->params = params;
    for (ZHTTPMiddlewareNative *middleware = server->app->middlewares; middleware != NULL; middleware = middleware->next) {
        ZValue result = z_http_invoke_handler(middleware->handler, request_value, response_value);
        if (result.tag == ZV_HTTP_RESPONSE) { ZHTTPResponseNative *returned = z_http_expect_response(result); z_http_merge_response_headers(response, returned); response_value = result; response = returned; z_http_write_response(stream, request, response, server->app); return; }
        if (result.tag == ZV_BOOL && !result.as.b) { z_http_write_response(stream, request, response, server->app); return; }
    }
    ZValue result = z_http_invoke_handler(route->handler, request_value, response_value);
    if (request->hijacked) return;
    if (result.tag == ZV_HTTP_RESPONSE) { ZHTTPResponseNative *returned = z_http_expect_response(result); z_http_merge_response_headers(response, returned); response_value = result; response = returned; }
    else if (result.tag != ZV_NULL && !response->written) { response->body = result; response->written = true; }
    z_http_write_response(stream, request, response, server->app);
}

static void *z_http_connection_worker(void *raw) {
    ZHTTPConnectionWork *work = (ZHTTPConnectionWork *)raw;
    jmp_buf trap; z_task_error_trap = &trap;
    if (setjmp(trap) == 0) z_http_handle_connection(work->server, work->stream);
    else {
        fprintf(stderr, "zumbra HTTP handler error: %s\n", z_task_error_message[0] == '\0' ? "unknown error" : z_task_error_message);
        const char *body = "Internal Server Error";
        ZHTTPBuffer response; z_http_buffer_init(&response);
        z_http_buffer_printf(&response, "HTTP/1.1 500 Internal Server Error\r\nContent-Type: text/plain\r\nContent-Length: %zu\r\nConnection: close\r\n\r\n%s", strlen(body), body);
        z_task_error_trap = NULL;
        if (!work->stream->closed) z_http_stream_write_all(work->stream, response.data, response.len);
        z_http_buffer_free(&response);
    }
    z_task_error_trap = NULL;
    ZHTTPRequestNative *unused = NULL; (void)unused;
    if (!work->stream->closed) z_stream_close_native(z_net_stream_value(work->stream));
    pthread_mutex_lock(&work->server->mutex); if (work->server->active_connections > 0) work->server->active_connections--; pthread_cond_broadcast(&work->server->connections_done); pthread_mutex_unlock(&work->server->mutex);
    free(work);
    pthread_mutex_lock(&z_task_count_mutex); z_active_tasks--; pthread_cond_broadcast(&z_task_count_condition); pthread_mutex_unlock(&z_task_count_mutex);
    return NULL;
}

static ZNetStreamNative *z_http_accept_stream(ZHTTPServerNative *server, int fd) {
    ZNetStreamNative *stream = (ZNetStreamNative *)z_alloc(sizeof(ZNetStreamNative));
    memset(stream, 0, sizeof(*stream)); stream->fd = fd; stream->closed = false;
#if defined(ZUMBRA_ENABLE_TLS)
    if (server->tls) {
        SSL *session = SSL_new(server->tls_context); if (session == NULL) { close(fd); return NULL; }
        SSL_set_fd(session, fd);
        if (SSL_accept(session) != 1) { SSL_free(session); close(fd); return NULL; }
        stream->tls = true;
        stream->tls_session = session;
        stream->owned_context = NULL;
    }
#endif
    return stream;
}

static void *z_http_server_worker(void *raw) {
    ZHTTPServerNative *server = (ZHTTPServerNative *)raw;
    for (;;) {
        pthread_mutex_lock(&server->mutex); bool stopping = server->stopping; pthread_mutex_unlock(&server->mutex);
        if (stopping) break;
        struct sockaddr_storage address; socklen_t length = sizeof(address);
        int fd = accept(server->fd, (struct sockaddr *)&address, &length);
        if (fd < 0) {
            if (errno == EINTR) continue;
            pthread_mutex_lock(&server->mutex); stopping = server->stopping; pthread_mutex_unlock(&server->mutex);
            if (stopping || errno == EBADF || errno == EINVAL) break;
            continue;
        }
        ZNetStreamNative *stream = z_http_accept_stream(server, fd); if (stream == NULL) continue;
        ZHTTPConnectionWork *work = (ZHTTPConnectionWork *)calloc(1, sizeof(ZHTTPConnectionWork));
        if (work == NULL) { z_stream_close_native(z_net_stream_value(stream)); continue; }
        work->server = server; work->stream = stream;
        pthread_mutex_lock(&server->mutex); server->active_connections++; pthread_mutex_unlock(&server->mutex);
        pthread_mutex_lock(&z_task_count_mutex); z_active_tasks++; pthread_mutex_unlock(&z_task_count_mutex);
        pthread_t thread;
        if (pthread_create(&thread, NULL, z_http_connection_worker, work) != 0) {
            pthread_mutex_lock(&z_task_count_mutex); z_active_tasks--; pthread_cond_broadcast(&z_task_count_condition); pthread_mutex_unlock(&z_task_count_mutex);
            pthread_mutex_lock(&server->mutex); server->active_connections--; pthread_cond_broadcast(&server->connections_done); pthread_mutex_unlock(&server->mutex);
            z_stream_close_native(z_net_stream_value(stream)); free(work); continue;
        }
        pthread_detach(thread);
    }
    pthread_mutex_lock(&server->mutex); server->running = false; pthread_mutex_unlock(&server->mutex);
    return NULL;
}

static ZValue z_http_start_server(ZValue app_value, const char *host, int64_t port, bool tls, const char *certificate, const char *key) {
    ZHTTPAppNative *app = z_http_expect_app(app_value);
    ZHTTPServerNative *server = (ZHTTPServerNative *)z_alloc(sizeof(ZHTTPServerNative)); memset(server, 0, sizeof(*server));
    pthread_mutex_init(&server->mutex, NULL); pthread_cond_init(&server->connections_done, NULL); server->active_connections = 0; server->app = app; server->tls = tls; server->running = true; server->address = z_strdup(host); server->port = port;
    server->fd = z_network_bound_socket(host, port, SOCK_STREAM, true, tls ? "httpServeTLS" : "httpServe");
    char actual_host[NI_MAXHOST]; int64_t actual_port = 0; z_network_address(server->fd, false, actual_host, sizeof(actual_host), &actual_port); server->port = actual_port; server->address = z_strdup(actual_host);
#if defined(ZUMBRA_ENABLE_TLS)
    if (tls) {
        server->tls_context = SSL_CTX_new(TLS_server_method()); if (server->tls_context == NULL) z_fatal("httpServeTLS could not create TLS context");
        SSL_CTX_set_min_proto_version(server->tls_context, TLS1_2_VERSION);
        if (SSL_CTX_use_certificate_chain_file(server->tls_context, certificate) != 1) z_fatal("httpServeTLS could not load certificate");
        if (SSL_CTX_use_PrivateKey_file(server->tls_context, key, SSL_FILETYPE_PEM) != 1 || SSL_CTX_check_private_key(server->tls_context) != 1) z_fatal("httpServeTLS could not load private key");
    }
#else
    if (tls) z_fatal("TLS support is not available");
#endif
    pthread_mutex_lock(&z_http_servers_mutex); server->next = z_http_servers; z_http_servers = server; pthread_mutex_unlock(&z_http_servers_mutex);
    if (pthread_create(&server->thread, NULL, z_http_server_worker, server) != 0) z_fatal("could not start HTTP server thread");
    server->thread_started = true;
    return z_http_value(ZV_HTTP_SERVER, ZK_HTTP_SERVER, server);
}

static bool z_http_stop_server(ZHTTPServerNative *server, int64_t timeout_ms) {
    pthread_mutex_lock(&server->mutex);
    if (!server->stopping) { server->stopping = true; server->running = false; }
    int fd = server->fd; server->fd = -1;
    pthread_mutex_unlock(&server->mutex);
    if (fd >= 0) { shutdown(fd, SHUT_RDWR); close(fd); }
    if (server->thread_started && !pthread_equal(pthread_self(), server->thread)) { pthread_join(server->thread, NULL); server->thread_started = false; }
    bool completed = true;
    pthread_mutex_lock(&server->mutex);
    if (timeout_ms < 0) { while (server->active_connections != 0) pthread_cond_wait(&server->connections_done, &server->mutex); }
    else {
        struct timespec deadline = z_deadline_after_ms(timeout_ms); int status = 0;
        while (server->active_connections != 0 && status != ETIMEDOUT) status = pthread_cond_timedwait(&server->connections_done, &server->mutex, &deadline);
        completed = server->active_connections == 0;
    }
    pthread_mutex_unlock(&server->mutex);
#if defined(ZUMBRA_ENABLE_TLS)
    if (completed && server->tls_context != NULL) { SSL_CTX_free(server->tls_context); server->tls_context = NULL; }
#endif
    return completed;
}

/* ---------------- HTTP client ---------------- */

static void z_http_body_for_request(ZValue value, const uint8_t **data, size_t *length, const char **content_type, ZHTTPBuffer *owned) {
    *data = NULL; *length = 0; *content_type = NULL;
    if (value.tag == ZV_NULL) return;
    if (value.tag == ZV_STRING) { *data = (const uint8_t *)value.as.s; *length = strlen(value.as.s); *content_type = "text/plain; charset=utf-8"; return; }
    if (value.tag == ZV_BUFFER) { *data = (const uint8_t *)value.as.buffer->data; *length = value.as.buffer->len; *content_type = "application/octet-stream"; return; }
    char *json = z_json_stringify(value); z_http_buffer_append(owned, json); *data = (const uint8_t *)owned->data; *length = owned->len; *content_type = "application/json";
}

static ZValue z_http_client_response_value(int status, const char *status_text, const char *url, ZValue headers, ZValue body) {
    ZHTTPClientResponseNative *response = (ZHTTPClientResponseNative *)z_alloc(sizeof(ZHTTPClientResponseNative));
    response->status = status; response->status_text = z_strdup(status_text); response->url = z_strdup(url); response->headers = headers; response->body = body; response->cookies = z_http_dict_new();
    if (headers.tag == ZV_DICT) {
        for (size_t index = 0; index < headers.as.dict->len; index++) {
            ZValue key = headers.as.dict->keys[index], value = headers.as.dict->values[index];
            if (key.tag != ZV_STRING || value.tag != ZV_STRING || !z_http_string_equal_ci(key.as.s, "set-cookie")) continue;
            char *copy = strdup(value.as.s); if (copy == NULL) continue; char *separator = strchr(copy, ';'); if (separator != NULL) *separator = '\0'; char *equals = strchr(copy, '=');
            if (equals != NULL) { *equals++ = '\0'; z_http_dict_set(response->cookies, copy, z_string(equals)); }
            free(copy);
        }
    }
    return z_http_value(ZV_HTTP_CLIENT_RESPONSE, ZK_HTTP_CLIENT_RESPONSE, response);
}

static ZValue z_http_client_request_depth(const char *method, const char *url, ZValue headers, ZValue body, int64_t timeout_ms, int redirects) {
    if (redirects > 5) z_fatal("httpRequest exceeded 5 redirects");
    ZHTTPURL target = z_http_parse_url(url);
    ZValue stream_value = target.tls ? z_tls_connect_native(target.host, target.port, target.host, false, timeout_ms) : z_tcp_connect_native(target.host, target.port, timeout_ms);
    ZNetStreamNative *stream = z_expect_net_stream(stream_value);
    if (timeout_ms >= 0) { z_network_set_timeout(stream->fd, SO_RCVTIMEO, timeout_ms); z_network_set_timeout(stream->fd, SO_SNDTIMEO, timeout_ms); }
    ZHTTPBuffer body_owned; z_http_buffer_init(&body_owned); const uint8_t *body_data = NULL; size_t body_length = 0; const char *content_type = NULL;
    z_http_body_for_request(body, &body_data, &body_length, &content_type, &body_owned);
    ZHTTPBuffer request; z_http_buffer_init(&request);
    z_http_buffer_printf(&request, "%s %s HTTP/1.1\r\nHost: %s", method, target.path, target.host);
    bool default_port = (!target.tls && target.port == 80) || (target.tls && target.port == 443);
    if (!default_port) z_http_buffer_printf(&request, ":%" PRId64, target.port);
    z_http_buffer_append(&request, "\r\nConnection: close\r\nAccept-Encoding: gzip\r\n");
    bool has_content_type = false;
    if (headers.tag == ZV_DICT) {
        for (size_t index = 0; index < headers.as.dict->len; index++) {
            ZValue key = headers.as.dict->keys[index], value = headers.as.dict->values[index];
            if (key.tag != ZV_STRING || value.tag != ZV_STRING) continue;
            if (z_http_string_equal_ci(key.as.s, "content-type")) has_content_type = true;
            if (z_http_string_equal_ci(key.as.s, "host") || z_http_string_equal_ci(key.as.s, "content-length") || z_http_string_equal_ci(key.as.s, "connection")) continue;
            z_http_buffer_printf(&request, "%s: %s\r\n", key.as.s, value.as.s);
        }
    }
    if (content_type != NULL && !has_content_type) z_http_buffer_printf(&request, "Content-Type: %s\r\n", content_type);
    if (body_length != 0) z_http_buffer_printf(&request, "Content-Length: %zu\r\n", body_length);
    z_http_buffer_append(&request, "\r\n");
    z_http_stream_write_all(stream, request.data, request.len); if (body_length != 0) z_http_stream_write_all(stream, body_data, body_length);
    z_http_buffer_free(&request); z_http_buffer_free(&body_owned);
    ZHTTPBuffer incoming; z_http_buffer_init(&incoming); size_t header_bytes = 0;
    if (!z_http_read_headers(stream, &incoming, &header_bytes)) { z_stream_close_native(stream_value); z_http_buffer_free(&incoming); z_fatal("httpRequest received no response"); }
    size_t header_text_length = header_bytes; while (header_text_length > 0 && (incoming.data[header_text_length - 1] == '\r' || incoming.data[header_text_length - 1] == '\n')) header_text_length--;
    char *header_copy = (char *)malloc(header_text_length + 1); if (header_copy == NULL) z_fatal("out of memory parsing HTTP response");
    memcpy(header_copy, incoming.data, header_text_length); header_copy[header_text_length] = '\0'; char *status_line = NULL; ZValue response_headers = z_http_parse_headers_block(header_copy, &status_line);
    char *status_save = NULL;
    char *version = strtok_r(status_line, " ", &status_save); char *status_text = strtok_r(NULL, " ", &status_save); char *reason = status_save;
    while (reason != NULL && *reason == ' ') reason++;
    if (version == NULL || status_text == NULL || strncmp(version, "HTTP/1.", 7) != 0) z_fatal("malformed HTTP response status line");
    int status = atoi(status_text); const char *transfer = z_http_dict_string(response_headers, "transfer-encoding", ""); const char *length_text = z_http_dict_string(response_headers, "content-length", "");
    ZValue response_body;
    if (strstr(transfer, "chunked") != NULL) response_body = z_http_read_chunked_body(stream, &incoming, header_bytes, Z_HTTP_MAX_BODY_LIMIT);
    else if (*length_text != '\0') response_body = z_http_read_fixed_body(stream, &incoming, header_bytes, (size_t)strtoull(length_text, NULL, 10), Z_HTTP_MAX_BODY_LIMIT);
    else response_body = z_http_read_until_close(stream, &incoming, header_bytes, Z_HTTP_MAX_BODY_LIMIT);
    if (strstr(z_http_dict_string(response_headers, "content-encoding", ""), "gzip") != NULL) response_body = z_http_gunzip(response_body);
    z_stream_close_native(stream_value); z_http_buffer_free(&incoming); free(header_copy);
    if (status == 301 || status == 302 || status == 303 || status == 307 || status == 308) {
        const char *location = z_http_dict_string(response_headers, "location", "");
        if (*location != '\0') {
            const char *next_method = status == 303 ? "GET" : method;
            ZValue next_body = status == 303 ? z_null() : body;
            if (strstr(location, "://") != NULL) return z_http_client_request_depth(next_method, location, headers, next_body, timeout_ms, redirects + 1);
            ZHTTPBuffer absolute; z_http_buffer_init(&absolute); z_http_buffer_printf(&absolute, "%s://%s", target.scheme, target.host);
            if (!default_port) z_http_buffer_printf(&absolute, ":%" PRId64, target.port);
            if (location[0] != '/') {
                z_http_buffer_append_bytes(&absolute, "/", 1);
            }
            z_http_buffer_append(&absolute, location);
            ZValue followed = z_http_client_request_depth(next_method, absolute.data, headers, next_body, timeout_ms, redirects + 1); z_http_buffer_free(&absolute); return followed;
        }
    }
    ZHTTPBuffer status_full; z_http_buffer_init(&status_full); z_http_buffer_printf(&status_full, "%d %s", status, reason == NULL ? z_http_reason_phrase(status) : reason);
    ZValue result = z_http_client_response_value(status, status_full.data, url, response_headers, response_body); z_http_buffer_free(&status_full); return result;
}

/* ---------------- WebSocket ---------------- */

static char *z_websocket_accept(const char *key) {
    ZHTTPBuffer input; z_http_buffer_init(&input); z_http_buffer_append(&input, key); z_http_buffer_append(&input, "258EAFA5-E914-47DA-95CA-C5AB0DC85B11");
    unsigned char digest[SHA_DIGEST_LENGTH]; SHA1((const unsigned char *)input.data, input.len, digest); z_http_buffer_free(&input);
    return z_base64_encode(digest, sizeof(digest), false, true);
}

static ZValue z_websocket_value(ZNetStreamNative *stream, bool client) {
    ZWebSocketNative *socket = (ZWebSocketNative *)z_alloc(sizeof(ZWebSocketNative)); memset(socket, 0, sizeof(*socket));
    pthread_mutex_init(&socket->read_mutex, NULL); pthread_mutex_init(&socket->write_mutex, NULL); socket->stream = stream; socket->client_side = client;
    return z_http_value(ZV_WEB_SOCKET, ZK_WEB_SOCKET, socket);
}

static void z_websocket_write_frame(ZWebSocketNative *socket, uint8_t opcode, const uint8_t *payload, size_t length) {
    if (socket->closed || socket->stream->closed) z_fatal("WebSocket is closed");
    if (length > Z_HTTP_MAX_WS_FRAME) z_fatal("WebSocket message exceeds 16 MiB");
    if (opcode >= 8 && length > 125) z_fatal("WebSocket control frames cannot exceed 125 bytes");
    pthread_mutex_lock(&socket->write_mutex);
    unsigned char header[14]; size_t header_length = 0; header[header_length++] = 0x80 | opcode; unsigned char mask_bit = socket->client_side ? 0x80 : 0;
    if (length < 126) header[header_length++] = mask_bit | (unsigned char)length;
    else if (length <= 65535) { header[header_length++] = mask_bit | 126; header[header_length++] = (unsigned char)(length >> 8); header[header_length++] = (unsigned char)length; }
    else { header[header_length++] = mask_bit | 127; uint64_t size = length; for (int shift = 56; shift >= 0; shift -= 8) header[header_length++] = (unsigned char)(size >> shift); }
    unsigned char mask[4]; unsigned char *encoded = NULL;
    if (socket->client_side) {
        if (RAND_bytes(mask, sizeof(mask)) != 1) { pthread_mutex_unlock(&socket->write_mutex); z_fatal("could not generate WebSocket mask"); }
        memcpy(header + header_length, mask, sizeof(mask)); header_length += sizeof(mask);
        encoded = (unsigned char *)malloc(length == 0 ? 1 : length); if (encoded == NULL) { pthread_mutex_unlock(&socket->write_mutex); z_fatal("out of memory writing WebSocket frame"); }
        for (size_t index = 0; index < length; index++) {
            encoded[index] = payload[index] ^ mask[index % 4];
        }
        payload = encoded;
    }
    z_http_stream_write_all(socket->stream, header, header_length); if (length != 0) z_http_stream_write_all(socket->stream, payload, length); free(encoded); pthread_mutex_unlock(&socket->write_mutex);
}

static bool z_websocket_read_exact(ZWebSocketNative *socket, void *data, size_t length, int64_t timeout_ms, bool timeout_mode) {
    if (timeout_mode) z_network_set_timeout(socket->stream->fd, SO_RCVTIMEO, timeout_ms);
    size_t offset = 0;
    while (offset < length) {
        bool eof = false; ssize_t count = z_stream_read_once(socket->stream, (uint8_t *)data + offset, length - offset, &eof);
        if (count == -2) { if (timeout_mode) z_network_set_timeout(socket->stream->fd, SO_RCVTIMEO, -1); return false; }
        if (count <= 0) { socket->closed = true; if (timeout_mode) z_network_set_timeout(socket->stream->fd, SO_RCVTIMEO, -1); return false; }
        offset += (size_t)count;
    }
    if (timeout_mode) {
        z_network_set_timeout(socket->stream->fd, SO_RCVTIMEO, -1);
    }
    return true;
}

static ZValue z_websocket_read_message(ZWebSocketNative *socket, int64_t timeout_ms, bool timeout_mode) {
    pthread_mutex_lock(&socket->read_mutex);
    ZHTTPBuffer message; z_http_buffer_init(&message); uint8_t message_opcode = 0; bool received = true;
    for (;;) {
        unsigned char header[2];
        if (!z_websocket_read_exact(socket, header, 2, timeout_ms, timeout_mode)) { received = false; break; }
        bool final = (header[0] & 0x80) != 0; uint8_t opcode = header[0] & 0x0F; bool masked = (header[1] & 0x80) != 0; uint64_t length = header[1] & 0x7F;
        if (length == 126) { unsigned char extended[2]; if (!z_websocket_read_exact(socket, extended, 2, timeout_ms, timeout_mode)) { received=false; break; } length=((uint64_t)extended[0]<<8)|extended[1]; }
        else if (length == 127) { unsigned char extended[8]; if (!z_websocket_read_exact(socket, extended, 8, timeout_ms, timeout_mode)) { received=false; break; } length=0; for(int i=0;i<8;i++) length=(length<<8)|extended[i]; }
        if (length > Z_HTTP_MAX_WS_FRAME || message.len + (size_t)length > Z_HTTP_MAX_WS_FRAME) { z_http_buffer_free(&message); pthread_mutex_unlock(&socket->read_mutex); z_fatal("WebSocket message exceeds 16 MiB"); }
        unsigned char mask[4] = {0}; if (masked && !z_websocket_read_exact(socket, mask, 4, timeout_ms, timeout_mode)) { received=false; break; }
        unsigned char *payload = (unsigned char *)malloc(length == 0 ? 1 : (size_t)length); if (payload == NULL) { z_http_buffer_free(&message); pthread_mutex_unlock(&socket->read_mutex); z_fatal("out of memory reading WebSocket frame"); }
        if (length != 0 && !z_websocket_read_exact(socket, payload, (size_t)length, timeout_ms, timeout_mode)) { free(payload); received=false; break; }
        if (masked) for (size_t index=0;index<(size_t)length;index++) payload[index]^=mask[index%4];
        if (opcode == 0x8) {
            if (!socket->close_sent) { z_websocket_write_frame(socket,0x8,payload,(size_t)length); socket->close_sent=true; }
            socket->closed=true;
            if (!socket->stream->closed) z_stream_close_native(z_net_stream_value(socket->stream));
            free(payload); message_opcode=0x8; break;
        }
        if (opcode == 0x9) { z_websocket_write_frame(socket,0xA,payload,(size_t)length); free(payload); continue; }
        if (opcode == 0xA) { free(payload); continue; }
        if (opcode != 0) message_opcode = opcode;
        z_http_buffer_append_bytes(&message, payload, (size_t)length); free(payload);
        if (final) break;
    }
    pthread_mutex_unlock(&socket->read_mutex);
    if (!received && timeout_mode && !socket->closed) { z_http_buffer_free(&message); ZValue values[4]={z_string(""),z_null(),z_bool(true),z_bool(false)}; return z_array_from(values,4); }
    const char *type = message_opcode==1?"text":message_opcode==2?"binary":message_opcode==8?"close":"";
    ZValue data = z_null();
    if (message_opcode == 1) data = z_string(message.data == NULL ? "" : message.data);
    else if (message_opcode == 2) { data=z_bytes(message.len); if(message.len!=0)memcpy(data.as.buffer->data,message.data,message.len); }
    bool open = !socket->closed && message_opcode != 8; z_http_buffer_free(&message);
    if (timeout_mode) { ZValue values[4]={z_string(type),data,z_bool(open),z_bool(true)}; return z_array_from(values,4); }
    ZValue values[3]={z_string(type),data,z_bool(open)}; return z_array_from(values,3);
}

static ZValue z_websocket_upgrade(ZValue request_value) {
    ZHTTPRequestNative *request = z_http_expect_request(request_value);
    const char *upgrade = z_http_dict_string(request->headers,"upgrade",""); const char *connection = z_http_dict_string(request->headers,"connection","");
    const char *version = z_http_dict_string(request->headers,"sec-websocket-version",""); const char *key = z_http_dict_string(request->headers,"sec-websocket-key","");
    if (!z_http_string_equal_ci(upgrade,"websocket") || !z_http_contains_ci(connection,"upgrade") || strcmp(version,"13")!=0 || *key=='\0') z_fatal("webSocketUpgrade received an invalid RFC 6455 request");
    char *accept = z_websocket_accept(key); ZHTTPBuffer response; z_http_buffer_init(&response);
    z_http_buffer_printf(&response,"HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n",accept);
    z_http_stream_write_all(request->stream,response.data,response.len); z_http_buffer_free(&response); request->hijacked=true; return z_websocket_value(request->stream,false);
}

static ZValue z_websocket_connect(const char *url, ZValue headers, int64_t timeout_ms) {
    ZHTTPURL target=z_http_parse_url(url); ZValue stream_value=target.tls?z_tls_connect_native(target.host,target.port,target.host,false,timeout_ms):z_tcp_connect_native(target.host,target.port,timeout_ms); ZNetStreamNative *stream=z_expect_net_stream(stream_value);
    unsigned char nonce[16]; if(RAND_bytes(nonce,sizeof(nonce))!=1)z_fatal("could not generate WebSocket key"); char *key=z_base64_encode(nonce,sizeof(nonce),false,true);
    ZHTTPBuffer request;z_http_buffer_init(&request);z_http_buffer_printf(&request,"GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: %s\r\n",target.path,target.host,key);
    if (headers.tag == ZV_DICT) {
        for (size_t index = 0; index < headers.as.dict->len; index++) {
            ZValue key_value = headers.as.dict->keys[index];
            ZValue header_value = headers.as.dict->values[index];
            if (key_value.tag == ZV_STRING && header_value.tag == ZV_STRING) {
                z_http_buffer_printf(&request, "%s: %s\r\n", key_value.as.s, header_value.as.s);
            }
        }
    }
    z_http_buffer_append(&request, "\r\n");
    z_http_stream_write_all(stream, request.data, request.len);
    z_http_buffer_free(&request);
    ZHTTPBuffer incoming;z_http_buffer_init(&incoming);size_t header_bytes=0;if(!z_http_read_headers(stream,&incoming,&header_bytes))z_fatal("WebSocket server returned no handshake");
    size_t length=header_bytes;while(length>0&&(incoming.data[length-1]=='\r'||incoming.data[length-1]=='\n'))length--;char *copy=(char*)malloc(length+1);if(copy==NULL)z_fatal("out of memory parsing WebSocket handshake");memcpy(copy,incoming.data,length);copy[length]='\0';char *line=NULL;ZValue response_headers=z_http_parse_headers_block(copy,&line);char *ws_save=NULL;char *version=strtok_r(line," ",&ws_save);char *status=strtok_r(NULL," ",&ws_save);(void)version;
    char *expected=z_websocket_accept(key);bool accepted=status!=NULL&&atoi(status)==101&&z_http_string_equal_ci(z_http_dict_string(response_headers,"upgrade",""),"websocket")&&z_http_contains_ci(z_http_dict_string(response_headers,"connection",""),"upgrade")&&strcmp(z_http_dict_string(response_headers,"sec-websocket-accept",""),expected)==0;
    free(copy);z_http_buffer_free(&incoming);if(!accepted){z_stream_close_native(stream_value);z_fatal("WebSocket server rejected the upgrade");}return z_websocket_value(stream,true);
}

/* ---------------- Builtin dispatch and object fields ---------------- */

typedef struct {
    ZValue receiver;
    char *name;
} ZHTTPBoundMethodNative;

static ZValue z_http_bound_method(ZValue receiver, const char *name) {
    ZHTTPBoundMethodNative *method = (ZHTTPBoundMethodNative *)z_alloc(sizeof(ZHTTPBoundMethodNative));
    method->receiver = receiver; method->name = z_strdup(name);
    return z_http_value(ZV_NATIVE_METHOD, ZK_FUNCTION, method);
}

static void z_http_header_append(ZHTTPResponseNative *response, const char *name, const char *value) {
    ZValue existing = z_http_dict_get(response->headers, name);
    if (existing.tag == ZV_NULL) { z_http_dict_set(response->headers, name, z_string(value)); return; }
    if (existing.tag == ZV_ARRAY) {
        ZArray *array = existing.as.array;
        if (array->len == array->cap) {
            size_t next = array->cap == 0 ? 4 : array->cap * 2;
            ZValue *grown = (ZValue *)z_alloc(sizeof(ZValue) * next); if (array->len != 0) memcpy(grown, array->items, sizeof(ZValue) * array->len); array->items = grown; array->cap = next;
        }
        array->items[array->len++] = z_string(value); return;
    }
    ZValue values[2] = {existing, z_string(value)}; z_http_dict_set(response->headers, name, z_array_from(values, 2));
}

static ZValue z_http_make_response(const char *kind, int status, ZValue body) {
    if (status < 100 || status > 599) z_fatal("HTTP status must be between 100 and 599");
    ZValue value = z_http_response_new(); ZHTTPResponseNative *response = z_http_expect_response(value); response->status = status; response->written = true;
    if (strcmp(kind, "json") == 0) { response->content_type = z_strdup("application/json"); response->body = body; }
    else if (strcmp(kind, "html") == 0) { response->content_type = z_strdup("text/html; charset=utf-8"); response->body = z_string(z_as_cstring(body)); }
    else if (strcmp(kind, "redirect") == 0) { response->content_type = z_strdup("text/plain; charset=utf-8"); response->body = z_string(z_http_reason_phrase(status)); z_http_header_append(response, "Location", z_as_cstring(body)); }
    else { response->content_type = z_strdup("text/plain; charset=utf-8"); response->body = body; }
    return value;
}

static char *z_http_cookie_string(const char *name, const char *value, ZValue options) {
    ZHTTPBuffer cookie; z_http_buffer_init(&cookie); z_http_buffer_printf(&cookie, "%s=%s", name, value);
    const char *path = z_http_dict_string(options, "path", "/"); if (*path != '\0') z_http_buffer_printf(&cookie, "; Path=%s", path);
    const char *domain = z_http_dict_string(options, "domain", ""); if (*domain != '\0') z_http_buffer_printf(&cookie, "; Domain=%s", domain);
    int64_t max_age = z_http_dict_int(options, "maxAge", 0); if (max_age != 0) z_http_buffer_printf(&cookie, "; Max-Age=%" PRId64, max_age);
    if (z_http_dict_bool(options, "secure", false)) z_http_buffer_append(&cookie, "; Secure");
    if (z_http_dict_bool(options, "httpOnly", false)) z_http_buffer_append(&cookie, "; HttpOnly");
    const char *same_site = z_http_dict_string(options, "sameSite", "Lax"); if (*same_site != '\0') z_http_buffer_printf(&cookie, "; SameSite=%s", same_site);
    char *result = z_strdup(cookie.data); z_http_buffer_free(&cookie); return result;
}

static ZValue z_http_file_response(int status, const char *path) {
    FILE *file = fopen(path, "rb"); if (file == NULL) z_fatal("httpFile could not open %s: %s", path, strerror(errno));
    if (fseek(file, 0, SEEK_END) != 0) { fclose(file); z_fatal("httpFile could not seek %s", path); }
    long size = ftell(file); if (size < 0 || (uint64_t)size > Z_HTTP_MAX_BODY_LIMIT) { fclose(file); z_fatal("httpFile exceeds 64 MiB safety limit"); }
    rewind(file); ZValue bytes = z_bytes((size_t)size); if (size > 0 && fread(bytes.as.buffer->data, 1, (size_t)size, file) != (size_t)size) { fclose(file); z_fatal("httpFile could not read %s", path); } fclose(file);
    ZValue value = z_http_response_new(); ZHTTPResponseNative *response = z_http_expect_response(value); response->status=status; response->raw_body=bytes; response->content_type=z_strdup(z_http_mime_type(path)); response->written=true;
    const char *base = strrchr(path, '/'); base = base == NULL ? path : base + 1; ZHTTPBuffer disposition; z_http_buffer_init(&disposition); z_http_buffer_printf(&disposition,"inline; filename=\"%s\"",base); z_http_header_append(response,"Content-Disposition",disposition.data); z_http_buffer_free(&disposition); return value;
}

static char *z_http_sse_event(const char *data, const char *event, const char *id, int64_t retry) {
    ZHTTPBuffer output; z_http_buffer_init(&output);
    if (event != NULL && *event != '\0') z_http_buffer_printf(&output, "event: %s\n", event);
    if (id != NULL && *id != '\0') z_http_buffer_printf(&output, "id: %s\n", id);
    if (retry > 0) z_http_buffer_printf(&output, "retry: %" PRId64 "\n", retry);
    char *copy = strdup(data); if (copy == NULL) z_fatal("out of memory creating SSE event"); char *save=NULL;
    for(char *line=strtok_r(copy,"\n",&save);line!=NULL;line=strtok_r(NULL,"\n",&save))z_http_buffer_printf(&output,"data: %s\n",line);
    if (*data == '\0') {
        z_http_buffer_append(&output, "data: \n");
    }
    z_http_buffer_append(&output, "\n");
    free(copy);
    char *result = z_strdup(output.data);
    z_http_buffer_free(&output);
    return result;
}

static ZValue z_http_call_builtin(const char *name, const ZValue *args, size_t argc, bool *handled);

static ZValue z_http_call_native_method(ZValue callable, const ZValue *args, size_t argc) {
    ZHTTPBoundMethodNative *method = (ZHTTPBoundMethodNative *)callable.as.p;
    if (method == NULL) z_fatal("invalid native HTTP method");

    if (method->receiver.tag == ZV_HTTP_APP) {
        ZValue forwarded[8];
        bool handled = false;
        forwarded[0] = method->receiver;
        if (strcmp(method->name, "get") == 0 || strcmp(method->name, "post") == 0 ||
            strcmp(method->name, "put") == 0 || strcmp(method->name, "patch") == 0 ||
            strcmp(method->name, "delete") == 0) {
            z_expect_args("app route", argc, 2);
            const char *verb = strcmp(method->name, "get") == 0 ? "GET" :
                               strcmp(method->name, "post") == 0 ? "POST" :
                               strcmp(method->name, "put") == 0 ? "PUT" :
                               strcmp(method->name, "patch") == 0 ? "PATCH" : "DELETE";
            forwarded[1] = z_string(verb);
            forwarded[2] = args[0];
            forwarded[3] = args[1];
            return z_http_call_builtin("httpRoute", forwarded, 4, &handled);
        }
        const char *builtin = NULL;
        if (strcmp(method->name, "route") == 0) builtin = "httpRoute";
        else if (strcmp(method->name, "use") == 0) builtin = "httpUse";
        else if (strcmp(method->name, "static") == 0) builtin = "httpStatic";
        else if (strcmp(method->name, "bodyLimit") == 0) builtin = "httpLimitBody";
        else if (strcmp(method->name, "compression") == 0) builtin = "httpCompression";
        else if (strcmp(method->name, "cors") == 0) builtin = "httpCors";
        else if (strcmp(method->name, "listen") == 0) builtin = "httpServe";
        else if (strcmp(method->name, "listenTls") == 0) builtin = "httpServeTLS";
        if (builtin != NULL) {
            if (argc + 1 > sizeof(forwarded) / sizeof(forwarded[0])) z_fatal("too many HTTP method arguments");
            for (size_t index = 0; index < argc; index++) forwarded[index + 1] = args[index];
            return z_http_call_builtin(builtin, forwarded, argc + 1, &handled);
        }
        z_fatal("unknown HttpApp method %s", method->name);
    }

    if (method->receiver.tag == ZV_HTTP_SERVER) {
        ZValue forwarded[3];
        bool handled = false;
        forwarded[0] = method->receiver;
        const char *builtin = NULL;
        if (strcmp(method->name, "shutdown") == 0) builtin = "httpShutdown";
        else if (strcmp(method->name, "port") == 0) builtin = "httpServerPort";
        else if (strcmp(method->name, "address") == 0) builtin = "httpServerAddress";
        else if (strcmp(method->name, "running") == 0) builtin = "httpServerRunning";
        if (builtin == NULL) z_fatal("unknown HttpServer method %s", method->name);
        if (argc + 1 > sizeof(forwarded) / sizeof(forwarded[0])) z_fatal("too many HTTP server method arguments");
        for (size_t index = 0; index < argc; index++) forwarded[index + 1] = args[index];
        return z_http_call_builtin(builtin, forwarded, argc + 1, &handled);
    }

    if (method->receiver.tag == ZV_HTTP_RESPONSE) {
        ZHTTPResponseNative *response = z_http_expect_response(method->receiver);
        if (strcmp(method->name,"status")==0) { z_expect_args("response.status",argc,1); response->status=(int)z_as_i64(args[0]); return method->receiver; }
        if (strcmp(method->name,"header")==0) { z_expect_args("response.header",argc,2); z_http_header_append(response,z_as_cstring(args[0]),z_as_cstring(args[1])); return method->receiver; }
        if (strcmp(method->name,"json")==0 || strcmp(method->name,"send")==0 || strcmp(method->name,"html")==0) {
            z_expect_args("response writer",argc,1); response->body=args[0]; response->written=true;
            if(strcmp(method->name,"json")==0)response->content_type=z_strdup("application/json");else if(strcmp(method->name,"html")==0)response->content_type=z_strdup("text/html; charset=utf-8");
            return method->receiver;
        }
        z_fatal("unknown HTTP response method %s",method->name);
    }

    z_fatal("invalid native HTTP method receiver");
    return z_null();
}

static ZValue z_http_get_field(ZValue value, const char *name, bool *handled) {
    *handled = true;
    if (value.tag == ZV_HTTP_APP) {
        if(strcmp(name,"route")==0||strcmp(name,"get")==0||strcmp(name,"post")==0||strcmp(name,"put")==0||strcmp(name,"patch")==0||strcmp(name,"delete")==0||strcmp(name,"use")==0||strcmp(name,"static")==0||strcmp(name,"bodyLimit")==0||strcmp(name,"compression")==0||strcmp(name,"cors")==0||strcmp(name,"listen")==0||strcmp(name,"listenTls")==0)return z_http_bound_method(value,name);
        z_fatal("HttpApp has no method %s",name);
    }
    if (value.tag == ZV_HTTP_SERVER) {
        if(strcmp(name,"shutdown")==0||strcmp(name,"port")==0||strcmp(name,"address")==0||strcmp(name,"running")==0)return z_http_bound_method(value,name);
        z_fatal("HttpServer has no method %s",name);
    }
    if (value.tag == ZV_HTTP_REQUEST) {
        ZHTTPRequestNative *request=z_http_expect_request(value);
        if (strcmp(name, "method") == 0) return z_string(request->method);
        if (strcmp(name, "scheme") == 0) return z_string(request->scheme);
        if (strcmp(name, "host") == 0) return z_string(request->host);
        if (strcmp(name, "path") == 0) return z_string(request->path);
        if (strcmp(name, "remoteAddress") == 0) return z_string(request->remote_address);
        if (strcmp(name, "remotePort") == 0) return z_int(request->remote_port);
        if (strcmp(name, "params") == 0) return request->params;
        if (strcmp(name, "query") == 0) return request->query;
        if (strcmp(name, "headers") == 0) return request->headers;
        if (strcmp(name, "cookies") == 0) return request->cookies;
        if (strcmp(name, "form") == 0) return request->form;
        if (strcmp(name, "files") == 0) return request->files;
        if (strcmp(name, "body") == 0) return request->body;
        if (strcmp(name, "rawBytes") == 0) return request->raw_body;
        if (strcmp(name, "rawBody") == 0) {
            ZBuffer *body = z_expect_byte_buffer(request->raw_body);
            char *text = (char *)z_alloc(body->len + 1);
            memcpy(text, body->data, body->len);
            text[body->len] = '\0';
            return z_string(text);
        }
        z_fatal("HttpRequest has no field %s", name);
    }
    if (value.tag == ZV_HTTP_RESPONSE) {
        if(strcmp(name,"status")==0||strcmp(name,"header")==0||strcmp(name,"json")==0||strcmp(name,"send")==0||strcmp(name,"html")==0)return z_http_bound_method(value,name);
        ZHTTPResponseNative *response = z_http_expect_response(value);
        if (strcmp(name, "statusCode") == 0) return z_int(response->status);
        if (strcmp(name, "headers") == 0) return response->headers;
        if (strcmp(name, "written") == 0) return z_bool(response->written);
        z_fatal("HttpResponse has no field or method %s", name);
    }
    if (value.tag == ZV_HTTP_CLIENT_RESPONSE) {
        ZHTTPClientResponseNative *response = z_http_expect_client_response(value);
        if (strcmp(name, "statusCode") == 0) return z_int(response->status);
        if (strcmp(name, "status") == 0) return z_string(response->status_text);
        if (strcmp(name, "url") == 0) return z_string(response->url);
        if (strcmp(name, "headers") == 0) return response->headers;
        if (strcmp(name, "cookies") == 0) return response->cookies;
        if (strcmp(name, "bytes") == 0) return response->body;
        if (strcmp(name, "body") == 0) {
            ZBuffer *body = z_expect_byte_buffer(response->body);
            char *text = (char *)z_alloc(body->len + 1);
            memcpy(text, body->data, body->len);
            text[body->len] = '\0';
            return z_string(text);
        }
        z_fatal("HttpClientResponse has no field %s", name);
    }
    if (value.tag == ZV_HTTP_FILE) {
        ZHTTPFileNative *file = (ZHTTPFileNative *)value.as.p;
        if (strcmp(name, "fieldName") == 0) return z_string(file->name);
        if (strcmp(name, "filename") == 0) return z_string(file->filename);
        if (strcmp(name, "contentType") == 0) return z_string(file->content_type);
        if (strcmp(name, "size") == 0) return z_int((int64_t)file->data.as.buffer->len);
        if (strcmp(name, "data") == 0) return file->data;
        z_fatal("HttpFile has no field %s", name);
    }
    *handled=false;return z_null();
}

static ZValue z_http_call_builtin(const char *name, const ZValue *args, size_t argc, bool *handled) {
    *handled=true;
    if(strcmp(name,"httpApp")==0){z_expect_args(name,argc,0);ZHTTPAppNative*app=(ZHTTPAppNative*)z_alloc(sizeof(ZHTTPAppNative));memset(app,0,sizeof(*app));pthread_mutex_init(&app->mutex,NULL);app->max_body_bytes=Z_HTTP_DEFAULT_BODY_LIMIT;app->cors_origins=z_array_from(NULL,0);app->cors_methods=z_array_from(NULL,0);app->cors_headers=z_array_from(NULL,0);return z_http_value(ZV_HTTP_APP,ZK_HTTP_APP,app);}
    if(strcmp(name,"httpRoute")==0){z_expect_args(name,argc,4);ZHTTPAppNative*app=z_http_expect_app(args[0]);ZHTTPRouteNative*route=(ZHTTPRouteNative*)z_alloc(sizeof(ZHTTPRouteNative));route->method=z_http_lower_copy(z_as_cstring(args[1]));for(char*p=route->method;*p;p++)*p=(char)toupper((unsigned char)*p);route->pattern=z_strdup(z_as_cstring(args[2]));route->handler=args[3];route->next=NULL;pthread_mutex_lock(&app->mutex);if(app->routes==NULL)app->routes=route;else{ZHTTPRouteNative*tail=app->routes;while(tail->next)tail=tail->next;tail->next=route;}pthread_mutex_unlock(&app->mutex);return args[0];}
    if(strcmp(name,"httpUse")==0){z_expect_args(name,argc,2);ZHTTPAppNative*app=z_http_expect_app(args[0]);ZHTTPMiddlewareNative*m=(ZHTTPMiddlewareNative*)z_alloc(sizeof(ZHTTPMiddlewareNative));m->handler=args[1];pthread_mutex_lock(&app->mutex);m->next=NULL;if(app->middlewares==NULL)app->middlewares=m;else{ZHTTPMiddlewareNative*t=app->middlewares;while(t->next)t=t->next;t->next=m;}pthread_mutex_unlock(&app->mutex);return args[0];}
    if(strcmp(name,"httpStatic")==0){z_expect_args(name,argc,3);ZHTTPAppNative*app=z_http_expect_app(args[0]);ZHTTPStaticNative*r=(ZHTTPStaticNative*)z_alloc(sizeof(ZHTTPStaticNative));r->prefix=z_strdup(z_as_cstring(args[1]));r->directory=z_strdup(z_as_cstring(args[2]));pthread_mutex_lock(&app->mutex);r->next=app->static_routes;app->static_routes=r;pthread_mutex_unlock(&app->mutex);return args[0];}
    if(strcmp(name,"httpLimitBody")==0){z_expect_args(name,argc,2);int64_t limit=z_as_i64(args[1]);if(limit<=0||limit>(int64_t)Z_HTTP_MAX_BODY_LIMIT)z_fatal("httpLimitBody must be between 1 and 64 MiB");z_http_expect_app(args[0])->max_body_bytes=(size_t)limit;return args[0];}
    if(strcmp(name,"httpCompression")==0){z_expect_args(name,argc,2);z_http_expect_app(args[0])->compression=z_as_bool(args[1]);return args[0];}
    if(strcmp(name,"httpCors")==0){z_expect_args(name,argc,6);ZHTTPAppNative*app=z_http_expect_app(args[0]);app->cors_enabled=true;app->cors_origins=args[1];app->cors_methods=args[2];app->cors_headers=args[3];app->cors_credentials=z_as_bool(args[4]);app->cors_max_age=z_as_i64(args[5]);return args[0];}
    if(strcmp(name,"httpServe")==0){z_expect_args(name,argc,3);return z_http_start_server(args[0],z_as_cstring(args[1]),z_as_i64(args[2]),false,NULL,NULL);}
    if(strcmp(name,"httpServeTLS")==0){z_expect_args(name,argc,5);return z_http_start_server(args[0],z_as_cstring(args[1]),z_as_i64(args[2]),true,z_as_cstring(args[3]),z_as_cstring(args[4]));}
    if(strcmp(name,"httpShutdown")==0){z_expect_args(name,argc,2);(void)args[1];return z_bool(z_http_stop_server(z_http_expect_server(args[0]),z_as_i64(args[1])));}
    if(strcmp(name,"httpServerPort")==0){z_expect_args(name,argc,1);return z_int(z_http_expect_server(args[0])->port);}
    if(strcmp(name,"httpServerAddress")==0){z_expect_args(name,argc,1);return z_string(z_http_expect_server(args[0])->address);}
    if(strcmp(name,"httpServerRunning")==0){z_expect_args(name,argc,1);ZHTTPServerNative*s=z_http_expect_server(args[0]);pthread_mutex_lock(&s->mutex);bool running=s->running;pthread_mutex_unlock(&s->mutex);return z_bool(running);}
    if(strcmp(name,"httpText")==0){z_expect_args(name,argc,2);return z_http_make_response("text",(int)z_as_i64(args[0]),args[1]);}
    if(strcmp(name,"httpJson")==0){z_expect_args(name,argc,2);return z_http_make_response("json",(int)z_as_i64(args[0]),args[1]);}
    if(strcmp(name,"httpHtml")==0){z_expect_args(name,argc,2);return z_http_make_response("html",(int)z_as_i64(args[0]),args[1]);}
    if(strcmp(name,"httpRedirect")==0){z_expect_args(name,argc,2);return z_http_make_response("redirect",(int)z_as_i64(args[0]),args[1]);}
    if(strcmp(name,"httpFile")==0){z_expect_args(name,argc,2);return z_http_file_response((int)z_as_i64(args[0]),z_as_cstring(args[1]));}
    if(strcmp(name,"httpHeader")==0){z_expect_args(name,argc,3);z_http_header_append(z_http_expect_response(args[0]),z_as_cstring(args[1]),z_as_cstring(args[2]));return args[0];}
    if(strcmp(name,"httpCookie")==0){z_expect_args(name,argc,4);char*cookie=z_http_cookie_string(z_as_cstring(args[1]),z_as_cstring(args[2]),args[3]);z_http_header_append(z_http_expect_response(args[0]),"Set-Cookie",cookie);return args[0];}
    if(strcmp(name,"httpStream")==0||strcmp(name,"httpSSE")==0){size_t expected=strcmp(name,"httpSSE")==0?2:3;z_expect_args(name,argc,expected);int status=(int)z_as_i64(args[0]);const char*content_type=strcmp(name,"httpSSE")==0?"text/event-stream":z_as_cstring(args[1]);ZValue channel=args[expected-1];z_expect_channel(channel);ZHTTPStreamNative*stream=(ZHTTPStreamNative*)z_alloc(sizeof(ZHTTPStreamNative));stream->channel=channel;stream->content_type=z_strdup(content_type);stream->sse=strcmp(name,"httpSSE")==0;ZValue response=z_http_response_new();ZHTTPResponseNative*r=z_http_expect_response(response);r->status=status;r->content_type=z_strdup(content_type);r->stream=z_http_value(ZV_HTTP_STREAM,ZK_HTTP_STREAM,stream);r->written=true;z_http_header_append(r,"Cache-Control","no-cache");return response;}
    if(strcmp(name,"sseEvent")==0){z_expect_args(name,argc,4);return z_string(z_http_sse_event(z_as_cstring(args[0]),z_as_cstring(args[1]),z_as_cstring(args[2]),z_as_i64(args[3])));}
    if(strcmp(name,"httpRequest")==0){z_expect_args(name,argc,5);return z_http_client_request_depth(z_as_cstring(args[0]),z_as_cstring(args[1]),args[2],args[3],z_as_i64(args[4]),0);}
    if(strcmp(name,"httpStatus")==0){z_expect_args(name,argc,1);return z_int(z_http_expect_client_response(args[0])->status);}
    if(strcmp(name,"httpBodyBytes")==0){z_expect_args(name,argc,1);return z_http_expect_client_response(args[0])->body;}
    if(strcmp(name,"httpBody")==0){z_expect_args(name,argc,1);ZBuffer*b=z_expect_byte_buffer(z_http_expect_client_response(args[0])->body);char*text=(char*)z_alloc(b->len+1);memcpy(text,b->data,b->len);text[b->len]='\0';return z_string(text);}
    if(strcmp(name,"httpBodyJSON")==0){z_expect_args(name,argc,1);ZBuffer*b=z_expect_byte_buffer(z_http_expect_client_response(args[0])->body);char*text=(char*)malloc(b->len+1);if(text==NULL)z_fatal("out of memory parsing response JSON");memcpy(text,b->data,b->len);text[b->len]='\0';ZValue parsed=z_json_parse(text);free(text);return parsed;}
    if(strcmp(name,"httpHeaders")==0){z_expect_args(name,argc,1);return z_http_expect_client_response(args[0])->headers;}
    if(strcmp(name,"jsonStringify")==0){z_expect_args(name,argc,1);return z_string(z_json_stringify(args[0]));}
    if(strcmp(name,"jsonParse")==0){z_expect_args(name,argc,1);return z_json_parse(z_as_cstring(args[0]));}
    if(strcmp(name,"jwtSignHS256")==0){z_expect_args(name,argc,3);return z_string(z_jwt_sign_hs256(args[0],z_as_cstring(args[1]),z_as_i64(args[2])));}
    if(strcmp(name,"jwtVerifyHS256")==0){z_expect_args(name,argc,2);return z_jwt_verify_hs256(z_as_cstring(args[0]),z_as_cstring(args[1]));}
    if(strcmp(name,"webSocketUpgrade")==0){z_expect_args(name,argc,1);return z_websocket_upgrade(args[0]);}
    if(strcmp(name,"webSocketConnect")==0){z_expect_args(name,argc,3);return z_websocket_connect(z_as_cstring(args[0]),args[1],z_as_i64(args[2]));}
    if(strcmp(name,"webSocketRead")==0){z_expect_args(name,argc,1);return z_websocket_read_message(z_http_expect_websocket(args[0]),-1,false);}
    if(strcmp(name,"webSocketReadTimeout")==0){z_expect_args(name,argc,2);return z_websocket_read_message(z_http_expect_websocket(args[0]),z_as_i64(args[1]),true);}
    if(strcmp(name,"webSocketWriteText")==0||strcmp(name,"webSocketPing")==0){z_expect_args(name,argc,2);const char*text=z_as_cstring(args[1]);z_websocket_write_frame(z_http_expect_websocket(args[0]),strcmp(name,"webSocketPing")==0?9:1,(const uint8_t*)text,strlen(text));return z_int((int64_t)strlen(text));}
    if(strcmp(name,"webSocketWriteBinary")==0){z_expect_args(name,argc,2);const uint8_t*data=NULL;size_t length=0;z_network_data(args[1],&data,&length);z_websocket_write_frame(z_http_expect_websocket(args[0]),2,data,length);return z_int((int64_t)length);}
    if(strcmp(name,"webSocketClose")==0){
        z_expect_args(name,argc,3);
        ZWebSocketNative*socket=z_http_expect_websocket(args[0]);
        uint16_t code=(uint16_t)z_as_i64(args[1]);
        const char*reason=z_as_cstring(args[2]);
        size_t reason_length=strlen(reason);
        if(reason_length>123)z_fatal("WebSocket close reason is too long");
        unsigned char payload[125];payload[0]=(unsigned char)(code>>8);payload[1]=(unsigned char)code;memcpy(payload+2,reason,reason_length);
        if(!socket->close_sent && !socket->stream->closed){z_websocket_write_frame(socket,8,payload,reason_length+2);socket->close_sent=true;}
        if(!socket->closed && !socket->stream->closed){(void)z_websocket_read_message(socket,1000,true);}
        socket->closed=true;
        if(!socket->stream->closed)z_stream_close_native(z_net_stream_value(socket->stream));
        return z_bool(true);
    }
    if(strcmp(name,"webSocketClosed")==0){z_expect_args(name,argc,1);ZWebSocketNative*socket=z_http_expect_websocket(args[0]);return z_bool(socket->closed||socket->stream->closed);}
    *handled=false;return z_null();
}

static void z_http_runtime_init(void) {
    pthread_mutex_lock(&z_http_servers_mutex); z_http_servers=NULL; pthread_mutex_unlock(&z_http_servers_mutex);
}

static void z_http_runtime_shutdown(void) {
    pthread_mutex_lock(&z_http_servers_mutex); ZHTTPServerNative *server=z_http_servers; pthread_mutex_unlock(&z_http_servers_mutex);
    while(server!=NULL){z_http_stop_server(server,-1);server=server->next;}
}

#endif
