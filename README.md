# OpenPeripherals (BETA)

Control peripheral settings from the browser over WebHID. No drivers, no vendor software.

## Dev Setup

```bash
npm install
npm run dev
```

## Linux Fix

By default on Linux your device is only readable, not writable, so the app connects but the lighting controls do nothing until a small udev rule grants access.

If "Connect Device" looks like it worked but nothing changes, run these three lines in a terminal, then unplug and replug your device (or reboot):

```
sudo tee /etc/udev/rules.d/55-openperipherals.rules <<'EOF'
KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="1532", MODE="0666", TAG+="uaccess"
KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="046d", MODE="0666", TAG+="uaccess"
KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="0c45", MODE="0666", TAG+="uaccess"
KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="3710", MODE="0666", TAG+="uaccess"
KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="3554", MODE="0666", TAG+="uaccess"
EOF
sudo udevadm control --reload-rules
sudo udevadm trigger
```

[Read on Chrome Docs](https://developer.chrome.com/docs/capabilities/hid#:~:text=On%20most%20Linux%20systems%2C%20HID%20devices%20are%20mapped%20with%20read%2Donly%20permissions%20by%20default.%20To%20allow%20Chrome%20to%20open%20an%20HID%20device%2C%20you%20will%20need%20to%20add%20a%20new%20udev%20rule.%20Create%20a%20file%20at%20/etc/udev/rules.d/50%2Dyourdevicename.rules%20with%20the%20following%20content%3A)

## How it works

Devices expose vendor HID interfaces that accept effect commands. This app builds and sends those report packets directly from the browser.

## Credits

The HID protocol details for Razer devices were reverse-engineered by the [OpenRazer](https://github.com/openrazer/openrazer) project.

The HID protocol details for Logitech and Redragon devices were reverse-engineered by the [OpenRGB](https://gitlab.com/CalcProgrammer1/OpenRGB) project.

## Contributing

Found a bug, or want to add support for a new device? Open an issue or pull request.

```bash
cd openperipherals
npm install
npm run dev        # start the dev server
```

Make your changes on a branch, then verify:

```bash
git checkout -b my-branch
npm run check      # svelte-check diagnostics
npm run build      # production build
git add .
git commit -m "fix: describe what you fixed"   # or feat: for new features
git push -u origin my-branch
```

Then open a pull request into `main` describing what changed and what you tested. It's highly recommended to describe what you add/fix and to include some screenshots or videos.
