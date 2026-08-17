# OneTiny 开发规范

本文档记录日常开发、代码组织、测试、生成物和发版规则。架构边界见 [architecture.md](architecture.md)。

## 基本约定

- 使用简体中文写项目内面向维护者的文档。
- Go module 路径统一使用 `github.com/tcp404/OneTiny`，不要恢复大小写混用或旧仓库路径。
- CLI 入口是 `./cmd/cli`，GUI 入口是 `./cmd/gui`。
- 项目不提供公共 Go SDK，因此不要新增 `pkg/`。
- 代码变更应尽量小步提交，PR 标题使用 Conventional Commits。
- 默认使用 squash merge，除非维护者明确要求保留 merge commit。

## 代码组织

优先把代码放在最接近使用方的位置：

- CLI-only：`cmd/cli/`
- GUI bootstrap-only：`cmd/gui/`
- Wails adapter-only：`internal/gui/`
- GUI 和 CLI 共享的应用服务：`internal/app/`
- HTTP server 生命周期和路由：`internal/server/`
- HTTP handler：`internal/server/handler/`
- 文件共享领域逻辑：`internal/share/`
- 临时列表领域逻辑：`internal/scratch/`
- 配置读写：`internal/config/`
- 当前进程状态：`internal/runtime/`
- 密码和凭据：`internal/security/`
- 访问日志：`internal/accesslog/`
- 自动更新共享逻辑：`internal/updater/`

不要为了“看起来复用”提前抽包。只有当职责稳定、调用方真实存在、依赖方向清楚时，才新增内部包。

## 配置变更规范

配置变更必须遵守一个可信源：

- 读配置：`store.Load()`。
- 写普通配置：`store.Patch(config.ConfigPatch{...})`。
- 写安全配置：`store.PatchSecurity(config.SecurityPatch{...})`。
- 校验安全配置：`store.ValidateSecureConfigFor(...)`。

不要在其他包中直接操作 Viper 或 YAML 文件。

入口行为：

- CLI 主命令 flag 只覆盖内存中的 `runtime.Runtime`。
- `onetiny config` 只写配置文件。
- `onetiny sec` 只写安全配置。
- GUI 变更先落盘，再用保存后的配置派生新的运行态。

新增配置字段时，同时更新：

- `internal/config.Config`
- `internal/config.ConfigPatch` 或 `internal/config.SecurityPatch`
- `configSettings`
- `configFromViper`
- `internal/runtime.PersistentConfig` 和 `runtime.Snapshot`
- CLI flag 或 GUI DTO，如果该字段对用户可见
- scratch 限制字段还需要更新 CLI root flag、`onetiny config`、GUI DTO 和 Wails bindings。
- 对应测试
- [architecture.md](architecture.md) 的配置分类

## 运行态变更规范

运行态来自两部分：

- 持久化配置：由 `config.Config` 转成 `runtime.PersistentConfig`。
- 进程信息：由 `runtime.NewProcess()` 创建。

不要把这些字段写进配置文件：

- `Output`
- `OS`
- `IP`
- `Pwd`
- `SessionVal`

如果一个字段只描述当前进程、当前窗口、当前 server 或当前请求，它应在 runtime、manager、service 或 request context 中，不应进 `config.Config`。

## HTTP 层规范

`internal/server` 的职责是运行 HTTP 服务，不是应用配置中心。

新增路由时：

- 路径常量放 `internal/server/routepath`。
- route 注册放 `internal/server/routes`。
- handler 放 `internal/server/handler/<domain>`。
- 需要访问当前配置时读取 `runtime.Snapshot`。
- 需要访问日志时注入或从 middleware 上下文拿 logger。
- 不在 handler 中写配置文件。

middleware 只做横切逻辑，例如 session、登录检查、访问日志、运行态注入。

## GUI 和前端规范

GUI 分三层：

- `frontend/src/`：界面和交互。
- `internal/gui/`：Wails adapter，负责窗口、托盘、dialog 和 service binding。
- `internal/app/`：应用服务，负责配置、运行态、server 和日志编排。

规则：

