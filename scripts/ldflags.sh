#!/bin/sh
# ldflags.sh — 打印构建时注入的 -ldflags 字符串。
#
# 由 mise tasks (build / build-mysql / build-postgres / build-pam) 共用,
# 取代原先 Makefile 与 justfile 里各自重复的那一段推导。
#
# 输出 (单行):
#   -s -w -X main.version=... -X main.commit=... -X main.date=...
#
# 版本号推导用 git describe, 与旧 Makefile 完全一致:
#   v0.1.1               恰好停在 tag 上
#   v0.1.1-3-g30f6222    距最近 tag 3 个提交
#   30f6222              仓库里还没有任何 tag 时的裸短哈希
#   ...-dirty            工作区有未提交改动
# 不在 git 仓库里 (例如从源码 tarball 构建) 时回退 0.1.0-dev。
set -eu

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# 用 printf '%s\n' 而不是 printf "<format>": 格式化字符串以 "-s" 开头,
# 部分 shell 的 printf 会把它当成选项解析。
printf '%s\n' "-s -w -X main.version=$VERSION -X main.commit=$COMMIT -X main.date=$DATE"
