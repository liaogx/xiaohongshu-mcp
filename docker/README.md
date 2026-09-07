# Docker 部署

本指南针对修改后的版本（见 [NOTICE](../NOTICE)）。从当前源码构建，避免误用不包含本仓库修复的上游镜像。

## 启动和更新

在本仓库根目录执行：

```bash
docker compose -f docker/docker-compose.yml up -d --build
docker compose -f docker/docker-compose.yml logs -f
```

代码更新后重新执行带 `--build` 的启动命令。停止服务：

```bash
docker compose -f docker/docker-compose.yml stop
```

当前镜像使用 Linux x64 浏览器；Apple Silicon Docker 环境需要 amd64 模拟支持。构建会下载 Go 依赖、系统依赖和浏览器，并校验浏览器 SHA256。首次构建需要联网。

## 数据与端口

- `docker/data/` 持久化登录文件与运行数据。
- `docker/images/` 映射为容器中的 `/app/images`；发布本地素材时传容器路径，不是宿主机路径。
- 默认端口仅绑定到本机 `127.0.0.1:18060`，不要直接暴露到公网。
- 上述目录含私人数据，不提交 Git，也不要上传为附件。

## 登录与验证

1. 运行 `npx @modelcontextprotocol/inspector`，打开其输出的本地页面。
2. 选择 `Streamable HTTP`，连接 `http://127.0.0.1:18060/mcp`。
3. 先调用 `check_login_status`；未登录时调用 `get_login_qrcode`，用小红书 App 扫码。
4. 重新检查登录。网页和手机的独立登录状态不能代替 MCP 自己的会话。
5. 如果服务要求额外安全验证，暂停操作并阅读[恢复说明](../docs/RECOVERY.md)；不要反复重试写操作。

根路径 `/` 不是操作后台，显示 404 并不代表 MCP 故障。`/health` 用于健康检查，`/mcp` 用于 MCP 客户端。

## 访问鉴权与代理

Compose 从宿主环境读取 `AUTH_TOKEN` 与 `XHS_PROXY`，请在本机安全注入，不要把真实值写入源码或示例。启用鉴权后，客户端需要 `Authorization: Bearer <YOUR_TOKEN>`。代理支持情况以服务实现为准；含认证信息的代理地址也是凭证。

不要把运行日志、二维码、cookies 文件或完整请求响应粘贴到公开 Issue。
