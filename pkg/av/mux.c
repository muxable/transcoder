#include "mux.h"

#include <stdint.h>

int cgoWritePacketFunc(void *opaque, uint8_t *buf, int buf_size)
{
    return goWritePacketFunc(opaque, buf, buf_size);
}