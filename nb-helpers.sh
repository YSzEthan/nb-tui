#!/usr/bin/env bash
# nb-helpers.sh — shared helpers for the `nb` NetBird control tool

PROFILE_DIR="$HOME/.config/netbird-profiles"
ACTIVE="/etc/netbird/config.json"
LAST_LOGOUT="$PROFILE_DIR/.last-logout.json"
CACHE="/tmp/nb-status-${USER}.cache"
CACHE_TMP="$CACHE.tmp"

mkdir -p "$PROFILE_DIR"

# ── primitive checks ─────────────────────────────────────────────
is_active()    { systemctl is-active --quiet netbird; }
is_connected() { grep -q "Management: Connected" "$CACHE" 2>/dev/null; }
has_login()    { sudo -n jq -e '.PrivateKey | length > 0' "$ACTIVE" >/dev/null 2>&1; }

# ── data extraction ──────────────────────────────────────────────
mgmt_url() {
    local f=${1:-$ACTIVE} reader=jq
    [[ $f == "$ACTIVE" ]] && reader="sudo -n jq"
    $reader -r '.ManagementURL // "?"' "$f" 2>/dev/null || echo "?"
}

profile_fingerprint() {
    local f=$1 reader=jq
    [[ $f == "$ACTIVE" ]] && reader="sudo -n jq"
    $reader -r '"\(.ManagementURL // "")|\(.PrivateKey // "")|\(.SetupKey // "")"' "$f" 2>/dev/null \
        | sha256sum | awk '{print $1}'
}

