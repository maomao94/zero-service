#!/usr/bin/env python3
"""校验仓库内 Markdown 文档的相对链接、页内锚点与代码块闭合。

只依赖 Python 标准库，CI 与本地使用同一份逻辑：

    python3 scripts/check_docs.py            # 校验全部
    python3 scripts/check_docs.py docs/      # 只校验指定目录

退出码：0 表示通过，1 表示存在文档问题。
"""

from __future__ import annotations

import os
import re
import sys
import urllib.parse

SKIP_DIRS = {".git", "node_modules", "vendor", ".worktrees", "dist"}
# 这些目录由第三方或运行时生成，不参与文档质量门禁
SKIP_PREFIXES = ("deploy/kingbase/data/", "third_party/")

FENCE_RE = re.compile(r"```.*?```", re.S)
INLINE_CODE_RE = re.compile(r"`[^`\n]*`")
LINK_RE = re.compile(r"!?\[[^\]]*\]\(([^)]+)\)")
HEADING_RE = re.compile(r"^(#{1,6})\s+(.*)$")


def iter_markdown(root: str):
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if d not in SKIP_DIRS]
        for name in filenames:
            if not name.endswith(".md"):
                continue
            path = os.path.normpath(os.path.join(dirpath, name))
            if path.startswith(SKIP_PREFIXES):
                continue
            yield path


def slug(heading: str) -> str:
    """按 GitHub 的标题锚点规则生成 slug。"""
    text = heading.strip().lower().replace("`", "")
    text = re.sub(r"[^\w\u4e00-\u9fff\- ]", "", text)
    return text.replace(" ", "-")


def headings_of(path: str) -> set[str]:
    result = set()
    with open(path, encoding="utf-8", errors="ignore") as handle:
        for line in handle:
            match = HEADING_RE.match(line)
            if match:
                result.add(slug(match.group(2)))
    return result


def main(argv: list[str]) -> int:
    roots = argv[1:] or ["."]
    files: list[str] = []
    for root in roots:
        files.extend(iter_markdown(root))
    files = sorted(set(files))

    broken_links: list[str] = []
    broken_anchors: list[str] = []
    unclosed_fences: list[str] = []
    heading_cache: dict[str, set[str]] = {}

    for path in files:
        with open(path, encoding="utf-8", errors="ignore") as handle:
            raw = handle.read()

        if raw.count("```") % 2:
            unclosed_fences.append(path)

        # 去掉围栏代码和行内代码，避免把示例里的方括号当成链接
        body = FENCE_RE.sub("", raw)
        body = INLINE_CODE_RE.sub("", body)

        for match in LINK_RE.finditer(body):
            target = match.group(1).strip()
            if target.startswith(("http://", "https://", "mailto:")):
                continue

            link_path, _, anchor = target.partition("#")
            link_path = link_path.split("?")[0]
            if link_path:
                resolved = os.path.normpath(
                    os.path.join(os.path.dirname(path), urllib.parse.unquote(link_path))
                )
                if not os.path.exists(resolved):
                    broken_links.append(f"{path}: {target}")
                    continue
            else:
                resolved = path

            if anchor and os.path.isfile(resolved):
                if resolved not in heading_cache:
                    heading_cache[resolved] = headings_of(resolved)
                if slug(urllib.parse.unquote(anchor)) not in heading_cache[resolved]:
                    broken_anchors.append(f"{path}: {target}")

    def report(title: str, items: list[str]) -> None:
        if not items:
            return
        print(f"\n{title}（{len(items)}）")
        for item in items:
            print(f"  - {item}")

    print(f"检查 {len(files)} 个 Markdown 文件")
    report("失效的相对链接", broken_links)
    report("失效的页内锚点", broken_anchors)
    report("未闭合的代码块", unclosed_fences)

    issues = len(broken_links) + len(broken_anchors) + len(unclosed_fences)
    if issues:
        print(f"\n未通过：共 {issues} 处问题")
        return 1

    print("通过：链接、锚点与代码块均正常")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
