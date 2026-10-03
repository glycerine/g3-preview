// Maintainer-only reference fixture generator. Tests never compile or run this.
#include "util/hash.h"
#include <cstdio>
struct Kind { unsigned operator()(unsigned)const{return 17;} };
struct Child { unsigned operator()(unsigned,unsigned i)const{return i*1234567+11;} };
int main(){
 for(unsigned x: {0u,1u,17u,0xffffffffu,0x80000000u}) printf("u %u %u\n",x,hash_u(x));
 for(unsigned long long x: {0ull,1ull,0xffffffffffffffffull,0x123456789abcdef0ull}) printf("ull %llu %u\n",x,hash_ull(x));
 for(unsigned n=0;n<=25;n++){ char s[25]; for(unsigned i=0;i<n;i++) s[i]=i*19+127; printf("s %u %u\n",n,string_hash(s,n,123)); }
 for(unsigned n=1;n<=10;n++) printf("c %u %u\n",n,get_composite_hash<unsigned,Kind,Child>(0,n));
 random_gen r(0xffffffffu); for(unsigned i=0;i<10;i++) printf("r %d\n",r());
 printf("comb %u mix %u\n",combine_hash(0xffffffffu,1234567u),mk_mix(0xffffffffu,1234567u,9876543u));
 printf("pair %u\n",hash_u_u(0xffffffffu,1234567u));
 random_gen z(0);for(int i=0;i<10;i++)printf("zero %d\n",z());
 random_gen gen(0);int values[]={0,1,2,3,4};shuffle(5,values,gen);for(int x:values)printf("shuffle %d\n",x);
}
