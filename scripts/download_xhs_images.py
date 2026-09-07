#!/usr/bin/env python3
"""Download images from a Xiaohongshu MCP search result (Apache-2.0).

This helper uses the local project's read-only HTTP API.  It never calls
comment, reply, like, favorite, publish, or follow operations.
"""

from __future__ import annotations

import argparse
import html
import json
import mimetypes
import os
import re
import sys
import time
from pathlib import Path
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode, urlparse
from urllib.request import HTTPRedirectHandler, Request, build_opener, urlopen


DEFAULT_BASE_URL = "http://127.0.0.1:18060"
MAX_IMAGE_BYTES = 30 * 1024 * 1024
USER_AGENT = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
    "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0 Safari/537.36"
)


class NoAPIRedirect(HTTPRedirectHandler):
    """Do not forward the local API authorization header to a redirect target."""

    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise HTTPError(req.full_url, code, "API redirect refused", headers, fp)


def safe_error(exc: Exception) -> str:
    # Exceptions can contain signed URLs or private paths; report type/status only.
    return f"HTTP {exc.code}" if isinstance(exc, HTTPError) else type(exc).__name__


def json_request(
    url: str,
    *,
    method: str = "GET",
    payload: dict[str, Any] | None = None,
    timeout: int,
) -> dict[str, Any]:
    data = None
    headers = {"Accept": "application/json", "User-Agent": USER_AGENT}
    if os.environ.get("AUTH_TOKEN"):
        headers["Authorization"] = "Bearer " + os.environ["AUTH_TOKEN"]
    if payload is not None:
        data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        headers["Content-Type"] = "application/json"
    request = Request(url, data=data, headers=headers, method=method)
    with build_opener(NoAPIRedirect()).open(request, timeout=timeout) as response:
        body = response.read().decode("utf-8")
    result = json.loads(body)
    if result.get("success") is not True:
        raise RuntimeError("API request failed; check service/login state")
    return result


def clean_name(value: str, fallback: str) -> str:
    value = re.sub(r"[\\/:*?\"<>|\r\n\t]+", " ", value or "")
    value = re.sub(r"\s+", " ", value).strip(" .")
    return (value or fallback)[:80]


def unique_urls(urls: list[str | None]) -> list[str]:
    result: list[str] = []
    for url in urls:
        if url and url not in result:
            result.append(url)
    return result


def cover_urls(feed: dict[str, Any]) -> list[str]:
    cover = feed.get("noteCard", {}).get("cover", {}) or {}
    urls = [cover.get("urlDefault"), cover.get("urlPre"), cover.get("url")]
    for item in cover.get("infoList") or []:
        if isinstance(item, dict):
            urls.append(item.get("url"))
    return unique_urls(urls)


def detail_image_urls(detail: dict[str, Any]) -> list[str]:
    note = detail.get("data", {}).get("data", {}).get("note", {}) or {}
    return unique_urls(
        [
            item.get("urlDefault") or item.get("urlPre")
            for item in note.get("imageList") or []
            if isinstance(item, dict)
        ]
    )


def extension_for(url: str, content_type: str) -> str:
    content_type = content_type.split(";", 1)[0].lower().strip()
    ext = mimetypes.guess_extension(content_type) if content_type else None
    if ext:
        return ext
    path = urlparse(url).path.lower()
    for candidate in (".jpg", ".jpeg", ".png", ".webp", ".gif"):
        if candidate in path:
            return candidate
    return ".img"


def download_image(url: str, destination: Path, timeout: int) -> tuple[bool, str]:
    headers = {
        "Accept": "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8",
        "Referer": "https://www.xiaohongshu.com/",
        "User-Agent": USER_AGENT,
    }
    request = Request(url, headers=headers)
    temporary = destination.with_suffix(destination.suffix + ".part")
    try:
        with urlopen(request, timeout=timeout) as response:
            content_type = response.headers.get("Content-Type", "")
            data = response.read(MAX_IMAGE_BYTES + 1)
        if len(data) > MAX_IMAGE_BYTES:
            return False, "图片超过 30MB 限制"
        if not data:
            return False, "空响应"
        destination = destination.with_suffix(extension_for(url, content_type))
        temporary = destination.with_suffix(destination.suffix + ".part")
        temporary.write_bytes(data)
        temporary.replace(destination)
        return True, str(destination)
    except (HTTPError, URLError, TimeoutError, OSError) as exc:
        try:
            temporary.unlink(missing_ok=True)
        except OSError:
            pass
        return False, safe_error(exc)


def fetch_detail(base_url: str, feed: dict[str, Any], timeout: int) -> dict[str, Any]:
    payload = {
        "feed_id": feed.get("id", ""),
        "xsec_token": feed.get("xsecToken", ""),
        "load_all_comments": False,
    }
    return json_request(
        f"{base_url.rstrip('/')}/api/v1/feeds/detail",
        method="POST",
        payload=payload,
        timeout=timeout,
    )


