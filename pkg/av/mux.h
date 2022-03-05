#ifndef MUX_H
#define MUX_H

#include <stdint.h>

extern int goWritePacketFunc(void *, uint8_t *, int);

int cgoWritePacketFunc(void *, uint8_t *, int);

#endif