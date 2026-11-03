# ARM64 移动 FFI 原生验收

本例程验证 Go→C、C→Go 的 ARM64 ABI。它没有窗口、事件循环或 GOUI 平台策略，
不依赖桌面；因维护者需要独立编译 C 参考代码，单独放在嵌套模块中，根目录
`go test ./...` 不会隐式启动或构建它。所有产物应写入仓库外的临时目录。

`bridge` 是基座生产端唯一的测试 cgo 包；`native/host.c` 是独立原生调用方。
C 函数的实参和期望值直接写在 C/Go 测试中，不使用同一份 Go ABI 计划生成双方。
`goui_probe` 仅为测试入口，不是正式移动平台入口。

## 依赖与本地开发

当前基座工具固定 Go 1.25.8。调用方向需要 `purego` 的移动 ABI 修复分支
（本地 `feature/mobile-base`，基于上游 v0.10.2）；根模块仍保持已发布的依赖，
不提交本机绝对路径 `replace`。验证时在仓库外建立显式工作区：

```text
go 1.25.8
use (
 /path/to/cgo
 /path/to/purego
 /path/to/cgo/testdata/mobileabi
)
```

给以下命令设置 `GOWORK=/absolute/path/to/go.work`。未经修复的 upstream purego
不能作为本例程的通过基线；其问题会被测试明确报告。

## Linux ARM64 对照组

维护者安装 ARM64 GCC 与 QEMU 用户态工具后，在此目录执行：

```sh
out=$(mktemp -d)
GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc \
  go build -trimpath -buildmode=c-shared -o "$out/abi.so" .
aarch64-linux-gnu-gcc -O2 -Wall -Wextra -Werror native/host.c \
  -o "$out/host" -ldl -lpthread
qemu-aarch64 -L /usr/aarch64-linux-gnu "$out/host" "$out/abi.so" 7
```

这只是对照组；不能代替 Android bionic 或 Apple ARM64 的原生 ABI 验收。

## Android / iOS 基座消费

维护者先用 NDK 或 Xcode 配置的 `CC`，通过 GOUI 的 `goui-base export` 导出
本模块实际使用的 cgo 归档；Android 使用 `c-shared`，iOS 使用 `c-archive`，
device/simulator 分别生产。之后清空 Go 构建缓存，将 `CC` / `CXX` 设为会失败
且记录调用的程序，按 GOUI `doc/PlanMobileBuild.md` 第 13、14 节使用 `GOUI_BASE`、
链接工具组件和目标输入组件重新构建。本模块的其余 Go 包必须从源码正常编译。
基座源码或 Go/目标配置变动后必须重新生产，不复用错误版本的归档。

- Android：由 NDK 单独构建 `native/host.c`，链接 `libdl`。将宿主与消费端
  `.so` 推送至专用模拟器的 `/data/local/tmp`，赋予宿主执行权限，运行
  `host /data/local/tmp/abi.so 7`；程序通过 `dlopen(RTLD_NOW|RTLD_GLOBAL)` 加载，进程结束前不卸载 Go runtime。
- iOS：维护者为对应 SDK 编译 `native/host.c`，增加 `-DSTATIC_HOST`。用
  `goui-link` 将此对象与消费端 Go `.a` 链接，并显式请求 `CoreFoundation`。
  模拟器可 ad-hoc 签名后用 `xcrun simctl spawn <UDID> /absolute/path/Probe 7`
  运行；真机必须另外完成签名、安装与实际运行，不能用设备目标编译代替。

只启动本次验证拥有的设备/ADB server；退出后清理推送文件并关闭该设备。
本例程不申请权限，不访问个人数据，不写入应用目录。

## 预期与覆盖

28 个用例均须输出 `PASS`，最终摘要为：

```text
SUMMARY variant=7 cases=28/28 threads=8 rounds=8 repeated_failures=0
```

失败用例打印编号和实际结果；最终进程必须非零退出。首次检查通过的用例还会
在 8 个独立 pthread 中各运行 8 轮。不能仅以启动成功或部分 `PASS` 视为通过。

| 编号 | 独立检查内容 |
|---|---|
| 1–4 | 浮点调用、HFA、混合结构体、大结构体间接参数与返回 |
| 5–10 | C 发起的浮点、结构体、窄整数及寄存器溢出回调 |
| 11–12 | 窄整数栈参数、GC 与 Go 栈增长 |
| 13–15 | 反向字段顺序、嵌套 HFA、数组 HFA |
| 16–18 | 普通结构体/HFA 整体溢出、寄存器银行耗尽、HFA 栈对齐 |
| 19–22 | 溢出后的间接参数、最后一个整数寄存器、非 HFA 大数组、嵌套混合类型 |
| 23–24 | C 调用回调执行 GC 后读取参数、混合结构体回调溢出 |
| 25–26 | HFA 回调栈布局、窄参数后的间接指针栈对齐 |
| 27–28 | 普通 12 字节结构体占用完整 8 字节字组的调用与回调 |

本例程的 `...any` 调用是固定 C 签名展开，不是 C 的变参函数。
指针与切片只在同步调用期间有效；不会允许 C 代码在返回后持有 Go 内存。
