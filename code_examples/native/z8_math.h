#ifndef ZUMBRA_Z8_MATH_H
#define ZUMBRA_Z8_MATH_H

#include <stdbool.h>
#include <stdint.h>

int32_t z8_add(int32_t left, int32_t right);
int32_t z8_apply(int32_t value, int32_t (*transform)(int32_t));
const char *z8_name(void);
void *z8_null_pointer(void);
bool z8_is_null(void *value);

#endif
