#pragma once

#include <stdint.h>

#if defined(__cplusplus)
extern "C" {
#endif

int verify_200_9(uint32_t *indices, const char *headernonce, const uint32_t headerlen);
int verify_48_5(uint32_t *indices, const char *headernonce, const uint32_t headerlen);

#if defined(__cplusplus)
}
#endif
