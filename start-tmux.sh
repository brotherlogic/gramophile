#!/bin/bash

# Ensure the 'gramophile' session exists
if ! tmux has-session -t gramophile 2>/dev/null; then
  # Create a new session named 'gramophile', detached
  cd /workspaces/gramophile
  tmux new-session -d -s gramophile
 fi
