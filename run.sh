#!/bin/bash

# ============================================
# GTG Project Runner
# Chia doi terminal: Backend (Go) | Frontend (Next.js)
# Tu dong cai dat dependencies neu thieu
# ============================================

set -e

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
SESSION_NAME="gtg"

# --- Mau sac cho output ---
BACKEND_PORT=8080
FRONTEND_PORT=3000

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

log_info()  { echo -e "${CYAN}[INFO]${NC} $1"; }
log_ok()    { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_err()   { echo -e "${RED}[ERROR]${NC} $1"; }

# --- Tim PID dang chiem port ---
find_pids_on_port() {
    local port=$1
    local all_pids=""

    # Thu tat ca cac phuong phap va gop ket qua
    if command -v lsof &>/dev/null; then
        local p=$(lsof -t -i:"$port" 2>/dev/null || true)
        [ -n "$p" ] && all_pids="$all_pids $p"
    fi
    if command -v fuser &>/dev/null; then
        local p=$(fuser "$port/tcp" 2>/dev/null | tr -s ' ' '\n' | grep -v '^$' || true)
        [ -n "$p" ] && all_pids="$all_pids $p"
    fi
    # ss luon co tren Linux
    local p=$(ss -tlnp 2>/dev/null | grep ":$port " | grep -oP 'pid=\K[0-9]+' || true)
    [ -n "$p" ] && all_pids="$all_pids $p"

    # Loc unique PIDs
    echo "$all_pids" | tr ' ' '\n' | grep -v '^$' | sort -u | tr '\n' ' '
}

# --- Kill process dang chiem port ---
kill_port() {
    local port=$1
    local pids
    pids=$(find_pids_on_port "$port")
    pids=$(echo "$pids" | xargs)  # trim whitespace

    if [ -n "$pids" ]; then
        log_warn "Port $port dang bi chiem boi PID: $pids. Dang kill..."
        echo "$pids" | tr ' ' '\n' | xargs kill -9 2>/dev/null || true
        sleep 1
        # Kiem tra lai
        local check_pids
        check_pids=$(find_pids_on_port "$port")
        check_pids=$(echo "$check_pids" | xargs)
        if [ -n "$check_pids" ]; then
            log_err "Khong the kill process tren port $port. Thu chay: sudo kill -9 $check_pids"
            exit 1
        fi
        log_ok "Da giai phong port $port."
    else
        log_ok "Port $port san sang."
    fi
}

# --- Kiem tra va cai dat dependencies ---
check_deps() {
    # Kiem tra tmux
    if ! command -v tmux &>/dev/null; then
        log_warn "tmux chua duoc cai dat. Dang cai dat..."
        if command -v apt-get &>/dev/null; then
            sudo apt-get update && sudo apt-get install -y tmux
        elif command -v pacman &>/dev/null; then
            sudo pacman -S --noconfirm tmux
        elif command -v dnf &>/dev/null; then
            sudo dnf install -y tmux
        elif command -v brew &>/dev/null; then
            brew install tmux
        else
            log_err "Khong the tu dong cai tmux. Vui long cai dat thu cong."
            exit 1
        fi
        log_ok "Da cai dat tmux."
    fi

    # Kiem tra Go
    if ! command -v go &>/dev/null; then
        log_err "Go chua duoc cai dat. Vui long cai dat tu: https://go.dev/dl/"
        exit 1
    fi
    log_ok "Go $(go version | awk '{print $3}')"

    # Kiem tra Node.js & npm
    if ! command -v node &>/dev/null || ! command -v npm &>/dev/null; then
        log_err "Node.js/npm chua duoc cai dat. Vui long cai dat tu: https://nodejs.org/"
        exit 1
    fi
    log_ok "Node $(node -v) | npm $(npm -v)"
}

# --- Cai dat Go dependencies ---
setup_backend() {
    log_info "Kiem tra Go dependencies..."
    cd "$PROJECT_DIR/backend"
    go mod download
    log_ok "Backend dependencies OK."
}

# --- Cai dat Frontend dependencies ---
setup_frontend() {
    log_info "Kiem tra Frontend dependencies..."
    cd "$PROJECT_DIR/frontend"
    if [ ! -d "node_modules" ] || [ "package.json" -nt "node_modules/.package-lock.json" ] 2>/dev/null; then
        log_warn "Dang cai dat npm dependencies..."
        npm install
    fi
    log_ok "Frontend dependencies OK."
}

# --- Main ---
main() {
    echo ""
    echo -e "${CYAN}╔══════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║         🚀 GTG Project Runner        ║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════╝${NC}"
    echo ""

    # === BUOC 1: Kill tmux session cu (neu co) ===
    log_info "Kill tmux session cu neu co..."
    tmux kill-session -t "$SESSION_NAME" 2>/dev/null || true
    sleep 1

    # === BUOC 2: Kiem tra va giai phong port ===
    log_info "Kiem tra va giai phong port..."
    kill_port $BACKEND_PORT
    kill_port $FRONTEND_PORT
    echo ""

    # === BUOC 3: Kiem tra dependencies va cai dat ===
    check_deps
    setup_backend
    setup_frontend

    echo ""
    log_info "Dang khoi dong project..."

    # Tao tmux session moi voi backend
    tmux new-session -d -s "$SESSION_NAME" -c "$PROJECT_DIR/backend" \
        "echo -e '${GREEN}=== 🔧 BACKEND (Go) ===${NC}'; echo ''; go run main.go; read"

    # Chia doc (split ngang) cho frontend
    tmux split-window -h -t "$SESSION_NAME" -c "$PROJECT_DIR/frontend" \
        "echo -e '${GREEN}=== 🌐 FRONTEND (Next.js) ===${NC}'; echo ''; npm run dev; read"

    # Can bang 2 pane
    tmux select-layout -t "$SESSION_NAME" even-horizontal

    echo ""
    log_ok "Da khoi dong thanh cong!"
    echo ""
    echo -e "  ${CYAN}Backend${NC}  → http://localhost:${BACKEND_PORT}"
    echo -e "  ${CYAN}Frontend${NC} → http://localhost:${FRONTEND_PORT}"
    echo ""
    echo -e "  ${YELLOW}Tips:${NC}"
    echo -e "    Ctrl+B rồi ←/→  : chuyen giua 2 pane"
    echo -e "    Ctrl+B rồi d     : detach (thoat ma khong tat)"
    echo -e "    tmux a -t gtg    : attach lai session"
    echo -e "    Ctrl+C           : tat server trong pane hien tai"
    echo ""

    # Attach vao session
    tmux attach-session -t "$SESSION_NAME"
}

main
