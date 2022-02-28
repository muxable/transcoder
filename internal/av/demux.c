#include "demux.h"

#include <stdint.h>

int cgoReadPacketFunc(void *opaque, uint8_t *buf, int buf_size)
{
    return goReadPacketFunc(opaque, buf, buf_size);
}