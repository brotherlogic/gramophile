sudo apt-get update
sudo apt-get install -y protobuf-compiler

go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Account for Ghostty
tic -x ghostty.terminfo

# Install tmux and emacs
sudo apt-get update && sudo apt-get install -y tmux emacs

# Setup tmux for Ghostty and graphics support
cat << 'EOF' > "$HOME/.tmux.conf"
# Set proper default terminal for better TUI rendering
set -g default-terminal "tmux-256color"

# Standard tmux best practice for modern TUIs
set -s escape-time 0

# Allow programs to use the terminal's graphics capabilities
set -g allow-passthrough on

# Support Ghostty terminal capabilities
set -as terminal-overrides ',xterm-ghostty:Sync:Tc'
EOF

# Add tmux auto-attach to .zshrc and .bashrc
for RC_FILE in "$HOME/.zshrc" "$HOME/.bashrc"; do
    if [ -f "$RC_FILE" ] && ! grep -q "Auto-attach to tmux session" "$RC_FILE"; then
        cat << 'EOF' >> "$RC_FILE"

# Auto-attach to tmux session
if [[ $- == *i* ]] && [[ -z "$TMUX" ]] && [[ -z "$SKIP_TMUX" ]] && [[ -t 0 ]]; then
    tmux new-session -A -s "default-${USER:-vscode}"
fi
EOF
    fi
done

git config --global user.email 'brotherlogic.automation@gmail.com'
git config --global user.name 'Brotherlogic Automation'