# OneTiny 架构规范

本文档沉淀 PR #16 后的架构边界。后续改代码时，优先维护这些边界，而不是按短期方便继续把职责塞回旧位置。

## 设计目标

OneTiny 是一个局域网文件共享工具，同时提供 CLI 和 GUI 两个入口。架构上需要满足三个目标：

- 配置只有一个可信源，读写都通过 `internal/config.Store`。
- 运行态从配置派生出来，只存在内存中，不反向写配置。
- CLI、GUI、HTTP server 各自是适配层，业务编排集中在清晰的内部包里。

## 目录职责

顶层目录的职责如下：

- `cmd/cli/`：CLI 入口和 CLI 专属命令/flag 解析。只放命令行进程生命周期、参数覆盖和命令实现。
- `cmd/gui/`：GUI 入口。只负责装配配置、运行态、服务和 Wails adapter。
- `frontend/`：前端源码和 Wails 生成的 TypeScript bindings。`frontend/bindings/` 是生成物，不手改。
- `internal/config/`：持久化配置的唯一可信源，负责 load、save、validate、patch。
- `internal/runtime/`：从持久化配置和当前进程信息派生出的运行态。
- `internal/updater/`：自动更新领域逻辑，负责 release 查询、版本比较、资产匹配、下载校验、解压、安装计划和 helper 替换流程。
- `internal/app/`：GUI 可调用的应用服务层，负责跨配置、运行态、server、日志的用例编排。
- `internal/gui/`：Wails adapter。负责窗口、托盘、dialog、前端服务暴露，不放核心业务规则。
- `internal/server/`：HTTP server 生命周期、Gin engine 装配、路由和 middleware。
- `internal/server/handler/`：HTTP handler。只处理 HTTP 请求/响应和调用底层能力。
- `internal/server/routepath/`：路由路径常量，避免 handler、middleware、routes 重复硬编码。
- `internal/share/`：文件共享领域逻辑，例如目录列举、路径展示、下载相关模型。
- `internal/scratch/`：临时列表领域逻辑，负责进程内文本/图片 item、hash ID、去重、容量淘汰和大小限制。
- `internal/security/`：密码哈希、凭据校验等安全规则。
- `internal/accesslog/`：访问日志读写和过滤。
- `internal/defaults/`：默认值常量。
- `internal/version/`：版本号注入点。
- `internal/kit/`：小而稳定的通用工具。新增工具必须有明确复用点，不能成为杂物间。
- `resource/`：旧版服务端模板和静态嵌入资源。

不要恢复这些旧边界：

- 不要新增 `pkg/`。当前项目没有对外 Go API 承诺。
- 不要恢复 `internal/conf`、`internal/runtimeconf`、`internal/control`、`internal/handle`、`internal/constant`。
- 不要把 CLI-only 或 GUI-only 代码放进泛化的 `internal` 包。

## 依赖方向

允许的主依赖方向：

```text
cmd/cli
  -> internal/config
  -> internal/runtime
  -> internal/server
  -> internal/updater
  -> internal/app/validation

cmd/gui
  -> internal/config
  -> internal/runtime
  -> internal/app
  -> internal/gui
  -> internal/server
  -> internal/updater

internal/gui
  -> internal/app
  -> Wails

internal/app
  -> internal/config
  -> internal/runtime
  -> internal/server
  -> internal/updater
  -> internal/accesslog
  -> internal/security

internal/server
  -> internal/runtime
  -> internal/accesslog
  -> internal/server/middleware
  -> internal/server/routes

internal/server/routes
  -> internal/server/handler/*
  -> internal/server/middleware
  -> internal/server/routepath

internal/server/handler/*
  -> internal/runtime
  -> internal/share
  -> internal/scratch
  -> internal/security
  -> internal/accesslog
  -> internal/kit/*
```

禁止的反向依赖：

- `internal/config` 不依赖 `runtime`、`app`、`server`、`gui`、`cmd`。
- `internal/runtime` 不依赖 `config`、`app`、`server`、`gui`、`cmd`。
- `internal/server` 不读取配置文件，不创建 `config.Store`。
- `internal/updater` 不读取或写入配置文件，不依赖 GUI、server 或 runtime。
- `internal/gui` 不直接写配置文件；配置变更走 `internal/app.Service`。
- `frontend` 不直接依赖 Go 实现细节，只通过生成的 Wails bindings 调用 GUI service。

## 配置可信源

`internal/config.Store` 是持久化配置的唯一可信源。

它负责：

- 计算默认配置路径。
- 确保配置文件存在。
- 从配置文件加载 `config.Config`。
- 按 patch 写回配置文件。
- 校验安全配置。
- 维护当前已加载的配置快照。

除 `internal/config` 外，生产代码不得直接调用：

- `viper.Set`
- `viper.WriteConfig`
- `os.WriteFile` 写配置文件
- YAML marshal 后自行写配置文件

配置字段分三类：

### 应持久化

这些字段属于用户配置，必须经 `config.Store` 落盘：

