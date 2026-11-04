#!/usr/bin/env bash
set -euo pipefail

# --- Shell setup first, so a failed tool install below never skips it ---

# Persistent bash history and mise data (named volumes)
sudo chown "$(id -u):$(id -g)" /commandhistory
sudo chown "$(id -u):$(id -g)" /home/vscode/.gemini
if [ -d /mnt/mise-data ]; then
  sudo chown -R "$(id -u):$(id -g)" /mnt/mise-data
fi
touch /commandhistory/.bash_history

# Idempotent block: drop any previous version, then append the current one
sed -i '/# >>> omed shell >>>/,/# <<< omed shell <<</d' ~/.bashrc
cat >> ~/.bashrc <<'BASHRC'
# >>> omed shell >>>
if [ -w /commandhistory ]; then
  export HISTFILE=/commandhistory/.bash_history
  export PROMPT_COMMAND='history -a'
fi
if command -v mise >/dev/null 2>&1; then
  eval "$(mise activate bash)"
fi
if command -v starship >/dev/null 2>&1; then
  eval "$(starship init bash)"
fi
# <<< omed shell <<<
BASHRC

# --- Tools via mise ---

mise trust --yes || true
mise install

# Playwright: Chromium only (version follows the one installed by bun install)
mise exec -- bunx playwright install --with-deps chromium
