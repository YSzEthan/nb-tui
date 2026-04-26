# nb-tui

NetBird 互動式控制中心（Linux）。fzf 雙窗格 TUI + 完整 CLI。

NetBird interactive control center for Linux. fzf double-pane TUI + full CLI.

---

## 需求 / Requirements

- `netbird` — [安裝說明](https://docs.netbird.io/how-to/installation)
- `fzf` — `brew install fzf`
- `jq` — `brew install jq`

---

## 安裝 / Install

```bash
git clone https://github.com/<you>/nb-tui.git ~/code/nb-tui
ln -sf ~/code/nb-tui/nb ~/.local/bin/nb
```

---

## 功能 / Features

- **狀態感知選單**：依當前狀態自動顯示/隱藏選項（已連線就看不到 Connect）
  State-aware menu: only shows relevant actions based on current state

- **即時狀態預覽**：Service / Login / Connected + peer 列表（背景每 2 秒更新）
  Live status preview updated every 2s via background daemon

- **多帳號 profile**：`~/.config/netbird-profiles/` 存放，mode 600
  Multi-account profiles stored at `~/.config/netbird-profiles/` (mode 600)

- **完整 CLI 模式**：可在 script 中使用
  Full CLI mode for scripting

---

## 互動式 TUI / Interactive TUI

```
打 `nb`（無參數）進入 TUI
Type `nb` (no args) to launch TUI

┌─ nb ──────────────┬─ Status (work) ──────────────────┐
│ > Disconnect      │ Profile:   work                   │
│   Switch profile..│ Service:   ● 開啟                 │
│   Peers...        │ Login:     ● 已登入               │
│   Save as...      │ Connected: ● 100.92.0.5           │
│   Logout          │                                   │
│   Stop service    │ Peers: 3/5 online                 │
│   Quit            │   laptop-mike   100.92.0.5  ● on  │
│                   │   server-prod   100.92.0.10 ● on  │
│                   │   phone-john    100.92.0.20 ○ off │
└───────────────────┴───────────────────────────────────┘
 ↑↓ Enter   /=篩選   Ctrl-R=刷新   Esc=離開
```

---

## CLI 指令 / CLI Commands

```bash
# 進入 TUI / Launch TUI
nb

# 狀態 / Status
nb status                # 顯示 Service / Login / Connected 四行

# 連線 / Connection
nb up                    # netbird up（登入/連線）
nb down                  # netbird down（斷線）
nb start                 # systemctl start netbird（啟動服務）
nb stop                  # systemctl stop netbird（停用服務）

# 帳號 / Account
nb logout                # 將憑證移至 .last-logout.json
nb restore-last          # 從 .last-logout.json 還原憑證

# Profile 管理 / Profile management
nb ls                    # 列出所有 profile（★ = 當前）
nb current               # 印當前 profile 名
nb save <name>           # 存當前 config 為 <name>
nb use [<name>]          # 切換 profile（無參數跳 fzf）
nb rename <old> <new>    # 重新命名
nb rm <name>             # 刪除
```

---

## Profile 安全設計 / Profile Security

Profile 存放位置 / Storage: `~/.config/netbird-profiles/<name>.json`（mode 600）

比對邏輯只用三個欄位 / Identity matching uses only 3 fields:
- `ManagementURL`
- `PrivateKey`
- `SetupKey`

登出流程 / Logout flow:
- `nb logout` → 將 `/etc/netbird/config.json` **移動**（不刪除）至 `.last-logout.json`
- `nb restore-last` → 還原，無需重新 OAuth

---

## 狀態感知選單對照 / State-aware Menu Matrix

| 狀態 State | 可用選項 Available actions |
|---|---|
| 服務未開 Service ○ | Start service / Switch profile / Quit |
| 服務開 + 未登入 Login ○ | Login (OAuth) / Switch profile / Stop service / Quit |
| 服務開 + 已登入 + 未連線 | Connect / Switch profile / Save as / Logout / Stop service / Quit |
| 服務開 + 已登入 + 已連線 | Disconnect / Switch profile / Peers / Save as / Logout / Stop service / Quit |
