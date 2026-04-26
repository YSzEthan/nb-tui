package ui

import (
	"context"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"netbird-tui/internal/config"
	"netbird-tui/internal/netbird"
	"netbird-tui/internal/svc"
)

// --- messages ---

type statusMsg struct {
	status      *netbird.Status
	isSvcActive bool
	profileName string
}

type errMsg struct{ err error }
type actionDoneMsg struct{ info string }
type profileListMsg struct{ names []string }

// --- tea.Cmds ---

func tickCmd(ctx context.Context, activePath string) tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return fetchStatusCmd(ctx, activePath)()
	})
}

func fetchStatusCmd(ctx context.Context, activePath string) tea.Cmd {
	return func() tea.Msg {
		s, err := netbird.GetStatus(ctx)
		if err != nil {
			return errMsg{err}
		}
		profile, _ := config.CurrentProfileName(activePath)
		return statusMsg{
			status:      s,
			isSvcActive: svc.IsActive(),
			profileName: profile,
		}
	}
}

func svcStartCmd() tea.Cmd {
	return func() tea.Msg {
		if err := svc.Start(); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"服務已啟動"}
	}
}

func svcStopCmd() tea.Cmd {
	return func() tea.Msg {
		if err := svc.Stop(); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"服務已停止"}
	}
}

func connectCmd() tea.Cmd {
	return func() tea.Msg {
		if err := netbird.Up(); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"已連線"}
	}
}

func disconnectCmd() tea.Cmd {
	return func() tea.Msg {
		if err := netbird.Down(); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"已斷線"}
	}
}

func logoutCmd(activePath string) tea.Cmd {
	return func() tea.Msg {
		if err := config.CoreLogout(activePath); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"已登出"}
	}
}

func switchProfileCmd(name, activePath string) tea.Cmd {
	return func() tea.Msg {
		src := config.ProfilePath(name)
		if err := svc.Stop(); err != nil {
			return errMsg{err}
		}
		if err := config.WriteActive(src, activePath); err != nil {
			return errMsg{err}
		}
		if err := svc.Start(); err != nil {
			return errMsg{err}
		}
		if err := netbird.Up(); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"已切換至 " + name}
	}
}

func saveProfileCmd(name, activePath string) tea.Cmd {
	return func() tea.Msg {
		if err := config.SaveProfile(name, activePath); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"已儲存為 " + name}
	}
}

func loadProfileListCmd() tea.Cmd {
	return func() tea.Msg {
		names, err := config.ListProfiles()
		if err != nil {
			return errMsg{err}
		}
		return profileListMsg{names}
	}
}

func sshCmd(peer string) tea.Cmd {
	cmd := netbird.SSHCmd(peer)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"SSH 結束"}
	})
}

func sudoWarmCmd() tea.Cmd {
	return func() tea.Msg {
		if netbird.NeedsSudoPrompt() {
			return actionDoneMsg{"需要 sudo 授權"}
		}
		return nil
	}
}

func startKeepaliveCmd(ctx context.Context, interval time.Duration) tea.Cmd {
	return func() tea.Msg {
		netbird.StartKeepalive(ctx, interval)
		return nil
	}
}

func sudoRefreshProc() *exec.Cmd {
	return exec.Command("sudo", "-v")
}

func toggleSSHCmd(enable bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if enable {
			err = netbird.EnableSSH()
		} else {
			err = netbird.DisableSSH()
		}
		if err != nil {
			return errMsg{err}
		}
		label := "SSH 伺服器已關閉"
		if enable {
			label = "SSH 伺服器已開啟"
		}
		return actionDoneMsg{label}
	}
}
