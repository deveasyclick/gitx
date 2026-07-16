#!/bin/sh
# GitX recommended git aliases
#
# Usage:
#   sh scripts/git-aliases.sh            # prints the git config commands
#   sh scripts/git-aliases.sh --install  # adds aliases to global git config

case "${1:-}" in
  --install)
    git config --global alias.gbc "branch --show-current"
    git config --global alias.gbd "branch -D"
    git config --global alias.gc "checkout"
    git config --global alias.gcn "checkout -b"
    git config --global alias.glo "log --oneline --graph"
    git config --global alias.gpc "push -u origin HEAD"
    git config --global alias.gpcf "push --force-with-lease origin HEAD"
    echo "Aliases installed. Verify with: git config --global --list | grep alias"
    ;;
  *)
    cat <<'EOF'
# Install gitx aliases
sh scripts/git-aliases.sh --install

This will run:
  git config --global alias.gbc  "branch --show-current"
  git config --global alias.gbd  "branch -D"
  git config --global alias.gc   "checkout"
  git config --global alias.gcn  "checkout -b"
  git config --global alias.glo  "log --oneline --graph"
  git config --global alias.gpc  "push -u origin HEAD"
  git config --global alias.gpcf "push --force-with-lease origin HEAD"
EOF
    ;;
esac
