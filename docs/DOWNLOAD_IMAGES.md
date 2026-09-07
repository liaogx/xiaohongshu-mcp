# 下载搜索结果图片

`scripts/download_xhs_images.py` 是本仓库的只读辅助工具，使用本地 MCP 的 HTTP 搜索和详情接口，再下载图片。它不是新增的 MCP 工具，不会评论、回复、点赞、收藏或发布。

先启动服务并确认 MCP 已登录，安装 Python 3.10 或更新版本，然后在仓库根目录运行：

```bash
# 每篇笔记下载一张主图，最多 20 篇；示例关键词可自行更换。
python3 scripts/download_xhs_images.py '咖啡' --count 20

# 获取详情，下载每篇笔记的全部可读取图片。
python3 scripts/download_xhs_images.py '咖啡' --count 20 --all-images
```

默认输出到用户下载目录 `Downloads/xiaohongshu-mcp/<关键词>/`，位于源码仓库之外。可用 `--output /absolute/private/output` 指定其他私有目录，`--timeout` 设置单次请求超时秒数。

默认连接 `http://127.0.0.1:18060`。如服务启用了鉴权，从本地环境设置同一个 `AUTH_TOKEN`；脚本只把此请求头发给指定 MCP API，不发给图片 CDN，也不跟随 API 重定向。不要把真实令牌放进命令参数、源码或公开文档。

输出包括图片、`index.html` 本地浏览索引及 `manifest.json` 下载清单。**这些都是私有运行产物，不得提交 Git 或公开上传。**清单可能包含笔记标题、作者和 ID，不保存 `xsec_token`。

搜索结果不足时按实际数量处理；图片链接过期或详情无法读取时会报告失败，不伪造成功。下载不意味着获得再分发许可，请仅保存有权访问和使用的内容。此脚本不会绕过验证码、登录限制或内容权限。
