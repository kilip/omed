#!/usr/bin/env bash
set -euo pipefail

# --- Shell setup first, so a failed tool install below never skips it ---

# Persistent bash history (named volume mounted at /commandhistory)
sudo chown "$(id -u):$(id -g)" /commandhistory
sudo chown "$(id -u):$(id -g)" /home/vscode/.gemini
touch /commandhistory/.bash_history

# Idempotent block: drop any previous version, then append the current one
sed -i '/# >>> omed shell >>>/,/# <<< omed shell <<</d' ~/.bashrc
cat >> ~/.bashrc <<'BASHRC'
# >>> omed shell >>>
if [ -w /commandhistory ]; then
  export HISTFILE=/commandhistory/.bash_history
  export PROMPT_COMMAND='history -a'
fi
if command -v starship >/dev/null 2>&1; then
  eval "$(starship init bash)"
fi
# <<< omed shell <<<
BASHRC

# --- Tools ---

# Starship: official installer (direct release download, no GitHub API rate limit)
curl -sS https://starship.rs/install.sh | sudo sh -s -- -y -v v1.26.0

# swag version must match github.com/swaggo/swag in go.mod (finance + blog)
go install github.com/air-verse/air@v1.67.4
go install github.com/swaggo/swag/v2/cmd/swag@latest

# Playwright: Chromium only (version follows the one installed by bun install)
bunx playwright install --with-deps chromium

bun db:push
