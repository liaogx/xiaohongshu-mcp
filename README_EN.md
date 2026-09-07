# xiaohongshu-mcp

[中文](README.md) | English · [Apache-2.0](LICENSE)

MCP for Xiaohongshu: search notes, retrieve recommendations and details, inspect public profiles, and perform user-authorized publishing and interactions.

This is an independently maintained, modified distribution of [xpzouying/xiaohongshu-mcp](https://github.com/xpzouying/xiaohongshu-mcp). It has fresh Git history, but retains the upstream license and attribution; see [NOTICE](NOTICE). It is not affiliated with Xiaohongshu.

### Changes in this distribution

- More accurate detection of expired login sessions and security-verification pages.
- Search-filter selection by group and leaf option, with verification of the selected state.
- Atomic, private cookie-file saves and a manual session-recovery tool.
- Offline regression fixtures, publication checks, and runtime-data exclusions.
- A generic [image download helper and local gallery index](docs/DOWNLOAD_IMAGES.md).

Feature demos and community tutorials below link to **public upstream examples**. No local account data, login QR images, private media, or real interaction receipts are distributed with this repository. UI details may differ by version.

## Project Overview

**Main Features**

> 💡 **Tip:** Click on the feature titles below to expand and view video demonstrations

<details>
<summary><b>1. Login and Check Login Status</b></summary>

The first step is required - RedNote needs to be logged in. You can check current login status.

**Login Demo:**

https://github.com/user-attachments/assets/8b05eb42-d437-41b7-9235-e2143f19e8b7

**Check Login Status Demo:**

https://github.com/user-attachments/assets/bd9a9a4a-58cb-4421-b8f3-015f703ce1f9

</details>

<details>
<summary><b>2. Publish Image and Text Content</b></summary>

Supports publishing image and text content to RedNote, including title, content description, and images.

**Image Support Methods:**

Supports two image input methods:

1. **HTTP/HTTPS Image Links**

   ```
   ["https://example.com/image1.jpg", "https://example.com/image2.png"]
   ```

2. **Local Image Absolute Paths** (Recommended)
   ```
   ["/Users/username/Pictures/image1.jpg", "/home/user/images/image2.png"]
   ```

**Why Local Paths are Recommended:**

- ✅ Better stability, not dependent on network
- ✅ Faster upload speed
- ✅ Avoid image link expiration issues
- ✅ Support more image formats

**Publish Image-Text Post Demo:**

https://github.com/user-attachments/assets/8aee0814-eb96-40af-b871-e66e6bbb6b06

</details>

<details>
<summary><b>3. Publish Video Content</b></summary>

Supports publishing video content to RedNote, including title, content description, and local video files.

**Video Support Methods:**

Only supports local video file absolute paths:

```
"/Users/username/Videos/video.mp4"
```

**Features:**

- ✅ Supports local video file upload
- ✅ Automatic video format processing
- ✅ Supports title, content description, and tags
- ✅ Automatically publishes after video processing is complete

**Important Notes:**

- Only supports local video files, not HTTP links
- Video processing takes longer, please be patient
- Recommended video file size should not exceed 1GB

</details>

<details>
<summary><b>4. Search Content</b></summary>

Search RedNote content by keywords.

**Search Posts Demo:**

https://github.com/user-attachments/assets/03c5077d-6160-4b18-b629-2e40933a1fd3

</details>

<details>
<summary><b>5. Get Recommendation List</b></summary>

Get RedNote homepage recommendation content list.

**Get Recommendation List Demo:**

https://github.com/user-attachments/assets/110fc15d-46f2-4cca-bdad-9de5b5b8cc28

</details>

<details>
<summary><b>6. Get Post Details (Including Interaction Data and Comments)</b></summary>

Get complete details of RedNote posts, including:

- Post content (title, description, images, etc.)
- User information
- Interaction data (likes, favorites, shares, comment count)
- Comment list and sub-comments

**⚠️ Important Note:**

- Both post ID and xsec_token are required (both parameters are essential)
- These two parameters can be obtained from Feed list or search results
- Must login first to use this feature

**Get Post Details Demo:**

https://github.com/user-attachments/assets/76a26130-a216-4371-a6b3-937b8fda092a

</details>

<details>
<summary><b>7. Post Comments to Posts</b></summary>

Supports automatically posting comments to RedNote posts.

**Feature Description:**

- Automatically locate comment input box
- Input comment content and publish
- Supports HTTP API and MCP tool calls

**⚠️ Important Note:**

- Must login first to use this feature
- Need to provide post ID, xsec_token, and comment content
- These parameters can be obtained from Feed list or search results

**Post Comment Demo:**

https://github.com/user-attachments/assets/cc385b6c-422c-489b-a5fc-63e92c695b80

</details>

<details>
<summary><b>8. Get User Profile</b></summary>

Get RedNote user's personal profile information, including basic user information and note content.

**Feature Description:**

- Get user basic information (nickname, bio, avatar, etc.)
- Get follower count, following count, likes count statistics
- Get user's published note content list
- Supports HTTP API and MCP tool calls

**⚠️ Important Note:**

- Must login first to use this feature
- Need to provide user ID and xsec_token
- These parameters can be obtained from Feed list or search results

**Returned Information Includes:**

- User basic info: nickname, bio, avatar, verification status
- Statistics: following count, follower count, likes count, note count
- Note list: all public notes published by the user

</details>

<details>
<summary><b>9. Reply to Comments</b></summary>

Reply to a specific comment under a note, supporting precise replies to specific users' comments.

**Feature Description:**

- Reply to a specific comment under a note
- Support locating target comment by comment ID or user ID
- Requires feed_id, xsec_token, comment_id/user_id, and reply content

**⚠️ Important Note:**

- Must login first to use this feature
- At least one of comment_id or user_id must be provided
- These parameters can be obtained from the comment list in post details

</details>

<details>
<summary><b>10. Like / Unlike</b></summary>

Like or unlike a note, with smart detection of current status to avoid duplicate operations.

**Feature Description:**

- Like or unlike a specified note
- Smart detection: skips liking if already liked, skips unliking if not liked
- Requires feed_id and xsec_token

**⚠️ Important Note:**

- Must login first to use this feature
- Default action is like, set unlike=true to unlike

</details>

<details>
<summary><b>11. Favorite / Unfavorite</b></summary>

Favorite a note or unfavorite it, with smart detection of current status to avoid duplicate operations.

**Feature Description:**

- Favorite or unfavorite a specified note
- Smart detection: skips favoriting if already favorited, skips unfavoriting if not favorited
- Requires feed_id and xsec_token

**⚠️ Important Note:**

- Must login first to use this feature
- Default action is favorite, set unfavorite=true to unfavorite

</details>

**Usage and privacy**

Login sessions, QR codes, downloaded media, logs, and interaction receipts are private runtime data, not source code. Keep them outside the repository. Browser authentication and security verification may expire independently; stop actions when verification is required. See [recovery instructions](docs/RECOVERY.md).

## 1. Usage Tutorial

### 1.1. Build this version from source

Install [Go](https://go.dev/doc/install) 1.24 or later and Git. Build from **this repository** to include its fixes; upstream binaries and container images do not automatically contain these changes.

```bash
git clone https://github.com/liaogx/xiaohongshu-mcp.git
cd xiaohongshu-mcp
go mod download
go build -o bin/xiaohongshu-mcp .
go build -o bin/xiaohongshu-login ./cmd/login
go build -o bin/xiaohongshu-recover ./cmd/recover
```

The bundled browser supports macOS Apple Silicon, Windows x64, and Linux x64. Windows executables should use `.exe` output names. macOS Intel and Linux ARM64 are not currently supported by the browser distribution. The first launch downloads and verifies the browser; later launches reuse the cache.

Optional module proxy: set `GOPROXY=https://goproxy.cn,direct` for the build if required by your network.

**Docker:** build this checkout with `docker compose -f docker/docker-compose.yml up -d --build`. See the [Docker guide](docker/README.md) and [Windows guide](docs/windows_guide.md).

### 1.2. Login

First time requires manual login to save RedNote login status.

Create a private runtime directory outside the repository and set the same `COOKIES_PATH` for the login tool, server, and recovery tool. Never commit login files or logs. Run the commands below with that environment; see [recovery and data isolation](docs/RECOVERY.md).

**Using Binary Files:**

```bash
# Run the login tool for your platform
./bin/xiaohongshu-login
```

**Using Source Code:**

```bash
go run ./cmd/login
```

### 1.3. Start MCP Service

Start xiaohongshu-mcp service.

For local use, append `-port=127.0.0.1:18060` to bind only to loopback. The existing `:18060` default listens on all interfaces; do not expose an unauthenticated server publicly.

**Using Binary Files:**

```bash
# Default: Headless mode, no browser interface
./bin/xiaohongshu-mcp

# Non-headless mode, with browser interface
./bin/xiaohongshu-mcp -headless=false
```

**Using Source Code:**

```bash
# Default: Headless mode, no browser interface
go run .

# Non-headless mode, with browser interface
go run . -headless=false
```

**Configure a proxy (optional)**:

If you need to go through a proxy, set the `XHS_PROXY` environment variable:

```bash
# Start with a proxy configured
XHS_PROXY=http://proxy.example:8080 ./bin/xiaohongshu-mcp

# Or from source
XHS_PROXY=http://proxy.example:8080 go run .
```

HTTP/HTTPS/SOCKS5 proxies are supported, and proxy credentials are automatically masked in the logs.

**Optional authentication**:

Authentication is disabled by default. In production, configure it with the `AUTH_TOKEN` environment variable; a non-empty startup flag takes precedence, while an empty value falls back to `AUTH_TOKEN`.

```bash
# Environment variable
AUTH_TOKEN=your-secret-token ./bin/xiaohongshu-mcp
AUTH_TOKEN=your-secret-token go run .

# Non-empty startup flag (takes precedence over the environment variable)
./bin/xiaohongshu-mcp -token=your-secret-token
go run . -token=your-secret-token
```

When authentication is enabled, every MCP client must configure the custom request header `Authorization: Bearer <token>`. Command-line arguments may be visible in process listings, so prefer `AUTH_TOKEN` in deployment environments.

For example, MCP clients that support custom request headers can use the following configuration:

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

### 1.4. Verify MCP

```bash
npx @modelcontextprotocol/inspector
```

[Run Inspector — upstream demo](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/run_inspect.png)

After running, open the red-marked link, configure MCP inspector, enter `http://localhost:18060/mcp`, and click the `Connect` button.

<img width="915" height="659" alt="bf9532dd0b7ba423491accf511a467de" src="https://github.com/user-attachments/assets/08bc3cef-73e7-42d2-b923-7ba9e6c8af30" />

**Note:** Check if the options in the left sidebar are correct.

After configuring MCP inspector as above, click the `List Tools` button to view all Tools.

The server root `http://127.0.0.1:18060/` does not host an administration page; a 404 there is expected. Inspector runs separately. Select `Streamable HTTP`, then call `check_login_status` first. If `AUTH_TOKEN` is configured, provide the corresponding request header.

### 1.5. Use MCP for Publishing

### Check Login Status

[Check Login Status — upstream demo](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/check_login.gif)

### Publish Image-Text

The example uses a random image from https://unsplash.com/ for testing.

[Publish Image-Text — upstream demo](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/inspect_mcp_publish.gif)

### Search Content

Use search functionality to search RedNote content by keywords:

Select `search_feeds` with example arguments:

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

Use the `id` and `xsecToken` from actual search results for the detail tool, never placeholder IDs or invented tokens. Unconfirmed filters are errors, not empty results. See the [HTTP API reference](docs/API.md).

[Search Content — upstream demo](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/search_result.png)

## 2. MCP Client Integration

This service supports the standard Model Context Protocol (MCP) and can integrate with various AI clients that support MCP.

### 2.1. Quick Start

#### Start MCP Service

```bash
# Start service (default headless mode)
go run .

# Or with interface mode
go run . -headless=false
```

Service will run at: `http://localhost:18060/mcp`

#### Verify Service Status

```bash
# Test MCP connection
curl -X POST http://localhost:18060/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Authorization: Bearer your-secret-token" \
  -d '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"example-client","version":"1.0.0"}},"id":1}'
```

#### Claude Code CLI Integration

```bash
# Add HTTP MCP server
claude mcp add --transport http xiaohongshu-mcp http://localhost:18060/mcp

# Check if MCP was added successfully (ensure MCP is already started before running this command)
claude mcp list
```

### 2.2. Supported Clients

<details>
<summary><b>Claude Code CLI</b></summary>

Official command line tool, already shown in the quick start section above:

```bash
# Add HTTP MCP server
claude mcp add --transport http xiaohongshu-mcp http://localhost:18060/mcp

# Check if MCP was added successfully (ensure MCP is already started before running this command)
claude mcp list
```

</details>

<details>
<summary><b>Open Code CLI</b></summary>

Add the MCP server with the interactive command:

```bash
opencode mcp add
```

Using `xiaohongshu-mcp` as an example:

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

Verify that it was added successfully (make sure the MCP service is running):

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

#### Configuration File Method

Create or edit MCP configuration file:

**Project-level configuration** (recommended):
Create `.cursor/mcp.json` in project root directory:

```json
{
  "mcpServers": {
    "xiaohongshu-mcp": {
      "url": "http://localhost:18060/mcp",
      "description": "RedNote content publishing service - MCP Streamable HTTP"
    }
  }
}
```

**Global configuration**:
Create `~/.cursor/mcp.json` in user directory (same content).

#### Usage Steps

1. Ensure RedNote MCP service is running
2. Save configuration file and restart Cursor
3. In Cursor chat, tools should be automatically available
4. You can view connected MCP tools through "Available Tools" in the chat interface

**Demo**

Plugin MCP integration:

[cursor_mcp_settings — upstream demo](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/cursor_mcp_settings.png)

Call MCP tools: (using check login status as example)

[cursor_mcp_check_login — upstream demo](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/cursor_mcp_check_login.png)

</details>

<details>
<summary><b>VSCode</b></summary>

#### Method 1: Configure using Command Palette

1. Press `Ctrl/Cmd + Shift + P` to open command palette
2. Run `MCP: Add Server` command
3. Select `HTTP` method.
4. Enter address: `http://localhost:18060/mcp`, or modify to corresponding Server address.
5. Enter MCP name: `xiaohongshu-mcp`.

#### Method 2: Direct Configuration File Edit

**Workspace configuration** (recommended):
Create `.vscode/mcp.json` in project root directory:

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

**View Configuration**:

[vscode_config — upstream demo](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/vscode_mcp_config.png)

1. Confirm running status.
2. Check if `tools` are correctly detected.

**Demo**

Using search post content as example:

[vscode_mcp_search — upstream demo](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/vscode_search_demo.png)

</details>

<details>
<summary><b>Google Gemini CLI</b></summary>

Configure in `~/.gemini/settings.json` or project directory `.gemini/settings.json`:

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

For more information, please refer to [Gemini CLI MCP Documentation](https://google-gemini.github.io/gemini-cli/docs/tools/mcp-server.html)

</details>

<details>
<summary><b>MCP Inspector</b></summary>

Debug tool for testing MCP connections:

```bash
# Start MCP Inspector
npx @modelcontextprotocol/inspector

# Connect in browser to: http://localhost:18060/mcp
```

Usage steps:

- Use MCP Inspector to test connection
- Test Ping Server functionality to verify connection
- Check that List Tools returns the tools exposed by the running server

</details>

<details>
<summary><b>Cline</b></summary>

Cline is a powerful AI programming assistant that supports MCP protocol integration.

#### Configuration Method

Add the following configuration to Cline's MCP settings:

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

#### Usage Steps

1. Ensure RedNote MCP service is running (`http://localhost:18060/mcp`)
2. Open MCP settings in Cline
3. Add the above configuration to the MCP server list
4. Save configuration and restart Cline
5. You can directly use RedNote-related features in conversations

#### Configuration Explanation

- `url`: MCP service address
- `type`: Use `streamableHttp` type for better performance
- `autoApprove`: Configurable auto-approve tool list (empty means manual approval)
- `disabled`: Set to `false` to enable this MCP service

#### Usage Examples

After configuration, you can use natural language to operate RedNote directly in Cline:

```
Help me check RedNote login status
```

```
Help me publish a spring-themed image-text post to RedNote, using this image: /path/to/spring.jpg
```

```
Search for content about "food" on RedNote
```

</details>
<details>
<summary><b>OpenClaw (via MCPorter)</b></summary>

> Make sure xiaohongshu-mcp is already deployed locally before you start. Handing the GitHub link to OpenClaw and letting it deploy the project for you is **not recommended**.

Since OpenClaw does not natively support MCP yet, the officially recommended way to call MCP services is through **MCPorter**.

> 💡 **Tip:** MCPorter is not the ideal way to call MCP — you may run into compatibility issues along the way, so please be aware.

#### Installation and Setup

Just hand the following three commands to OpenClaw in one go (via the Control UI, Telegram, Feishu, etc.), and OpenClaw will set up MCPorter for you.

```
npm i -g mcporter
npx mcporter config add xiaohongshu-mcp http://localhost:18060/mcp
npx mcporter list xiaohongshu-mcp
```

Once that is done, you can use every xiaohongshu-mcp feature from OpenClaw through natural language.

</details>
<details>
<summary><b>Other HTTP MCP Supporting Clients</b></summary>

Any client supporting HTTP MCP protocol can connect to: `http://localhost:18060/mcp`

Basic configuration template:

```json
{
  "name": "xiaohongshu-mcp",
  "url": "http://localhost:18060/mcp",
  "type": "http"
}
```

</details>

### 2.3. Available MCP Tools

After successful connection, you can use the following MCP tools:

- `check_login_status` - Check RedNote login status (no parameters)
- `get_login_qrcode` - Get login QR code, returns Base64 image and timeout (no parameters)
- `delete_cookies` - Delete cookies file, reset login status, requires re-login after deletion (no parameters)
- `publish_content` - Publish image-text content to RedNote (required: title, content, images)
  - `images`: Image path list (minimum 1), supports HTTP links or local absolute paths, local paths recommended
  - `tags`: Topic tags list (optional), e.g. `["food", "travel", "lifestyle"]`
  - `schedule_at`: Scheduled publish time (optional), ISO8601 format, supports 1 hour to 14 days ahead
  - `is_original`: Declare as original content (optional), default is not declared
  - `visibility`: Visibility scope (optional), supports `公开可见` / public (default), `仅自己可见` / self-only, `仅互关好友可见` / mutual-followers-only
  - `products`: Product keyword list (optional), used to attach products for social commerce. Provide a product name or product ID; the system searches automatically and picks the first match. Requires the product feature to be enabled on your account. Example: [面膜, 防晒霜SPF50]
- `publish_with_video` - Publish video content to RedNote (required: title, content, video)
  - `video`: Local video file absolute path (single file only)
  - `tags`: Topic tags list (optional), e.g. `["food", "travel", "lifestyle"]`
  - `schedule_at`: Scheduled publish time (optional), ISO8601 format, supports 1 hour to 14 days ahead
  - `visibility`: Visibility scope (optional), supports `公开可见` / public (default), `仅自己可见` / self-only, `仅互关好友可见` / mutual-followers-only
  - `products`: Product keyword list (optional), used to attach products for social commerce. Provide a product name or product ID; the system searches automatically and picks the first match. Requires the product feature to be enabled on your account. Example: [面膜, 防晒霜SPF50]
- `list_feeds` - Get RedNote homepage recommendation list (no parameters)
- `search_feeds` - Search RedNote content (required: keyword)
  - `filters`: Filter options (optional). Values must be passed exactly as the Chinese strings below — they match the labels on the RedNote filter panel.
    - `sort_by`: Sort by - `综合` / comprehensive (default) | `最新` / latest | `最多点赞` / most liked | `最多评论` / most comments | `最多收藏` / most saved
    - `note_type`: Note type - `不限` / any (default) | `视频` / video | `图文` / image-text
    - `publish_time`: Publish time - `不限` / any (default) | `一天内` / last day | `一周内` / last week | `半年内` / last 6 months
    - `search_scope`: Search scope - `不限` / any (default) | `已看过` / viewed | `未看过` / not viewed | `已关注` / followed
    - `location`: Location - `不限` / any (default) | `同城` / same city | `附近` / nearby
- `get_feed_detail` - Get post details including interaction data and comments (required: feed_id, xsec_token)
  - `load_all_comments`: Whether to load all comments (optional), default false returns only first 10 top-level comments
  - `limit`: Limit number of top-level comments to load (optional), only effective when load_all_comments=true, default 20
  - `click_more_replies`: Whether to expand nested replies (optional), only effective when load_all_comments=true, default false
  - `reply_limit`: Skip comments with too many replies (optional), only effective when click_more_replies=true, default 10
  - `scroll_speed`: Scroll speed (optional), `slow` | `normal` | `fast`, only effective when load_all_comments=true
- `post_comment_to_feed` - Post comments to RedNote posts (required: feed_id, xsec_token, content)
- `reply_comment_in_feed` - Reply to a specific comment under a note (required: feed_id, xsec_token, content, and at least one of comment_id or user_id)
- `like_feed` - Like / unlike a note (required: feed_id, xsec_token)
  - `unlike`: Whether to unlike (optional), true to unlike, default is like
- `favorite_feed` - Favorite / unfavorite a note (required: feed_id, xsec_token)
  - `unfavorite`: Whether to unfavorite (optional), true to unfavorite, default is favorite
- `user_profile` - Get user profile information (required: user_id, xsec_token)
- `get_my_profile` - Get the current account profile, including supported note/favorite/liked tabs
- `get_unread_count` - Get unread notification counts
- `list_notifications` - List notifications
- `reply_notification` - Reply to a comment notification, with user authorization
- `like_notification` - Like a comment notification, with user authorization

Use the running server's `List Tools` schemas for complete arguments. IDs, tokens, and content in documentation examples are placeholders.

### 2.4. Usage Examples

Using Claude Code to publish content to RedNote:

**Example 1: Using HTTP Image Links**

```
Help me write a post to publish on RedNote,
with image: https://example.com/your-licensed-image.jpg
Use an image you have permission to publish; the URL above is a placeholder.

Use xiaohongshu-mcp for publishing.
```

**Example 2: Using Local Image Paths (Recommended)**

```
Help me write a post about spring to publish on RedNote,
using these local images:
- /Users/username/Pictures/spring_flowers.jpg
- /Users/username/Pictures/cherry_blossom.jpg

Use xiaohongshu-mcp for publishing.
```

**Example 3: Publishing Video Content**

```
Help me write a video post about cooking tutorials to publish on RedNote,
using this local video file:
- /Users/username/Videos/cooking_tutorial.mp4

Use xiaohongshu-mcp's video publishing feature.
```

[claude-cli publishing — upstream demo](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/claude_push.gif)

**Publishing Result:**

[Upstream example](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/assets/publish_result.jpeg)

### 2.5. 💬 MCP FAQ

---

> ⚠️ The following are known risks when using OpenClaw + MCPorter. Please read them carefully before you start:

- OpenClaw's automated AI deployment behavior is outside the scope of this project's maintenance, and its results cannot be guaranteed
- As an intermediate layer, MCPorter may introduce additional compatibility issues that have nothing to do with xiaohongshu-mcp itself
- If you hit connection failures or abnormal tool calls, please check MCPorter's own configuration first instead of filing an Issue
- Before asking in the community or the groups, please confirm whether the problem also reproduces **without OpenClaw**

If you do not specifically need OpenClaw, we strongly recommend switching to a client with native HTTP MCP support such as [Claude Code CLI](#claude-code-cli), [Cursor](#cursor) or [Cline](#cline) — the experience is much more stable.

---

**Q:** How is login validated?
**A:** This version checks a real, non-guest account state. A sidebar, another signed-in browser, or an existing cookie file does not prove that the MCP session is usable. See [recovery instructions](docs/RECOVERY.md).

---

**Q:** It shows publish success but the post doesn't actually appear?
**A:** Troubleshooting steps:

1. Verify whether publishing already succeeded before retrying; use **non-headless mode** for diagnosis if necessary.
2. Retry only after confirming the prior action did not succeed.
3. Login to RedNote web version and check if the account has been **restricted from web publishing due to risk control**.
4. Check if the **image size** is too large.
5. Make sure there are **no Chinese characters in the image path**.
6. If using network image URLs, confirm the **image links are accessible**.

---

**Q:** The MCP program crashes on my device, how to resolve?
**A:**

1. It is recommended to **build from source**.
2. Or use **Docker to install xiaohongshu-mcp**, refer to:
   - [Install xiaohongshu-mcp with Docker](https://github.com/xpzouying/xiaohongshu-mcp#:~:text=%E6%96%B9%E5%BC%8F%E4%B8%89%EF%BC%9A%E4%BD%BF%E7%94%A8%20Docker%20%E5%AE%B9%E5%99%A8%EF%BC%88%E6%9C%80%E7%AE%80%E5%8D%95%EF%BC%89)
   - [X-MCP Project Page](https://github.com/xpzouying/x-mcp/)

---

**Q:** When verifying MCP with `http://localhost:18060/mcp`, it shows connection error?
**A:**

- In a **Docker environment**, please use
  [http://host.docker.internal:18060/mcp](http://host.docker.internal:18060/mcp)
- In a **non-Docker environment**, please use your **local IPv4 address** to access.

---

## 3. 🌟 Community Showcases

> 💡 **Highly Recommended**: These are real-world use cases from community contributors, featuring detailed configuration steps and practical experiences!

### 📚 Complete Tutorial List

1. **[n8n Complete Integration Tutorial](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/examples/n8n/README.md)** - Workflow automation platform integration
2. **[Cherry Studio Complete Configuration Tutorial](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/examples/cherrystudio/README.md)** - Perfect AI client integration
3. **[Claude Code + Kimi K2 Integration Tutorial](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/examples/claude-code/claude-code-kimi-k2.md)** - If Claude Code's barrier is too high, then integrate with Kimi domestic LLM!
4. **[AnythingLLM Complete Guide](https://github.com/xpzouying/xiaohongshu-mcp/blob/main/examples/anythingLLM/readme.md)** - AnythingLLM is an all-in-one multimodal AI client that supports workflow definition, multiple LLMs, and plugin extensions.

> 🎯 **Tip**: Click the links above to view detailed step-by-step tutorials for quick setup of various integration solutions!
>

## License

Distributed under [Apache License 2.0](LICENSE). The upstream copyright and license are retained. See [NOTICE](NOTICE) for source attribution and the modification summary.
