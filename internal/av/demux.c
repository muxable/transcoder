#include "demux.h"

#include <stdint.h>

int cgoReadPacketFunc(void *opaque, uint8_t *buf, int buf_size)
{
    return goReadPacketFunc(opaque, buf, buf_size);
}

int cgoWriteRTCPPacketFunc(void *opaque, uint8_t *buf, int buf_size)
{
    return goWriteRTCPPacketFunc(opaque, buf, buf_size);
}