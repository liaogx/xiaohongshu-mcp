# 发行构建

本仓库从 `v1.0.0` 开始独立编号，与上游版本号无对应关系。许可证保持 Apache-2.0，上游来源见 [NOTICE](../NOTICE)。

## 构建工具与平台

- Go 1.24 或更新版本、Python 3.9+、Git；使用指定的本地 Go，不自动升级工具链。
- 只构建内置浏览器实际支持的 `darwin/arm64`、`linux/amd64`、`windows/amd64`。
- 从 v1.0.1 起，每个平台只提供一个主程序，内置 `login` 与 `recover`，共 3 个可执行文件；旧版 v1.0.0 附件不变。
- `-version` 仅显示版本信息，不下载浏览器、不读取登录文件、不启动服务。
- 使用 `CGO_ENABLED=0`、`-trimpath` 和基线 CPU 目标；去除调试符号，并嵌入版本、源码提交和 UTC 构建时间。

## 准备与执行

1. 更新 `pkg/buildinfo.DefaultVersion`、Dockerfile 的默认 `VERSION`，以及版本相关文档。
2. 新增 `docs/releases/vX.Y.Z.md`，其中只保留一个 `<!-- BUILD_INFO -->` 标记；构建脚本会替换为实际信息。
3. 完成测试并提交修改。构建脚本要求源码工作区干净；已有同名 tag 必须指向当前提交。
4. 执行：

```bash
go test ./...
go vet ./...
python3 -m unittest discover -s scripts -p '*_test.py'
python3 scripts/build_release.py --tag v1.0.1
# 可用 --go /absolute/path/to/go 指定 Go；默认输出 dist/v1.0.1。
```

输出目录必须是本仓库内被 Git 忽略的新目录。脚本不覆盖已有构建产物，不创建标签、不推送、不发布、不部署或操作账号；构建失败时保留产物供排查，不能当成完整版本上传。

## 产物与验证

- 3 个可执行文件（每个平台一个），不再分发独立登录或恢复程序。
- `BUILD-INFO.json`：版本、tag、提交、Go 版本、时间、浏览器版本、各二进制的 SHA256/大小/目标平台。
- `SHA256SUMS`：可执行文件、构建信息和许可附件的哈希，不包含校验文件自身。
- `LICENSE`、`NOTICE`、`THIRD_PARTY_NOTICES.txt`：项目许可、来源声明与依赖许可文本。
- `RELEASE_NOTES.md`：用于 GitHub Release 正文，不必再作为下载附件。

检查本机可执行文件的 `-version` 与 `BUILD-INFO.json` 一致，并核对各目标的二进制格式和校验值。跨平台编译成功不等于已经在目标操作系统完成运行测试，Release 正文必须如实说明验证范围。若要检验网页业务功能，应另行安排有授权的测试账号；发行构建本身不做实际发布或互动。

确认 tag、附件和正文后可使用 GitHub CLI 创建草稿 Release，检查附件后再发布。不要覆盖已经发布的版本：后续修复使用新的版本号。旧 `cmd/login`、`cmd/recover` 仅保留为源码兼容入口，共用主程序的实现，不加入新版下载附件。
