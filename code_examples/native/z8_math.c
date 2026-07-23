#include "z8_math.h"
#include <stddef.h>

int32_t z8_add(int32_t left, int32_t right) {
    return left + right;
}

int32_t z8_apply(int32_t value, int32_t (*callback)(int32_t)) {
    return callback(value);
}

const char *z8_name(void) {
    return "zumbra";
}

void *z8_null_pointer(void) {
    return NULL;
}

bool z8_is_null(void *value) {
    return value == NULL;
}
