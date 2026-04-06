#include <stdio.h>

#ifdef _MSC_VER
#define export __declspec(dllexport)
#else
#define export __attribute__((visibility("default")))
#endif

export int count = 0;

export void increment() {
    count++;
}

export int add(int a, int b) {
    return a + b;
}

export float getFloat() {
    return 3.14;
}

export double getDouble() {
    return 3.1415;
}

export void print(int a, float b, double c) {
    printf("a=%d b=%f c=%lf\n", a, b, c);
}

export void printString(const char* s)
{
    puts(s);
}

typedef struct Point {
    float x, y;
} Point;

export Point addPos(float x, float y)
{
    Point p = { x*2, y*2 };
    return p;
}

export const char* getName() {
    return "test";
}

typedef struct Data {
    int a;
    float b;
    double c;
} Data;

export Data getData(int a, float b, double c) {
    Data data = {a, b, c};
    return data;
}

export void printData(Data data) {
    printf("data: a=%d b=%f c=%lf\n", data.a, data.b, data.c);
}
