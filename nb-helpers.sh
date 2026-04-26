#!/usr/bin/env bash
# nb-helpers.sh — shared helpers for the `nb` NetBird control tool

PROFILE_DIR="$HOME/.config/netbird-profiles"
LAST_LOGOUT="$PROFILE_DIR/.last-logout.json"

# ── platform detection ───────────────────────────────────────────
OS="$(uname -s)"   # Linux | Darwin

detect_active_config() {
    local candidates=("/var/lib/netbird/default.json" "/etc/netbird/config.json")
    for f in "${candidates[@]}"; do
        sudo test -f "$f" 2>/dev/null && { echo "$f"; return 0; }
    done
    # fallback: parse --config from service definition
    if [[ $OS == Linux ]] && command -v systemctl >/dev/null 2>&1; then
        systemctl cat netbird 2>/dev/null \
            | grep -oE -- '--config[= ][^ "]+' | head -1 | awk '{print $NF}' | tr -d '='
    else
        echo "/var/lib/netbird/default.json"
    fi
}
ACTIVE="$(detect_active_config)"

svc_is_active() {
    case "$OS" in
        Linux)  systemctl is-active netbird 2>/dev/null || echo inactive ;;
        Darwin) launchctl print system/io.netbird.client 2>/dev/null \
                    | grep -q 'state = running' && echo active || echo inactive ;;
    esac
}
svc_start() {
    case "$OS" in
        Linux)  sudo systemctl start netbird ;;
        Darwin) sudo launchctl bootstrap system \
                    /Library/LaunchDaemons/io.netbird.client.plist ;;
    esac
}
svc_stop() {
    case "$OS" in
        Linux)  sudo systemctl stop netbird ;;
        Darwin) sudo launchctl bootout system/io.netbird.client 2>/dev/null || true ;;
    esac
}
CACHE="/tmp/nb-status-${USER}.cache"
CACHE_TMP="$CACHE.tmp"

mkdir -p "$PROFILE_DIR"

# ── state checks — all read from cache (no sudo at menu-build time) ──
is_active()    { grep -q "^service=active"    "$CACHE" 2>/dev/null; }
has_login()    { grep -q "^login=yes"         "$CACHE" 2>/dev/null; }
is_connected() { grep -q "^connected=yes"     "$CACHE" 2>/dev/null; }
cached_ip()    { grep "^ip=" "$CACHE" 2>/dev/null | cut -d= -f2; }
cached_mgmt()  { grep "^mgmt=" "$CACHE" 2>/dev/null | cut -d= -f2-; }
cached_profile(){ grep "^profile=" "$CACHE" 2>/dev/null | cut -d= -f2-; }

