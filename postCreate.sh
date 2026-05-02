sudo apt-get update
sudo apt-get install -y protobuf-compiler

go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Account for Ghostty
tic -x ghostty.terminfo

# Install tmux and emacs
sudo apt-get update && sudo apt-get install -y tmux emacs

# Add tmux auto-attach to .zshrc and .bashrc
for RC_FILE in "$HOME/.zshrc" "$HOME/.bashrc"; do
    if [ -f "$RC_FILE" ] && ! grep -q "Auto-attach to tmux session" "$RC_FILE"; then
        cat << 'EOF' >> "$RC_FILE"

# Auto-attach to tmux session
if [[ -z "$TMUX" ]] && [[ -n "$PS1" ]] && [[ -t 0 ]]; then
    tmux attach-session -t default || tmux new-session -s default
fi
EOF
    fi
done

git config --global user.email 'brotherlogic.automation@gmail.com'
git config --global user.name 'Brotherlogic Automation'