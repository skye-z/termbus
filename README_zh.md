# Termbus
> 一款面向终端优先的 DevOps SSH 终端工具 - 以 SSH 为传输总线的「远程操作系统」

Termbus 是基于 Go 开发的现代化 TUI 风格 SSH 客户端，定位为以 SSH 协议为核心的全功能「远程操作系统」。它将 SSH 连接、SFTP 文件管理、端口转发、多会话管理和插件扩展能力统一到原生终端体验中，完全遵循 Unix 工具设计哲学和 DevOps 工作流。

## ✨ 核心特性
- **完整的SSH能力**：支持密码/密钥/Agent 认证、SSH Config 全兼容、堡垒机跳转、连接复用。
- **终端优先的TUI界面**：响应式UI（类 lazygit/k9s）、tmux 风格窗格/标签页、全键盘操作。
- **SFTP文件管理器**：双栏浏览器、上传/下载、增量同步、内置编辑器。
- **端口转发管理**：可视化管理本地/远程/动态 SOCKS 隧道，状态实时监控。
- **多会话管理**：并发连接、会话状态持久化、断线自动重连。
- **插件扩展系统**：支持 Docker/K8s/Redis/MySQL 等插件（进程隔离，安全可控）。
- **AI智能助手（测试版）**：自然语言驱动的远程运维，内置安全沙箱。
- **可脚本化**：批量执行、命令别名、CI/CD 集成能力。

## 🚀 快速安装
### macOS/Linux
```bash
# Go 安装（需要 Go 1.21+）
go install github.com/skye-z/termbus/cmd/termbus@latest

# Homebrew 安装
brew install termbus/tap/termbus

# Paru 安装（Arch Linux/AUR）
paru -S termbus-bin
```

### Windows
```bash
# 1. 通过 npm/yarn 安装（推荐）
npm install -g termbus
# 或
yarn global add termbus

# 2. 通过 Scoop 安装（Windows 版 brew）
scoop bucket add termbus https://github.com/skye-z/termbus-scoop.git
scoop install termbus

# 3. 通过 curl 下载二进制文件
# 适用于 x64 架构
curl -L https://github.com/skye-z/termbus/releases/latest/download/termbus_windows_amd64.exe -o termbus.exe
# 添加到系统 PATH（或移动到 C:\Windows\System32 目录）
move termbus.exe C:\Windows\System32\

# 4. 通过 Bun 安装
bun install -g termbus
```

## 🎮 快速上手
1. 启动 Termbus：`tb`
2. 核心命令（输入 `:` 打开命令栏）：
   - `:connect <host>`: 连接远程主机
   - `:sftp`: 打开 SFTP 文件浏览器
   - `:port 8080:localhost:80`: 创建本地端口转发
   - `:ai <prompt>`: 使用 AI 助手执行远程运维操作
3. 常用快捷键：
   - `Ctrl+N`: 新建会话标签页
   - `Ctrl+S/V`: 分割窗格（水平/垂直）
   - `Tab`: 切换窗格焦点
   - `?`: 打开帮助菜单

## 📋 运行前提
- Windows：Windows 10/11（64位）、PowerShell 5.1+ 或命令提示符
- WSL2：推荐使用 Ubuntu/Debian/Arch（可获得完整功能支持）
- npm/yarn：需要 Node.js 16+（仅 npm/yarn 安装方式需要）

## 🤝 贡献指南
欢迎各类贡献！请参考 [CONTRIBUTING.md](CONTRIBUTING.md) 了解 PR、Issue 和代码风格的规范。

## 📄 许可证
Termbus 基于 **MIT 许可证** 开源——可免费用于个人/商业用途。