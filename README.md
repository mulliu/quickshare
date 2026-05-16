# QuickShare

> Zero-setup LAN file sharing — double-click to start, scan QR with your phone, done.
> 双击即用，扫码即传，无需任何配置。

Double-click the exe, your browser opens automatically with a QR code. Scan it with your phone — upload files, download files, share text, all over your local network at full speed. No app installation, no account, no cloud.

双击运行，浏览器自动打开页面。掏出手机扫码，传文件、收文件、发文字，全部局域网直连跑满带宽。无需安装 App，无需账号，不上传云端。

## Quick Start / 快速开始

**Windows（推荐）**：从 Releases 下载 `quickshare-windows-x86_64.exe`，双击运行即可。

**终端 / Terminal**：
```bash
# Download from Releases, then:
./quickshare

# Or with Go installed:
go install github.com/mulliu/quickshare@latest
quickshare
```

## Features / 功能

| English | 中文 |
|---------|------|
| **Upload** — Select files from your phone, streamed directly to disk with no memory buffering. Supports files up to 4GB (configurable). | **上传** — 手机选文件直接写入磁盘，不占内存。支持最大 4GB（可配置）。 |
| **Download** — Files appear in a list on the phone. Tap to download via browser. | **下载** — 文件列表实时刷新，点击即下载到手机。 |
| **Text sharing** — Type on either device, text syncs to the other in real time. | **文本共享** — 任意一端输入文字，另一端实时同步显示。 |
| **Pre-share files** — Use `-s file.pdf` to make a file available before anyone connects. | **预设发送** — 启动时用 `-s` 参数预置文件，对方连接即可下载。 |
| **Auto-scan existing** — Files in the output directory are auto-registered on startup. | **自动扫描** — 启动时自动注册下载目录中的已有文件。 |
| **QR in terminal & web** — QR code printed in terminal and also shown on the web page. | **双二维码** — 终端和网页同时显示二维码。 |
| **Auto-cleanup** — Expired files are removed automatically (default 1h TTL). | **自动清理** — 文件到期自动删除（默认 1 小时）。 |
| **Auto-shutdown** — Server shuts down when browser tab is closed (8s timeout). | **自动关停** — 关闭浏览器页面后服务自动退出，不留孤儿进程。 |

## Usage / 使用方法

```
quickshare [flags]

Flags:
  -p int        port (default: try 8080, 3000, 8000, then random / 默认尝试 8080、3000、8000，最后随机)
  -o string     output directory (default / 默认: ./downloads)
  -max-size int max upload size in bytes (default / 默认: 4GB)
  -ttl duration file TTL before auto-cleanup (default / 默认: 1h, 0 = no cleanup / 不清理)
  -s string     pre-share a file at startup / 启动时预设一个文件
  -n            don't auto-open browser on Windows / 不自动打开浏览器
```

### Examples / 示例

```bash
# Default / 默认启动
quickshare

# Custom port / 指定端口
quickshare -p 9000

# Custom output directory / 指定下载目录
quickshare -o ~/Downloads/incoming

# Pre-share a file / 预设文件
quickshare -s ./presentation.pdf

# 8GB limit, no cleanup / 限制 8GB，不自动清理
quickshare -max-size 8589934592 -ttl 0
```

## How It Works / 工作原理

1. QuickShare starts an HTTP server on your LAN / 在局域网启动 HTTP 服务
2. Terminal prints URL + QR code / 终端显示 URL 和二维码
3. Phone scans QR or opens URL / 手机扫码或输入网址
4. Upload, download, or share text over LAN / 在网页上上传、下载或共享文本
5. Transfer is direct peer-to-peer at maximum LAN speed / 局域网直连，速度跑满带宽

### Upload / 上传（手机 → 电脑）
Tap the upload area on your phone, select a file — it's saved to `./downloads` on the computer. Large files are streamed to disk without buffering in memory.

手机端点击上传区域，选择文件后直接流式写入电脑磁盘，不占用内存。

### Download / 下载（电脑 → 手机）
Files appear in the "Available to download" list on the phone. Tap to download. Use `-s file.pdf` to pre-seed a file before anyone connects.

文件显示在手机页面的下载列表中，点击即下载。也可以用 `-s` 参数预设文件。

### Text Sharing / 文本共享
Type text on either device, tap share — it syncs to the other side in real time (polling every 2s).

任意一端输入文字，点击共享后实时同步到另一端（2 秒轮询）。

### Auto-Shutdown / 自动关停
The server monitors heartbeat signals from the browser. When all browser tabs are closed, the server shuts down after 8 seconds of inactivity — no orphan processes.

服务通过心跳检测浏览器是否存活。所有页面关闭后 8 秒无心跳，服务自动退出，不留孤儿进程。

## Building from Source / 源码编译

Requires Go 1.26+. / 需要 Go 1.26+。

```bash
git clone https://github.com/mulliu/quickshare
cd quickshare
go build -o quickshare .
```

Cross-compile for all platforms / 跨平台编译:

```bash
make build-all
```

## Why QuickShare? / 为什么用 QuickShare？

| vs | QuickShare | PairDrop | Python http.server | LocalSend |
|----|-----------|----------|-------------------|-----------|
| Start / 启动 | 1 command | Node.js/Docker | `python3 -m http.server` | Requires install |
| Phone app / 手机 App | ❌ | ❌ | ❌ | ✅ |
| Phone upload / 手机上传感 | ✅ | ❌ | ❌ | ✅ |
| Terminal QR / 终端二维码 | ✅ | ❌ | ❌ | ❌ |
| File size / 文件大小 | Configurable / 可配 | Browser limited / 浏览器限制 | Unlimited / 无限制 | Unlimited / 无限制 |
| Speed / 速度 | Fastest (raw TCP) | Medium (WebRTC) | Fast | Fast |

## License / 许可证

MIT
