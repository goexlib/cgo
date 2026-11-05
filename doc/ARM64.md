# ARM64 typed native calls

goexlib/cgo owns typed argument and result marshalling on Linux, Android,
macOS and iOS ARM64. The backend depends on the unmodified, published purego
v0.10.2. It does not require the experimental purego mobile branch.

## Boundaries

- purego owns dynamic loading and the runtime's native-stack/callback bridge.
- goexlib/cgo owns aggregate classification, registers, stack arguments, results,
  callbacks and the synchronous pointer lifetime contract.
- GUI/platform consumers own native bindings and their OS-level semantics.

`Call`, `CallRet`, `RegisterFunc` and `RegisterLibFunc` share the ARM64 backend.
Fixed registrations prepare their argument plan once. A trailing `...any` or
`[]any` is expanded by actual type at invocation; this models a fixed native
signature and is not C variadic support. Other architectures retain their
existing backend. `CallRaw` retains its integer/address contract.

## Invocation

An invocation owns a frame containing X0-X7, D0-D7, stack bytes, the X8 indirect
result pointer, and return registers. The C-ABI assembly entry is invoked through
the public `purego.SyscallN` with one frame pointer. It therefore reuses purego's
runtime stack switch without relying on its private argument classifier or
copying runtime internals. The assembly installs the actual target's arguments,
calls it and stores its results.

The target stack area is limited to 256 bytes and checked before native entry.
Frames, indirect aggregates and required pointer arguments are kept alive and
pinned for the synchronous call. There is no shared mutable frame or global
signature cache. Native retention beyond return still requires explicit
ownership; `uintptr` is not a Go owner. See [PointerLifetime](PointerLifetime.md).

The classifier is shared with ARM64 callbacks. It handles nested/array HFAs,
ordinary composites, whole-aggregate register exhaustion, indirect large
non-HFA parameters/results, and the distinction between standard and Apple
ARM64 stack allocation. Go and C struct memory layouts must still match.

The implementation follows [AAPCS64 2025Q1](https://github.com/ARM-software/abi-aa/blob/2025Q1/aapcs64/aapcs64.rst)
and [Apple's ARM64 ABI](https://developer.apple.com/documentation/xcode/writing-arm64-code-for-apple-platforms).
Independent compiler-generated C callers/callees in `testdata/mobileabi` are the
primary ABI reference. Go callback roundtrips supplement, rather than replace,
that validation. Consumers that call purego directly do not use these extensions;
Objective-C struct messages and complex IMPs must use the goexlib/cgo path.

## Verification on 2026-11-05

Go 1.25.8 was used for the following checks with official purego v0.10.2,
without a purego workspace replacement:

- Linux amd64 package tests and race tests passed. Linux ARM64 under QEMU 6.2
  passed the 28 independent C ABI cases and 8 native threads x 8 repeated rounds.
  Emulation is recorded separately from physical Linux ARM64 validation.
- macOS 15.7.7 ARM64 package tests: 56 passed, one existing raw-call test skipped;
  race tests and all 28 independent C cases with repeated native threads passed.
  Two typed stack-call tests previously skipped for purego now run and pass.
  The remaining raw-call test cannot encode Apple's compact int32 stack slots
  using only uintptr arguments; its callback is also checked by a real C caller.
- Android API 30 ARM64 and iOS 26.3 ARM64 Simulator each passed all 28 independent
  cases, including typed and expanded registrations, and 8 threads x 8 rounds.
- Windows 10 amd64 executed the cross-built test binary against locally compiled
  C fixtures; calls, symbol registration, callbacks, pointer reentry and memory
  checks passed. This was a headless test, not desktop interaction validation.
- GOUI's isolated X11 source-copy regression passed 4102 tests/subtests with
  15 existing conditional skips. The copy preserved existing untracked tests.
- Production GLES apps built on Linux for Android, iOS Simulator and unsigned
  iOS device IPA. Android/iOS Simulator each passed 1x/2x GPU FBO pixel checks
  and the official Go test selection (137 pass, 1 original short-mode skip).
  Android background/resume preserved state. Device IPA execution remains untested.
- Production app compilation reused matched native archives with native compiler
  and JDK rejection tools on PATH; no consumer native compiler calls were recorded.

Ordinary vet reports three pre-existing unsafe.Pointer diagnostics in cast.go,
extern.go and utils.go. Other analyzers passed with `-unsafeptr=false` on Linux
amd64 and ARM64; the unfiltered commands are not reported as passing.

Detailed logs, source copies and artifacts are retained outside the repositories
under `/tmp/goui-ffi-stable-1dj6hiys` on the build host and the dedicated Mac.
Only owned simulator/ADB instances were used and stopped; the iOS instances were
deleted after validation. Native fixtures are maintainer tools and are not a
requirement for consumers of this package.
