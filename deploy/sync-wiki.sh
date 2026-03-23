#!/usr/bin/env bash
#
# sync-wiki.sh - 将 .qoder/repowiki 内容同步到所有 git remote 的 Wiki 仓库
#
# 用法: ./sync-wiki.sh [remote名称|all]
#   all           - 同步到所有远程仓库 (默认)
#   <remote名称>  - 只同步到指定的远程仓库 (如 origin, allcore)

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REPOWIKI_CONTENT="$PROJECT_ROOT/.qoder/repowiki/zh/content"
WIKI_WORK_DIR="/tmp/zero-service-wiki-sync"

# 从 git remote 自动遍历所有远程仓库，拼接 .wiki.git
# 用 "name|url|branch" 格式存储，兼容 bash 3.x
WIKI_ENTRIES=()
while IFS= read -r line; do
    name="$(echo "$line" | awk '{print $1}')"
    url="$(echo "$line" | awk '{print $2}')"
    type="$(echo "$line" | awk '{print $3}')"
    [ "$type" = "(push)" ] && continue
    wiki_url="${url%.git}.wiki.git"
    # 通过 ls-remote 检测远端默认分支
    default_branch="$(git ls-remote --symref "$wiki_url" HEAD 2>/dev/null | awk '/^ref:/{sub("refs/heads/","",$2); print $2}')"
    default_branch="${default_branch:-master}"
    WIKI_ENTRIES+=("${name}|${wiki_url}|${default_branch}")
done < <(git -C "$PROJECT_ROOT" remote -v 2>/dev/null)

