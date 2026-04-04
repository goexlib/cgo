# cgo

**Pure-Go FFI Library for Windows, Linux, and macOS - No CGO Required**

## Overview

**cgo** is a zero-c-dependency, pure-Go Foreign Function Interface (FFI) library that enables calling C shared libraries directly from Go code without requiring the CGO toolchain. Built on top of https://github.com/ebitengine/purego, it provides a unified API for Windows, Linux, and macOS with enhanced features for modern Go development.

And in addition, it adds support for **struct parameters and return values** on the Windows platform.

## Features
- 🚫 **No CGO** - Eliminates cross-compilation complexities and C toolchain dependencies
- 🎯 **Unified API** - Consistent interface across all supported platforms
- 📦 **Zero Dependencies** - Pure Go implementation, only standard library + purego
- 🔍 **Extern Variable Support** - Read and write C `extern` global variables
- 🚀 **Fast Development** - No recompilation needed when changing Go code
- 🏃 **Cross-Platform Support**:  Windows (`dll`), Linux (`so`), macOS (`dylib`)

## Quick Start

### Installation

```bash
go get github.com/goexlib/cgo
```

### 1. Basic Library Usage
```go
package main

import (
    "fmt"
    "github.com/goexlib/cgo"
)

func main() {
    // Load a shared library file
    lib, err := cgo.LoadLibrary("path/to/library.file")
    if err != nil {
        panic(err)
    }
    defer lib.Close()

    // Load a function symbol
    sym, err := lib.GetSymbol("add")
    if err != nil {
        panic(err)
    }

    // Call the C function
    result := cgo.CallRet[cgo.Int](sym.Addr(), cgo.Int(5), cgo.Int(3))
    fmt.Printf("5 + 3 = %d\n", result)
}
```

### 2. Working with Extern Variables
```go
package main

import (
    "fmt"
    "github.com/goexlib/cgo"
)

func main() {
    lib, err := cgo.LoadLibrary("mylib")
    if err != nil {
        panic(err)
    }
    defer lib.Close()
	
    // Access an extern integer variable
    counter,err := cgo.GetExternVariant[cgo.Int](lib, "counter")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Counter value: %d\n", *counter) // 0
	
	fnIncrement, err := lib.GetSymbol("increment")
	if err != nil {
		panic(err)
	}
	cgo.Call(fnIncrement)
	
	fmt.Printf("Counter value: %d\n", *counter) // 1
	
	*counter = 999
	cgo.Call(fnIncrement)
	fmt.Printf("Counter value: %d\n", *counter) // 1000
}
```

### 3. Platform-Specific Code Paths
```go
package main

import (
    "runtime"
    "github.com/goexlib/cgo"
)

func main() {
    var libPath string
    
    switch runtime.GOOS {
    case "windows":
        libPath = "C:\\libs\\mylib.dll"
    case "darwin":
        libPath = "/usr/local/lib/libmylib.dylib"
    case "linux":
        libPath = "/usr/lib/libmylib.so"
    }
    
    lib, err := cgo.LoadLibrary(libPath)
    if err != nil {
        panic(err)
    }
    defer lib.Close()
    
    // Use library...
}
```

## Platform-Specific Notes

### Windows
- Searches in: application directory, system directories, `PATH`
- Supports both `stdcall` and `cdecl` calling conventions (X64 ABI)

### Linux
- Respects `LD_LIBRARY_PATH` environment variable

### macOS
- Searges in: `@rpath`, `@executable_path`, system directories
- Supports framework bundles

## License

This project is licensed under the MIT License
