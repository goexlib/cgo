/* 独立 C 调用方；操作和逐项预期见 ../README.md。失败必须非零退出。 */
#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include <pthread.h>
#include <stdatomic.h>
#ifndef STATIC_HOST
#include <dlfcn.h>
#else
extern int64_t goui_probe(int64_t);
extern void fixture_keep(void);
#endif
typedef int64_t(*Probe)(int64_t);
static Probe probe;
static atomic_int failures;
static int modes[28];
static void *worker(void *arg) {
 (void)arg;
 for(int n=0;n<8;n++)for(int i=1;i<=28;i++)if(modes[i-1]){
  if(probe(i)!=1)atomic_fetch_add(&failures,1);
 }
 return NULL;
}
int main(int argc,char **argv){
#ifndef STATIC_HOST
 if(argc<3)return 2;
 void *handle=dlopen(argv[1],RTLD_NOW|RTLD_GLOBAL);
 if(!handle){fprintf(stderr,"dlopen: %s\n",dlerror());return 3;}
 probe=(Probe)dlsym(handle,"goui_probe");
 int64_t expected=strtoll(argv[2],NULL,10);
#else
 fixture_keep();probe=goui_probe;
 int64_t expected=argc>1?strtoll(argv[1],NULL,10):7;
#endif
 if(!probe || probe(0)!=expected){fprintf(stderr,"entry or variant mismatch\n");return 4;}
 int count=0;
 for(int i=1;i<=28;i++){
  int ok=probe(i)==1;modes[i-1]=ok;count+=ok;
  printf("CASE %d %s\n",i,ok?"PASS":"FAIL");fflush(stdout);
 }
 pthread_t ts[8];
 for(int i=0;i<8;i++)if(pthread_create(&ts[i],NULL,worker,NULL))return 5;
 for(int i=0;i<8;i++)if(pthread_join(ts[i],NULL))return 6;
 printf("SUMMARY variant=%lld cases=%d/28 threads=8 rounds=8 repeated_failures=%d\n",(long long)expected,count,atomic_load(&failures));
 return count==28 && atomic_load(&failures)==0?0:1;
}
