# NetBird-TUI

互動式 TUI + CLI 工具，用來管理 [NetBird](https://netbird.io/) VPN 連線。以 Go 撰寫，shell out 呼叫 `netbird` CLI，支援 Linux 與 macOS。

---

## Features

- **互動式 TUI**：Bubble Tea 主畫面，左側選單 + 右側即時狀態，每 2 秒自動刷新
- **Profile 管理**：儲存、切換、改名、刪除 config profile（`~/.config/netbird-profiles/`）
- **Service 控制**：`start` / `stop` netbird daemon（Linux: systemctl，macOS: launchctl）
- **連線操作**：`up` / `down`，對應 `sudo netbird up/down`
- **SSH 互動**：TUI 暫停後進入 `sudo netbird ssh`，結束後自動復原
- **自動偵測 config 路徑**：
  1. `/var/lib/netbird/default.json`（NetBird 0.27+）
  2. `/etc/netbird/config.json`（legacy）
  3. `systemctl cat netbird` 解析 `--config` flag（Linux fallback）
- **登出 / 還原**：`logout` 備份目前 config，`restore-last` 一鍵還原

---

## Requirements

| 項目 | 版本 |
|---|---|
| `netbird` CLI | 任意（建議 0.27+） |
| `sudo` | 系統內建 |
| Go | 1.26+ |

---

## Install

```bash
git clone <this-repo>
cd nb-tui
make build          # 產生 ./NetBird-TUI
make install        # 安裝至 ~/.local/bin/NetBird-TUI
```

> 確保 `~/.local/bin` 在 `PATH` 裡。若沒有：
> ```bash
> echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc && source ~/.bashrc
> ```

---

## Usage

### 啟動 TUI

```bash
NetBird-TUI
```

主畫面鍵盤操作：

| 鍵 | 動作 |
|---|---|
| `↑` / `↓` | 移動選單 |
| `enter` | 選擇 |
| `r` | 立即刷新狀態 |
| `q` / `Ctrl-C` | 退出 |
| `s`（peers 畫面） | SSH 至選中的 peer |
| `Esc` | 取消 / 返回 |

### CLI 子指令

```
NetBird-TUI status       印出目前 NetBird 狀態
NetBird-TUI ls           列出所有 profile（★ 標示目前使用中）
NetBird-TUI current      印出目前使用的 profile 名稱
NetBird-TUI save <name>  將目前 config 儲存為 profile
NetBird-TUI use [<name>] 切換到指定 profile（無參數時印可用清單）
NetBird-TUI rename <old> <new>
NetBird-TUI rm <name>
NetBird-TUI up           連線 (sudo netbird up)
NetBird-TUI down         斷線 (sudo netbird down)
NetBird-TUI start        啟動 netbird service
NetBird-TUI stop         停止 netbird service
NetBird-TUI logout       登出並備份目前 config
NetBird-TUI restore-last 還原最後一次登出的 config
NetBird-TUI version      印出版本
```

---

## Configuration

### Active config path

程式啟動時依序探測：

1. `/var/lib/netbird/default.json`（NetBird 0.27+，Linux）
2. `/etc/netbird/config.json`（legacy，Linux）
3. 解析 `systemctl cat netbird` 取得 `--config <path>`（Linux fallback）
4. `/etc/netbird/config.json`（macOS）

### Profile 目錄

```
~/.config/netbird-profiles/
├── home.json          # 自訂名稱的 profile
├── office.json
└── .last-logout.json  # logout 備份（隱藏）
```

所有 profile 檔案 mode 600，由 user 擁有（私鑰保護）。

---

## Migration from bash `nb`

Profile 檔案格式相同，**不需遷移**。如果之前跑過舊的 bash 版，可選擇清掉背景 daemon 殘留檔：

```bash
rm -f /tmp/nb-status-*.cache
```

---

## Development

```bash
make vet            # go vet ./...
make vet-cross      # 同時 vet Linux + Darwin
make fmt            # gofmt -l -w .
make lint           # golangci-lint run ./... (需自行安裝)
make test           # go test ./...
make build          # 產生 ./NetBird-TUI (with ldflags version)
make help           # 列出所有 target
```

### Cross-build sanity check

```bash
GOOS=linux  go build ./...
GOOS=darwin go build ./...
```

---

## Architecture

```
cmd/netbird-tui/     Cobra root + 每個子指令一個檔
internal/
  netbird/           shell out 至 netbird CLI；Status/Peer 型別；sudo keepalive
  config/            偵測 active config 路徑；profile CRUD
  svc/               service start/stop（Linux: systemctl / macOS: launchctl）
  ui/                Bubble Tea model、views、commands（三個檔）
```
