# xiaohongshu-mcp

中文 | [English](README_EN.md) · [Apache-2.0](LICENSE)

MCP for 小红书 / xiaohongshu.com。让 AI 助手搜索笔记、获取推荐和详情、查看公开主页，并执行用户授权的发布和互动操作。

本仓库基于 [xpzouying/xiaohongshu-mcp](https://github.com/xpzouying/xiaohongshu-mcp) 修改，是独立维护的衍生版本。Git 历史重新初始化，但保留上游许可证与来源声明，详见 [NOTICE](NOTICE)。本项目与小红书官方无隶属关系。

### 本版本的主要修改

- 改进登录失效与安全验证页面识别，避免将验证问题误判为普通搜索超时。
- 搜索筛选按分组和实际选项定位，并核验选中状态。
- 登录文件使用私有权限原子保存，增加人工扫码恢复工具。
- 补充离线回归测试与运行数据排除规则。
- 增加通用的[图片下载脚本与本地浏览索引](docs/DOWNLOAD_IMAGES.md)。

以下功能演示保留的是**上游公开示例链接**，不同版本的界面可能有差异。

## 项目简介

**主要功能**

> 💡 **提示：** 点击下方功能标题可展开查看视频演示

<details>
<summary><b>1. 登录和检查登录状态</b></summary>

第一步必须，小红书需要进行登录。可以检查当前登录状态。

**登录演示：**

https://github.com/user-attachments/assets/8b05eb42-d437-41b7-9235-e2143f19e8b7

**检查登录状态演示：**

https://github.com/user-attachments/assets/bd9a9a4a-58cb-4421-b8f3-015f703ce1f9

</details>

<details>
<summary><b>2. 发布图文内容</b></summary>

支持发布图文内容到小红书，包括标题、内容描述和图片。

**图片支持方式：**

支持两种图片输入方式：

1. **HTTP/HTTPS 图片链接**

   ```
   ["https://example.com/image1.jpg", "https://example.com/image2.png"]
   ```

2. **本地图片绝对路径**（推荐）
   ```
   ["/Users/username/Pictures/image1.jpg", "/home/user/images/image2.png"]
   ```

**为什么推荐使用本地路径：**

- ✅ 稳定性更好，不依赖网络
- ✅ 上传速度更快
- ✅ 避免图片链接失效问题
- ✅ 支持更多图片格式

**发布图文帖子演示：**

https://github.com/user-attachments/assets/8aee0814-eb96-40af-b871-e66e6bbb6b06

</details>

<details>
<summary><b>3. 发布视频内容</b></summary>

支持发布视频内容到小红书，包括标题、内容描述和本地视频文件。

**视频支持方式：**

仅支持本地视频文件绝对路径：

```
"/Users/username/Videos/video.mp4"
```

**功能特点：**

- ✅ 支持本地视频文件上传
- ✅ 自动处理视频格式转换
- ✅ 支持标题、内容描述和标签
- ✅ 等待视频处理完成后自动发布

**注意事项：**

- 仅支持本地视频文件，不支持 HTTP 链接
- 视频处理时间较长，请耐心等待
- 建议视频文件大小不超过 1GB

</details>

<details>
<summary><b>4. 搜索内容</b></summary>

根据关键词搜索小红书内容。

**搜索帖子演示：**

https://github.com/user-attachments/assets/03c5077d-6160-4b18-b629-2e40933a1fd3

</details>

<details>
<summary><b>5. 获取推荐列表</b></summary>

获取小红书首页推荐内容列表。

**获取推荐列表演示：**

https://github.com/user-attachments/assets/110fc15d-46f2-4cca-bdad-9de5b5b8cc28

</details>

<details>
<summary><b>6. 获取帖子详情（包括互动数据和评论）</b></summary>

获取小红书帖子的完整详情，包括：

- 帖子内容（标题、描述、图片等）
- 用户信息
- 互动数据（点赞、收藏、分享、评论数）
- 评论列表及子评论

**⚠️ 重要提示：**

- 需要提供帖子 ID 和 xsec_token（两个参数缺一不可）
- 这两个参数可以从 Feed 列表或搜索结果中获取
- 必须先登录才能使用此功能

**获取帖子详情演示：**

https://github.com/user-attachments/assets/76a26130-a216-4371-a6b3-937b8fda092a

</details>

<details>
<summary><b>7. 发表评论到帖子</b></summary>

支持自动发表评论到小红书帖子。

**功能说明：**

- 自动定位评论输入框
- 输入评论内容并发布
- 支持 HTTP API 和 MCP 工具调用

**⚠️ 重要提示：**

- 需要先登录才能使用此功能
- 需要提供帖子 ID、xsec_token 和评论内容
- 这些参数可以从 Feed 列表或搜索结果中获取

**发表评论演示：**

https://github.com/user-attachments/assets/cc385b6c-422c-489b-a5fc-63e92c695b80

</details>

<details>
<summary><b>8. 获取用户个人主页</b></summary>

获取小红书用户的个人主页信息，包括用户基本信息和笔记内容。

**功能说明：**

- 获取用户基本信息（昵称、简介、头像等）
- 获取关注数、粉丝数、获赞量统计
- 获取用户发布的笔记内容列表
- 支持 HTTP API 和 MCP 工具调用

**⚠️ 重要提示：**

- 需要先登录才能使用此功能
- 需要提供用户 ID 和 xsec_token
- 这些参数可以从 Feed 列表或搜索结果中获取

**返回信息包括：**

- 用户基本信息：昵称、简介、头像、认证状态
- 统计数据：关注数、粉丝数、获赞量、笔记数
- 笔记列表：用户发布的所有公开笔记

</details>

<details>
<summary><b>9. 回复评论</b></summary>

回复笔记下的指定评论，支持精准回复特定用户的评论。

**功能说明：**

- 回复指定笔记下的特定评论
- 支持通过评论 ID 或用户 ID 定位目标评论
- 需要提供 feed_id、xsec_token、comment_id/user_id 和回复内容

**⚠️ 重要提示：**

- 需要先登录才能使用此功能
- comment_id 和 user_id 至少提供一个
- 这些参数可以从帖子详情的评论列表中获取

</details>

<details>
<summary><b>10. 点赞/取消点赞</b></summary>

为笔记点赞或取消点赞，智能检测当前状态避免重复操作。

**功能说明：**

- 为指定笔记点赞或取消点赞
- 智能检测：已点赞时跳过点赞，未点赞时跳过取消点赞
- 需要提供 feed_id 和 xsec_token

**⚠️ 重要提示：**

- 需要先登录才能使用此功能
- 默认为点赞操作，设置 unlike=true 可取消点赞

</details>

<details>
<summary><b>11. 收藏/取消收藏</b></summary>

收藏笔记或取消收藏，智能检测当前状态避免重复操作。

**功能说明：**

- 收藏指定笔记或取消收藏
- 智能检测：已收藏时跳过收藏，未收藏时跳过取消收藏
- 需要提供 feed_id 和 xsec_token

**⚠️ 重要提示：**

- 需要先登录才能使用此功能
- 默认为收藏操作，设置 unfavorite=true 可取消收藏

</details>

**使用与隐私说明**

登录会话、二维码、下载素材、日志和实际互动回执都是私有运行数据，不属于项目源码，请保存在仓库外。登录状态和安全验证可能分别失效；遇到验证要求时应暂停操作。详见[登录与安全验证恢复](docs/RECOVERY.md)。

## 1. 使用教程

### 1.1. 从本仓库源码编译（推荐）

安装 [Go](https://go.dev/doc/install) 1.24 或更新版本以及 Git。要使用本仓库的修改，请编译**本仓库源码**；上游发布的二进制和 Docker 镜像不会自动包含这些修复。

```bash
git clone https://github.com/liaogx/xiaohongshu-mcp.git
cd xiaohongshu-mcp
go mod download
go build -o bin/xiaohongshu-mcp .
go build -o bin/xiaohongshu-login ./cmd/login
go build -o bin/xiaohongshu-recover ./cmd/recover
```

内置浏览器支持 macOS Apple Silicon、Windows x64、Linux x64。Windows 编译时请为输出文件增加 `.exe` 后缀；macOS Intel、Linux ARM64 暂无对应内置浏览器。首次启动会下载并校验浏览器，后续复用缓存。

国内网络如有需要，可为编译命令设置 `GOPROXY=https://goproxy.cn,direct`。

**Docker 部署：**在本仓库根目录执行 `docker compose -f docker/docker-compose.yml up -d --build`，从当前源码构建。详见 [Docker 指南](docker/README.md)和 [Windows 指南](docs/windows_guide.md)。

### 1.2. 登录

第一次需要手动登录，需要保存小红书的登录状态。

建议先在仓库外创建私有运行目录，并为登录工具、服务、恢复工具统一设置 `COOKIES_PATH`。不要把登录文件和日志提交 Git。下面的命令应在同一配置环境中运行，详见[恢复与数据隔离说明](docs/RECOVERY.md)。

**使用二进制文件**：

```bash
# 运行对应平台的登录工具
./bin/xiaohongshu-login
```

**使用源码**：

```bash
go run ./cmd/login
```

### 1.3. 启动 MCP 服务

启动 xiaohongshu-mcp 服务。

本地使用建议追加 `-port=127.0.0.1:18060`，仅允许本机访问；程序原有默认 `:18060` 会监听全部网卡。不要把未鉴权的服务暴露到公网。

**使用二进制文件**：

```bash
# 默认：无头模式，没有浏览器界面
./bin/xiaohongshu-mcp

# 非无头模式，有浏览器界面
./bin/xiaohongshu-mcp -headless=false
```

**使用源码**：

```bash
# 默认：无头模式，没有浏览器界面
go run .

# 非无头模式，有浏览器界面
go run . -headless=false
```

**配置代理（可选）**：

如果需要通过代理访问，可以设置 `XHS_PROXY` 环境变量：

```bash
# 设置代理后启动
XHS_PROXY=http://proxy.example:8080 ./bin/xiaohongshu-mcp

# 或使用源码
XHS_PROXY=http://proxy.example:8080 go run .
```

支持 HTTP/HTTPS/SOCKS5 代理，日志中会自动隐藏代理的认证信息。

**访问鉴权（可选）**：

默认关闭鉴权。生产环境建议使用 `AUTH_TOKEN` 环境变量配置；非空的启动参数优先于环境变量，留空则读取 `AUTH_TOKEN`。

```bash
# 环境变量
AUTH_TOKEN=your-secret-token ./bin/xiaohongshu-mcp
AUTH_TOKEN=your-secret-token go run .

# 非空启动参数（优先于环境变量）
./bin/xiaohongshu-mcp -token=your-secret-token
go run . -token=your-secret-token
```

启用鉴权后，所有 MCP 客户端都必须配置自定义请求头 `Authorization: Bearer <token>`。命令行参数可能被进程列表看到，部署环境优先使用 `AUTH_TOKEN`。

例如，支持自定义请求头的 MCP 客户端可使用以下配置：

```json
{
  "mcpServers": {
    "xiaohongshu-mcp": {
      "url": "http://localhost:18060/mcp",
      "headers": { "Authorization": "Bearer your-secret-token" }
    }
  }
}
```

### 1.4. 验证 MCP

```bash
npx @modelcontextprotocol/inspector
```

[运行 Inspector — 上游演示](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/run_inspect.png)

运行后，打开红色标记的链接，配置 MCP inspector，输入 `http://localhost:18060/mcp` ，点击 `Connect` 按钮。

<img width="915" height="659" alt="bf9532dd0b7ba423491accf511a467de" src="https://github.com/user-attachments/assets/08bc3cef-73e7-42d2-b923-7ba9e6c8af30" />

**注意：** 左侧边框中的选项是否正确。

按照上面配置 MCP inspector 后，点击 `List Tools` 按钮，查看所有的 Tools。

`http://127.0.0.1:18060/` 是服务根路径，不提供管理网页，返回 404 属正常情况；Inspector 是独立启动的调试页面。请选择 `Streamable HTTP`，并先执行 `check_login_status`。如启用了 `AUTH_TOKEN`，连接时填写相应请求头。

### 1.5. 使用 MCP 发布

### 检查登录状态

[检查登录状态 — 上游演示](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/check_login.gif)

### 发布图文

示例中是从 https://unsplash.com/ 中随机找了个图片做测试。

[发布图文 — 上游演示](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/inspect_mcp_publish.gif)

### 搜索内容

使用搜索功能，根据关键词搜索小红书内容：

选择 `search_feeds` 并传入示例参数：

```json
{
  "keyword": "咖啡",
  "filters": {
    "sort_by": "最多评论",
    "note_type": "不限",
    "publish_time": "一周内",
    "search_scope": "未看过",
    "location": "不限"
  }
}
```

从真实搜索结果取 `id` 和 `xsecToken`，再调用详情工具；不要使用文档中的占位 ID 或伪造令牌。筛选状态无法确认时，本版本会报错，不把该情况当作“没有结果”。[完整 HTTP API 说明](docs/API.md)。

[搜索内容 — 上游演示](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/search_result.png)

## 2. MCP 客户端接入

本服务支持标准的 Model Context Protocol (MCP)，可以接入各种支持 MCP 的 AI 客户端。

### 2.1. 快速开始

#### 启动 MCP 服务

```bash
# 启动服务（默认无头模式）
go run .

# 或者有界面模式
go run . -headless=false
```

服务将运行在：`http://localhost:18060/mcp`

#### 验证服务状态

```bash
# 测试 MCP 连接
curl -X POST http://localhost:18060/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Authorization: Bearer your-secret-token" \
  -d '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"example-client","version":"1.0.0"}},"id":1}'
```

#### Claude Code CLI 接入

```bash
# 添加 HTTP MCP 服务器
claude mcp add --transport http xiaohongshu-mcp http://localhost:18060/mcp

# 检查 MCP 是否添加成功（确保 MCP 已经启动的前提下，运行下面命令）
claude mcp list
```

### 2.2. 支持的客户端

<details>
<summary><b>Claude Code CLI</b></summary>

官方命令行工具，已在上面快速开始部分展示：

```bash
# 添加 HTTP MCP 服务器
claude mcp add --transport http xiaohongshu-mcp http://localhost:18060/mcp

# 检查 MCP 是否添加成功（确保 MCP 已经启动的前提下，运行下面命令）
claude mcp list
```

</details>

<details>
<summary><b>Open Code CLI</b></summary>

使用交互式命令添加 MCP Server：

```bash
opencode mcp add
```

以添加 `xiaohongshu-mcp` 为例：

```
┌  Add MCP server
│
◇  Enter MCP server name
│  xiaohongshu-mcp
│
◇  Select MCP server type
│  Remote
│
◇  Enter MCP server URL
│  http://localhost:18060/mcp
│
◇  Does this server require OAuth authentication?
│  No
│
◆  MCP server "xiaohongshu-mcp" added to C:\Users\admin\.config\opencode\opencode.json
│
└  MCP server added successfully
```

验证是否添加成功（确保 MCP 已启动的前提下）：

```bash
opencode mcp list
```

```
┌  MCP Servers
│
●  ✓ xiaohongshu-mcp connected
```

</details>

<details>
<summary><b>Cursor</b></summary>

#### 配置文件的方式

创建或编辑 MCP 配置文件：

**项目级配置**（推荐）：
在项目根目录创建 `.cursor/mcp.json`：

```json
{
  "mcpServers": {
    "xiaohongshu-mcp": {
      "url": "http://localhost:18060/mcp",
      "description": "小红书内容发布服务 - MCP Streamable HTTP"
    }
  }
}
```

**全局配置**：
在用户目录创建 `~/.cursor/mcp.json` (同样内容)。

#### 使用步骤

1. 确保小红书 MCP 服务正在运行
2. 保存配置文件后，重启 Cursor
3. 在 Cursor 聊天中，工具应该自动可用
4. 可以通过聊天界面的 "Available Tools" 查看已连接的 MCP 工具

**Demo**

插件 MCP 接入：

[cursor_mcp_settings — 上游演示](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/cursor_mcp_settings.png)

调用 MCP 工具：（以检查登录状态为例）

[cursor_mcp_check_login — 上游演示](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/cursor_mcp_check_login.png)

</details>

<details>
<summary><b>VSCode</b></summary>

#### 方法一：使用命令面板配置

1. 按 `Ctrl/Cmd + Shift + P` 打开命令面板
2. 运行 `MCP: Add Server` 命令
3. 选择 `HTTP` 方式。
4. 输入地址： `http://localhost:18060/mcp`，或者修改成对应的 Server 地址。
5. 输入 MCP 名字： `xiaohongshu-mcp`。

#### 方法二：直接编辑配置文件

**工作区配置**（推荐）：
在项目根目录创建 `.vscode/mcp.json`：

```json
{
  "servers": {
    "xiaohongshu-mcp": {
      "url": "http://localhost:18060/mcp",
      "type": "http"
    }
  },
  "inputs": []
}
```

**查看配置**：

[vscode_config — 上游演示](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/vscode_mcp_config.png)

1. 确认运行状态。
2. 查看 `tools` 是否正确检测。

**Demo**

以搜索帖子内容为例：

[vscode_mcp_search — 上游演示](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/vscode_search_demo.png)

</details>

<details>
<summary><b>Google Gemini CLI</b></summary>

在 `~/.gemini/settings.json` 或项目目录 `.gemini/settings.json` 中配置：

```json
{
  "mcpServers": {
    "xiaohongshu": {
      "httpUrl": "http://localhost:18060/mcp",
      "timeout": 30000
    }
  }
}
```

更多信息请参考 [Gemini CLI MCP 文档](https://google-gemini.github.io/gemini-cli/docs/tools/mcp-server.html)

</details>

<details>
<summary><b>MCP Inspector</b></summary>

调试工具，用于测试 MCP 连接：

```bash
# 启动 MCP Inspector
npx @modelcontextprotocol/inspector

# 在浏览器中连接到：http://localhost:18060/mcp
```

使用步骤：

- 使用 MCP Inspector 测试连接
- 测试 Ping Server 功能验证连接
- 检查 List Tools 是否返回工具列表，数量以当前服务实际返回为准

</details>

<details>
<summary><b>Cline</b></summary>

Cline 是一个强大的 AI 编程助手，支持 MCP 协议集成。

#### 配置方法

在 Cline 的 MCP 设置中添加以下配置：

```json
{
  "xiaohongshu-mcp": {
    "url": "http://localhost:18060/mcp",
    "type": "streamableHttp",
    "autoApprove": [],
    "disabled": false
  }
}
```

#### 使用步骤

1. 确保小红书 MCP 服务正在运行（`http://localhost:18060/mcp`）
2. 在 Cline 中打开 MCP 设置
3. 添加上述配置到 MCP 服务器列表
4. 保存配置并重启 Cline
5. 在对话中可以直接使用小红书相关功能

#### 配置说明

- `url`: MCP 服务地址
- `type`: 使用 `streamableHttp` 类型以获得更好的性能
- `autoApprove`: 可配置自动批准的工具列表（留空表示手动批准）
- `disabled`: 设置为 `false` 启用此 MCP 服务

#### 使用示例

配置完成后，可以在 Cline 中直接使用自然语言操作小红书：

```
帮我检查小红书登录状态
```

```
帮我发布一篇关于春天的图文到小红书，使用这张图片：/path/to/spring.jpg
```

```
搜索小红书上关于"美食"的内容
```

</details>
<details>
<summary><b>OpenClaw（通过 MCPorter）</b></summary>

> 使用前请确保 xiaohongshu-mcp 已完成本地部署。**不建议**将 GitHub 链接直接丢给 OpenClaw 让其代为部署。

由于 OpenClaw 目前不原生支持 MCP，官方推荐通过 **MCPorter** 来调用 MCP 服务。

> 💡 **提示：** MCPorter 并非调用 MCP 的最佳方案，使用过程中可能出现一些兼容性问题，请知悉。

#### 安装与配置步骤

直接一次性将一下三行命令丢给 OpenClaw（可以是 Control UI、Telegram、Feishu等方式），Openclaw 会代为部署 MCPorter。

```
npm i -g mcporter
npx mcporter config add xiaohongshu-mcp http://localhost:18060/mcp
npx mcporter list xiaohongshu-mcp
```

完成上述步骤后，即可在 OpenClaw 中通过自然语言调用 xiaohongshu-mcp 的所有功能。

</details>
<details>
<summary><b>其他支持 HTTP MCP 的客户端</b></summary>

任何支持 HTTP MCP 协议的客户端都可以连接到：`http://localhost:18060/mcp`

基本配置模板：

```json
{
  "name": "xiaohongshu-mcp",
  "url": "http://localhost:18060/mcp",
  "type": "http"
}
```

</details>

### 2.3. 可用 MCP 工具

连接成功后，可使用以下 MCP 工具：

- `check_login_status` - 检查小红书登录状态（无参数）
- `get_login_qrcode` - 获取登录二维码，返回 Base64 图片和超时时间（无参数）
- `delete_cookies` - 删除 cookies 文件，重置登录状态，删除后需要重新登录（无参数）
- `publish_content` - 发布图文内容到小红书（必需：title, content, images）
  - `images`: 图片路径列表（至少1张），支持 HTTP 链接或本地绝对路径，推荐使用本地路径
  - `tags`: 话题标签列表（可选），如 `["美食", "旅行", "生活"]`
  - `schedule_at`: 定时发布时间（可选），ISO8601 格式，支持 1 小时至 14 天内
  - `is_original`: 是否声明原创（可选），默认不声明
  - `visibility`: 可见范围（可选），支持 `公开可见`（默认）、`仅自己可见`、`仅互关好友可见`
  - `products`: 商品关键词列表（可选），用于绑定带货商品。填写商品名称或商品ID，系统会自动搜索并选择第一个匹配结果。需账号已开通商品功能。示例: [面膜, 防晒霜SPF50]
- `publish_with_video` - 发布视频内容到小红书（必需：title, content, video）
  - `video`: 本地视频文件绝对路径（仅支持单个视频文件）
  - `tags`: 话题标签列表（可选），如 `["美食", "旅行", "生活"]`
  - `schedule_at`: 定时发布时间（可选），ISO8601 格式，支持 1 小时至 14 天内
  - `visibility`: 可见范围（可选），支持 `公开可见`（默认）、`仅自己可见`、`仅互关好友可见`
  - `products`: 商品关键词列表（可选），用于绑定带货商品。填写商品名称或商品ID，系统会自动搜索并选择第一个匹配结果。需账号已开通商品功能。示例: [面膜, 防晒霜SPF50]
- `list_feeds` - 获取小红书首页推荐列表（无参数）
- `search_feeds` - 搜索小红书内容（必需：keyword）
  - `filters`: 筛选选项（可选）
    - `sort_by`: 排序依据 - `综合`（默认）| `最新` | `最多点赞` | `最多评论` | `最多收藏`
    - `note_type`: 笔记类型 - `不限`（默认）| `视频` | `图文`
    - `publish_time`: 发布时间 - `不限`（默认）| `一天内` | `一周内` | `半年内`
    - `search_scope`: 搜索范围 - `不限`（默认）| `已看过` | `未看过` | `已关注`
    - `location`: 位置距离 - `不限`（默认）| `同城` | `附近`
- `get_feed_detail` - 获取帖子详情，包括互动数据和评论（必需：feed_id, xsec_token）
  - `load_all_comments`: 是否加载全部评论（可选），默认 false 仅返回前 10 条一级评论
  - `limit`: 限制加载的一级评论数量（可选），仅当 load_all_comments=true 时生效，默认 20
  - `click_more_replies`: 是否展开二级回复（可选），仅当 load_all_comments=true 时生效，默认 false
  - `reply_limit`: 跳过回复数过多的评论（可选），仅当 click_more_replies=true 时生效，默认 10
  - `scroll_speed`: 滚动速度（可选），`slow` | `normal` | `fast`，仅当 load_all_comments=true 时生效
- `post_comment_to_feed` - 发表评论到小红书帖子（必需：feed_id, xsec_token, content）
- `reply_comment_in_feed` - 回复笔记下的指定评论（必需：feed_id, xsec_token, content，以及 comment_id 或 user_id 至少一个）
- `like_feed` - 点赞/取消点赞（必需：feed_id, xsec_token）
  - `unlike`: 是否取消点赞（可选），true 为取消点赞，默认为点赞
- `favorite_feed` - 收藏/取消收藏（必需：feed_id, xsec_token）
  - `unfavorite`: 是否取消收藏（可选），true 为取消收藏，默认为收藏
- `user_profile` - 获取用户个人主页信息（必需：user_id, xsec_token）
- `get_my_profile` - 获取当前登录账号主页，可选择笔记、收藏或点赞 tab
- `get_unread_count` - 获取当前账号未读通知数量
- `list_notifications` - 获取通知列表
- `reply_notification` - 回复通知中的评论，需要明确授权
- `like_notification` - 点赞通知中的评论，需要明确授权

各工具参数以运行中的 `List Tools` 返回结构为准；文档示例中的 ID、令牌和内容均为占位说明。

### 2.4. 使用示例

使用 Claude Code 发布内容到小红书：

**示例 1：使用 HTTP 图片链接**

```
帮我写一篇帖子发布到小红书上，
配图为：https://example.com/your-licensed-image.jpg
请使用已经获得发布许可的图片；上面的地址是占位示例。

使用 xiaohongshu-mcp 进行发布。
```

**示例 2：使用本地图片路径（推荐）**

```
帮我写一篇关于春天的帖子发布到小红书上，
使用这些本地图片：
- /Users/username/Pictures/spring_flowers.jpg
- /Users/username/Pictures/cherry_blossom.jpg

使用 xiaohongshu-mcp 进行发布。
```

**示例 3：发布视频内容**

```
帮我写一篇关于美食制作的视频发布到小红书上，
使用这个本地视频文件：
- /Users/username/Videos/cooking_tutorial.mp4

使用 xiaohongshu-mcp 的视频发布功能。
```

[claude-cli 进行发布 — 上游演示](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/claude_push.gif)

**发布结果：**

[上游示例](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/publish_result.jpeg)

### 2.5. 💬 MCP 使用常见问题解答

---

> ⚠️ 以下是使用 OpenClaw + MCPorter 时的已知风险，使用前请充分了解：

- OpenClaw 的 AI 自动部署行为不在本项目的维护范围内，部署结果无法保证
- MCPorter 作为中间层可能引入额外的兼容性问题，与 xiaohongshu-mcp 本身无关
- 若遇到连接失败、工具调用异常等问题，请先排查 MCPorter 自身的配置，而非提交 Issue
- 在提问社区或群组前，请先确认问题是否能在**不使用 OpenClaw** 的情况下复现

如果你没有强烈的 OpenClaw 使用需求，强烈建议改用 [Claude Code CLI](#claude-code-cli)、[Cursor](#cursor) 或 [Cline](#cline) 等原生支持 HTTP MCP 的客户端，体验会更稳定。

---

**Q:** 登录状态怎样确认？
**A:** 本版本会核对真实的非游客账号状态；仅看到侧栏、普通网页已登录或本地存在 cookies 文件，不能证明 MCP 会话可用。安全验证需要单独完成，详见[恢复说明](docs/RECOVERY.md)。

---

**Q:** 显示发布成功后，但实际上没有显示？
**A:** 排查步骤如下：

1. 先核对是否已经发布成功，结果不确定时不要重复发布；必要时用 **非无头模式** 诊断。
2. 确认未成功后再决定是否重试。
3. 登录网页版小红书，查看账号是否被 **风控限制网页版发布**。
4. 检查 **图片大小** 是否过大。
5. 确认 **图片路径中没有中文字符**。
6. 若使用网络图片地址，请确认 **图片链接可正常访问**。

---

**Q:** 在设备上运行 MCP 程序出现闪退如何解决？
**A:**

1. 建议 **从源码安装**。
2. 或使用 **Docker 安装 xiaohongshu-mcp**，教程参考：
   - [使用 Docker 安装 xiaohongshu-mcp](https://github.com/xpzouying/xiaohongshu-mcp#:~:text=%E6%96%B9%E5%BC%8F%E4%B8%89%EF%BC%9A%E4%BD%BF%E7%94%A8%20Docker%20%E5%AE%B9%E5%99%A8%EF%BC%88%E6%9C%80%E7%AE%80%E5%8D%95%EF%BC%89)
   - [X-MCP 项目页面](https://github.com/xpzouying/x-mcp/)

---

**Q:** 使用 `http://localhost:18060/mcp` 进行 MCP 验证时提示无法连接？
**A:**

- 在 **Docker 环境** 下，请使用
  👉 [http://host.docker.internal:18060/mcp](http://host.docker.internal:18060/mcp)
- 在 **非 Docker 环境** 下，请使用 **本机 IPv4 地址** 访问。

---

## 许可证

采用 [Apache License 2.0](LICENSE) 开源，保留上游版权声明与完整许可证。来源与修改说明见 [NOTICE](NOTICE)。
