// Package ui implements the Bubble Tea TUI for NetBird-TUI.
// The single model holds all view states (viewMain/viewSwitch/viewPeers/viewSave/viewConfirm).
package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"netbird-tui/internal/config"
	"netbird-tui/internal/netbird"
)

// viewState represents which screen is active.
type viewState int

const (
	viewUnknown viewState = iota // zero value sentinel — uninitialized model
	viewMain
	viewSwitch
	viewPeers
	viewSave
	viewConfirm
)

// menuItem implements list.Item.
type menuItem struct {
	id    string
	label string
}

func (m menuItem) Title() string       { return m.label }
func (m menuItem) Description() string { return "" }
func (m menuItem) FilterValue() string { return m.label }

// model is the root Bubble Tea model.
type model struct {
	ctx        context.Context
	activePath string

	// current view
	state viewState

	// main view
	menu        list.Model
	lastStatus  *netbird.Status
	isSvcActive bool
	profileName string

	// switch view
	profiles    []string
	profileList list.Model

	// peers view
	peers    []netbird.Peer
	peerList list.Model

	// save view
	input textinput.Model

	// confirm view
	confirmMsg    string
	confirmAction func() tea.Cmd
	confirmCursor int // 0=yes, 1=no

	// status bar
	statusBar    string
	statusExpiry time.Time

	width, height int
}

func initialModel(ctx context.Context) model {
	activePath := config.DetectActive()

	// Main menu list (items populated dynamically)
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	menuList := list.New(nil, delegate, 0, 0)
	menuList.SetShowTitle(false)
	menuList.SetShowHelp(false)
	menuList.SetShowStatusBar(false)
	menuList.SetFilteringEnabled(false)

	// Profile list
	profList := list.New(nil, delegate, 0, 0)
	profList.SetShowTitle(true)
	profList.Title = "選擇 Profile"
	profList.SetShowHelp(false)
	profList.SetShowStatusBar(false)
	profList.SetFilteringEnabled(true)

	// Peer list
	peerListM := list.New(nil, delegate, 0, 0)
	peerListM.SetShowTitle(true)
	peerListM.Title = "Peers（Enter=SSH, q=返回）"
	peerListM.SetShowHelp(false)
	peerListM.SetShowStatusBar(false)
	peerListM.SetFilteringEnabled(false)

	// Text input for save
	ti := textinput.New()
	ti.Placeholder = "profile 名稱"
	ti.CharLimit = 64

	return model{
		ctx:         ctx,
		activePath:  activePath,
		state:       viewMain,
		menu:        menuList,
		profileList: profList,
		peerList:    peerListM,
		input:       ti,
	}
}

// Run starts the Bubble Tea program.
func Run(ctx context.Context) error {
	m := initialModel(ctx)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchStatusCmd(m.ctx, m.activePath), startKeepaliveCmd(m.ctx, 60*time.Second))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.menu.SetSize(msg.Width/2-2, msg.Height-5)
		m.profileList.SetSize(msg.Width-4, msg.Height-5)
		m.peerList.SetSize(msg.Width-4, msg.Height-5)

	case statusMsg:
		m.lastStatus = msg.status
		m.isSvcActive = msg.isSvcActive
		m.profileName = msg.profileName
		m.rebuildMenu()
		if m.state == viewPeers && msg.status != nil {
			m.updatePeerList(netbird.PeersFromStatus(msg.status))
		}
		return m, tickCmd(m.ctx, m.activePath)

	case errMsg:
		m.setStatus(styleError.Render("錯誤: " + msg.err.Error()))
		return m, tickCmd(m.ctx, m.activePath)

	case actionDoneMsg:
		m.setStatus(msg.info)
		return m, fetchStatusCmd(m.ctx, m.activePath)

	case profileListMsg:
		m.profiles = msg.names
		items := make([]list.Item, len(msg.names))
		for i, n := range msg.names {
			items[i] = menuItem{id: n, label: n}
		}
		m.profileList.SetItems(items)
		m.state = viewSwitch

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case viewMain:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			return m, fetchStatusCmd(m.ctx, m.activePath)
		case "enter":
			if i, ok := m.menu.SelectedItem().(menuItem); ok {
				return m.handleMenuAction(i.id)
			}
		}
		var cmd tea.Cmd
		m.menu, cmd = m.menu.Update(msg)
		return m, cmd

	case viewSwitch:
		switch msg.String() {
		case "q", "esc":
			m.state = viewMain
			return m, nil
		case "enter":
			if i, ok := m.profileList.SelectedItem().(menuItem); ok {
				m.state = viewMain
				return m, switchProfileCmd(i.id, m.activePath)
			}
		}
		var cmd tea.Cmd
		m.profileList, cmd = m.profileList.Update(msg)
		return m, cmd

	case viewPeers:
		switch msg.String() {
		case "q", "esc":
			m.state = viewMain
			return m, nil
		case "enter", "s":
			if i, ok := m.peerList.SelectedItem().(menuItem); ok {
				return m, sshCmd(i.id)
			}
		}
		var cmd tea.Cmd
		m.peerList, cmd = m.peerList.Update(msg)
		return m, cmd

	case viewSave:
		switch msg.String() {
		case "esc":
			m.state = viewMain
			m.input.Blur()
			return m, nil
		case "enter":
			name := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			m.input.Blur()
			m.state = viewMain
			if name != "" {
				return m, saveProfileCmd(name, m.activePath)
			}
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case viewConfirm:
		switch msg.String() {
		case "left", "h":
			m.confirmCursor = 0
		case "right", "l":
			m.confirmCursor = 1
		case "enter":
			yes := m.confirmCursor == 0
			m.state = viewMain
			if yes && m.confirmAction != nil {
				return m, m.confirmAction()
			}
			return m, nil
		case "esc", "q":
			m.state = viewMain
			return m, nil
		}
	}
	return m, nil
}