current_profile() {
    local active_fp; active_fp=$(profile_fingerprint "$ACTIVE")
    [[ -n $active_fp ]] || return
    local f
    for f in "$PROFILE_DIR"/*.json; do
        [[ -e $f ]] || continue
        [[ "$(profile_fingerprint "$f")" == "$active_fp" ]] && { basename "$f" .json; return; }
    done
}

# ── shared core operations ───────────────────────────────────────
# Save current ACTIVE config to a named profile file
save_profile_file() {
    local name=$1
    sudo cp "$ACTIVE" "$PROFILE_DIR/$name.json"
    sudo chown "$USER:" "$PROFILE_DIR/$name.json"
    chmod 600 "$PROFILE_DIR/$name.json"
}

# Switch netbird to use a profile file (stop→install→start→up)
apply_profile() {
    local src=$1
    sudo systemctl stop netbird
    sudo install -m 600 -o root -g root "$src" "$ACTIVE"
    sudo systemctl start netbird
    sudo netbird up
}

# Move ACTIVE config to LAST_LOGOUT (stop first if connected)
core_logout() {
    sudo netbird down 2>/dev/null || true
    sudo mv "$ACTIVE" "$LAST_LOGOUT"
    sudo chown "$USER:" "$LAST_LOGOUT"
    chmod 600 "$LAST_LOGOUT"
}

# fzf wrapper: show labels (col 2+), return id (col 1)
fzf_pick_id() { fzf --delimiter='|' --with-nth=2.. "$@" | cut -d'|' -f1; }

# ── status snapshot ──────────────────────────────────────────────
render_status() {
    local cur mgmt nb_status connected ip

    cur=$(current_profile || true)
    mgmt=$(mgmt_url)

    # Single call — used for both connected check and peer list
    nb_status=$(netbird status -d 2>/dev/null || true)
    connected=$(echo "$nb_status" | grep -c "Management: Connected" || true)
    ip=$(echo "$nb_status" | awk '/^NetBird IP/ {print $NF; exit}')

    echo "Profile:   ${cur:-(unsaved)}   ${mgmt}"
    is_active  && echo "Service:   ● 開啟"   || echo "Service:   ○ 未開啟"
    has_login  && echo "Login:     ● 已登入" || echo "Login:     ○ 未登入"
    if (( connected > 0 )); then
        echo "Connected: ● ${ip:-yes}"
        echo "Management: Connected"
    else
        echo "Connected: ○ 未連線"
    fi
    echo
    echo "Peers:"
    echo "$nb_status" \
        | awk '/Peers detail:/{flag=1;next} flag && NF' \
        | head -20 \
        | sed 's/^/  /'
}

# ── background status daemon ─────────────────────────────────────
status_daemon() {
    while :; do
        # Refresh sudo ticket every cycle so sudo -n calls keep working
        sudo -v 2>/dev/null || true
        render_status > "$CACHE_TMP" 2>/dev/null && mv -f "$CACHE_TMP" "$CACHE"
        sleep 2
    done
}

start_daemon() {
    sudo -v 2>/dev/null || true
    status_daemon &
    DAEMON_PID=$!
    trap 'kill $DAEMON_PID 2>/dev/null; rm -f "$CACHE" "$CACHE_TMP"' EXIT INT TERM
    # Wait for first cache write before entering fzf
    local i=0
    until [[ -s $CACHE ]] || (( i++ > 10 )); do sleep 0.2; done
}

# ── state-aware menu (id|label format) ──────────────────────────
build_menu() {
    if ! is_active; then
        printf '%s\n' "start_svc|Start service" "switch|Switch profile..." "quit|Quit"
        return
    fi
    if ! has_login; then
        printf '%s\n' "login|Login (OAuth)" "switch|Switch profile..." "stop_svc|Stop service" "quit|Quit"
        return
    fi
    is_connected && echo "disconnect|Disconnect" || echo "connect|Connect"
    echo "switch|Switch profile..."
    is_connected && echo "peers|Peers..."
    printf '%s\n' "save|Save as..." "logout|Logout" "stop_svc|Stop service" "quit|Quit"
}

# ── interactive sub-UIs ──────────────────────────────────────────
switch_profile_ui() {
    local profiles
    mapfile -t profiles < <(find "$PROFILE_DIR" -maxdepth 1 -name '*.json' ! -name '.*' 2>/dev/null)
    if (( ${#profiles[@]} == 0 )); then
        printf '\nNo profiles yet — use "Save as..." first.\n' >&2
        sleep 1.5
        return
    fi

    local lines=() f name url
    for f in "${profiles[@]}"; do
        name=$(basename "$f" .json)
        url=$(mgmt_url "$f")
        lines+=("$(printf '%s|%-20s %s' "$name" "$name" "$url")")
    done

    local name
    name=$(printf '%s\n' "${lines[@]}" \
        | fzf_pick_id --reverse \
              --header="Switch profile (Enter=apply, Esc=cancel)" \
              --preview="jq -r '\"Mgmt: \\(.ManagementURL // \"?\")\"' \"$PROFILE_DIR/{1}.json\" 2>/dev/null" \
              --preview-window=right:40%) || return
    [[ -n $name && -f "$PROFILE_DIR/$name.json" ]] || return

    if [[ "$(current_profile)" == "$name" ]]; then
        printf '\nAlready on %s\n' "$name" >&2; sleep 1; return
    fi
    apply_profile "$PROFILE_DIR/$name.json"
}

peers_ui() {
    local peer name
    peer=$(netbird status -d 2>/dev/null \
        | awk '/Peers detail:/{flag=1;next} flag && NF' \
        | fzf --reverse \
              --header="Peers — Enter to ssh, Esc to back" \
              --preview-window=hidden) || return
    [[ -n $peer ]] || return
    name=$(awk '{print $1}' <<< "$peer")
    [[ -n $name ]] || return
    clear
    sudo netbird ssh "$name" || true
    read -rsn1 -p "(SSH ended; press any key)"
}

save_ui() {
    has_login || { printf '\nNot logged in.\n' >&2; sleep 1; return; }
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
    yn=$(printf 'no|No, cancel\nyes|Yes, logout (move credentials to .last-logout.json)\n' \
        | fzf_pick_id --reverse --header="Confirm logout?") || return
    [[ $yn == yes ]] || return
    core_logout 2>/dev/null || true
}

restore_last() {
    [[ -f $LAST_LOGOUT ]] || { echo "No .last-logout.json to restore." >&2; return 1; }
    sudo systemctl stop netbird 2>/dev/null || true
    sudo install -m 600 -o root -g root "$LAST_LOGOUT" "$ACTIVE"
    sudo systemctl start netbird
    rm -f "$LAST_LOGOUT"
    echo "Restored last logout credentials."
}
