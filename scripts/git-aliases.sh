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
  --uninstall)
    git config --global --unset alias.gbc 2>/dev/null || true
    git config --global --unset alias.gbd 2>/dev/null || true
    git config --global --unset alias.gc 2>/dev/null || true
    git config --global --unset alias.gcn 2>/dev/null || true
    git config --global --unset alias.glo 2>/dev/null || true
    git config --global --unset alias.gpc 2>/dev/null || true
    git config --global --unset alias.gpcf 2>/dev/null || true
    echo "GitX aliases removed."
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

# Uninstall gitx aliases
sh scripts/git-aliases.sh --uninstall
EOF
    ;;
esac