- 前端通过生成的 Wails bindings 调用 Go service。
- `frontend/bindings/` 不手改。
- `internal/gui/webassets/dist/` 不手改。
- Wails service 方法只做 adapter 和 DTO 转换，核心规则放 `internal/app`。
- GUI 对配置的修改必须走 `internal/app.Service`。
- 自动更新共享逻辑放 `internal/updater/`。CLI 只处理命令行交互，GUI 只处理 Wails 调用和退出重启。

## 生成物和构建产物

需要提交的生成物：

- `frontend/bindings/`
- `internal/gui/webassets/`
- release 所需的静态资源和 manifest

不应提交的本地构建产物：

- `build/bin/`
- 临时压缩包
- 本地测试输出
- `cmd/gui/rsrc_windows_*.syso`，除非某次发布流程明确要求提交

改前端或 Wails 暴露面后，至少跑：

```bash
rtk wails3 generate bindings -ts -b -names ./cmd/gui
rtk npm run build --prefix frontend
rtk go build ./cmd/gui
```

## 品牌资源规范

品牌资源采用一源多产物：

- 源文件：`resource/logo/logo.png` 和 `resource/logo/favicon.ico`。
- HTTP 页面：`/logo.png` 和 `/favicon.ico` 从 `resource/logo/` 嵌入资源读取。
- GUI 前端：`frontend/vite.config.ts` 的 `publicDir` 指向 `../resource/logo`，构建后进入 `internal/gui/webassets/dist/`。
- GUI/Wails 运行时图标：`internal/gui/assets/appicon.png` 是从 `resource/logo/logo.png` 生成的受控副本，用于 Go embed，不手改。
- 打包图标：`build/appicon.png`、`build/darwin/icons.icns`、`build/windows/icon.ico` 是本地或 CI 构建产物，不作为源文件维护。

更新 logo 时，修改 `resource/logo/logo.png` 后运行：

```bash
rtk just _icons
rtk npm run build --prefix frontend
```

`README/logo.svg` 仅作为 README 展示素材，不参与运行时或发布构建。

## 测试规范

日常 Go 变更至少跑：

```bash
rtk go test ./...
```

影响入口、依赖注入、配置或 server 生命周期时，额外跑：

```bash
rtk go build ./cmd/cli ./cmd/gui
rtk proxy git diff --check
```

影响前端时，额外跑：

```bash
rtk npm run build --prefix frontend
```

测试应该优先验证行为，不验证实现细节。配置和运行态相关测试应覆盖：

- 默认配置创建。
- patch 后重新 load。
- CLI flag 不落盘。
- GUI patch 先落盘再刷新运行态。
- 安全配置缺失时不能开启登录保护。

## PR 规范

PR 说明使用 `.github/PULL_REQUEST_TEMPLATE.md`。不要删掉模板里的核心小节，确实不适用时在对应位置说明原因。

PR 应包含：

- Summary：说明用户可见变化或架构边界变化。
- Architecture / Config Impact：说明是否影响目录职责、依赖方向、配置可信源或运行态。
- Test Plan：列出实际执行过的命令。
- 如果移动目录或改 import path，说明旧路径为什么不能保留。
- 如果改配置字段，说明持久化/运行态分类。
- 如果修改生成物，说明来源命令。

默认合并方式是 squash merge。

PR checklist 中关于 `internal/config.Store`、运行态字段、旧目录和 release-please 的条目是架构守门项。勾选前应看实际 diff，不要把 checklist 当成形式化文本。

## 发版规范

OneTiny 使用 release-please。

正常发版流程：

1. release-please 自动创建 release PR。
2. 确认版本号和 changelog。
3. 合并 release PR。
4. 等 Release Please workflow 和 release build workflow 完成。
5. 如果 release 被创建为 draft，确认附件齐全后发布 draft。

不要手动创建 tag 或 `gh release create`。如果 release-please 的版本号不符合项目阶段，应先调整 release-please 配置，再让机器人重新生成 release PR。

当前项目仍处于 `0.x` 阶段。breaking change 默认应升 minor，而不是进入 `1.0.0`。这个规则由 `release-please-config.json` 中的 `bump-minor-pre-major` 保持。