func (m *model) handleMenuAction(id string) (tea.Model, tea.Cmd) {
	switch id {
	case "start_svc":
		return m, svcStartCmd()
	case "stop_svc":
		return m, svcStopCmd()
	case "login", "connect":
		if netbird.NeedsSudoPrompt() {
			return m, tea.ExecProcess(sudoRefreshProc(), func(err error) tea.Msg {
				if err != nil {
					return errMsg{err}
				}
				return actionDoneMsg{"sudo 已授權，請再次選擇連線"}
			})
		}
		return m, connectCmd()
	case "disconnect":
		return m, disconnectCmd()
	case "switch":
		return m, loadProfileListCmd()
	case "peers":
		m.state = viewPeers
		if m.lastStatus != nil {
			m.updatePeerList(netbird.PeersFromStatus(m.lastStatus))
		}
		return m, nil
	case "ssh_on":
		if m.lastStatus != nil {
			m.lastStatus.SSHServer.Enabled = true
		}
		m.rebuildMenu()
		return m, toggleSSHCmd(true)
	case "ssh_off":
		if m.lastStatus != nil {
			m.lastStatus.SSHServer.Enabled = false
		}
		m.rebuildMenu()
		return m, toggleSSHCmd(false)
	case "save":
		m.state = viewSave
		m.input.Focus()
		return m, textinput.Blink
	case "logout":
		m.confirmMsg = "確定要登出嗎？"
		m.confirmAction = func() tea.Cmd { return logoutCmd(m.activePath) }
		m.confirmCursor = 1
		m.state = viewConfirm
		return m, nil
	case "quit":
		return m, tea.Quit
	}
	return m, nil
}

func (m *model) rebuildMenu() {
	items := []list.Item{}
	if !m.isSvcActive {
		items = append(items, menuItem{"start_svc", "▶ 啟動服務"})
	} else {
		items = append(items, menuItem{"stop_svc", "■ 停止服務"})
		if m.lastStatus != nil {
			if !m.lastStatus.IsLoggedIn() {
				items = append(items, menuItem{"login", "⇥ 登入 (OAuth)"})
			} else if !m.lastStatus.IsConnected() {
				items = append(items, menuItem{"connect", "▶ 連線"})
			} else {
				items = append(items, menuItem{"disconnect", "⏹ 斷線"})
				items = append(items, menuItem{"peers", "⊞ Peers..."})
				if m.lastStatus.SSHEnabled() {
					items = append(items, menuItem{"ssh_off", "⛔ 關閉 SSH 伺服器"})
				} else {
					items = append(items, menuItem{"ssh_on", "⌨ 開啟 SSH 伺服器"})
				}
			}
		}
	}
	items = append(items,
		menuItem{"switch", "⇄ 切換 Profile..."},
		menuItem{"save", "⊕ 另存 Profile..."},
		menuItem{"logout", "⇤ 登出"},
		menuItem{"quit", "✕ 離開"},
	)
	m.menu.SetItems(items)
}

