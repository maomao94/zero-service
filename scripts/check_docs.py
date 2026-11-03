#!/usr/bin/env python3
"""校验仓库内 Markdown 文档的相对链接、页内锚点与代码块闭合。

只依赖 Python 标准库，CI 与本地使用同一份逻辑：

    python3 scripts/check_docs.py            # 校验全部
    python3 scripts/check_docs.py docs/      # 只校验指定目录

覆盖范围：

- Markdown 链接，含图片语法 ``![alt](path)``
- HTML ``<img src>`` 与 ``<a href>``（本仓库图表统一用 ``<img>`` 引用 PNG）
- 页内锚点，按 GitHub slug 规则生成，重复标题依次追加 ``-1``、``-2``
- 代码块闭合，```` 与 ``~~~`` 两种围栏，支持四反引号以上的嵌套
- Mermaid 源文件与导出 PNG 的配对

退出码：0 表示通过，1 表示存在文档问题。
"""

from __future__ import annotations

import os
import re
import sys
import urllib.parse

SKIP_DIRS = {".git", "node_modules", "vendor", ".worktrees", "dist", ".openchamber"}
# 这些目录由第三方或运行时生成，不参与文档质量门禁
SKIP_PREFIXES = ("deploy/kingbase/data/", "third_party/")

INLINE_CODE_RE = re.compile(r"`[^`\n]*`")
LINK_RE = re.compile(r"!?\[[^\]]*\]\(([^)]+)\)")
HTML_LINK_RE = re.compile(
    r"<(?:img|a)\b[^>]*?\b(?:src|href)\s*=\s*(?:\"([^\"]+)\"|'([^']+)')",
    re.IGNORECASE,
)
HEADING_RE = re.compile(r"^(#{1,6})\s+(.*)$")
FENCE_RE = re.compile(r"^\s*(`{3,}|~{3,})(.*)$")

SCHEME_PREFIXES = ("http://", "https://", "mailto:", "data:", "tel:")


def iter_markdown(root: str):
    root = os.path.normpath(root)
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if d not in SKIP_DIRS]
        for name in filenames:
            if not name.endswith(".md"):
                continue
            path = os.path.normpath(os.path.join(dirpath, name))
            if path.replace(os.sep, "/").startswith(SKIP_PREFIXES):
                continue
            yield path


def slug(heading: str) -> str:
    """按 GitHub 的标题锚点规则生成 slug。"""
    text = heading.strip().lower().replace("`", "")
    text = re.sub(r"[^\w\u4e00-\u9fff\- ]", "", text)
    return text.replace(" ", "-")


def strip_fences(text: str) -> tuple[str, bool]:
    """把围栏代码块的内容替换为空行，返回处理后的正文与是否存在未闭合围栏。

    按行解析而不是用正则整段匹配：这样才能正确处理 ``~~~`` 围栏，
    以及四反引号围栏里嵌套的三反引号示例。
    """
    fence: tuple[str, int] | None = None
    unclosed = False
    lines: list[str] = []

    for line in text.splitlines():
        if fence is None:
            match = FENCE_RE.match(line)
            # 反引号围栏的 info string 不允许再出现反引号，否则是行内代码
            if match and not (match.group(1)[0] == "`" and "`" in match.group(2)):
                fence = (match.group(1)[0], len(match.group(1)))
                lines.append("")
                continue
            lines.append(line)
            continue

        char, length = fence
        stripped = line.strip()
        if stripped and len(stripped) >= length and set(stripped) == {char}:
            fence = None
        lines.append("")

    if fence is not None:
        unclosed = True
    return "\n".join(lines), unclosed


def anchors_of(text: str) -> set[str]:
    """收集文档中所有可跳转的锚点。

    GitHub 对重复标题依次追加 ``-1``、``-2``，因此 ``#foo-1`` 是合法锚点。
    """
    body, _ = strip_fences(text)
    used: set[str] = set()
    for line in body.splitlines():
        match = HEADING_RE.match(line)
        if not match:
            continue
        base = slug(match.group(2))
        if not base:
            continue
        candidate = base
        suffix = 0
        while candidate in used:
            suffix += 1
            candidate = f"{base}-{suffix}"
        used.add(candidate)
    return used


def local_target(path: str, raw: str) -> tuple[str, str] | None:
    """把相对引用解析成本地路径，返回 (路径, 锚点)；外部链接返回 None。"""
    target = raw.strip()
    if not target or target.startswith(SCHEME_PREFIXES) or target.startswith("//"):
        return None
    link_path, _, anchor = target.partition("#")
    link_path = link_path.split("?")[0]
    if not link_path:
        return path, anchor
    resolved = os.path.normpath(
        os.path.join(os.path.dirname(path), urllib.parse.unquote(link_path))
    )
    return resolved, anchor


def check_markdown(files: list[str]):
    broken_links: list[str] = []
    broken_anchors: list[str] = []
    unclosed_fences: list[str] = []
    anchor_cache: dict[str, set[str]] = {}

    for path in files:
        with open(path, encoding="utf-8", errors="ignore") as handle:
            raw = handle.read()

        body, unclosed = strip_fences(raw)
        if unclosed:
            unclosed_fences.append(path)

        # 去掉行内代码，避免把示例里的方括号和标签当成真实引用
        body = INLINE_CODE_RE.sub("", body)

        targets = [match.group(1) for match in LINK_RE.finditer(body)]
        targets += [
            match.group(1) or match.group(2)
            for match in HTML_LINK_RE.finditer(body)
        ]

        for target in targets:
            parsed = local_target(path, target)
            if parsed is None:
                continue
            resolved, anchor = parsed

            if resolved != path and not os.path.exists(resolved):
                broken_links.append(f"{path}: {target}")
                continue

            if anchor and os.path.isfile(resolved):
                if resolved not in anchor_cache:
                    with open(resolved, encoding="utf-8", errors="ignore") as handle:
                        anchor_cache[resolved] = anchors_of(handle.read())
                if slug(urllib.parse.unquote(anchor)) not in anchor_cache[resolved]:
                    broken_anchors.append(f"{path}: {target}")

    return broken_links, broken_anchors, unclosed_fences


def check_diagrams(roots: list[str]) -> list[str]:
    """每个 Mermaid 源文件都要有同名导出的 PNG。"""
    missing: list[str] = []
    seen: set[str] = set()
    for root in roots:
        for dirpath, dirnames, filenames in os.walk(root):
            dirnames[:] = [d for d in dirnames if d not in SKIP_DIRS]
            for name in filenames:
                if not name.endswith(".mmd"):
                    continue
                source = os.path.normpath(os.path.join(dirpath, name))
                if source in seen or source.replace(os.sep, "/").startswith(SKIP_PREFIXES):
                    continue
                seen.add(source)
                if not os.path.exists(os.path.join(dirpath, name[:-4] + ".png")):
                    missing.append(f"{source}: 缺少同名 PNG")
    return missing


def main(argv: list[str]) -> int:
    roots = argv[1:] or ["."]
    files: list[str] = []
    for root in roots:
        files.extend(iter_markdown(root))
    files = sorted(set(files))

    broken_links, broken_anchors, unclosed_fences = check_markdown(files)
    missing_png = check_diagrams(roots)

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
    report("缺少导出 PNG 的 Mermaid 源文件", missing_png)

    issues = (
        len(broken_links) + len(broken_anchors) + len(unclosed_fences) + len(missing_png)
    )
    if issues:
        print(f"\n未通过：共 {issues} 处问题")
        return 1

    print("通过：链接、锚点、代码块与 Mermaid 配图均正常")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
