#include "util/hash.h"
#include <cstdio>
int main(){random_gen r(0);for(int i=0;i<10;i++)printf("r %d\n",r());char s[8]={1,0,0,0,3,0,0,0};for(int n=0;n<=8;n++)printf("h %d %u\n",n,string_hash(s,n,0));}
