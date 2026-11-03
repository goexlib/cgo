#include <stdint.h>
typedef struct { double x, y; } Point;
typedef struct { uint64_t n; double x; } Mixed;
typedef struct { uint64_t a, b, c; double d; } Big;
__attribute__((visibility("default"))) void fixture_keep(void) {}
double fixture_float(double a, float b, uint64_t c) { return a+b+(double)c; }
Point fixture_point(Point p, double factor) { Point r={p.x*factor,p.y*factor}; return r; }
Mixed fixture_mixed(Mixed p, uint64_t n, double x) { Mixed r={p.n+n,p.x+x};return r; }
Big fixture_big(Big p, uint64_t n) { Big r={p.a+n,p.b+2*n,p.c+3*n,p.d+0.5};return r; }
double fixture_cb_float(uintptr_t fn) { return ((double(*)(double,float,uint64_t))fn)(1.25,2.5f,7); }
double fixture_cb_point(uintptr_t fn) {
 Point p={1.5,2.25};Point r=((Point(*)(Point))fn)(p);return r.x+2*r.y;
}
double fixture_cb_mixed(uintptr_t fn) {
 Mixed p={7,1.25};Mixed r=((Mixed(*)(Mixed))fn)(p);return (double)r.n+r.x;
}
double fixture_cb_big(uintptr_t fn) {
 Big p={1,2,3,4.5};Big r=((Big(*)(Big))fn)(p);return r.a+r.b+r.c+r.d;
}
typedef int64_t(*packed_fn)(int64_t,int64_t,int64_t,int64_t,int64_t,int64_t,int64_t,int64_t,int8_t,uint8_t,int16_t,uint16_t,int32_t,uint32_t,int64_t);
int64_t fixture_cb_packed(uintptr_t fn) { return ((packed_fn)fn)(1,2,3,4,5,6,7,8,-9,10,-11,12,-13,14,15); }
double fixture_cb_spill(uintptr_t fn) {
 typedef double(*Fn)(int64_t,int64_t,int64_t,int64_t,int64_t,int64_t,int64_t,int64_t,int64_t,int64_t,double,double,double,double,double,double,double,double,double,double);
 return ((Fn)fn)(1,2,3,4,5,6,7,8,9,10,0.5,1.5,2.5,3.5,4.5,5.5,6.5,7.5,8.5,9.5);
}
int64_t fixture_packed(int64_t a,int64_t b,int64_t c,int64_t d,int64_t e,int64_t f,int64_t g,int64_t h,int8_t i,uint8_t j,int16_t k,uint16_t l,int32_t m,uint32_t n,int64_t o) {
 return a+b+c+d+e+f+g+h+i+j+k+l+m+n+o;
}

#include <string.h>
typedef struct { double x; uint64_t n; } Reverse;
typedef struct { Point a,b; } Nested;
typedef struct { float v[3]; } Float3;
typedef struct { double v[5]; } Float5;
typedef struct { double x; } OnlyFloat;
typedef struct { uint64_t n; OnlyFloat p; } NestedMixed;
Reverse fixture_reverse(Reverse p) { Reverse r={p.x+0.5,p.n+10};return r; }
Nested fixture_nested(Nested p) { Nested r={{p.a.x*2,p.a.y*2},{p.b.x*2,p.b.y*2}};return r; }
Float3 fixture_array(Float3 p) { for(int i=0;i<3;i++)p.v[i]*=2;return p; }
double fixture_mixed_spill(uint64_t a,uint64_t b,uint64_t c,uint64_t d,uint64_t e,uint64_t f,uint64_t g,Mixed p,uint64_t tail,double z) {return a+b+c+d+e+f+g+p.n+p.x+tail+z;}
double fixture_hfa_spill(double a,double b,double c,double d,double e,double f,Float3 p,float tail) {return a+b+c+d+e+f+p.v[0]+p.v[1]+p.v[2]+tail;}
double fixture_packed_hfa(uint64_t a,uint64_t b,uint64_t c,uint64_t d,uint64_t e,uint64_t f,uint64_t g,uint64_t h,double i,double j,double k,double l,double m,double n,double o,double p,int8_t before,Float3 value,int8_t after) {return a+b+c+d+e+f+g+h+i+j+k+l+m+n+o+p+before+value.v[0]+value.v[1]+value.v[2]+after;}
double fixture_big_after_float(double a,double b,double c,double d,double e,double f,double g,double h,double i,double j,Big p,uint64_t tail) {return a+b+c+d+e+f+g+h+i+j+p.a+p.b+p.c+p.d+tail;}
double fixture_big_last_reg(uint64_t a,uint64_t b,uint64_t c,uint64_t d,uint64_t e,uint64_t f,uint64_t g,Big p,uint64_t tail) {return a+b+c+d+e+f+g+p.a+p.b+p.c+p.d+tail;}
Float5 fixture_float5(Float5 p) {for(int i=0;i<5;i++)p.v[i]*=2;return p;}
NestedMixed fixture_nested_mixed(NestedMixed p) {NestedMixed r={p.n+10,{p.p.x+0.5}};return r;}
int64_t fixture_keepalive(uint64_t a,uint64_t b,uint64_t c,uint64_t d,uint64_t e,uint64_t f,uint64_t g,uint64_t h,const char *text,const uint8_t *data,uint64_t size,uintptr_t callback) {
 ((void(*)(void))callback)();
 return a+b+c+d+e+f+g+h==36 && !strcmp(text,"goui ffi") && size==3 && data[0]=='a' && data[1]=='b' && data[2]=='c';
}
double fixture_cb_mixed_spill(uintptr_t callback) {
 typedef double(*Fn)(uint64_t,uint64_t,uint64_t,uint64_t,uint64_t,uint64_t,uint64_t,Mixed,uint64_t,double);
 Mixed p={9,1.25};return ((Fn)callback)(1,2,3,4,5,6,7,p,11,0.5);
}

double fixture_cb_packed_hfa(uintptr_t callback) {
 typedef double(*Fn)(uint64_t,uint64_t,uint64_t,uint64_t,uint64_t,uint64_t,uint64_t,uint64_t,double,double,double,double,double,double,double,double,int8_t,Float3,int8_t);
 Float3 p={{1.5f,2.5f,3.5f}};return ((Fn)callback)(1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,9,p,11);
}
double fixture_cb_big_stack(uintptr_t callback) {
 typedef double(*Fn)(uint64_t,uint64_t,uint64_t,uint64_t,uint64_t,uint64_t,uint64_t,uint64_t,int8_t,Big,uint64_t);
 Big p={1,2,3,4.5};return ((Fn)callback)(1,2,3,4,5,6,7,8,9,p,11);
}

typedef struct { uint32_t v[3]; } Int3;
double fixture_int3_stack(uint64_t a,uint64_t b,uint64_t c,uint64_t d,uint64_t e,uint64_t f,uint64_t g,uint64_t h,int8_t before,Int3 p,int8_t after) {return a+b+c+d+e+f+g+h+before+p.v[0]+p.v[1]+p.v[2]+after;}
double fixture_cb_int3_stack(uintptr_t callback) {
 typedef double(*Fn)(uint64_t,uint64_t,uint64_t,uint64_t,uint64_t,uint64_t,uint64_t,uint64_t,int8_t,Int3,int8_t);
 Int3 p={{1,2,3}};return ((Fn)callback)(1,2,3,4,5,6,7,8,9,p,11);
}
