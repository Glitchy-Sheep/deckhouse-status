#!/usr/bin/env bash
set -euo pipefail

REPO="glitchy-sheep/deckhouse-status"
BINARY="deckhouse-status"
INSTALL_DIR="/usr/local/bin"

# ── Colors & Symbols ─────────────────────────────────────────────────
if [ -t 1 ] && command -v tput &>/dev/null && [ "$(tput colors 2>/dev/null || echo 0)" -ge 8 ]; then
  BOLD=$(tput bold)
  DIM=$(tput dim)
  RESET=$(tput sgr0)
  GREEN=$(tput setaf 2)
  CYAN=$(tput setaf 6)
  YELLOW=$(tput setaf 3)
  RED=$(tput setaf 1)
  MAGENTA=$(tput setaf 5)
else
  BOLD="" DIM="" RESET="" GREEN="" CYAN="" YELLOW="" RED="" MAGENTA=""
fi

OK="${GREEN}✔${RESET}"
FAIL="${RED}✖${RESET}"
ARROW="${CYAN}▸${RESET}"

# ── Helpers ───────────────────────────────────────────────────────────
step()  { printf "  %s %s\n" "$ARROW" "$1"; }
ok()    { printf "  %s %s\n" "$OK" "$1"; }
fail()  { printf "  %s %s\n" "$FAIL" "${RED}$1${RESET}" >&2; exit 1; }

# Progress bar: draw a bar that fills to $1 % with label $2
progress() {
  local pct=$1 label=$2 width=30
  local filled=$(( pct * width / 100 ))
  local empty=$(( width - filled ))
  local bar="${GREEN}"
  for ((i=0; i<filled; i++)); do bar+="█"; done
  bar+="${DIM}"
  for ((i=0; i<empty; i++)); do bar+="░"; done
  bar+="${RESET}"
  printf "\r  %s %s %3d%%\033[K" "$bar" "$label" "$pct"
}

# ── Banner ────────────────────────────────────────────────────────────
echo ""
echo "  ${BOLD}${CYAN}┌──────────────────────────────────────┐${RESET}"
echo "  ${BOLD}${CYAN}│  ${MAGENTA}⚓ Deckhouse Status — Installer${CYAN}     │${RESET}"
echo "  ${BOLD}${CYAN}└──────────────────────────────────────┘${RESET}"
echo ""

# ── Detect platform ───────────────────────────────────────────────────
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       fail "Unsupported architecture: $ARCH" ;;
esac

case "$OS" in
  linux|darwin) ;;
  *)            fail "Unsupported OS: $OS" ;;
esac

# ── Download ──────────────────────────────────────────────────────────
ASSET="${BINARY}-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# Get the real file size (follow GitHub's redirect to CDN)
TOTAL_SIZE=$(curl -fsSLI "$URL" 2>/dev/null | grep -i '^content-length:' | tail -1 | tr -dc '0-9')
TOTAL_SIZE=${TOTAL_SIZE:-0}

# Download in background
curl -fSL "$URL" -o "${TMP}/${BINARY}" 2>/dev/null &
CURL_PID=$!

# Show real progress by polling file size
while kill -0 "$CURL_PID" 2>/dev/null; do
  if [ -f "${TMP}/${BINARY}" ] && [ "$TOTAL_SIZE" -gt 0 ]; then
    CURRENT=$(wc -c < "${TMP}/${BINARY}" 2>/dev/null | tr -d ' ')
    PCT=$(( CURRENT * 100 / TOTAL_SIZE ))
    [ "$PCT" -gt 100 ] && PCT=100
    progress "$PCT" "downloading..."
  fi
  sleep 0.1
done

wait "$CURL_PID" || fail "Download failed. Check your network or if the release exists."
progress 100 "done"
printf "\r\033[K"

# ── Install ───────────────────────────────────────────────────────────
chmod +x "${TMP}/${BINARY}"

if [ -w "$INSTALL_DIR" ]; then
  mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  step "${YELLOW}Root privileges required${RESET}"
  sudo mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

ok "Installed to ${BOLD}${INSTALL_DIR}/${BINARY}${RESET}"

# ── Done ──────────────────────────────────────────────────────────────
echo ""
echo "  ${GREEN}${BOLD}Installation complete!${RESET}"
echo ""
echo "  ${BOLD}Quick start:${RESET}"
echo "    ${DIM}\$${RESET} ${CYAN}deckhouse-status${RESET}              ${DIM}# show cluster status${RESET}"
echo "    ${DIM}\$${RESET} ${CYAN}deckhouse-status -s${RESET}           ${DIM}# compact 3-line output${RESET}"
echo ""
echo "  ${BOLD}${YELLOW}⚡ Auto-show on login:${RESET}"
echo "    ${DIM}\$${RESET} ${CYAN}sudo deckhouse-status install-motd${RESET}"
echo ""
