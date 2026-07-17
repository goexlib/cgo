// libcallback.c exercises cgo.NewCallback from real C code.
//
// Every test function receives a Go callback pointer, invokes it with
// hard-coded arguments and verifies the result in C, returning 0 on success
// and a small error code otherwise. This keeps the Go side free of float
// arguments so the tests also run on windows.
//
// Build:
//   linux:   gcc -shared -fPIC -O2 -o libcallback.so libcallback.c
//   macOS:   clang -dynamiclib -O2 -o libcallback.dylib libcallback.c
//   windows: gcc -shared -O2 -o libcallback.dll libcallback.c   (mingw-w64)
//            cl /LD /O2 /Felibcallback.dll libcallback.c        (MSVC)

#ifdef _MSC_VER
#define export __declspec(dllexport)
#else
#define export __attribute__((visibility("default")))
#endif

export int test_cb_int(int (*cb)(int, int)) {
    return cb(7, 8) == 78 ? 0 : 1;
}

export int test_cb_double(double (*cb)(double, double)) {
    return cb(1.5, 2.25) == 3.75 ? 0 : 1;
}

export int test_cb_float(float (*cb)(float, float)) {
    return cb(1.5f, 3.0f) == 4.5f ? 0 : 1;
}

export int test_cb_float_int_mix(double (*cb)(int, float, double, int)) {
    return cb(10, 1.5f, 2.25, 20) == 33.75 ? 0 : 1;
}

// More arguments than there are registers: 10 ints and 10 doubles.
export int test_cb_stack(double (*cb)(int, int, int, int, int, int, int, int, int, int,
                                      double, double, double, double, double,
                                      double, double, double, double, double)) {
    double r = cb(1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
                  0.5, 1.5, 2.5, 3.5, 4.5, 5.5, 6.5, 7.5, 8.5, 9.5);
    return r == 105.0 ? 0 : 1;
}

typedef struct PointF {
    float x, y;
} PointF;

export int test_cb_pointf(PointF (*cb)(float, float)) {
    PointF p = cb(1.5f, 2.5f);
    if (p.x != 3.0f) return 1;
    if (p.y != 5.0f) return 2;
    return 0;
}

typedef struct Vec3 {
    float x, y, z;
} Vec3;

export int test_cb_vec3(Vec3 (*cb)(float)) {
    Vec3 v = cb(1.5f);
    if (v.x != 1.5f) return 1;
    if (v.y != 2.5f) return 2;
    if (v.z != 3.5f) return 3;
    return 0;
}

typedef struct ArrayHFA {
    float v[2];
    float w;
} ArrayHFA;

// Homogeneous float aggregate flattened through an array field.
export int test_cb_array_hfa(ArrayHFA (*cb)(void)) {
    ArrayHFA a = cb();
    if (a.v[0] != 1.5f) return 1;
    if (a.v[1] != 2.5f) return 2;
    if (a.w != 3.5f) return 3;
    return 0;
}

typedef struct Quad {
    double a, b, c, d;
} Quad;

export int test_cb_quad(Quad (*cb)(double)) {
    Quad q = cb(1.5);
    if (q.a != 1.5) return 1;
    if (q.b != 3.0) return 2;
    if (q.c != 4.5) return 3;
    if (q.d != 6.0) return 4;
    return 0;
}

typedef struct Mixed {
    int a;
    float b;
    double c;
} Mixed;

export int test_cb_mixed(Mixed (*cb)(int, float, double)) {
    Mixed m = cb(10, 1.5f, 2.5);
    if (m.a != 11) return 1;
    if (m.b != 2.0f) return 2;
    if (m.c != 2.75) return 3;
    return 0;
}

typedef struct Big {
    long long a, b, c;
} Big;

export int test_cb_big(Big (*cb)(long long)) {
    Big b = cb(5);
    if (b.a != 5) return 1;
    if (b.b != 10) return 2;
    if (b.c != 15) return 3;
    return 0;
}

// Hidden struct return pointer combined with register and stack arguments.
export int test_cb_big_args(Big (*cb)(int, int, int, int, int, int, int, double)) {
    Big b = cb(1, 2, 3, 4, 5, 6, 7, 42.0);
    if (b.a != 10) return 1;
    if (b.b != 18) return 2;
    if (b.c != 42) return 3;
    return 0;
}

typedef struct FloatIntFloat {
    float a;
    int b;
    float c;
} FloatIntFloat;

export int test_cb_float_int_float(FloatIntFloat (*cb)(void)) {
    FloatIntFloat f = cb();
    if (f.a != 1.5f) return 1;
    if (f.b != 2) return 2;
    if (f.c != 3.5f) return 3;
    return 0;
}

typedef struct OneFloat {
    float v;
} OneFloat;

export int test_cb_one_float(OneFloat (*cb)(float)) {
    OneFloat f = cb(1.5f);
    return f.v == 1.75f ? 0 : 1;
}

typedef struct IntPair {
    long long a, b;
} IntPair;

export int test_cb_int_pair(IntPair (*cb)(long long, long long)) {
    IntPair p = cb(11, 22);
    if (p.a != 22) return 1;
    if (p.b != 66) return 2;
    return 0;
}

export int test_cb_bool_ret(_Bool (*cb)(int)) {
    if (cb(1) != 1) return 1;
    if (cb(0) != 0) return 2;
    return 0;
}

export int test_cb_void(void (*cb)(int), int (*get)(void)) {
    cb(42);
    return get() == 42 ? 0 : 1;
}

// Invoke the callback from a foreign (non-Go) thread.

typedef struct ThreadData {
    double (*cb)(double, double);
    int result;
} ThreadData;

#ifdef _WIN32

#include <windows.h>

static DWORD WINAPI thread_main(LPVOID arg) {
    ThreadData* d = (ThreadData*)arg;
    d->result = d->cb(1.5, 2.25) == 3.75 ? 0 : 1;
    return 0;
}

export int test_cb_thread(double (*cb)(double, double)) {
    ThreadData d = {cb, -1};
    HANDLE h = CreateThread(NULL, 0, thread_main, &d, 0, NULL);
    if (h == NULL) return 2;
    WaitForSingleObject(h, INFINITE);
    CloseHandle(h);
    return d.result;
}

#else

#include <pthread.h>

static void* thread_main(void* arg) {
    ThreadData* d = (ThreadData*)arg;
    d->result = d->cb(1.5, 2.25) == 3.75 ? 0 : 1;
    return 0;
}

export int test_cb_thread(double (*cb)(double, double)) {
    ThreadData d = {cb, -1};
    pthread_t t;
    if (pthread_create(&t, NULL, thread_main, &d) != 0) return 2;
    pthread_join(t, NULL);
    return d.result;
}

#endif