def write_index(output_dir: Path, records: list[dict[str, Any]]) -> None:
    cards = []
    for record in records:
        title = html.escape(record.get("title") or "未命名笔记")
        author = html.escape(record.get("author") or "未知作者")
        files = record.get("files") or []
        images = "".join(
            f'<img src="{html.escape(str(Path(file).name))}" alt="{title}">'
            for file in files
        )
        status = "下载成功" if files else html.escape(record.get("error") or "下载失败")
        cards.append(
            "<article>"
            f"<h2>{record.get('rank', '')}. {title}</h2>"
            f"<p class=author>{author}</p>"
            f"<div class=images>{images or '<p>暂无图片</p>'}</div>"
            f"<p class=status>{html.escape(status)}</p>"
            "</article>"
        )
    document = """<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8">
<title>小红书图片下载结果</title>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,"PingFang SC",sans-serif;margin:32px;background:#f7f7f7;color:#222}
main{max-width:1100px;margin:auto} article{background:#fff;border-radius:14px;padding:18px;margin:18px 0;box-shadow:0 2px 12px #00000010}
h2{font-size:20px;margin:0 0 4px}.author,.status{color:#666}.images{display:flex;flex-wrap:wrap;gap:12px}.images img{max-width:280px;max-height:360px;object-fit:contain;border-radius:8px;background:#eee}
</style></head><body><main><h1>小红书图片下载结果</h1>
""" + "\n".join(cards) + "\n</main></body></html>"
    (output_dir / "index.html").write_text(document, encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser(description="下载小红书搜索结果的主图")
    parser.add_argument("keyword", help="搜索关键词")
    parser.add_argument("--count", type=int, default=20, help="下载数量，默认 20")
    parser.add_argument("--output", type=Path, help="私有输出目录，默认用户 Downloads/xiaohongshu-mcp/<关键词>")
    parser.add_argument("--base-url", default=DEFAULT_BASE_URL, help="本地 MCP API 地址")
    parser.add_argument("--timeout", type=int, default=90, help="单次请求超时秒数")
    parser.add_argument(
        "--all-images",
        action="store_true",
        help="逐条读取详情并下载每篇笔记的全部图片，而不是每篇只下载主图",
    )
    args = parser.parse_args()
    if args.count < 1:
        parser.error("--count 必须大于 0")

    parsed_base = urlparse(args.base_url)
    if parsed_base.scheme not in ("http", "https") or not parsed_base.hostname or parsed_base.username or parsed_base.password:
        parser.error("--base-url 必须是没有嵌入账号密码的 HTTP(S) MCP 服务地址")
    if args.timeout <= 0:
        parser.error("--timeout 必须大于 0")
    output_dir = args.output or Path.home() / "Downloads" / "xiaohongshu-mcp" / clean_name(args.keyword, "search")
    output_dir.mkdir(parents=True, exist_ok=True)

    search_url = f"{args.base_url.rstrip('/')}/api/v1/feeds/search?{urlencode({'keyword': args.keyword})}"
    try:
        response = json_request(search_url, timeout=args.timeout)
    except Exception as exc:  # noqa: BLE001 - CLI should show a concise actionable error.
        print(f"搜索失败（{safe_error(exc)}），请检查 MCP 登录和服务状态。", file=sys.stderr)
        return 1

    feeds = (response.get("data") or {}).get("feeds") or []
    feeds = feeds[: args.count]
    if not feeds:
        print("没有找到搜索结果。")
        return 0

    records: list[dict[str, Any]] = []
    success_count = 0
    for rank, feed in enumerate(feeds, start=1):
        note_card = feed.get("noteCard") or {}
        title = note_card.get("displayTitle") or "视频笔记"
        author = (note_card.get("user") or {}).get("nickname") or "未知作者"
        stem = f"{rank:02d}_{clean_name(title, 'untitled')}_{clean_name(str(feed.get('id') or ''), 'unknown')}"
        urls = detail_image_urls({}) if args.all_images else cover_urls(feed)
        detail_error = ""

        if args.all_images or not urls:
            try:
                detail = fetch_detail(args.base_url, feed, args.timeout)
                urls = detail_image_urls(detail)
            except Exception as exc:  # noqa: BLE001 - keep other results downloading.
                detail_error = f"详情读取失败：{safe_error(exc)}"

        files: list[str] = []
        download_errors: list[str] = []
        for image_index, url in enumerate(urls if args.all_images else urls[:1], start=1):
            target = output_dir / f"{stem}_{image_index:02d}"
            ok, result = download_image(url, target, args.timeout)
            if ok:
                files.append(str(Path(result).relative_to(output_dir)))
            else:
                download_errors.append(result)
                if not args.all_images:
                    # If the search cover is blocked/expired, fetch a fresh detail image.
                    try:
                        detail = fetch_detail(args.base_url, feed, args.timeout)
                        fresh_urls = detail_image_urls(detail)
                        for fresh_url in fresh_urls:
                            ok, result = download_image(fresh_url, target, args.timeout)
                            if ok:
                                files.append(str(Path(result).relative_to(output_dir)))
                                break
                            download_errors.append(result)
                    except Exception as exc:  # noqa: BLE001
                        detail_error = f"详情读取失败：{safe_error(exc)}"
                break

        if files:
            success_count += 1
        error = detail_error or (download_errors[-1] if not files and download_errors else "")
        record = {
            "rank": rank,
            "id": feed.get("id"),
            "title": title,
            "author": author,
            "files": files,
            "error": error,
        }
        records.append(record)
        print(f"[{rank:02d}/{len(feeds):02d}] {'完成' if files else '失败'} {title}")
        time.sleep(0.15)

    (output_dir / "manifest.json").write_text(
        json.dumps(
            {
                "keyword": args.keyword,
                "count": len(records),
                "downloaded_posts": success_count,
                "records": records,
            },
            ensure_ascii=False,
            indent=2,
        ),
        encoding="utf-8",
    )
    write_index(output_dir, records)
    print(f"\n已下载 {success_count}/{len(records)} 条帖子的图片。")
    print(f"图片目录：{output_dir}")
    print(f"查看索引：{output_dir / 'index.html'}")
    return 0 if success_count == len(records) else 2


if __name__ == "__main__":
    raise SystemExit(main())
