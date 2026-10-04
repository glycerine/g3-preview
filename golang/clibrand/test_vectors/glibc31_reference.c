/* Source dependency: src/math/polynomial/polynomial.cpp:4058; src/test/lp.cpp:1459.
 * Licensed under the MIT license. This oracle calls the host library only;
 * the Go implementation and tests do not require a C compiler or libc.
 */
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <gnu/libc-version.h>
#if !defined(__GLIBC__)
#error This oracle records the selected glibc profile.
#endif
static void draws(void) {
    putchar('[');
    for (unsigned i = 0; i < 128; ++i) {
        if (i) putchar(',');
        printf("%u", (unsigned)rand());
    }
    putchar(']');
}
int main(void) {
    printf("{\"profile\":\"glibc31\",\"libc_version\":\"%s\",\"rand_max\":%u,\"default\":", gnu_get_libc_version(), (unsigned)RAND_MAX);
    draws();
    printf(",\"seeds\":[");
    for (unsigned i = 0; i < 262; ++i) {
        uint32_t seed;
        if (i < 256) seed = i;
        else {
            static const uint32_t edge[] = {0x7ffffffeU,0x7fffffffU,0x80000000U,0x80000001U,0xfffffffeU,0xffffffffU};
            seed = edge[i-256];
        }
        if (i) putchar(',');
        printf("{\"seed\":%u,\"values\":", seed);
        srand(seed);
        draws();
        putchar('}');
    }
    puts("]}");
    return 0;
}
