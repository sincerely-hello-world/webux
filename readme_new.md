# Webux — 开发指南（教学向）

> Webux 是一个基于 **Go 后端 + Svelte 5 前端** 的 Linux 服务器管理面板，最终交付为**单个可执行文件**（单二进制架构）。本文面向想要上手开发、测试、调试和部署本项目的开发者，从项目组成讲到工具链、再到日常开发流程。

---

## 目录

1. [项目组成](#一项目组成)
2. [开发工具链（mise + nub）](#二开发工具链mise--nub)
3. [目录结构速览](#三目录结构速览)
4. [Go 后端：主要库包与设计](#四go-后端主要库包与设计)
5. [Svelte 前端：组件与路由](#五svelte-前端组件与路由)
6. [环境安装（首次使用）](#六环境安装首次使用)
7. [日常开发流程](#七日常开发流程)
8. [测试与代码检查](#八测试与代码检查)
9. [调试技巧](#九调试技巧)
10. [构建与部署](#十构建与部署)
11. [常见问题速查表](#十一常见问题速查表)

**排查问题时最常用的三节**：

- [6.1 系统前提：inotify 配额](#61-系统前提inotify-配额linux-桌面用户必读) —— `mise run dev` 起不来、报 `too many open files`
- [7.4 端口与访问地址速查](#74-端口与访问地址速查dev-下最常问的问题) —— 5173 / 8989 该开哪个、dev 下配置为何不生效
- [9.5 前端连不上后端 / WebSocket 反复重连](#95-前端连不上后端--websocket-反复重连) —— 日志里每 3 秒一条 TLS 报错

---

## 一、项目组成

### 1.1 单二进制架构

```
┌──────────────────────────────────────────────┐
│              webux (单个可执行文件)             │
│  ┌────────────────────────────────────────┐  │
│  │  Go 后端                               │  │
│  │  - HTTP/HTTPS 服务 (:8989)             │  │
│  │  - REST API (/api/*)                   │  │
│  │  - WebSocket 实时事件 (/ws)             │  │
│  │  - SQLite 持久化 (webux.db)             │  │
│  └────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────┐  │
│  │  内嵌前端 (go:embed 静态资源)            │  │
│  │  - Svelte 5 SPA (dist/)                │  │
│  └────────────────────────────────────────┘  │
└──────────────────────────────────────────────┘
```

核心思想：

- **前端构建产物通过 `go:embed` 打进 Go 二进制**，所以部署时只需拷贝一个文件，无需额外安装 Node.js 或 Nginx。
- 开发时前后端**分离运行**（vite dev server + air 热重载），通过代理互联；生产时是一个整体。
- 关键衔接点：`cmd/webux/embed.go` 中的 `//go:embed dist`。**如果 `cmd/webux/dist/` 不存在，后端根本编译不过**——这是新手最先踩的坑，下面会反复强调。

### 1.2 运行时行为

| 项目     | 说明                                                             |
| -------- | ---------------------------------------------------------------- |
| 监听端口 | 默认`:8989`（HTTPS，强制 TLS）                                 |
| 数据目录 | 默认`/var/lib/webux`（可通过 `WEBUX_DATA_DIR` 环境变量覆盖） |
| 数据库   | SQLite 单文件`webux.db`（嵌入式，无需单独安装数据库）          |
| 证书     | 开发模式用`tlsutil` 自动生成的自签证书                         |
| 鉴权     | 默认开启；登录走 `/auth/login`，非 root 读不到 `/etc/shadow` 时必然失败（见 4.3）                                           |

---

## 二、开发工具链（mise + nub）

本项目采用一套现代化的本地工具链，全部工具版本与全部开发命令都由同一份 `mise.toml` 管理 —— **它既是版本锁定文件，也是唯一的命令入口**（原先并存的 `Makefile` 与 `justfile` 已删除）：

| 工具                 | 作用                                  | 版本       | 管理方式   |
| -------------------- | ------------------------------------- | ---------- | ---------- |
| **mise**       | 多语言版本管理器 + 任务运行器         | —         | 系统级安装 |
| **Go**         | 后端语言                              | `1.26.5` | mise 管理  |
| **air**        | Go 热重载（改代码自动重启）           | `1.67.4` | mise 管理  |
| **nub**        | Node 运行时 + 包管理器二合一          | `0.9.0`  | mise 管理  |
| **goreleaser** | 交叉编译 + 归档 + deb/rpm/pacman 打包 | `2.18.1` | mise 管理  |

分工逻辑：

- **工具链由 mise 统一提供**，版本在 `mise.toml [tools]` 中锁定，`mise install` 后得到完全一致的环境。缺工具时 mise 会在跑任务前**自动补装**，所以命令里不再需要 `_require_xxx` 这类前置检查。
- **所有开发命令都是 mise 任务**（定义在 `mise.toml [tasks]`）。约定：*能敲 `mise run xxx` 就不直接敲 go/nub 命令*。查看全部命令用 **`mise tasks ls`**（取代原先的 `make help` / `just --list`）。
- **Node 运行时与前端包管理统一交给 nub**：Node 版本由仓库根的 **`.node-version`**（`24`，LTS）锁定，依赖用 `nub install` / `nub ci`，脚本用 `nub run <script>`。Node **不由 mise 托管**，也不在 `mise.toml [tools]` 里。
- **Go 侧不需要 CGO**：`mise.toml` 的 `[env] CGO_ENABLED = "0"` 统一关闭了它（三个数据库驱动全是纯 Go），只有 `mise run build-pam` 这一条任务局部改为 `1`。

> 💡 类比教学：mise ≈ 自动装好"指定版本的编译器们" **加上**项目里的"快捷键总表"；nub ≈ npm + nvm 的合体。

---

## 三、目录结构速览

```
webux/
├── mise.toml              # 工具链版本锁定 + 全部命令（mise tasks）
├── .node-version          # Node 版本锁定（24 LTS，由 nub 读取）
├── .goreleaser.yaml       # 交叉编译 / 归档 / deb·rpm·pacman 打包配置
├── .github/workflows/     # CI（ci.yml 启用；release.yml 待命）
├── .air.toml              # Go 热重载配置（开发模式 --no-auth）
├── go.mod / go.sum        # Go 模块依赖
├── cmd/
│   └── webux/
│       ├── main.go        # 后端入口：flag 解析、路由注册、TLS 启动
│       ├── embed.go       # //go:embed dist —— 前端产物嵌入点
│       └── dist/          # 前端构建产物（不入库，需 mise run web 生成）
├── internal/
│   ├── api/               # chi 路由 + handlers（HTTP 层）
│   ├── auth/              # 鉴权：shadow 认证 / PAM / bypass token
│   ├── config/            # 配置加载（yaml + 环境变量覆盖）
│   ├── db/                # SQLite 打开 + 迁移（migrations/*.sql）
│   ├── system/            # 系统探测：容器/磁盘/网络/服务/包管理...
│   ├── ws/                # WebSocket hub（实时事件推送）
│   ├── tlsutil/           # 自签证书生成（开发 HTTPS）
│   ├── learn/             # "最近操作" 学习记录
│   └── migration/         # 配置迁移模板
├── web/                   # Svelte 5 前端
│   ├── index.html         # Vite 入口 HTML
│   ├── package.json       # 前端依赖与脚本（nub 管理）
│   ├── nub.lock           # 锁文件（替代 package-lock.json，必须入库）
│   ├── .npmrc             # 含 node-linker=hoisted（⚠️ 必须保留）
│   ├── vite.config.ts     # dev 代理：/api、/ws → https://localhost:8989
│   ├── src/
│   │   ├── App.svelte     # 根组件：hash 路由 + 登录态 + 布局
│   │   ├── main.ts        # 挂载入口
│   │   ├── routes/        # 23 个页面组件
│   │   ├── components/    # 共享组件（Sidebar/Topbar/...）
│   │   └── lib/           # api.ts（fetch 封装）、ws.ts（WebSocket store）
└── scripts/               # 安装与打包脚本（install.sh、PAM 容器构建、ldflags.sh…）
```

### 分层认知（教学重点）

```
浏览器
  │  HTTPS / WSS
  ▼
Go 后端 (internal/api/router.go — chi 路由)
  ├── middleware 链：RequestID → RealIP → Logger → Recoverer → Compress → Auth
  ├── handlers/  按领域拆分：system / services / processes / users / disks ...
  │     └─ 只做 HTTP 协议转换（解析请求、写 JSON）
  ├── system/    真正的系统能力：detect 探测发行版、docker/podman 容器、iptables 防火墙...
  │     └─ 只做"读系统/写系统"的脏活，不关心 HTTP
  ├── db/        SQLite + 迁移
  └── ws/        Hub 广播：指标/日志/CLI 回显/告警 四类事件
```

> 教学要点：**handler 与 system/ 分离**是好习惯。handler 负责"接口长什么样"，system/ 负责"系统怎么操作"。新增一个页面时，通常要同时动两层：`internal/api/handlers/xxx.go` + `internal/system/xxx/`。

---

## 四、Go 后端：主要库包与设计

### 4.1 依赖清单（go.mod 主要库）

| 库                                  | 用途                 | 一句话教学                                            |
| ----------------------------------- | -------------------- | ----------------------------------------------------- |
| `github.com/go-chi/chi/v5`        | HTTP 路由            | 轻量、兼容`net/http` 的 Go 路由器，天然支持中间件链 |
| `github.com/ncruces/go-sqlite3`   | 嵌入式 SQLite driver | 纯 Go 的 SQLite 实现，免 CGO，适合单二进制分发        |
| `github.com/gorilla/websocket`    | WebSocket            | 前端实时数据（指标、日志、告警）的推送通道            |
| `github.com/go-sql-driver/mysql`  | MySQL 驱动           | `-tags mysql` 构建时启用（数据库连通性检测）        |
| `github.com/jackc/pgx/v5`         | PostgreSQL 驱动      | `-tags postgres` 构建时启用                         |
| `github.com/godbus/dbus/v5`       | D-Bus 通信           | 与桌面/系统服务通信（如 systemd 用户实例）            |
| `github.com/creack/pty`           | 伪终端               | 网页版 Terminal 的 PTY 支持                           |
| `golang.org/x/crypto`             | 密码学扩展           | 加密 / 口令散列                                       |
| `github.com/openwall/yescrypt-go` | yescrypt 口令散列    | 校验`/etc/shadow` 中的口令                          |
| `gopkg.in/yaml.v3`                | YAML 解析            | 配置文件解析                                          |

### 4.2 后端启动流程（`cmd/webux/main.go`）

```
解析 flags（--config / --no-auth / --version）
  → 加载配置（internal/config：yaml + 环境变量覆盖）
  → 打开 SQLite 并跑迁移（internal/db/migrations/*.sql）
  → 初始化鉴权管理器（authMgr）
  → 创建 WebSocket Hub（internal/ws）
  → 注册 chi 路由（internal/api/router.go）
  → 加载/生成 TLS 证书（internal/tlsutil）
  → ListenAndServeTLS(:8989)
```

### 4.3 鉴权设计（`internal/auth`）

- 读取 `/etc/shadow` 做 **shadow 口令校验**（非 root 用户读不到会失败 → 所以默认 401）。
- 支持 SSSD / PAM（`crypt_pure.go` 纯 Go 实现、`crypt_cgo.go` CGO 加速、`pam_stub.go` 无 PAM 时降级）。
- 开发模式用 `--no-auth` 完全关闭鉴权；生产可用 `X-Webux-Token` bypass header。
- 前端登录：`POST /auth/login` 成功后在 `localStorage` 存 `webux_token`，之后所有请求带 `Authorization: Bearer ...`。

### 4.4 WebSocket 实时通道（`internal/ws`）

`Hub` 管理连接注册/注销，广播 4 类事件：

| 事件类型     | 场景                           |
| ------------ | ------------------------------ |
| `metric`   | 系统指标实时推送（CPU/内存等） |
| `log`      | 日志流推送                     |
| `cli_echo` | 终端命令回显同步               |
| `alert`    | 告警（如磁盘使用率超阈值）     |

前端 `web/src/lib/ws.ts` 自动重连（3 秒），协议随页面自动切换 `ws://` / `wss://`。

---

## 五、Svelte 前端：组件与路由

### 5.1 技术栈

- **Svelte 5**（`svelte.config.js` 开启 `compilerOptions.runes: true`）——Svelte 5 的新响应式写法：`$state`、`$derived`、`$props` 取代旧的 `let` + `$:`。
- **Vite 6**：构建工具 + dev server（端口 5173）。
- **TypeScript strict**：路径别名 `$lib`（src/lib）、`$components`（src/components）、`$stores`（src/stores），见 `tsconfig.json` 与 `vite.config.ts` 的 `resolve.alias`。

> 教学提示：看到以 `$` 开头的 import（如 `import { api } from '$lib/api'`），就是上面配置的别名，不是 npm 包。

### 5.2 入口与路由（`src/App.svelte`）

- 使用 **hash 路由**：页面 URL 形如 `#/disks`、`#/services`，映射表在 App.svelte 内（20+ 条）。
- 启动时请求 `/auth/whoami` 校验登录态，未登录跳 `#/login`。
- 整体布局：`Sidebar`（侧边栏）+ `Topbar`（顶栏）+ `page-content`（页面区）+ `CLIEchoPane`（终端回显面板）。

### 5.3 共享组件（`src/components/`）

| 组件                      | 职责                                           |
| ------------------------- | ---------------------------------------------- |
| `Sidebar.svelte`        | 左侧导航（按领域分组：系统/网络/存储/配置...） |
| `Topbar.svelte`         | 顶栏（标题、实时状态、设置入口）               |
| `CLIEchoPane.svelte`    | 同步显示后端 CLI 回显事件的浮层                |
| `HardeningScore.svelte` | 安全加固分数展示（配合`/api/hardening`）     |
| `HealthChecks.svelte`   | 健康检查项展示（配合`/api/health/checks`）   |

### 5.4 页面路由（`src/routes/`，23 个）

Dashboard、Ports、Migration、Services、Processes、Users、Network、Firewall、Containers、Databases、Webservers、Files、Cron、Packages、Puppet、Terminal、Ansible、AIAssistant、Disks、Logs、Settings、Login、NotFound。

每个页面组件的通用模式：

```svelte
<script lang="ts">
  // 1. 用 $state 声明响应式数据
  let items = $state<Item[]>([]);
  let loading = $state(true);

  // 2. 挂载时调用 api 封装拉数据
  onMount(async () => {
    const res = await api.get<{ items: Item[] }>('/api/xxx');
    items = res.items;
    loading = false;
  });
</script>

<!-- 3. 模板用 {#each} 渲染，加载中显示骨架/提示 -->
```

### 5.5 统一请求封装（`src/lib/api.ts`）

```ts
import { api } from '$lib/api';

// GET
const data = await api.get<T>('/api/system/info');

// POST（带 body）
await api.post('/api/services', { name: 'nginx', action: 'restart' });

// DELETE（body 可选，兼容后端部分 DELETE 接口会解析 body）
await api.delete('/api/cron', { id: 42 });
```

- 自动附带 `localStorage` 里的 `webux_token`（Bearer）。
- 401 时前端会自动清理 token 并跳转登录页（在 App.svelte 处理）。

---

## 六、环境安装（首次使用）

> 前提：系统已安装 **mise**（[mise.jdx.dev](https://mise.jdx.dev) 有各平台安装方式）。

```bash
# 1) 按 mise.toml 安装锁定版本的工具链（go / air / nub / goreleaser）
mise install

# 2) 安装项目全部依赖（go mod tidy + 前端 nub install）
mise run setup
```

Node 版本已由仓库根的 `.node-version` 锁定为 `24`（LTS），nub 会按它自动下载并切换，**无需手工 `nub pin`**。想预热缓存（CI 里就是这么做的）：`nub node install`。

### 6.1 系统前提：inotify 配额（Linux 桌面用户必读）

`air` 与 `vite` 都靠 **inotify** 监听文件变化。Linux 对**每个用户**能创建的 inotify 实例数有硬上限，默认值很小：

```bash
cat /proc/sys/fs/inotify/max_user_instances   # 常见默认值 128
```

在同时开着 VS Code、浏览器与桌面环境（KDE/GNOME）的机器上，128 很容易被占满。一旦占满，**任何进程都建不出新的 inotify 实例**，两个 dev 进程会以完全不同的面貌失败：

| 进程   | 症状                                                                                                                                       |
| ------ | ------------------------------------------------------------------------------------------------------------------------------------------ |
| `air`  | 只打印 `too many open files` 就退出；连 `air --version`、`air init` 都会失败（它启动时就要建实例）                                          |
| `vite` | `Error: EMFILE: too many open files, watch` + `errno: -24, syscall: 'watch', path: '.../vite.config.ts'` 并直接崩掉                          |

> ⚠️ **这不是 ulimit 问题。** `ulimit -n` 通常高达 1048576，报错与它无关；真正的限制是 `fs.inotify.max_user_instances`。所以看到 `too many open files` 时不要去调 `ulimit`。

修复（需要 root）：

```bash
# 临时生效
sudo sysctl fs.inotify.max_user_instances=1024
sudo sysctl fs.inotify.max_user_watches=524288

# 持久化（推荐）
printf 'fs.inotify.max_user_instances=1024\nfs.inotify.max_user_watches=524288\n' \
  | sudo tee /etc/sysctl.d/99-inotify-dev.conf
sudo sysctl --system
```

**自检**：连第 0 个实例都建不出来，就是耗尽了。

```bash
python3 -c "
import ctypes, os
libc = ctypes.CDLL('libc.so.6', use_errno=True)
fd = libc.inotify_init1(0)
print('OK' if fd >= 0 else 'EXHAUSTED: ' + os.strerror(ctypes.get_errno()))
"

# 谁占着：按进程统计 inotify 类型的 fd 数量
for p in /proc/[0-9]*; do
  [ "$(stat -c %u "$p" 2>/dev/null)" = "$(id -u)" ] || continue
  c=$(find "$p/fd" -lname 'anon_inode:inotify' 2>/dev/null | wc -l)
  [ "$c" -gt 0 ] && echo "$c  $(tr -d '\0' < "$p/comm" 2>/dev/null)  (pid ${p#/proc/})"
done | sort -rn | head
```

> 若因权限无法改 sysctl（受限容器 / WSL1），前端侧可退化为轮询：`CHOKIDAR_USEPOLLING=1 mise run dev-frontend`。但 `air` **没有**轮询模式，后端热重载仍必须靠提高配额。

### ⚠️ 三件"必须知道"的事

1. **`cmd/webux/dist` 不入库**（`.gitignore` 忽略）。后端 `/*.go` 里的 `//go:embed dist` 要求该目录存在才能编译。首次克隆后必须先：

   ```bash
   mise run web    # = nub install + 前端构建 + 把产物复制到 cmd/webux/dist
   ```
2. **`web/.npmrc` 里的 `node-linker=hoisted` 绝不能删**。nub 默认是 isolated（不提升依赖）布局，会导致 vite 相关插件的 peerDependency（如 `vite`）解析失败，报 `ERR_MODULE_NOT_FOUND`。
3. **非 root 运行**：数据目录默认 `/var/lib/webux` 普通用户不可写，且非 root 读不了 `/etc/shadow`（鉴权必然 401）。开发时统一用环境变量覆盖：`WEBUX_DATA_DIR=/tmp/webux-data`，并给后端加 `--no-auth`。以上两点 `mise run dev` 系列命令已经帮你处理好。

---

## 七、日常开发流程

### 7.1 推荐：两个终端分开跑（联调体验最好）

```bash
# 终端 A —— 后端热重载（HTTPS :8989，鉴权关闭）
mise run dev-backend
# 任务内已注入数据目录；手动跑等价于:
#   WEBUX_DATA_DIR=/tmp/webux-data air -c .air.toml

# 终端 B —— 前端 dev server（http://localhost:5173）
mise run dev-frontend
```

浏览器访问 **http://localhost:5173**：

- vite 把 `/api/*` 代理到 `https://localhost:8989`（`changeOrigin: true, secure: false`）；
- `/ws` 代理到 `wss://localhost:8989`（WebSocket 也走代理）；
- 后端是自签证书，浏览器首次访问 **https://localhost:8989** 会提示不安全 → 点"高级 → 继续前往"即可（开发环境正常现象）。

### 7.2 或：单终端一条命令（前端后台、后端前台，Ctrl-C 全退）

```bash
mise run dev
```

内部逻辑：`ensure-dist` 检查（`cmd/webux/dist` 必须存在，否则 `go:embed` 编译不过）→ `mkdir -p /tmp/webux-data` → 后台启动 vite → 前台跑 `air`，退出时自动 `kill` 前端进程。

### 7.3 开发闭环（改代码 → 看效果）

| 改了哪里                                      | 会发生什么                               |
| --------------------------------------------- | ---------------------------------------- |
| `internal/**` 或 `cmd/**` 的 `.go` 文件 | air 自动重编译并重启后端（约 0.5s 延迟） |
| `web/src/**` 的组件/逻辑                    | vite HMR 秒级热更新，页面状态保留        |
| `web/package.json`                          | 需要重新`cd web && nub install`        |
| 新增了前端路由/页面                           | 无需任何操作，hash 路由直接可用          |

### 7.4 端口与访问地址速查（dev 下最常问的问题）

dev 模式同时跑着**两个服务、两个端口**，务必分清：

| 端口        | 谁在听  | 协议           | 作用                                                          |
| ----------- | ------- | -------------- | ------------------------------------------------------------- |
| **5173**    | vite    | HTTP           | 前端 dev server —— **日常开发就用这个地址**                   |
| **8989**    | Go 后端 | HTTPS（自签）  | REST API + WebSocket；由 vite 反向代理，通常不直接访问        |

```bash
# ✅ 日常开发入口（推荐）
http://localhost:5173        # 本机
http://<本机IP>:5173         # 局域网 —— vite 带 --host，绑定 0.0.0.0

# ✅ 只调试 API 时直连后端（浏览器会报自签证书告警）
https://localhost:8989
```

两个端口的值从哪里来（要改端口就改这两处）：

| 值        | 定义位置                                                                                    |
| --------- | ------------------------------------------------------------------------------------------- |
| vite 5173 | `web/package.json` 的 `dev` 脚本 `--port 5173`，与 `vite.config.ts` 的 `server.port: 5173`（刻意保持一致） |
| 后端 8989 | `internal/config/config.go` 的 `defaults()` → `ListenAddr: ":8989"`                        |
| 代理目标  | `web/vite.config.ts` 的 `server.proxy` → `https://localhost:8989`（`/api` 与 `/ws`）        |

实测确认监听的命令：

```bash
ss -ltnp | grep -E '5173|8989'
# LISTEN *:5173   users:(("MainThread",...))     <- vite
# LISTEN *:8989   users:(("webux",...))          <- 后端
```

#### ⚠️ dev 模式下 `scripts/config.yaml` 完全不生效

`.air.toml` 的 `entrypoint` 传的是 `--config /dev/null`。空文件 → `yaml.Unmarshal` 不覆盖任何字段 → 走 `defaults()`。所以：

- **在 `scripts/config.yaml` 里改 `listen_addr`，对 dev 没有任何影响**，后端恒定监听 `:8989`（那份文件只在安装成服务/包之后才被读取）；
- 要在 dev 改端口，改 `.air.toml` 里 `entrypoint` 的 `--config` 参数，或加环境变量 `WEBUX_LISTEN_ADDR`。

看日志确认 dev 实际生效的配置：

```
msg="webux listening" addr=https://:8989 auth=DISABLED
```

`auth=DISABLED` 说明 `--no-auth` 生效，`addr` 就是实际监听地址。两个参数都来自同一行 `entrypoint`：

```toml
entrypoint = [".air/webux", "--config", "/dev/null", "--no-auth"]
```

#### ⚠️ 为什么不要用 `https://<本机IP>:8989` 当开发入口

那个页面**能打开**（浏览器点「高级 → 继续前往」即可），但页面里的 WebSocket 会**永远连不上**。原因：`ws.ts` 用 `location.host` 拼地址，直连 8989 时它会去连 `wss://<本机IP>:8989/ws`，而自签证书下浏览器会拒掉这个 WSS 握手，于是触发每 3 秒一次的重连循环：

```
后端日志：      http: TLS handshake error from <本机IP>:xxxxx: remote error: tls: unknown certificate
浏览器控制台：  WebSocket connection to 'wss://<本机IP>:8989/ws' failed
```

这不是故障，是「连错入口」的症状。**dev 统一走 5173**：那里 `/ws` 会被 vite 代理成 `wss://localhost:8989`（`secure: false`，跳过证书校验），完全正常。详见 9.5。

---

## 八、测试与代码检查

### 8.1 命令总表

| 命令                   | 实际执行                                                                                          | 需要`cmd/webux/dist` | 典型耗时 |
| ---------------------- | ------------------------------------------------------------------------------------------------- | ---------------------- | -------- |
| `mise run test-go`   | `go test ./internal/...`                                                                        | 否                     | 秒级     |
| `mise run check-web` | `cd web && nub run check`（svelte-check）                                                       | 否                     | 十秒级   |
| `mise run web`       | `nub install` + 前端构建 + 同步 `dist`                                                        | —                     | 首次较慢 |
| `mise run vet`       | `mise run web` → `go vet ./...`                                                              | 是                     | 分钟级   |
| `mise run test`      | `mise run web` → `go test ./...` → `nub run check`                                        | 是                     | 分钟级   |
| `mise run fmt`       | `gofmt -w` 全部受版本控制的 `.go`                                                             | 否                     | 秒级     |
| `mise run ci`        | `web-ci` → gofmt 门禁 → `go vet` → `go test` → `svelte-check` → `goreleaser check` | 是                     | 分钟级   |

`nub run check` 内部是 `svelte-check --tsconfig ./tsconfig.json`，会同时校验 Svelte 模板与 TS 类型。**前端类型错误（如调用 `api.delete(path, body)` 缺参）在这里就能抓出来**。

### 8.2 为什么 `test` / `vet` 要求先有前端产物

`cmd/webux/embed.go` 声明了 `//go:embed dist`。`go test ./...` 与 `go vet ./...` 都会编译 `cmd/webux` 这个包，因此 `cmd/webux/dist/` 必须存在，否则直接报错：

```
pattern dist: no matching files found
```

对策：先跑 `mise run web`。**只想验证后端逻辑时用 `mise run test-go`** —— 它只编 `./internal/...`，完全绕开 embed，是日常最快的反馈回路。

### 8.3 单包 / 单测 / 竞态

```bash
go test ./internal/system/containers/...           # 只测一个包
go test -run TestNegotiateAPIVersion ./internal/system/containers/
go test -v ./internal/system/containers/           # 列出用例名
go test -race ./internal/...                       # 竞态检测（明显更慢）
```

> ⚠️ `mise.toml` 的 `[env] CGO_ENABLED = "0"` 会让 `-race` 失效（race detector 需要 CGO）。真要跑就用 `CGO_ENABLED=1 go test -race ./internal/...` 临时覆盖。

### 8.4 当前测试覆盖现状（务必知道）

`internal/**` 下目前只有 `internal/system/containers/` 带 `_test.go`。也就是说：

- `mise run test` 通过 **≠** 逻辑正确，多数包只是"能编译"。
- 改 `internal/system/*` 里的解析逻辑（`/proc` 解析、命令输出解析、磁盘/网络/服务探测）**没有回归网**，只能靠手工验证。

这些包大多属于"喂一段文本 → 断言解析结果"的纯函数，**不需要 root、也不需要真实机器**，是补测性价比最高的地方（如 `internal/system/disks`、`internal/system/processes`、`internal/system/logs`）。建议新增或修改功能时顺手补 `_test.go`。

### 8.5 提交前推荐顺序

```bash
mise run fmt          # 统一格式，减少 review 噪声
mise run test-go      # 快；抓后端编译错误与逻辑错误
mise run check-web    # 抓前端 TS / Svelte 类型错误
mise run test         # 仅当改动涉及 embed、接口签名、前后端契约时再跑完整版
mise run ci           # 完整门禁，与 GitHub Actions 逐字等价（提交前最省事的一条）
```

### 8.6 说明

- 本仓库**已有 CI**：`.github/workflows/ci.yml` 在 push / PR 时跑 `mise run ci`。因为 CI 与本地调用的**是同一条 mise 任务**，所以本地绿 = CI 绿，不存在"本地通过、CI 报错"的两套逻辑。
- `release.yml`（GoReleaser 发版）**已写好但处于待命状态**：它只接受手动触发，push tag 不会自动发版。要启用就把文件里 `push: tags:` 那段注释打开。
- gofmt 已是**阻断式**门禁（`mise run ci` 的第一步）。仓库当前全部 Go 文件都已 gofmt 干净；若新增代码没格式化，`mise run fmt` 可一次性修好。

---

## 九、调试技巧

### 9.1 后端日志

- air 终端直接输出 slog 日志，带颜色高亮（`main=黄色`、`watcher=青色`、`build=绿色`）。
- 之前用 `nohup` 挂后台跑的话，日志在指定的文件里（如 `/tmp/webux-dev-air.log`）。

### 9.2 直接打 REST API（绕过浏览器）

后端是 HTTPS + 自签证书，curl 要加 `-k`：

```bash
# 后端直连（dev 模式鉴权已关）
curl -sk https://localhost:8989/api/system/info

# 经 vite 代理（走 5173，可用来确认代理是否工作）
curl -s http://localhost:5173/api/system/info
```

> 如果 5173 代理报 `502`/`ECONNREFUSED`，多半是后端进程没起、或端口不是 8989。用 `ss -ltnp | grep 8989` 确认。

### 9.3 常见的 500 排查路径

| 现象                                            | 根因                                     | 对策                                        |
| ----------------------------------------------- | ---------------------------------------- | ------------------------------------------- |
| 编译报`pattern dist: no matching files found` | `cmd/webux/dist` 缺失                  | `mise run web`                            |
| 所有页面 500                                    | 数据目录不可写（默认`/var/lib/webux`） | dev 时`WEBUX_DATA_DIR=/tmp/webux-data`    |
| 请求 500 但后端日志无异常                       | 后端进程没在跑，vite 代理落空            | 另开终端`mise run dev-backend`            |
| curl 后端 401                                   | 非 root 读不了`/etc/shadow`            | dev 加`--no-auth`（`.air.toml` 已内置） |
| 前端报 SSL 证书错                               | 自签证书                                 | 浏览器手动信任，或 curl 加`-k`            |

### 9.4 前端调试

- Vue/React 系的 React DevTools 不适用；Svelte 5 直接在浏览器 DevTools 看组件状态即可。
- 网络面板看 `fetch` 到 `/api/*` 的状态码；WebSocket 帧在 Network → WS 面板看。
- `api.ts` 的 401 处理、`ws.ts` 的重连逻辑都可以加 `console.log` 观察。

### 9.5 前端连不上后端 / WebSocket 反复重连

`web/src/lib/ws.ts` 内建**无限重连**：`onclose` 里 `setTimeout(connect, 3000)`。

- ✅ 好处：后端重启后前端会自动恢复连接，无需刷新页面。
- ⚠️ 代价：连不上时它会**安静地每 3 秒重试一次，把后端日志刷满**，很容易被误判成「后端坏了」。

两个判据：

| 症状                                                              | 含义                                                        |
| ----------------------------------------------------------------- | ----------------------------------------------------------- |
| 后端刷 `remote error: tls: unknown certificate`，**每 3 秒一条**   | 客户端在直连 8989，且 `wss://` 握手被自签证书拒掉 → 见 7.4   |
| 浏览器控制台 `WebSocket connection to '...' failed`               | 同上；先看地址里的端口是不是 5173                           |

正确的连接（经 vite 代理）应当是一次 `101 Switching Protocols`：

```bash
curl -s -i -m 4 -H "Connection: Upgrade" -H "Upgrade: websocket" \
     -H "Sec-WebSocket-Version: 13" \
     -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
     http://localhost:5173/ws
# 期望：HTTP/1.1 101 Switching Protocols
```

> 另外：`ws.ts` 是在**模块加载时**就调用 `connect()` 的（`export const wsStore = createWSStore()` 那行的副作用），**不是**在某个组件挂载时。所以 WS 何时建连与你在哪个路由无关 —— 排查时不要拿路由里的轮询周期去解释 WS 报错的节奏（`Processes.svelte` 是 3 秒、`Dashboard.svelte` 是 5 秒，都是独立机制）。

---

## 十、构建与部署

### 10.0 先分清三种"运行"

| 场景           | 用什么                                         | 权限     | 数据目录            | 鉴权                  |
| -------------- | ---------------------------------------------- | -------- | ------------------- | --------------------- |
| 日常开发       | `mise run dev`（vite 5173 + air 8989）       | 普通用户 | `/tmp/webux-data` | 关闭（`--no-auth`） |
| 本地试用二进制 | `mise run run`                               | 普通用户 | `/tmp/webux-data` | 见 10.1               |
| 生产           | `mise run install` / deb / rpm / tar.gz 解压 | root     | `/var/lib/webux`  | 开启（shadow 或 PAM） |

### 10.1 运行：本地试用与真实鉴权

```bash
mise run run      # 等价于 mise run build + WEBUX_DATA_DIR=/tmp/webux-data ./build/webux
```

访问 **https://localhost:8989**（强制 HTTPS，自签证书 → 浏览器"高级 → 继续前往"，或 curl 加 `-k`）。

**`--no-auth` 的行为边界（最容易误解的一点）**：

```bash
WEBUX_DATA_DIR=/tmp/webux-data ./build/webux --no-auth
# 等价：WEBUX_AUTH_DISABLED=true WEBUX_DATA_DIR=/tmp/webux-data ./build/webux
```

`--no-auth` 只让 `/api/*` 与 `/ws` **跳过鉴权中间件**；而前端启动时会调用 `GET /auth/whoami`，该接口在"无 token"时**仍返回 401**，所以**前端依旧会显示登录页**。想真正跳过登录页，用下面两个办法之一：

1. 用真实系统账号登录一次，拿到会话 cookie；
2. 配置 bypass token（`auth.bypass_token` 或 `WEBUX_BYPASS_TOKEN`），请求带 `X-Webux-Token: <token>` 头即可顺带种下会话 cookie。

**真实鉴权运行**（单二进制默认 shadow 后端，需读 `/etc/shadow`）：

```bash
sudo WEBUX_DATA_DIR=/tmp/webux-data ./build/webux
```

用户名 = 机器上的 Linux 账号，密码 = 该账号的系统密码。**不加 sudo 时非 root 读不到 `/etc/shadow`，登录必然失败**（表现为 `invalid credentials`；若字段为空则前端先报 `Username and password are required`，属于前端本地校验，根本没发请求）。

查看当前使用的鉴权后端：

```bash
./build/webux --version
# 含 "auth backend: shadow (rebuild with -tags pam ...)" 或 "PAM (...)"

sudo ./build/webux --no-auth &   # 前台/后台运行皆可
ss -ltnp | grep 8989             # 确认端口
```

### 10.2 构建入口：统一的 mise 任务

**只有一条入口** —— `mise.toml` 的 `[tasks]`。原先的 `Makefile` 与 `justfile` 已删除，"开发用哪份、发布用哪份"的对照表本身也就不需要了：

| 任务                        | 作用                                                             |
| --------------------------- | ---------------------------------------------------------------- |
| `mise run web`            | 前端构建 + 同步到`cmd/webux/dist`（`nub install`，本地迭代） |
| `mise run web-ci`         | 同上，但用`nub ci`（从锁文件干净安装；发布与 CI 用）           |
| `mise run build`          | `web` + 后端编译 → `build/webux`（含 mysql/postgres 驱动）  |
| `mise run build-mysql`    | 仅 MySQL 驱动 →`build/webux-mysql`                            |
| `mise run build-postgres` | 仅 PostgreSQL 驱动 →`build/webux-postgres`                    |
| `mise run build-pam`      | PAM 鉴权 + 全驱动 →`build/webux-pam`（`CGO_ENABLED=1`）     |
| `mise run snapshot`       | 本地试跑打包（无需 tag）→`build/dist/`                        |
| `mise run release`        | 正式发布（读当前 git tag 并上传 GitHub Releases）                |
| `mise tasks ls`           | 查看全部任务（取代原先的`make help` / `just --list`）        |

> **默认构建 tag 已统一为 `mysql postgres`**：`mise run build*`、`.air.toml` 的热重载构建、以及 `.goreleaser.yaml` 的发布构建**都**带上这两个 tag。
> 不加 tag 时 `internal/system/databases/mysql_stub.go`（`//go:build !mysql`）会让 `connectMySQL` 直接返回 `ErrDriverNotCompiled` —— 只能探测实例、不能执行查询，而发布出去的包是带驱动的，**本地与线上行为会静默不一致**，所以刻意保持全量对齐。

`nub install` 与 `nub ci` 对锁文件的用法有意不同，这是**唯一需要留意的差异**：

| 命令            | 语义                                                          | 适用                           |
| --------------- | ------------------------------------------------------------- | ------------------------------ |
| `nub install` | 按`package.json` 解析并对齐锁文件                           | `mise run web`（日常开发）   |
| `nub ci`      | 从锁文件**干净安装**，不修改锁文件（等价于 `npm ci`） | `mise run web-ci`（CI/发布） |

> `web/` 已迁移到 nub：仓库里只有 `web/nub.lock`（pnpm 格式），没有 `package-lock.json`。`nub` 由 mise 提供，缺失时 mise 会在跑任务前自动补装。

其他容易看错的地方：

- 真正需要 CGO 的只有 `build-pam`（任务级 `env = { CGO_ENABLED = "1" }`）。其余全部构建（含所有发布产物）都继承 `mise.toml` 里 `[env] CGO_ENABLED = "0"` 的静态链接设置，目标机不需要 Go/Node，也不需要联网。
- **CGO 与数据库无关**：MySQL（`go-sql-driver/mysql`）、PostgreSQL（`jackc/pgx/v5`）、SQLite（`ncruces/go-sqlite3`，经 WASM）三个驱动全是纯 Go；`mysql` / `postgres` 两个 tag 决定的是"编入哪个驱动实现"，不是 CGO 开关。
- 单驱动任务 `build-mysql` / `build-postgres` 刻意保留，用于排查"只装了一种数据库"的场景；产物名带后缀，不会覆盖标准产物。
- 交叉编译、归档与打包已全部交给 **GoReleaser**（见 10.3），mise 任务里没有任何按架构或按格式展开的条目。

### 10.3 发布构建：GoReleaser

交叉编译矩阵、归档、deb / rpm / pkg.tar.zst 的生成**全部改为声明式**，配置只有一份：仓库根的 **`.goreleaser.yaml`**。mise 任务里**刻意没有**按架构或按格式的条目 —— snapshot / release 各自只调用 GoReleaser 一次，由 GoReleaser 自己遍历全部目标。

```bash
mise run snapshot     # 本地试跑：构建全部产物但不发布，不需要 git tag
mise run release      # 正式发布：读取当前 git tag 并上传到 GitHub Releases

# 等价的手工调用
goreleaser release --snapshot --clean
goreleaser check                     # 只校验 .goreleaser.yaml（mise run check-release）
```

**为什么可以整条替换掉 fpm 链路**：

| 旧链路                                                                                                                      | 现在                                        |
| --------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------- |
| 4 个手工维护的`make release-<arch>` 目标                                                                                  | `.goreleaser.yaml` → `builds.targets`  |
| `scripts/package.sh`、`scripts/build-packages.sh`、仓库根 `build-packages.sh`（**三个功能重叠的脚本，已删除**） | nfpm，配置内联在同一份文件里                |
| fpm（Ruby gem）+`rpmbuild` + `bsdtar`                                                                                   | **全部不再需要**（nfpm 是纯 Go 实现） |
| 手工`make checksums`                                                                                                      | `checksum` 段，自动生成 `checksums.txt` |
| 手工`gh release upload`                                                                                                   | `goreleaser release` 直接发布             |

构建参数与旧链路一致：`CGO_ENABLED=0`（静态链接）+ `-trimpath` + `-ldflags "-s -w -X main.version=… -X main.commit=… -X main.date=…"`，并额外把归档内文件的 mtime 固定为提交时间，使同一提交可复现。版本号：`mise run release` 取自 git tag（GoReleaser 的 `{{ .Version }}` **不含前导 `v`**）；`mise run snapshot` 用 `{{ incpatch .Version }}.{{ .ShortCommit }}`，因此快照版形如 `0.1.2.<短提交号>`（前半段来自**最近的可达 tag**：当前为 v0.1.1 → 0.1.2）。

**目标平台**（`.goreleaser.yaml` → `builds.targets`）：

| 构建目标           | deb             | rpm               | pacman           | 归档后缀  |
| ------------------ | --------------- | ----------------- | ---------------- | --------- |
| `linux_amd64_v1` | amd64           | x86_64            | x86_64           | `amd64` |
| `linux_386`      | i386            | i386              | i686             | `386`   |
| `linux_arm64`    | arm64           | aarch64           | aarch64          | `arm64` |
| `linux_arm_7`    | **armhf** | **armv7hl** | **armv7h** | `armv7` |

> 四个目标 × 5 种文件 = **20 个产物**（再加 `checksums.txt` 共 **21 个**，见 10.4）。
>
> 相比旧链路**仍然不产出 `armv6`** —— GoReleaser 没有 `linux_arm_6` 目标，旧打包脚本里也只是半途支持、从未真正产出过产物。`386` 则是**本次重新加回来的**。
>
> ⚠️ `386` 那一行的三种格式**架构名并不统一**（实测值，不是推测）：deb 与 rpm 都是 `i386`，只有 pacman 是 `i686`，而 tar.gz 归档的后缀又是 `386`。搜产物时别只 grep 一个词。rpm 用 `i386` 而非 `i686` 在 i686 机器上可以正常安装（rpm 的向后兼容规则），不影响使用。
>
> ⚠️ `linux_386` 在 GoReleaser 内部会被规范化成 **`linux_386_sse2`**（即 `GO386=sse2`，Go 自身对 386 的默认值，要求 P4 级别以上的 CPU）；中间产物目录因此叫 `build/dist/webux_linux_386_sse2/`，但**归档名与 `install.sh` 用的仍然是 `386`**。真要跑在 486/386 上得把目标改成 `linux_386_softfloat`。

**PAM 变体刻意不在 GoReleaser 范围内，也不进任何包**

PAM 版是**同一份源码 + `-tags pam`** 的另一个构建，与上面那 21 个产物是两套完全独立的交付物：

| 变体 | 构建入口                        | 产物                                         | 说明                                                                               |
| ---- | ------------------------------- | -------------------------------------------- | ---------------------------------------------------------------------------------- |
| PAM  | `mise run build-pam`          | `build/webux-pam`                          | 本机构建；`CGO_ENABLED=1`，**需本机 libpam 开发头**，产物依赖目标机 libpam |
| PAM  | `scripts/build-pam-ubuntu.sh` | `build/release/webux-pam-linux-amd64`      | 在 Ubuntu 24.04 容器内构建（glibc 基线较新）                                       |
| PAM  | `scripts/build-pam-rhel.sh`   | `build/release/webux-pam-linux-amd64-rhel` | 在 UBI9 容器内构建（RHEL 基线），`-rhel` 后缀仅用于区分                          |

**为什么它出局**：`.goreleaser.yaml` 的 `CGO_ENABLED=0` + `tags: [mysql, postgres]` 是全局设定，里面**没有 `pam`** —— `grep -i pam .goreleaser.yaml` 是**零命中**。PAM 版是 `CGO_ENABLED=1` 且动态链接**目标机的** `libpam.so`，不具备其余产物那种"静态、可任意交叉编译"的性质：放进 GoReleaser 只会让矩阵里多出一批在本机能编译、到目标机却因 libpam 版本/位置对不上而跑不起来的产物。这也顺带解决了旧链路里两个脚本自称产出 `webux-pam-linux-amd64`、而打包脚本却去 `webux-linux-<arch>` 查找、始终对不上的问题。

**由此带来的四条硬约束（最容易踩的地方）**：

1. **`mise run snapshot` / `mise run release` 的 21 个产物全部是 `CGO_ENABLED=0` 的 shadow 鉴权版**。装完 deb/rpm/pacman 后 `--version` 会打印 `auth backend: shadow (rebuild with -tags pam for full PAM support)`。想要 PAM 必须另行 `mise run build-pam` 并手工覆盖二进制。
2. **不产出任何 PAM 包** —— 没有 `webux-pam-*.deb` / `.rpm` / `.pkg.tar.zst`，`nfpms` 段里没有任何 PAM 条目。
3. **PAM 裸二进制只有 amd64** —— 两个容器脚本都硬编码 `-o build/release/webux-pam-linux-amd64`，没有 arm64 / armv7 / 386 变体。
4. **分发全手工** —— 不进 `checksums.txt`、CI 里没有 PAM 步骤、仓库也不装 `gh`；两个脚本末尾只打印一行 `gh release upload v<ver> build/release/... --clobber` 提示，得自己执行。

**`scripts/webux.pam` 目前是个孤儿文件**：仓库里**没有任何安装路径引用它** —— `scripts/pkg-postinstall.sh`、`scripts/install.sh`、`.goreleaser.yaml` 的归档与 `nfpms.contents` 全都没有它，它也不会被任何归档收进去。`internal/auth/pam.go` 里那句注释 *"we'll create /etc/pam.d/webux in the install scripts"* 从未落地。想让它生效只能自己复制：

```bash
sudo cp scripts/webux.pam /etc/pam.d/webux
```

**⚠️ PAM 查找顺序与 `login` 回退（实测，务必知道）**

`AuthenticatePAM`（`internal/auth/pam.go`）把服务名**硬编码为 `"webux"`**，一旦认证未返回 `PAM_SUCCESS`，就会**无条件**再用服务名 `"login"` 重试一次。PAM 的实际解析顺序是：

```
/etc/pam.d/webux  →  /usr/lib/pam.d/webux  →  /etc/pam.d/other  →  (代码兜底)  /etc/pam.d/login
```

两个后果：

- **`/etc/pam.d/webux` 不存在时并不会优雅降级回 shadow** —— 它会一路走到 `/etc/pam.d/login`，`/etc/pam.d` 里若两者皆无则直接 401 失败。
- **`login` 回退会静默绕过二次验证**。实测：把 `/etc/pam.d/webux` 设为 `pam_unix required` + `pam_deny required`（模拟一个不满足的 TOTP），`/etc/pam.d/login` 只保留 `pam_unix`，用**正确密码**登录返回 **HTTP 200**；再给 `login` 也加上 `pam_deny` 后返回 **HTTP 401** —— 证明 200 正是来自这个回退。也就是说只要 `/etc/pam.d/login` 比 `/etc/pam.d/webux` 宽松（webux 配了 2FA / LDAP 而 login 没配），**2FA 就被跳过了**。
- 结论：**必须**把 `scripts/webux.pam` 复制成 `/etc/pam.d/webux`。只依赖 `login` 等于放弃 webux 自己的鉴权策略。

PAM 开发依赖：

```bash
sudo apt install libpam0g-dev      # Debian / Ubuntu
sudo dnf install pam-devel         # RHEL / Fedora
sudo pacman -S pam                 # Arch
```

> 两个容器脚本需要 podman 或 docker，会在容器内自行安装 Node 24 LTS 与 nub，并完成前端构建 + 同步 `cmd/webux/dist`，**不依赖宿主机是否构建过前端**（首次运行会拉取镜像并安装依赖，较慢）。

### 10.4 打包分发：deb / rpm / pkg.tar.zst / tar.gz

**不需要任何前置工具**。旧链路要装的 `fpm`（Ruby gem）、`rpmbuild`、`bsdtar` 全部由 GoReleaser 内置的 nfpm（纯 Go）取代 —— 一条 `mise run snapshot` 即可在本机产出全部格式，任何格式生成失败都会让 `goreleaser` 直接以非 0 退出（旧链路 `make package` 缺 `rpmbuild` / `bsdtar` 时"静默少产物但仍 `EXIT=0`"的坑随之消失）。

**产物清单**（输出目录 `build/dist/`，共 **21** 个发布产物 = 4 架构 × 5 种 + 1 个校验和）：

| 类别            | 命名                                    | 数量 | 说明                                       |
| --------------- | --------------------------------------- | ---- | ------------------------------------------ |
| 安装树 tar.gz   | `webux_<ver>_linux_<arch>.tar.gz`     | 4    | **推荐主产物**：解压即得到完整目录树 |
| 纯二进制 tar.gz | `webux_<ver>_linux_<arch>_bin.tar.gz` | 4    | 归档里只有一个`webux`，无目录前缀        |
| deb             | `webux_<ver>_<arch>.deb`              | 4    | `<arch>` ∈ amd64 / i386 / arm64 / armhf |
| rpm             | `webux-<ver>-1.<arch>.rpm`            | 4    | `<arch>` ∈ x86_64 / i386 / aarch64 / armv7hl |
| pacman          | `webux-<ver>-1-<arch>.pkg.tar.zst`    | 4    | `<arch>` ∈ x86_64 / i686 / aarch64 / armv7h |
| 校验和          | `webux_<ver>_checksums.txt`           | 1    | sha256，共 20 行，覆盖上面 20 个            |

> 两类 tar.gz 的区别：**安装树**用同一份 `usr/local/bin/webux` 作为构建产物名，归档时保留完整相对路径，解压后可直接 `-C /`；**纯二进制**归档额外启用 `strip_binary_directory`，把 `usr/local/bin/webux` 压回根目录的单个 `webux` 文件。`install.sh` 依赖的是**安装树**（它会去找 `etc/webux/config.yaml` 作为模板），在线安装请下载不带 `_bin` 的那个。
>
> `build/dist/` 里除了上述 21 个发布产物，还会有 goreleaser 自己的 `metadata.json` / `artifacts.json` / `config.yaml`，以及按目标命名的中间二进制目录（`webux_linux_amd64_v1`、`webux_linux_386_sse2` 等）—— 它们不会被发布，但会出现在目录列表里。

**产物命名规则**（快照版以 `0.1.2.<短提交号>` 为例，正式版则为 `0.1.1`）：

| 格式            | 命名                                    | 快照版实际落地                                                       |
| --------------- | --------------------------------------- | -------------------------------------------------------------------- |
| deb             | `webux_<ver>_<arch>.deb`              | `webux_0.1.2.<sha>_amd64.deb`，`Version: 0.1.2.<sha>`            |
| rpm             | `webux-<ver>-1.<arch>.rpm`            | `webux-0.1.2.<sha>-1.x86_64.rpm`，`VERSION=0.1.2.<sha>`          |
| pacman          | `webux-<ver>-1-<arch>.pkg.tar.zst`    | `webux-0.1.2.<sha>-1-x86_64.pkg.tar.zst`，`pkgver=0.1.2.<sha>-1` |
| 安装树 tar.gz   | `webux_<ver>_linux_<arch>.tar.gz`     | `webux_0.1.2.<sha>_linux_amd64.tar.gz`                             |
| 纯二进制 tar.gz | `webux_<ver>_linux_<arch>_bin.tar.gz` | `webux_0.1.2.<sha>_linux_amd64_bin.tar.gz`                         |
| 校验和          | `webux_<ver>_checksums.txt`           | 覆盖上面 20 个                                                       |

> **前半段数字跟着 tag 走**：快照版形如 `0.1.2.<短提交号>`（`{{ incpatch .Version }}.{{ .ShortCommit }}`），其中 `0.1.2` 是 `incpatch(最近的可达 tag)`。当前最近的 tag 是 **v0.1.1**，所以是 0.1.2；在打 v0.1.1 之前最近的是 v0.1.0，那时是 0.1.1。
>
> **为什么刻意不带 `snapshot` 这类自造词**：发行版惯例就是"版本号 + 短提交号"，而 GoReleaser 的内置默认值 `{{ incpatch .Version }}-SNAPSHOT-<sha>` 会把大写 `SNAPSHOT` 留在每个文件名里。
>
> **分隔符为什么是 `.`**（三种都实测过，只有 `.` 三方通吃）：
>
> | 分隔符 | deb                  | rpm          | pacman               | 结论                                                                                                |
> | ------ | -------------------- | ------------ | -------------------- | --------------------------------------------------------------------------------------------------- |
> | `_`  | ❌**不可安装** | ✅           | ✅                   | dpkg 拒绝解析：`'Version' 字段值 '0.1.1_30f6222': 版本号含无效字符`。deb 是主要分发渠道，一票否决 |
> | `-`  | ⚠️ 变`~`         | ⚠️ 变`~` | ⚠️**被删掉** | pacman 得到`0.1.2<sha>`（两段数字粘连），必须靠 `g` 之类字母兜底                                |
> | `.`  | ✅                   | ✅           | ✅                   | 版本号合法字符，**文件名与元数据完全一致**，无需任何兜底前缀                                  |
>
> 也就是说历史上用过的 `0.1.1-g30f6222` / `0.1.1~g30f6222` / `0.1.1g30f6222` 这一串变形，现在被 `0.1.2.<sha>` 一个形式统一掉了 —— 四个产物名字段完全对齐，deb 文件名也正好回到 Debian 规范的 `<name>_<version>_<arch>.deb`。
>
> **正式 tag（如 `v0.1.1` → `0.1.1`）不受影响**，不含短提交号。deb 与 rpm 都要求版本号以数字开头，GoReleaser 会自动去掉 git tag 的前导 `v`。
>
> ⚠️ `.` 的唯一代价是**排序**：`0.1.2.<sha>` 在版本比较上**高于** `0.1.2`（旧的 `~` 语义是低于）。若同一 apt 仓库里同时存在快照与对应正式版，快照会被视为可升级版本。快照只用于本地演练、**不要上传到正式 apt 仓库**、也不要打进 release 资产即可规避。

**架构名映射**（与旧链路输出一致，均为实测值）：

| 构建目标           | deb             | rpm               | pacman           |
| ------------------ | --------------- | ----------------- | ---------------- |
| `linux_amd64_v1` | amd64           | x86_64            | x86_64           |
| `linux_386`      | i386            | i386              | i686             |
| `linux_arm64`    | arm64           | aarch64           | aarch64          |
| `linux_arm_7`    | **armhf** | **armv7hl** | **armv7h** |

> 注意 `linux_386` 一行三种格式**并不一致**（deb/rpm = `i386`，pacman = `i686`），而 tar.gz 归档后缀用的是 GoReleaser 的 `386`。这是实测值，不是笔误。

**包内文件与声明的依赖**：

| 格式            | 安装内容                                                                                                                                             | 依赖                                 | 配置文件保护                                 |
| --------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------ | -------------------------------------------- |
| deb             | `/usr/local/bin/webux`、`/etc/webux/config.yaml`、`/usr/lib/systemd/system/webux.service`                                                      | `libxcrypt2 \| libcrypt1 \| libc6`   | dpkg conffiles                               |
| rpm             | 同 deb 布局                                                                                                                                          | `libxcrypt`                        | `%config(noreplace)` → 新版留 `.rpmnew` |
| pacman          | 同 deb 布局                                                                                                                                          | `libxcrypt`（声明在 `.PKGINFO`） | `backup=` → 新版留 `.pacnew`            |
| 安装树 tar.gz   | `usr/local/bin/webux`、`etc/systemd/system/webux.service`、`etc/init.d/webux`、`etc/webux/config.yaml`、`usr/local/share/webux/install.sh` | 无                                   | 无（`install.sh` 检测到已存在即跳过）      |
| 纯二进制 tar.gz | 只有`webux` 一个文件                                                                                                                               | 无                                   | 不适用                                       |

> 归档里的路径**不带前导 `/`**（tar 惯例），`tar xzf … -C /` 即还原成上表的绝对路径。所有条目属主 `root:root`，权限按类型固定（二进制 0755，配置与单元文件 0644），mtime 固定为提交时间。
> deb 的 `Priority` / `Section` 与 rpm 的 `Group` 分别为 `optional` / `admin` / `Applications/System`，packager 与 maintainer 均为 `sincerely-hello-world <sincerely.hello.world@gmail.com>`（本仓库是 fork，包元数据指向自己而不是上游）。

**配置保护**：三种包格式都声明 `config.yaml` 属于"配置文件"（deb 走 dpkg conffiles、rpm 走 `%config(noreplace)`、pacman 走 `backup=` 数组）。升级行为：

- **deb** — dpkg 升级时提示（默认"保留你现在的版本"），`--force-confold` 可强制保留；
- **rpm** — 保留旧文件并把新版写成 `config.yaml.rpmnew`；
- **pacman** — 保留旧文件并把新版写成 `config.yaml.pacnew`。

验证方式（值得在发布前抽查一次）：

```bash
# deb: conffiles 应恰好是 /etc/webux/config.yaml
dpkg-deb -I  build/dist/webux_<ver>_amd64.deb conffiles

# rpm: config.yaml 的 FILEFLAGS 应为 17（'c'=config, 'n'=noreplace）
rpm -qp --qf '%{FILEFLAGS} %{FILENAMES}\n' build/dist/webux-<ver>-1.x86_64.rpm | grep config.yaml

# pacman: .PKGINFO 里应有一行 backup = etc/webux/config.yaml
bsdtar -xOf build/dist/webux-<ver>-1-x86_64.pkg.tar.zst .PKGINFO | grep backup

# 全部校验和
cd build/dist && sha256sum -c webux_<ver>_checksums.txt
```

**pacman 的 `.INSTALL` 与另外两种格式不同（值得单独记住）**：nfpm 的 archlinux 后端把钩子脚本**原样**写成 `.INSTALL` 里的同名 shell 函数：

```sh
function post_install() {
  ...脚本原文...
}
function post_remove() {
  ...脚本原文...
}
```

因此 `.INSTALL` 是 **0644**（不会被当可执行文件调用），脚本里的 `$1` 也**不是** `install` / `upgrade` / `0`。升级判断只能靠副作用 —— pacman 在 `post_remove` **之后**才删除文件，所以"`/usr/local/bin/webux` 仍然存在"就代表这次是升级。为此 `scripts/pkg-postremove-pacman.sh` 是**专供 pacman** 的版本，与 deb/rpm 共用的 `scripts/pkg-postremove.sh` 分开：

| 脚本                                 | 适用                   | `$1` 语义                                                         |
| ------------------------------------ | ---------------------- | ------------------------------------------------------------------- |
| `scripts/pkg-postremove.sh`        | deb / rpm              | `remove` / `purge` / `0`（卸载）、`upgrade` / `1`（升级） |
| `scripts/pkg-postremove-pacman.sh` | pacman（`.INSTALL`） | 无意义；靠"二进制是否还在"判断                                      |

> ⚠️ 未在真实 Arch 环境实测过 pacman 的升级 / 卸载路径（手头没有 Arch 机器），以上为静态分析 + 产物检查的结论。
> 两个脚本都**保留 `/var/lib/webux` 数据**。
>
> 📌 这一点与 `mise run uninstall` **故意不同**：发行版包的 `postremove` 走的是包管理器语义（`apt remove` 保留数据、`apt purge` 才清），而 `mise run uninstall` 是开发/调试用的"彻底还原"操作，会连同 `/var/lib/webux` 一起删干净。二者不要混用。

安装钩子 `scripts/pkg-postinstall.sh` 会：创建 `/var/lib/webux`（750）与 `/etc/webux`（750），然后 `systemctl daemon-reload`、**enable**、**restart**（无 systemd 时退回 init 脚本，并用 `rc-update` / `update-rc.d` / `chkconfig` 登记自启），最后打印 `https://<本机IP>:<端口>`。

> **装完包即为开机自启 + 立即运行**，不需要再手动动服务。想改回手动管理：`sudo systemctl disable webux`。
>
> 判断 systemd 是否在用的是 `[ -d /run/systemd/system ]`，**不是** `systemctl is-system-running`。后者在 `degraded`（即系统基本可用但有个别单元失败）时返回非 0，会让整个启动分支被静默跳过 —— 装完包服务不启动、也没有任何报错，很难排查。
>
> **升级时不会重复折腾服务**：`pkg-postremove.sh` 会先看 `$1`（rpm 的 `%postun` 在升级时也会执行，值为剩余版本数）—— 只要不是真正的卸载就直接退出。否则每次升级都会把刚启动的新服务停掉，并抹掉管理员的 `enable`。

**安装 / 升级 / 卸载**：

```bash
# Debian / Ubuntu
sudo dpkg -i  build/dist/webux_1.0.0_amd64.deb
sudo apt-get -f install            # 缺依赖时补齐
sudo dpkg -r  webux                # 卸载（保留数据与配置）
sudo dpkg -P  webux                # 连配置一起清除

# RHEL / Fedora / Rocky / Alma / openSUSE
sudo rpm -i  build/dist/webux-1.0.0-1.x86_64.rpm
sudo rpm -U  build/dist/webux-1.0.0-1.x86_64.rpm     # 升级
sudo rpm -e  webux                                       # 卸载

# Arch / Manjaro / CachyOS
sudo pacman -U build/dist/webux-1.0.0-1-x86_64.pkg.tar.zst

# 通用安装树 tar.gz（含二进制、systemd/OpenRC 单元与配置模板）
sudo tar xzf build/dist/webux_1.0.0_linux_amd64.tar.gz -C /
# 或交给安装脚本（会识别 init 系统、保留已有配置）
sudo sh build/dist/usr/local/share/webux/install.sh
```

服务与端口：

```bash
systemctl status webux
systemctl restart webux
journalctl -u webux -f
ss -ltnp | grep 8989
```

### 10.5 部署方式三选一

**方式 A — 只拷二进制（最快，适合单机/容器）**

```bash
# 二选一：mise run snapshot 后的纯二进制归档，或直接拿本机 mise run build 的产物
tar xzf build/dist/webux_<ver>_linux_amd64_bin.tar.gz -C /tmp
scp /tmp/webux root@server:/usr/local/bin/webux
ssh root@server 'chmod +x /usr/local/bin/webux && mkdir -p /var/lib/webux /etc/webux'
ssh root@server 'WEBUX_DATA_DIR=/var/lib/webux /usr/local/bin/webux'   # 前台试跑
```

**方式 B — `mise run install`（写系统目录 + systemd 单元）**

```bash
mise run install         # = mise run build（含 web → nub ci + nub run build）+ 安装二进制/配置/systemd 单元并 enable
sudo systemctl start webux
mise run uninstall       # 卸载：删二进制/单元/配置/数据（见下方表格）
```

⚠️ **不能**写成 `sudo mise run install` —— sudo 会丢掉 mise 的 PATH 与环境变量，任务里的 `go` 等工具就找不到了。任务内部对每条特权命令单独加了 `sudo`，所以**普通用户直接敲 `mise run install` 即可**（中途会问 sudo 密码）。

它需要 root 权限（写 `/usr/local/bin`、`/etc`、`systemctl`），**不会**覆盖已存在的 `/etc/webux/config.yaml`，并且把 systemd 单元装到 **`/usr/lib/systemd/system/`** —— 与三种发行包（nfpm）一致，不再是早期的 `/etc/systemd/system/`。

### 10.5.1 `mise run uninstall` 到底动了什么

| 动作 | 路径 |
| --- | --- |
| 🗑️ 停止并禁用服务 | `systemctl disable --now webux`，或 `rc-service` / `rc-update` / `update-rc.d` / `chkconfig`  |
| 🗑️ 删二进制与单元 | `/usr/local/bin/webux`、`/usr/lib/systemd/system/webux.service`、`/etc/systemd/system/webux.service`、`/etc/init.d/webux`、`/run/webux.pid`、`/usr/local/share/webux` |
| 🗑️ 删配置与数据 | `/etc/webux`、`/var/lib/webux`（含 `webux.db`、JWT 密钥、自签证书） |
| ✅ 保留（开发环境） | `build/`、`web/dist`、`cmd/webux/dist`、`.air/`、`web/node_modules`、`/tmp/webux-data`、`mise.toml` |
| ✅ 保留（跨项目缓存） | `~/.cache/nub`、Go build cache（模块缓存用 `go clean -modcache`） |

> ⚠️ **删数据前请备份**：`/var/lib/webux/webux.db` 是全部状态（用户、会话、配置历史）。任务在检测到该文件时会先打印一行提醒，但**不会**询问确认。要留档就先执行
> `sudo cp /var/lib/webux/webux.db ~/webux-backup-$(date +%F).db`。
> 删除后所有已登录会话立即失效，重装即是全新实例。
>
> 要清的是**构建产物**而不是安装结果，请用 `mise run clean` —— 两者互不重叠。

卸载是**幂等**的：重复执行不会报错，第二次会直接跳过所有已不存在的路径。

**方式 C — 发行版包（推荐规模化部署）**

见 10.4。**离线/内网注意**：二进制静态链接，目标机不需要 Go/Node、也不需要联网；运行时依赖仅 `libxcrypt`（PAM 变体另需 `libpam`）。

> **sudo 的使用场景**：生产安装（写 `/usr/local/bin` 与 `/etc`）、读取 `/etc/shadow` 做鉴权、管理其他用户的进程、用 `systemctl` 控制服务。**纯前端/后端开发不需要 sudo** —— dev 已用 `--no-auth` + 可写数据目录（`/tmp/webux-data`）绕开了权限问题。

### 10.6 在线安装脚本

`scripts/install.sh` 会从 GitHub Releases 下载对应架构的 tar.gz 并配置好服务：

```bash
curl -fsSL https://raw.githubusercontent.com/sincerely-hello-world/webux/feature/mise-just-nub-toolchain/scripts/install.sh \
  | sudo WEBUX_REPO=sincerely-hello-world/webux sh
sudo sh scripts/install.sh --version v0.1.1     # 指定版本
sudo sh scripts/install.sh --no-service         # 只装二进制，不装服务
```

> ⚠️ **fork 必须覆盖 `WEBUX_REPO`**。脚本默认去**上游** `brendan4linux/webux` 找 release 资产，即 `REPO="${WEBUX_REPO:-brendan4linux/webux}"`；本仓库作为独立 fork 发布时资产在 `sincerely-hello-world/webux` 下，不覆盖就会装到上游版本（或直接 404）。
>
> 另外 `sudo` 会清空环境变量，所以写法是 `sudo WEBUX_REPO=... sh`（或 `sudo -E sh`），而不是 `WEBUX_REPO=... sudo sh` —— 后者变量在 sudo 之前就丢了。

它会依次：要求 root → 识别架构（amd64 / 386 / arm64 / armv7）与发行版 → 识别 init 系统（systemd / openrc / sysv，并据此生成对应服务定义）→ 从 `https://github.com/${WEBUX_REPO:-brendan4linux/webux}/releases/download/<tag>/webux_<ver>_linux_<arch>.tar.gz` 下载并解压 → 安装二进制与配置（已存在的配置不覆盖）→ 启动服务。

> ⚠️ **该脚本依赖“安装树”归档（不带 `_bin` 后缀）**，因为它是从归档里的 `etc/webux/config.yaml` 取配置模板。下载 `webux_<ver>_linux_<arch>_bin.tar.gz`（纯二进制）会因找不到模板而失败。
>
> 相比旧版本：资产名与 GoReleaser 的命名精确对齐（脚本内部用 `ASSET_VERSION="${VERSION#v}"` 把 `v0.1.0` 还原成 `0.1.0`），旧写法 `webux-linux-<arch>.tar.gz` 会 404；架构识别已改为与发布产物一致的 `amd64 / 386 / arm64 / armv7`（不再接受 `armv6`）。
>
> ⚠️ `detect_arch()` 回显的是 **GoReleaser 的 `{{ .Arch }}`**（即 `386`），**不是**发行版包名里的 `i386` / `i686` —— 资产名 `webux_<ver>_linux_386.tar.gz` 用的是后者以外的那个。32 位 x86 上 `uname -m` 可能是 `i386` / `i486` / `i586` / `i686` 四种，已全部归到 `386`。
> 末尾打印的访问地址已改为按 `listen_addr` 推导 —— 旧版本硬编码的 `http://<IP>:9090` 永远不可能正确（配置是 `:8989` + HTTPS）。

### 10.7 配置文件、环境变量与命令行参数

配置文件默认 **`/etc/webux/config.yaml`**（`--config` 可改）。**实际会被解析的键**（见 `internal/config/config.go`）：

| YAML 键                | 默认值             | 说明                                                            |
| ---------------------- | ------------------ | --------------------------------------------------------------- |
| `listen_addr`        | `:8989`          | 监听地址（含冒号形式）                                          |
| `data_dir`           | `/var/lib/webux` | 数据目录（含`webux.db` 与自签证书）                           |
| `tls_cert_file`      | 空                 | 留空则自动生成自签证书并存于`data_dir`                        |
| `tls_key_file`       | 空                 | 同上                                                            |
| `log_level`          | `info`           | `debug` / `info` / `warn` / `error`                     |
| `learn_mode`         | `true`           | Learn Mode（CLI 回显）                                          |
| `auth.bypass_token`  | 空                 | 旁路令牌（`X-Webux-Token` 头），建议 `openssl rand -hex 32` |
| `auth.jwt_secret`    | 空                 | 留空则首次运行自动生成并存入 DB                                 |
| `auth.disabled`      | `false`          | 关闭鉴权（**仅开发用**）                                  |
| `auth.allowed_users` | 空                 | 非空时仅允许列出的 Unix 用户名登录                              |
| `files.root`         | `/home`          | 文件管理器可访问的根目录                                        |

环境变量覆盖（优先级高于 YAML 文件）：

| 变量                    | 作用                        |
| ----------------------- | --------------------------- |
| `WEBUX_LISTEN_ADDR`   | 覆盖`listen_addr`         |
| `WEBUX_DATA_DIR`      | 覆盖`data_dir`            |
| `WEBUX_BYPASS_TOKEN`  | 覆盖`auth.bypass_token`   |
| `WEBUX_AUTH_DISABLED` | 设为`true`/`1` 关闭鉴权 |

命令行参数：

| 参数                | 默认值                     | 说明                             |
| ------------------- | -------------------------- | -------------------------------- |
| `--config <path>` | `/etc/webux/config.yaml` | 配置文件路径                     |
| `--no-auth`       | 关闭                       | 等价于设置`auth.disabled=true` |
| `--version`       | —                         | 打印版本与鉴权后端后退出         |

> ⚠️ **YAML 解析会静默忽略未知键，改配置务必对照上表**。`Config` 结构体里**没有** `server` / `log` 段落，只有顶层的 `listen_addr` / `log_level`；键写错位置既不报错也不生效，端口会悄悄停留在默认 `:8989`。
>
> 这个坑原先存在于 `scripts/config.yaml` 与 `install.sh` 生成的配置里，现已修正（连带 `scripts/webux.service` / `scripts/webux.initd`）。新构建的包不再包含 `server:` / `log:` 段落。相关约定：
>
> - 二进制**只认 `--config`**，不读 `WEBUX_CONFIG` 环境变量。服务模板已改为在 systemd 的 `ExecStart=`、OpenRC 的 `command_args=`、SysV 的 `DAEMON_OPTS` 里显式传 `--config /etc/webux/config.yaml`，不再依赖"默认路径恰好一致"这一巧合。
> - `WEBUX_DATA_DIR` 是**会被读取**的，systemd 单元里的取值必须与配置里的 `data_dir` 保持一致（`StateDirectory=webux` 会在必要时自动创建 `/var/lib/webux`）。
> - systemd 单元使用 `Wants=`/`After=network-online.target` 而非 `network.target`（后者排序很早，不保证地址已可用），并加了 `StartLimitIntervalSec=60` / `StartLimitBurst=5` 防止重启风暴。
> - 单元**刻意不加** `ProtectSystem=` / `ProtectHome=` / `NoNewPrivileges=` 等沙箱指令：webux 需要 root 权限管理服务、软件包与用户，并要按 `files.root` 浏览 `/home`，套上沙箱会直接破坏核心功能。
> - 日志只走 journald（`StandardOutput=journal`），不再另外写文件，避免绕过日志轮转。

### 10.8 升级、回滚与数据迁移

```bash
# 包管理升级（默认不覆盖已有配置）
sudo dpkg -i webux_<new>.deb          # 或 sudo rpm -U webux-<new>.rpm
sudo systemctl restart webux

# 手工升级（先备份，便于回滚）
sudo systemctl stop webux
sudo cp /usr/local/bin/webux /usr/local/bin/webux.bak
tar xzf build/dist/webux_<ver>_linux_amd64_bin.tar.gz -C /tmp
sudo install -m 755 /tmp/webux /usr/local/bin/webux
sudo systemctl start webux

# 回滚
sudo systemctl stop webux
sudo mv /usr/local/bin/webux.bak /usr/local/bin/webux
sudo systemctl start webux
```

**数据与迁移**：数据库位于 `<data_dir>/webux.db`。JWT 签名密钥以 `auth.jwt_secret` 形式存放在库内，**跨机器/跨版本迁移时务必带上 `webux.db`**，否则所有会话失效（重新登录即可）。自签证书同样位于 `data_dir`，删除后会重新生成，浏览器需要重新信任。

> 要跑 `mise run uninstall` 做彻底还原之前，先 `sudo cp /var/lib/webux/webux.db ~/webux-backup-$(date +%F).db` —— 该任务会直接删掉 `/var/lib/webux`（见 10.5.1）。

> **一个曾经的坑（已修）**：`internal/db/db.go` 原先按 mattn/go-sqlite3 的写法传 `?_journal=WAL&_timeout=5000&_fk=true`，但本项目用的是 `github.com/ncruces/go-sqlite3` —— 它只在 DSN 以 `file:` 开头时才解析参数，且 PRAGMA 要写成 `_pragma=name(value)`。结果那串参数被当成**文件名的一部分**，磁盘上出现的是 `webux.db?_journal=WAL&_timeout=5000&_fk=true` 这个文件，而 WAL、busy timeout、`foreign_keys` **全都没生效**，且不报任何错。现已改为 `file:<path>?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)`。
>
> 因此从旧版本升上来的机器若发现"数据空了"，实际是驱动开始正常读写 `webux.db`，而旧数据在那个带 `?` 的文件里。迁移一行命令即可（服务先停）：
>
> ```bash
> sudo systemctl stop webux
> sudo mv "/var/lib/webux/webux.db?_journal=WAL&_timeout=5000&_fk=true" /var/lib/webux/webux.db
> sudo systemctl start webux
> ```

### 10.9 发布检查清单

```bash
# 0. 无前置工具（fpm / rpmbuild / bsdtar 都不再需要）
goreleaser check                       # 只校验 .goreleaser.yaml
# 1. 质量门禁（一条抵四条：gofmt 门禁 + go vet + go test + svelte-check + goreleaser check）
mise run ci
# 2. 打包（goreleaser 的 before.hooks 会自动先跑 mise run web-ci → nub ci + nub run build）
mise run snapshot
# 3. 核对产物数量（应恰好 21 个发布产物 = 4 deb + 4 rpm + 4 zst + 4 安装树 + 4 二进制 + 1 校验和）
#    注意 build/dist/ 里还有 goreleaser 自己的 metadata.json / artifacts.json / config.yaml
#    以及按目标命名的中间二进制目录，所以不要用 `ls | wc -l` 计数
ls -1 build/dist | grep -c '\.deb$'                                 # 4
ls -1 build/dist | grep -c '\.rpm$'                                 # 4
ls -1 build/dist | grep -c '\.zst$'                                 # 4
ls -1 build/dist | grep -cE '_linux_(amd64|386|arm64|armv7)\.tar\.gz$'  # 4（安装树）
ls -1 build/dist | grep -c '_bin\.tar\.gz$'                        # 4（纯二进制）
# 4. 抽查包元数据与校验和
dpkg-deb -I  build/dist/webux_<version>_amd64.deb
rpm -qip     build/dist/webux-<version>-1.x86_64.rpm
cd build/dist && sha256sum -c webux_<version>_checksums.txt
# 5. 发布（mise run release 会自动上传；手工上传时资产名需与 install.sh 期望一致，见 10.6）
mise run release
```

> ⚠️ `mise run release` 要求 **tag 在 HEAD 上**且工作区干净；否则会提示找不到 tag。本地验证一律用 `mise run snapshot`。

> **实测基线（2026-09，GoReleaser 2.18.1 / Go 1.26）**：`goreleaser release --snapshot --clean` 约 **9–14s** 完成（旧链路的打包阶段约 50s），产出上述 21 个文件；`sha256sum -c` **20/20** 通过。PAM 变体需另外单独构建与上传。
> 注意中文 locale 下 `sha256sum -c` 的输出是“OK: 20”／“成功”，不是 `: OK`，脚本里不要按英文关键字 grep。

---

## 十一、常见问题速查表

| 问题                                                         | 一句话答案                                                                                                                                                 |
| ------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `mise install` 找不到工具？                                | 确认 mise 已安装且`~/.local/share/mise` 可写；或用 `mise doctor` 诊断                                                                                  |
| 前端`ERR_MODULE_NOT_FOUND`？                               | `web/.npmrc` 的 `node-linker=hoisted` 被删了，恢复后 `nub install`                                                                                   |
| 编译报 embed dist 缺失？                                     | 先`mise run web` 生成 `cmd/webux/dist`                                                                                                                 |
| 登录永远 401？                                               | dev 用`--no-auth`；生产用 root 运行或配置 bypass token                                                                                                   |
| 登录页提示`Username and password are required`？           | 这是**前端本地校验**（字段为空，常见于浏览器自动填充未触发 input 事件）；先确认两个输入框真的填了值                                                  |
| 登录报`invalid credentials`？                              | 非 root 读不到`/etc/shadow`；用 `sudo` 运行，或用真实 Linux 账号登录                                                                                   |
| `mise run vet` / `mise run test` 报 embed dist 错误？    | 先`mise run web`；只想快跑后端逻辑用 `mise run test-go`（不碰 embed）                                                                                  |
| `mise run web` / `nub ci` 相关报错？                     | 确认已`mise install`（`nub` 由 mise 提供）；`nub ci` 会**清空并重装** `web/node_modules`（见 10.2）                                          |
| 打包报错找不到 fpm / rpmbuild / bsdtar？                     | **已不需要它们** —— 打包改由 GoReleaser + nfpm（纯 Go）完成；如有报错直接看 `goreleaser` 的非 0 退出信息（见 10.3）                              |
| `mise run release` 报“no tag”？                          | `mise run release` 需要当前 HEAD 上有 git tag；本地验证请用 `mise run snapshot`（见 10.3）                                                             |
| 在线安装 404？                                               | 资产名必须是**安装树** `webux_<ver>_linux_<arch>.tar.gz`（不带 `_bin`），见 10.6                                                                 |
| 改了 config.yaml 的`server: port:` 但端口没变？            | `Config` 没有 `server:` / `log:` 段落，未知键会被**静默忽略**；端口只能用顶层 `listen_addr:` 或 `WEBUX_LISTEN_ADDR`（模板已修正，见 10.7） |
| `mise run dev` 报 `too many open files`／vite 报 `EMFILE, errno: -24`？ | inotify 实例配额耗尽（默认仅 128），**与 ulimit 无关**；提 `fs.inotify.max_user_instances`（见 6.1） |
| dev 下改了 `scripts/config.yaml` 的端口却没反应？           | dev 用的是 `--config /dev/null`，该文件**完全不生效**；端口恒为 `defaults()` 的 `:8989`，要看效果得改 `.air.toml` 的 `entrypoint`（见 7.4） |
| 浏览器控制台刷 `WebSocket ... failed`，后端刷 `TLS handshake error`？ | 你在用 `https://<IP>:8989` 直连页面，自签证书下 WSS 必然被拒 → `ws.ts` 每 3 秒重连一次。dev 请统一用 `http://<IP>:5173`（见 7.4、9.5） |
| air 打印 `build.bin is deprecated`？                        | 旧写法残留。`.air.toml` 已改用 `entrypoint = [".air/webux", "--config", "/dev/null", "--no-auth"]` |
| 装完 deb/rpm 后服务没启动？                                  | 钩子会 enable + start（见 10.4）；若仍没起来看`sudo journalctl -u webux -n 30`，常见原因是 8989 被占或 `/etc/webux/config.yaml` 写坏了                 |
| 升级包后`/etc/webux/config.yaml` 被覆盖？                  | 已修复：三种格式都把这个文件声明为配置文件（deb conffiles / rpm`%config(noreplace)` / pacman `backup=`），升级时保留旧文件（见 10.4）                  |
| `systemctl status webux` 显示 `ExecStart` 找不到二进制？ | 单元里是`/usr/local/bin/webux`；若你手动装到别处（`mise run install` 也是这个路径），同步改 `ExecStart` 或改用 `--config` 那条命令                 |
| 5173 打开白屏？                                              | 看 vite dev server 是否启动成功（终端 B）；后端不在也能出壳，接口会 502                                                                                    |
| 端口被占？                                                   | `ss -ltnp \| grep 8989` 查占用者，或改 `listen_addr`；dev 前端的 5173 也可能被占                                                                        |
| 升级工具链？                                                 | `mise upgrade golang air nub goreleaser`                                                                                                                 |
| 如何知道当前工具版本？                                       | `mise run info`                                                                                                                                          |

---

## 附：常用命令速查

```bash
mise install               # 装齐工具链（首次）
mise run setup             # 装项目依赖（go + 前端）
mise run web               # 构建前端产物到 cmd/webux/dist（embed 必需）
mise run dev               # 一键前后端开发
mise run dev-backend       # 只跑后端（air 热重载）
mise run dev-frontend      # 只跑前端（vite）
mise run test              # 后端测试 + 前端类型检查
mise run ci                # 完整门禁（与 GitHub Actions 逐字等价）
mise run build             # 产出 build/webux
mise run run               # 构建并本地运行
mise run clean             # 清空构建产物
mise tasks ls              # 查看全部命令及说明

# ── dev 访问地址 ─────────────────────────────────────────
# 前端（日常开发用这个）  : http://localhost:5173   或  http://<本机IP>:5173
# 后端（自签证书, 会有告警）: https://localhost:8989
# 注意: 不要把 https://<本机IP>:8989 当开发入口 —— 自签证书下 WebSocket 会被拒,
#       前端会每 3 秒重连一次并刷满日志（见 7.4 / 9.5）
```

```bash
# ── 发布构建 ────────────────────────────────────────────
mise run web                                    # 前端产物（= nub install + build + 同步 dist）
mise run web-ci                                 # 同上但用 nub ci（干净安装，发布更可复现）
mise run snapshot                               # 本地试跑：全部产物 → build/dist/（21 个文件）
mise run release                                # 正式发布：读 git tag 并上传 GitHub Release
mise run build-pam                              # PAM 鉴权变体（需 libpam 开发头，不进 GoReleaser）
mise run check-release                          # 只校验 .goreleaser.yaml

# ── 打包分发（GoReleaser + nfpm，无外部依赖）───────────────────────
# 无需 fpm / rpmbuild / bsdtar —— 全部由 GoReleaser 内置的 nfpm（纯 Go）生成

# ── 安装 ────────────────────────────────────────────────
mise run install                                # 本机安装（二进制 + /etc/webux + systemd 单元）
mise run uninstall                              # 彻底卸载（连 /etc/webux 与 /var/lib/webux 一起删！先备份 webux.db）
sudo dpkg -i  build/dist/webux_1.0.0_amd64.deb
sudo rpm -U   build/dist/webux-1.0.0-1.x86_64.rpm
sudo pacman -U build/dist/webux-1.0.0-1-x86_64.pkg.tar.zst
sudo tar xzf  build/dist/webux_1.0.0_linux_amd64.tar.gz -C /

# ── 运行与运维 ──────────────────────────────────────────
./build/webux --version                          # 版本 + 鉴权后端
sudo WEBUX_DATA_DIR=/tmp/webux-data ./build/webux   # 真实鉴权试跑
systemctl status webux && journalctl -u webux -f
ss -ltnp | grep 8989
```

> 本指南是**开发向的权威文档**：任何「这条命令该怎么敲」的疑问，先 `mise tasks ls` 看全部任务，再对照第八章（测试）与第十章（构建发布）。
>
> 文档分工：[`README.md`](README.md) = 对外门面（是什么、怎么装、发布产物到底有哪些）；[`RUNNING_LOCALLY.md`](RUNNING_LOCALLY.md) = "本机跑起来"的最短路径；**本文** = 完整开发指南。