- `RootPath`
- `Port`
- `MaxLevel`
- `ScratchMaxItems`
- `ScratchMaxItemSize`
- `IsAllowUpload`
- `IsSecure`
- `Username`
- `PasswordHash`
- `PasswordHashAlgo`
- `LegacyPassword`，仅用于旧配置迁移和拒绝不安全启动

### 只在运行时存在

这些字段来自当前进程或当前会话，不属于配置文件：

- `Output`
- `OS`
- `IP`
- `Pwd`
- `SessionVal`
- 临时列表 item 内容、item bytes、item hash ID 和当前内存队列
- HTTP server/listener 状态
- GUI 当前窗口、托盘、dialog 状态
- GUI 端口重启确认状态，例如 pending port

临时列表限制属于持久化配置；临时列表内容属于运行进程内存，不写配置，不进入 `RootPath`。

### 选择性持久化

同一个字段可能被不同入口以不同方式修改：

- CLI 主命令 flag 覆盖当前 `runtime.Runtime`，不落盘。
- `onetiny config` 命令直接 patch `config.Store`，只落盘，不启动 server。
- `onetiny sec` 命令直接 patch `config.Store` 的安全字段。
- GUI 修改配置时，先 patch `config.Store`，再用保存后的配置派生新的运行态。
- GUI 修改端口时，如果服务正在运行，必须显式确认重启后才切换 server 监听端口。

## 启动流程

CLI 启动流程：

```text
config.DefaultPath()
  -> config.NewStore(path)
  -> store.Load()
  -> runtime.New(runtime.SnapshotFromConfig(config, runtime.NewProcess()))
  -> CLI flags patch runtime only
  -> validate runtime snapshot
  -> server.Manager.Start()
```

CLI 的 `config` 和 `sec` 子命令是例外：它们只操作 `config.Store` 并退出，不进入 server 启动流程。

GUI 启动流程：

```text
config.DefaultPath()
  -> config.NewStore(path)
  -> store.Load()
  -> runtime.New(runtime.SnapshotFromConfig(config, runtime.NewProcess()))
  -> app.NewService(config store, runtime, server manager, logger)
  -> gui.Run(...)
```

GUI 的所有前端请求都应先进入 `internal/gui.Service`，再转给 `internal/app.Service`。不要让前端或 Wails adapter 绕过应用服务直接操作 `server.Manager` 或 `config.Store`。

## Server 边界

`internal/server.Manager` 只负责 HTTP server 生命周期：

- start
- stop
- restart
- runtime patch
- 当前运行状态查询

它不负责：

- 配置文件加载和保存
- CLI flag 解析
- GUI 状态管理
- 密码创建
- 前端 DTO

HTTP 层通过 middleware 从 `runtime.Runtime` 读取运行态。handler 可以使用 `runtime.Snapshot` 做请求判断，但不能把持久化配置逻辑塞入 handler。

## GUI 边界

`internal/gui` 是 Wails adapter，不是业务层。

允许放在 `internal/gui`：

- Wails application/window/tray/dialog 逻辑
- single instance 行为
- 前端可绑定的 GUI service 方法
- Wails 事件适配

不允许放在 `internal/gui`：

- 配置文件写入规则
- server 启停的核心状态机
- 文件共享领域逻辑
- 密码哈希规则

这些逻辑应进入 `internal/app`、`internal/server`、`internal/share` 或 `internal/security`。

## 前端和生成物

前端源码在 `frontend/src/`。

这些文件是生成物或构建产物：

- `frontend/bindings/`
- `internal/gui/webassets/dist/`
- `internal/gui/webassets/assets.go`
- `cmd/gui/rsrc_windows_*.syso`

规则：

- `frontend/bindings/` 使用正式生成 bindings，不手写临时类型。
- 改 Go service 暴露面后，重新生成 bindings，再提交生成结果。
- 改前端源码后，运行前端构建并提交嵌入资源。
- 不在生成物里做业务修复，修复应回到源代码。

## 新增包判断

新增 `internal` 包前先回答：

1. 它的职责能否用一句话说明？
2. 它是否有两个以上真实调用方，或代表一个稳定领域边界？
3. 它是否会引入反向依赖？
4. 它是否只是为了躲开循环依赖？
5. 它是否会变成 `util`、`common`、`helper` 这类无边界容器？

如果答案不清楚，先放在调用方附近。CLI-only 放 `cmd/cli/`，GUI-only 放 `internal/gui/` 或 `cmd/gui/`，server-only 放 `internal/server/...`。

## 架构回归检查

涉及架构边界的 PR 至少跑：

```bash
rtk go test ./...
rtk go build ./cmd/cli ./cmd/gui
rtk proxy git diff --check
```

涉及配置读写时，额外检查：

```bash
rtk grep -n "viper\\.Set\\|viper\\.WriteConfig" cmd internal --glob '*.go' --glob '!*_test.go'
rtk grep -n "internal/conf\\|internal/runtimeconf\\|internal/control\\|internal/handle\\|internal/constant" cmd internal docs --glob '!frontend/node_modules/**'
```

这些检查命中时，不要机械删除。先判断命中是否说明边界被打破，再修正设计。
