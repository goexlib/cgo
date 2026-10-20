# Native pointer lifetime

`CallRaw`, `Symbol.CallRaw` and `LazySymbol.CallRaw` are synchronous native
call boundaries. Each entry has `//go:uintptrescapes`, so a pointer conversion
written directly in its argument list forces its Go target onto the heap and
keeps it alive for the entire call, including lazy resolution and Go callbacks.
The annotation is needed at every forwarding entry, not only the final syscall.

```go
var value uint64
symbol.CallRaw(uintptr(unsafe.Pointer(&value)))
```

Do not store a Go address in an integer first, or prebuild a `[]uintptr` of Go
addresses and expect a later call to recover ownership. Keep the value typed as
a pointer until the annotated entry, or explicitly pin its owner. A higher-level
wrapper accepting uintptr arguments needs its own matching annotation.

`Call` and `CallRet` take typed arguments: pass pointers as pointers, rather
than converting them to uintptr. On Windows x64, pointer/string arguments,
indirect struct arguments and hidden return buffers remain pinned until the
native call finishes. Typed arguments remain GC roots through result decoding.
Native handles, function pointers, integer/float bits and borrowed C allocations
are not Go object addresses and retain their normal ABI semantics.

These rules cover synchronous calls only. If C retains an address after return,
the owner must retain and pin the Go allocation until native use has ended, or
use an appropriate native allocation. A temporary pin or `runtime.KeepAlive`
does not provide that later lifetime. Nested Go pointer fields also need an
appropriate owner; the bridge does not recursively pin arbitrary object graphs.
Callback-returned pointers similarly remain the callback owner's responsibility.

The Unix typed bridge uses purego v0.10.0's typed argument retention. The raw
bridge preserves pointer conversions before entering purego/syscall; the inner
entry cannot repair information already lost by an outer Go wrapper.

References, checked with Go 1.25.8:

- [Compiler directives](https://pkg.go.dev/cmd/compile@go1.25.8#hdr-Compiler_Directives)
- [unsafe.Pointer](https://pkg.go.dev/unsafe@go1.25.8#Pointer)
- [runtime.Pinner](https://pkg.go.dev/runtime@go1.25.8#Pinner)

`raw_pointer_test.go` exercises raw/symbol/lazy/Memset calls from fresh stacks at
varying depths against real libc writes and fixed canaries. All four cases fail
before the annotations and pass with them. `pointer_reentry_test.go` builds a
temporary C fixture, grows the Go stack and runs GC while C retains the address,
then verifies native writes and indirect struct arguments/returns. Its C-only
fixture is skipped explicitly if no C compiler exists; raw pointer tests do not
require a compiler or desktop.
