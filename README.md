# Termbus
> A terminal-first DevOps SSH shell — the "remote operating system" with SSH as the transport bus.

Termbus is a modern, TUI-based SSH client built with Go, designed to be a comprehensive "remote operating system" centered around the SSH protocol. It unifies SSH, SFTP, port forwarding, multi-session management, and plugin extensibility into a single terminal-native experience, following Unix tool philosophy and DevOps workflows.

## ✨ Core Features
- **Full SSH Capabilities**: Password/key/agent auth, SSH config support, jump hosts, connection multiplexing.
- **Terminal-First TUI**: Responsive UI (like lazygit/k9s), tmux-style panes/tabs, keyboard-first navigation.
- **SFTP File Manager**: Two-pane browser, upload/download, incremental sync, built-in editor.
- **Port Forwarding**: Local/remote/dynamic SOCKS tunnels with visual management.
- **Multi-Session Management**: Concurrent connections, session persistence, auto-reconnect.
- **Plugin System**: Extensible with Docker/K8s/Redis/MySQL plugins (process-isolated, secure).
- **AI Agent (Beta)**: Natural language-driven remote ops with safety sandbox.
- **Scriptable**: Batch execution, command aliases, CI/CD integration.

## 🚀 Quick Install
### macOS/Linux
```bash
# Go install (requires Go 1.21+)
go install github.com/skye-z/termbus/cmd/termbus@latest

# Homebrew
brew install termbus/tap/termbus

# Paru (Arch Linux/AUR)
paru -S termbus-bin
```

### Windows
```bash
# 1. Via npm/yarn (recommended)
npm install -g termbus
# or
yarn global add termbus

# 2. Via Scoop (Windows brew alternative)
scoop bucket add termbus https://github.com/skye-z/termbus-scoop.git
scoop install termbus

# 3. Via curl (direct binary download)
# For x64
curl -L https://github.com/skye-z/termbus/releases/latest/download/termbus_windows_amd64.exe -o termbus.exe
# Add to PATH (or move to C:\Windows\System32)
move termbus.exe C:\Windows\System32\

# 4. Via Bun
bun install -g termbus
```

## 🎮 Quick Start
1. Launch Termbus: `tb`
2. Core commands (type `:` to open command bar):
   - `:connect <host>`: Connect to a remote host
   - `:sftp`: Open SFTP file browser
   - `:port 8080:localhost:80`: Create local port forward
   - `:ai <prompt>`: Use AI agent for remote ops
3. Key shortcuts:
   - `Ctrl+N`: New session tab
   - `Ctrl+S/V`: Split pane (horizontal/vertical)
   - `Tab`: Switch focus between panes
   - `?`: Open help menu

## 📋 Prerequisites
- Windows: Windows 10/11 (64-bit), PowerShell 5.1+ or Command Prompt
- WSL2: Ubuntu/Debian/Arch (recommended for full feature support)
- npm/yarn: Node.js 16+ (for npm/yarn installation)

## 🤝 Contributing
Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on PRs, issues, and code style.

## 📄 License
Termbus is open-source under the **MIT License** — free for personal/commercial use.