if [ ${#WIKI_ENTRIES[@]} -eq 0 ]; then
    echo "错误: 未找到任何 git remote"
    exit 1
fi

echo "==> 检测到 ${#WIKI_ENTRIES[@]} 个远程仓库:"
for entry in "${WIKI_ENTRIES[@]}"; do
    IFS='|' read -r name url branch <<< "$entry"
    echo "    $name -> $url ($branch)"
done

TARGET="${1:-all}"

if [ ! -d "$REPOWIKI_CONTENT" ]; then
    echo "错误: repowiki 内容目录不存在: $REPOWIKI_CONTENT"
    exit 1
fi

# 准备工作目录
rm -rf "$WIKI_WORK_DIR"
mkdir -p "$WIKI_WORK_DIR"

echo "==> 复制 repowiki 内容到工作目录..."

# 扁平化复制所有 md 文件，并动态生成 Home.md 和 _Sidebar.md
REPOWIKI_CONTENT="$REPOWIKI_CONTENT" WIKI_WORK_DIR="$WIKI_WORK_DIR" python3 << 'PYEOF'
import os, shutil

src_root = os.environ["REPOWIKI_CONTENT"]
dst_root = os.environ["WIKI_WORK_DIR"]

# 记录复制后的文件名映射，用于生成侧边栏
# key: 原始相对路径, value: wiki 文件名(不含.md)
file_map = {}
count = 0

for dirpath, dirnames, filenames in os.walk(src_root):
    dirnames.sort()
    for fname in sorted(filenames):
        if not fname.endswith('.md'):
            continue
        src_path = os.path.join(dirpath, fname)
        dst_name = fname
        dst_path = os.path.join(dst_root, dst_name)

        if os.path.exists(dst_path):
            prefix = os.path.basename(dirpath)
            dst_name = f"{prefix}---{fname}"
            dst_path = os.path.join(dst_root, dst_name)

        shutil.copy2(src_path, dst_path)
        rel_path = os.path.relpath(src_path, src_root)
        file_map[rel_path] = dst_name[:-3]  # 去掉 .md
        count += 1

print(f"    复制了 {count} 个文件")

# --- 扫描目录结构，构建树 ---
# 顶层: content 下的直属目录和 .md 文件
# 每个目录作为一个分类，目录下的 .md 作为子页面（只取一层深度用于侧边栏）

top_dirs = []
top_files = []

for item in sorted(os.listdir(src_root)):
    full = os.path.join(src_root, item)
    if os.path.isdir(full):
        top_dirs.append(item)
    elif item.endswith('.md'):
        top_files.append(item[:-3])

# --- 生成 _Sidebar.md ---
sidebar_lines = []

# 顶层散落的 md 文件
if top_files:
    for f in top_files:
        wiki_name = file_map.get(f + ".md", f)
        sidebar_lines.append(f"- [{f}]({wiki_name})")
    sidebar_lines.append("")

for d in top_dirs:
    sidebar_lines.append(f"## {d}")
    dir_path = os.path.join(src_root, d)

    # 收集该目录下所有 md (包括子目录的)，但侧边栏只列直属的
    entries = []
    for item in sorted(os.listdir(dir_path)):
        item_path = os.path.join(dir_path, item)
        if item.endswith('.md'):
            name = item[:-3]
            rel = os.path.relpath(item_path, src_root)
            wiki_name = file_map.get(rel, name)
            entries.append((name, wiki_name))
        elif os.path.isdir(item_path):
            # 子目录：找同名 md 作为入口
            sub_md = os.path.join(item_path, item + ".md")
            if os.path.exists(sub_md):
                rel = os.path.relpath(sub_md, src_root)
                wiki_name = file_map.get(rel, item)
                entries.append((item, wiki_name))
            else:
                # 没有同名 md，列子目录名
                entries.append((item, item))

    for name, wiki_name in entries:
        sidebar_lines.append(f"- [{name}]({wiki_name})")
    sidebar_lines.append("")

with open(os.path.join(dst_root, "_Sidebar.md"), "w") as f:
    f.write("\n".join(sidebar_lines))

print("    生成 _Sidebar.md")

# --- 生成 Home.md ---
# 从项目概述目录的主 md 中提取第一段作为简介，若没有则用通用描述
intro = ""
overview_md = os.path.join(src_root, "项目概述", "项目概述.md")
if os.path.exists(overview_md):
    with open(overview_md) as f:
        lines = f.readlines()
    # 跳过标题和 cite 块，取第一个正文段落
    in_cite = False
    for line in lines:
        stripped = line.strip()
        if stripped.startswith("<cite"):
            in_cite = True
            continue
        if stripped.startswith("</cite"):
            in_cite = False
            continue
        if in_cite:
            continue
        if stripped.startswith("#"):
            continue
        if not stripped:
            if intro:
                break
            continue
        intro += stripped + " "

intro = intro.strip()
if not intro:
    intro = "项目 Wiki 文档，由 repowiki 自动生成。"

home_lines = ["# Wiki", "", intro, "", "## 快速导航", ""]
for d in top_dirs:
    wiki_name = file_map.get(os.path.join(d, d + ".md"), d)
    home_lines.append(f"- [{d}]({wiki_name})")

with open(os.path.join(dst_root, "Home.md"), "w") as f:
    f.write("\n".join(home_lines) + "\n")

print("    生成 Home.md")
PYEOF

echo "==> 文件准备完成"

# 推送函数
push_wiki() {
    local name="$1"
    local url="$2"
    local branch="$3"

    echo "==> 推送到 $name ($branch 分支)..."

    local repo_dir="/tmp/wiki-push-$name"
    rm -rf "$repo_dir"

    # 克隆或初始化
    if git ls-remote "$url" &>/dev/null; then
        git clone --depth 1 "$url" "$repo_dir" 2>/dev/null || {
            mkdir -p "$repo_dir"
            cd "$repo_dir"
            git init
            git remote add origin "$url"
        }
    else
        mkdir -p "$repo_dir"
        cd "$repo_dir"
        git init
        git remote add origin "$url"
    fi

    cd "$repo_dir"

    # 清除旧内容，复制新内容
    find . -maxdepth 1 -name '*.md' -delete
    cp "$WIKI_WORK_DIR"/*.md .

    git add -A
    if git diff --cached --quiet; then
        echo "    $name: 无变更，跳过"
        return
    fi

    git commit -m "docs: sync wiki from repowiki $(date +%Y-%m-%d)"

    # 推送
    if git push origin HEAD:"$branch" 2>&1; then
        echo "    $name: 推送成功"
    else
        echo "    $name: 推送失败，尝试 force push..."
        git push -f origin HEAD:"$branch" 2>&1 && echo "    $name: force push 成功" || echo "    $name: 推送失败"
    fi
}

# 根据目标执行推送
found=0
for entry in "${WIKI_ENTRIES[@]}"; do
    IFS='|' read -r name url branch <<< "$entry"
    if [ "$TARGET" = "all" ] || [ "$TARGET" = "$name" ]; then
        push_wiki "$name" "$url" "$branch"
        found=1
    fi
done

if [ "$found" -eq 0 ]; then
    echo "错误: 未找到远程仓库 '$TARGET'"
    echo "可用的远程仓库:"
    for entry in "${WIKI_ENTRIES[@]}"; do
        IFS='|' read -r name _ _ <<< "$entry"
        echo "    $name"
    done
    echo "用法: $0 [remote名称|all]"
    exit 1
fi

echo "==> 同步完成!"
