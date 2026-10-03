// Maintainer-only fixture generator for the source fixed_bit_vector hash call.
#include "util/hash.h"
#include <cstdio>
int main(){unsigned words[]={0x80000001u,0xf0000003u,7u};for(unsigned n: {0u,1u,7u,8u,15u,16u,17u,24u,31u,32u,33u,40u,48u,56u,63u,64u,65u})printf("%u %u\n",n,string_hash(reinterpret_cast<char*>(words),n/8,n));}