# ── data extraction (used by daemon with active sudo) ────────────
mgmt_url() {
    local f=${1:-$ACTIVE} reader=jq
    [[ $f == "$ACTIVE" ]] && reader="sudo jq"
    $reader -r '
      if (.ManagementURL | type) == "object"
      then "\(.ManagementURL.Scheme)://\(.ManagementURL.Host)\(.ManagementURL.Path // "")"
      else (.ManagementURL // "?")
      end
    ' "$f" 2>/dev/null || echo "?"
}

profile_fingerprint() {
    local f=$1 reader=jq
    [[ $f == "$ACTIVE" ]] && reader="sudo jq"
    $reader -r '
      ( if (.ManagementURL | type) == "object"
        then "\(.ManagementURL.Scheme)://\(.ManagementURL.Host)\(.ManagementURL.Path // "")"
        else (.ManagementURL // "")
        end ) + "|" + (.PrivateKey // "") + "|" + (.SetupKey // "")
    ' "$f" 2>/dev/null | sha256sum | awk '{print $1}'
}

current_profile_live() {
    local active_fp; active_fp=$(profile_fingerprint "$ACTIVE") || return
    [[ -n $active_fp ]] || return
    local f
    for f in "$PROFILE_DIR"/*.json; do
        [[ -e $f && $(basename "$f") != .* ]] || continue
        [[ "$(profile_fingerprint "$f")" == "$active_fp" ]] && { basename "$f" .json; return; }
    done
}

# ── shared core operations ───────────────────────────────────────
save_profile_file() {
    local name=$1
    sudo cp "$ACTIVE" "$PROFILE_DIR/$name.json"
    sudo chown "$USER:" "$PROFILE_DIR/$name.json"
    chmod 600 "$PROFILE_DIR/$name.json"
}

apply_profile() {
    local src=$1
    svc_stop
    sudo install -m 600 -o root -g root "$src" "$ACTIVE"
    svc_start
    sudo netbird up
}

core_logout() {
    sudo netbird down 2>/dev/null || true
    sudo mv "$ACTIVE" "$LAST_LOGOUT"
    sudo chown "$USER:" "$LAST_LOGOUT"
    chmod 600 "$LAST_LOGOUT"
}

fzf_pick_id() { fzf --delimiter='|' --with-nth=2.. "$@" | cut -d'|' -f1; }

# ── daemon: write cache (machine flags + human display) ──────────
render_cache() {
    local nb_out nb_detail svc login connected ip mgmt profile

    svc=$(svc_is_active)
    mgmt=$(mgmt_url "$ACTIVE")

    nb_out=$(netbird status 2>/dev/null || true)
    nb_detail=$(netbird status -d 2>/dev/null || true)
    nb_json=$(netbird status --json 2>/dev/null || true)

    # daemonStatus is the authoritative signal:
    #   NeedsLogin → not logged in; Connected/Idle/Disconnected → logged in
    local dstatus
    dstatus=$(echo "$nb_json" | jq -r '.daemonStatus // ""' 2>/dev/null)
    case "$dstatus" in
        NeedsLogin|LoginFailed) login=no ;;
        "")
            # daemon unreachable — fall back to credential presence
            if sudo jq -e '.PrivateKey | length > 0' "$ACTIVE" >/dev/null 2>&1; then
                login=yes
            else
                login=no
            fi
            ;;
        *) login=yes ;;
    esac

    connected=no; ip=""
    if echo "$nb_out" | grep -q "Management: Connected"; then
        connected=yes
        ip=$(echo "$nb_out" | awk '/^NetBird IP/{print $NF; exit}')
    fi

    profile=$(current_profile_live || true)

    # Machine-readable flags (read by is_active / has_login / is_connected)
    cat <<EOF
service=$svc
login=$login
connected=$connected
ip=$ip
mgmt=$mgmt
profile=${profile:-(unsaved)}
---
EOF

    # Human-readable display (cat'd by fzf preview)
    printf 'Profile:   %-20s %s\n' "${profile:-(unsaved)}" "$mgmt"
    [[ $svc == active ]]    && echo "Service:   ● 開啟"   || echo "Service:   ○ 未開啟"
    [[ $login == yes ]]     && echo "Login:     ● 已登入" || echo "Login:     ○ 未登入"
    [[ $connected == yes ]] && echo "Connected: ● $ip"    || echo "Connected: ○ 未連線"
    echo

    # Peers — only name / IP / status (no raw detail)
    local peers_summary
    peers_summary=$(echo "$nb_detail" | awk '
        /^ [A-Za-z]/ && /:$/ {
            name=$0; gsub(/^ +/,"",name); gsub(/:$/,"",name); gsub(/\.netbird\.cloud$/,"",name)
        }
        /^  NetBird IP:/ { ip=$NF }
        /^  Status:/     { printf "  %-22s %-18s %s\n", name, ip, $2 }
    ')
    if [[ -n $peers_summary ]]; then
        echo "Peers:"
        echo "$peers_summary"
    else
        echo "Peers: (none)"
    fi
}

status_daemon() {
    while :; do
        sudo -v 2>/dev/null || true
        render_cache > "$CACHE_TMP" 2>/dev/null && mv -f "$CACHE_TMP" "$CACHE"
        sleep 2
    done
}

start_daemon() {
    sudo -v 2>/dev/null || true
    status_daemon &
    DAEMON_PID=$!
    trap 'kill $DAEMON_PID 2>/dev/null; rm -f "$CACHE" "$CACHE_TMP"' EXIT INT TERM
    local i=0
    until [[ -s $CACHE ]] || (( i++ > 15 )); do sleep 0.2; done
}

# ── state-aware menu (id|label) ──────────────────────────────────
build_menu() {
    if ! is_active; then
        printf '%s\n' "start_svc|▶ Start service" "switch|⇄ Switch profile..." "quit|✕ Quit"
        return
    fi
    if ! has_login; then
        printf '%s\n' "login|⇥ Login (OAuth)" "switch|⇄ Switch profile..." "stop_svc|■ Stop service" "quit|✕ Quit"
        return
    fi
    is_connected && echo "disconnect|⏹ Disconnect" || echo "connect|▶ Connect"
    echo "switch|⇄ Switch profile..."
    is_connected && echo "peers|⊞ Peers..."
    printf '%s\n' "save|⊕ Save as..." "logout|⇤ Logout" "stop_svc|■ Stop service" "quit|✕ Quit"
}

# ── nb status (CLI) ──────────────────────────────────────────────
render_status() {
    grep -A 999 '^---$' "$CACHE" 2>/dev/null | tail -n +2 \
        || { echo "Cache not ready — run 'nb' (TUI) first or wait 2s"; return 1; }
}

# ── interactive sub-UIs ──────────────────────────────────────────
switch_profile_ui() {
    local profiles
    mapfile -t profiles < <(find "$PROFILE_DIR" -maxdepth 1 -name '*.json' ! -name '.*' 2>/dev/null)
    if (( ${#profiles[@]} == 0 )); then
        printf '\nNo profiles yet — use "Save as..." first.\n'; sleep 1.5; return
    fi
    local lines=() f name url
    for f in "${profiles[@]}"; do
        name=$(basename "$f" .json)
        url=$(mgmt_url "$f")
        lines+=("$(printf '%s|%-20s  %s' "$name" "$name" "$url")")
    done
    local pick
    pick=$(printf '%s\n' "${lines[@]}" \
        | fzf_pick_id --reverse \
              --header="Switch profile  (Enter=apply  Esc=cancel)" \
              --preview="jq -r '\"Mgmt: \\(.ManagementURL // \"?\")\"' \"$PROFILE_DIR/{1}.json\" 2>/dev/null" \
              --preview-window=right:40%) || return
    [[ -n $pick && -f "$PROFILE_DIR/$pick.json" ]] || return
    if [[ "$(cached_profile)" == "$pick" ]]; then
        printf '\nAlready on %s\n' "$pick"; sleep 1; return
    fi
    apply_profile "$PROFILE_DIR/$pick.json"
}

peers_ui() {
    local peer name
    peer=$(netbird status -d 2>/dev/null \
        | awk '
            /^ [A-Za-z]/ && /:$/ {
                name=$0; gsub(/^ +/,"",name); gsub(/:$/,"",name); gsub(/\.netbird\.cloud$/,"",name)
            }
            /^  NetBird IP:/ { ip=$NF }
            /^  Status:/     { printf "%-22s %-18s %s\n", name, ip, $2 }
        ' \
        | fzf --reverse --header="Peers  (Enter=ssh  Esc=back)") || return
    [[ -n $peer ]] || return
    name=$(awk '{print $1}' <<< "$peer")
    [[ -n $name ]] || return
    clear
    sudo netbird ssh "$name" || true
    read -rsn1 -p "(SSH ended; press any key)"
}

save_ui() {
    has_login || { printf '\nNot logged in.\n'; sleep 1; return; }
    local name
    read -rp "Profile name: " name
    [[ -n $name ]] || return
    if [[ -f "$PROFILE_DIR/$name.json" ]]; then
        local yn
        yn=$(printf 'no|Cancel\nyes|Overwrite %s\n' "$name" \
            | fzf_pick_id --reverse --header="Profile exists") || return
        [[ $yn == yes ]] || return
    fi
    save_profile_file "$name"
}

logout_ui() {
    local yn
    yn=$(printf 'no|No, cancel\nyes|Yes — move credentials to .last-logout.json\n' \
        | fzf_pick_id --reverse --header="Confirm logout?") || return
    [[ $yn == yes ]] || return
    core_logout 2>/dev/null || true
}

restore_last() {
    [[ -f $LAST_LOGOUT ]] || { echo "No .last-logout.json to restore." >&2; return 1; }
    svc_stop
    sudo install -m 600 -o root -g root "$LAST_LOGOUT" "$ACTIVE"
    svc_start
    rm -f "$LAST_LOGOUT"
    echo "Restored last logout credentials."
}
