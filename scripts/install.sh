#!/usr/bin/env bash

set -eu -o pipefail

if [[ ! -z ${DEBUG-} ]]; then
    set -x
fi

: ${PREFIX:=/usr/local}
BINDIR="$PREFIX/bin"

if [[ $# -gt 0 ]]; then
  BINDIR=$1
fi

_can_install() {
  if [[ ! -d "$BINDIR" ]]; then
    mkdir -p "$BINDIR" 2> /dev/null
  fi
  [[ -d "$BINDIR" && -w "$BINDIR" ]]
}

if ! _can_install && [[ $EUID != 0 ]]; then
  echo "Run script as sudo"
  exit 1
fi

if ! _can_install; then
  echo "Can't install to $BINDIR"
  exit 1
fi

machine=$(uname -m)

case $(uname -s) in
    Linux)
        os="linux"
        ;;
    Darwin)
        os="darwin"
        ;;
    *)
        echo "OS not supported"
        exit 1
        ;;
esac

latest="$(curl -sL 'https://api.github.com/repos/pgollangi/fastget/releases/latest' | grep 'tag_name' | grep --only 'v[0-9\.]\+' | cut -c 2-)"
curl -sL "https://github.com/pgollangi/fastget/releases/download/v${latest}/fastget_${latest}_${os}_${machine}.tar.gz" | tar -C /tmp/ -xzf -
install -m755 /tmp/fastget $BINDIR/fastget
echo "Successfully installed fastget into $BINDIR/"