func (m *model) updatePeerList(peers []netbird.Peer) {
	m.peers = peers
	items := make([]list.Item, len(peers))
	for i, p := range peers {
		status := "○"
		if p.Connected {
			status = "●"
		}
		items[i] = menuItem{id: p.IP, label: fmt.Sprintf("%s %s  %s", status, p.Name, p.IP)}
	}
	m.peerList.SetItems(items)
}

func (m *model) setStatus(msg string) {
	m.statusBar = msg
	m.statusExpiry = time.Now().Add(3 * time.Second)
}

func (m model) View() string {
	if m.width == 0 {
		return "載入中..."
	}

	switch m.state {
	case viewSwitch:
		return styleBorder.Render(m.profileList.View())
	case viewPeers:
		return styleBorder.Render(m.peerList.View())
	case viewSave:
		return m.renderSave()
	case viewConfirm:
		return m.renderConfirm()
	}

	return m.renderMain()
}

func (m model) renderMain() string {
	menuPane := styleBorder.Width(m.width/2 - 4).Render(m.menu.View())
	statusPane := styleBorder.Width(m.width/2 - 4).Render(m.renderStatus())

	body := lipgloss.JoinHorizontal(lipgloss.Top, menuPane, statusPane)

	// Status bar
	bar := ""
	if m.statusBar != "" && time.Now().Before(m.statusExpiry) {
		bar = m.statusBar
	}
	help := styleHelp.Render("↑↓ 移動 · enter 選擇 · r 刷新 · q 離開")

	return lipgloss.JoinVertical(lipgloss.Left, body, help, bar)
}

func (m model) renderStatus() string {
	if m.lastStatus == nil {
		return styleInactive.Render("讀取中...")
	}
	s := m.lastStatus

	svcLine := styleInactive.Render("○ 未開啟")
	if m.isSvcActive {
		svcLine = styleActive.Render("● 開啟")
	}
	loginLine := styleInactive.Render("○ 未登入")
	if s.IsLoggedIn() {
		loginLine = styleActive.Render("● 已登入")
	}
	connLine := styleInactive.Render("○ 未連線")
	if s.IsConnected() {
		connLine = styleActive.Render("● " + s.IP)
	}
	profile := styleInactive.Render("(unsaved)")
	if m.profileName != "" {
		profile = styleTitle.Render(m.profileName)
	}

	lines := []string{
		styleTitle.Render("Profile:  ") + profile,
		"",
		"Service:  " + svcLine,
		"Login:    " + loginLine,
		"Conn:     " + connLine,
		"Mgmt:     " + styleHelp.Render(s.ManagementURL()),
		"SSH:      " + func() string {
			if s.SSHEnabled() {
				return styleActive.Render("● 開啟")
			}
			return styleInactive.Render("○ 關閉")
		}(),
	}

	if len(m.peers) > 0 && s.IsConnected() {
		lines = append(lines, "", styleTitle.Render("Peers:"))
		for _, p := range m.peers {
			icon := styleInactive.Render("○")
			if p.Connected {
				icon = styleActive.Render("●")
			}
			lines = append(lines, fmt.Sprintf("  %s %s  %s", icon, p.Name, styleHelp.Render(p.IP)))
		}
	}

	return strings.Join(lines, "\n")
}

func (m model) renderSave() string {
	content := lipgloss.JoinVertical(lipgloss.Left,
		styleTitle.Render("另存 Profile"),
		"",
		m.input.View(),
		"",
		styleHelp.Render("Enter 確認 · Esc 取消"),
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		styleBorder.Render(content))
}

func (m model) renderConfirm() string {
	yes := "  是  "
	no := "  否  "
	if m.confirmCursor == 0 {
		yes = styleSelected.Render("[ 是 ]")
	} else {
		no = styleSelected.Render("[ 否 ]")
	}
	content := lipgloss.JoinVertical(lipgloss.Center,
		styleTitle.Render(m.confirmMsg),
		"",
		lipgloss.JoinHorizontal(lipgloss.Top, yes, "  ", no),
		"",
		styleHelp.Render("← → 選擇 · Enter 確認 · Esc 取消"),
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		styleBorder.Render(content))
}
