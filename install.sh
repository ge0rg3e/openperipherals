#!/bin/sh
# Install OpenPeripherals for the current user on Ubuntu 24.04.
set -eu

fail() { printf '%s\n' "$*" >&2; exit 1; }

[ "$(uname -s)" = Linux ] || fail 'This installer is for Linux.'
[ "$(id -u)" -ne 0 ] || fail 'Run this as your regular user, without sudo.'
case "$(uname -m)" in
  x86_64|amd64) ;;
  *) fail 'The Linux release supports x86-64 only.' ;;
esac
for tool in curl tar sudo apt-get udevadm; do
  command -v "$tool" >/dev/null 2>&1 || fail "Missing $tool. This installer targets Ubuntu 24.04."
done

install_dir="$HOME/.local/opt/openperipherals"
binary="$install_dir/openperipherals"
config_dir=${XDG_CONFIG_HOME:-"$HOME/.config"}
data_dir=${XDG_DATA_HOME:-"$HOME/.local/share"}
asset=openperipherals-linux-amd64.tar.gz

# Escape the quoted Exec argument, then the desktop entry value.
case "$binary" in
  *[[:cntrl:]]*) fail 'The installation path contains an unsupported control character.' ;;
esac
desktop_exec=$(printf '%s' "$binary" | sed 's/\\/\\\\/g; s/["$`]/\\&/g; s/%/%%/g; s/\\/\\\\/g')

workdir=$(mktemp -d)
staged_binary=
cleanup() {
  if [ -n "$staged_binary" ]; then rm -f -- "$staged_binary"; fi
  rm -rf -- "$workdir"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

printf 'Downloading OpenPeripherals...\n'
curl --fail --location --show-error --proto '=https' --proto-redir '=https' \
  "https://github.com/ge0rg3e/openperipherals/releases/latest/download/$asset" -o "$workdir/$asset"
tar -xzf "$workdir/$asset" -C "$workdir" openperipherals 70-openperipherals.rules
[ -s "$workdir/openperipherals" ] || fail 'The archive is missing the OpenPeripherals executable.'
[ -s "$workdir/70-openperipherals.rules" ] || fail 'The archive is missing the device permission rules.'

sudo apt-get update
sudo apt-get install -y libgtk-3-0t64 libwebkit2gtk-4.1-0
sudo install -m 0644 "$workdir/70-openperipherals.rules" /etc/udev/rules.d/
sudo udevadm control --reload-rules

first_install=false
if [ ! -e "$binary" ]; then first_install=true; fi
mkdir -p "$install_dir" "$data_dir/applications"
staged_binary=$(mktemp "$install_dir/.openperipherals.XXXXXX")
install -m 0755 "$workdir/openperipherals" "$staged_binary"
mv -f -- "$staged_binary" "$binary"
staged_binary=

cat > "$workdir/openperipherals.desktop" <<DESKTOP
[Desktop Entry]
Type=Application
Name=OpenPeripherals
Comment=Configure supported gaming peripherals
Exec="$desktop_exec"
Terminal=false
Categories=Settings;HardwareSettings;
DESKTOP
install -m 0644 "$workdir/openperipherals.desktop" "$data_dir/applications/openperipherals.desktop"
if [ "$first_install" = true ]; then
  mkdir -p "$config_dir/autostart"
  if [ ! -e "$config_dir/autostart/openperipherals.desktop" ]; then
    install -m 0644 "$workdir/openperipherals.desktop" "$config_dir/autostart/openperipherals.desktop"
  fi
fi

printf '\nInstalled OpenPeripherals to %s.\n' "$binary"
printf 'Reconnect your peripherals to apply the device permissions. Startup is managed in Settings.\n'
if [ -n "${DISPLAY:-}${WAYLAND_DISPLAY:-}" ]; then
  if command -v pgrep >/dev/null 2>&1 && pgrep -u "$(id -u)" -x openperipherals >/dev/null; then
    printf 'OpenPeripherals is already running. Restart it to use the installed version.\n'
  else
    printf 'Opening OpenPeripherals...\n'
    nohup "$binary" </dev/null >/dev/null 2>&1 &
  fi
else
  printf 'Launch OpenPeripherals from your applications menu after signing in.\n'
fi
