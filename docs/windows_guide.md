# Windows 安装与验证

本指南已为修改后的源码版本更新，来源见 [NOTICE](../NOTICE)。当前内置浏览器支持 Windows x64。

## 直接下载（推荐）

从[最新 Release](https://github.com/liaogx/xiaohongshu-mcp/releases/latest) 只下载 `xiaohongshu-mcp-windows-amd64.exe`，无需安装 Go 或 Git。该文件已包含服务、`login` 和 `recover` 入口。下载版的完整启动命令见[快速启动](releases/v1.0.1.md)，可跳过下面的源码编译步骤。

## 从源码编译：安装依赖

安装 Git、Go 1.24 或更新版本，以及运行 MCP Inspector 所需的 Node.js。可使用官方安装包或 Winget：

```powershell
winget install Git.Git
winget install GoLang.Go
winget install OpenJS.NodeJS.LTS
```

安装后重新打开终端，检查 `git --version`、`go version`、`node --version`。

## 编译本仓库

```powershell
git clone https://github.com/liaogx/xiaohongshu-mcp.git
cd xiaohongshu-mcp
go mod download
go build -o bin/xiaohongshu-mcp.exe .
```

上游预编译安装包不一定包含本仓库的修改。首次运行会下载并校验内置浏览器。网络失败时检查连接后重试，不要关闭安全软件或给整个临时目录添加排除规则。遇到安全告警应先核对下载来源和告警原因。

## 私有登录文件与启动

将示例替换为本人私有运行目录，先创建该目录，不要放在公开仓库中：

```powershell
$env:COOKIES_PATH = 'C:\private-xhs-runtime\cookies.json'
.\bin\xiaohongshu-mcp.exe login
.\bin\xiaohongshu-mcp.exe -port=127.0.0.1:18060
```

用小红书 App 完成弹窗中的扫码。下载版将 `.\bin\xiaohongshu-mcp.exe` 路径替换为下载文件的实际路径。服务默认无头运行；需要诊断时才使用 `-headless=false`。所有启动入口应使用同一 `COOKIES_PATH`。代理与访问令牌只通过本地环境注入，不提交真实值。

需要人工验证时先暂停任务，运行 `.\bin\xiaohongshu-mcp.exe recover`；重新扫码使用 `.\bin\xiaohongshu-mcp.exe recover -fresh`。不需要另外一个程序。

## 验证连接与搜索

```powershell
npx @modelcontextprotocol/inspector
```

打开 Inspector 输出的本地链接，选择 `Streamable HTTP`，URL 填写 `http://127.0.0.1:18060/mcp`。点击连接、列出工具，先调用 `check_login_status`，再尝试 `search_feeds`。根路径返回 404 不代表 MCP 服务故障。

遇到安全验证或登录失效时，先暂停写操作，按[恢复说明](RECOVERY.md)处理。不要公开上传二维码、运行日志、cookies 或完整请求回执。
