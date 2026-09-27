# tendactl

[![Go](https://img.shields.io/badge/go-1.26%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**tendactl** is a command-line tool for Tenda routers. It talks to the same web
API the admin UI uses, so anything you would click through in the browser (WiFi,
port forwarding, DHCP reservations, parental control, VPN, reboots) you can do
from a terminal or a script.

```console
$ tendactl status
AC10 (router) @ 192.168.0.1
5G:[OFF] 2.4G:[ON]WiFi-16                WAN: 203.0.113.10 (Down:521.28 KB/s Up:7.23 KB/s)
Clients: 13    MAC: 02:00:00:00:00:0F
Firmware: V15.03.06.50_multi
```

## Features

* Covers almost the whole admin UI: WiFi, guest network, port forwarding,
  DHCP, bandwidth caps, parental control, MAC filtering, firewall, DMZ, UPnP,
  static routes, DDNS, the PPTP/L2TP VPN, and system maintenance.
* Commands and output use the router UI's own wording, so the manual and the
  web pages still apply.
* Readable tables by default, and `-o json` on every command for scripting.
* Safe by default: anything that can cut your connection or wipe settings is
  refused unless you pass `--yes`.
* Logs in once and caches the session, so later commands need no password.
* `tendactl api` reaches any router endpoint directly, including the ones
  without a dedicated command.

## Contents

* [How to use](#how-to-use)
  * [Connecting to the router](#connecting-to-the-router)
  * [See what's on your network](#see-whats-on-your-network)
  * [Forward a port](#forward-a-port)
  * [Give a device a fixed IP](#give-a-device-a-fixed-ip)
  * [Change the WiFi](#change-the-wifi)
  * [Run a guest network](#run-a-guest-network)
  * [Block or limit a device](#block-or-limit-a-device)
  * [Parental control](#parental-control)
  * [Turn things off at night](#turn-things-off-at-night)
  * [Firewall, DMZ, UPnP and routes](#firewall-dmz-upnp-and-routes)
  * [Dynamic DNS](#dynamic-dns)
  * [VPN](#vpn)
  * [Look after the router](#look-after-the-router)
  * [LAN settings](#lan-settings)
  * [Scripting with JSON](#scripting-with-json)
  * [Calling the API directly](#calling-the-api-directly)
* [Hazardous commands and `--yes`](#hazardous-commands-and---yes)
* [Command reference](#command-reference)
* [Installation](#installation)
* [Compatibility](#compatibility)
* [License](#license)

## How to use

Every command group follows the same shape: run it on its own to see the
current state, then use `show`/`list`, `set`, `add`/`rm` or `enable`/`disable`
to change it. `tendactl <command> --help` lists every flag.

### Connecting to the router

tendactl connects to `192.168.0.1` by default. Point it somewhere else with
`--host` or `TENDA_HOST`, and give it the admin password through
`TENDA_PASSWORD` (or type it when prompted):

```bash
export TENDA_HOST=192.168.1.1
export TENDA_PASSWORD='your-admin-password'
tendactl status
```

The session cookie is cached per host under `$XDG_RUNTIME_DIR` (mode 0600), so
you only need the password again once the router expires the session.

| Flag | Environment | Default | Meaning |
|---|---|---|---|
| `--host` | `TENDA_HOST` | `192.168.0.1` | Router address, `host[:port]` |
| | `TENDA_PASSWORD` | (prompt) | Admin password |
| `-o`, `--output` | | `table` | `table` or `json` |
| `-y`, `--yes` | | off | Allow [hazardous requests](#hazardous-commands-and---yes) |
| `--timeout` | | `15s` | HTTP timeout per request |

### See what's on your network

`clients` (alias `online`) lists every device with its live upload and download
speed. Guest network devices are marked `[Guest]`.

```console
$ tendactl clients
Device-11 @ 192.168.0.2 (02:00:00:00:00:02)

DEVICE NAME IP ADDRESS    ↑KB/s ↓KB/s TYPE
─────────── ───────────── ───── ───── ────
Device-3    192.168.0.151     5   144
Device-5    192.168.0.169     0     0
Device-4    192.168.0.170     0    31
```

Give a device a friendlier name:

```bash
tendactl clients rename 02:00:00:00:00:04 "Living room TV"
```

### Forward a port

`nat` manages port forwarding (the UI's "Virtual Server"). The WAN port
identifies each rule and defaults to the LAN port.

```console
$ tendactl nat add 192.168.0.5 25565 --proto tcp
$ tendactl nat
IP          INTERNAL PORT EXTERNAL PORT PROTOCOL
─────────── ───────────── ───────────── ────────
192.168.0.5 25565         25565         TCP
```

Forward a different WAN port, then remove it again:

```bash
tendactl nat add 192.168.0.100 80 8080 --proto tcp   # WAN 8080 -> LAN 80
tendactl nat rm 8080
```

### Give a device a fixed IP

A DHCP reservation takes effect the next time the device renews its lease.
Adding a MAC that is already reserved updates its reservation.

```bash
tendactl dhcp add 02:00:00:00:00:08 192.168.0.31 --name nas
tendactl dhcp                    # reservations, then the other DHCP clients
tendactl dhcp rm 02:00:00:00:00:08
```

### Change the WiFi

```console
$ tendactl wifi show
2.4 GHz:             on
2.4 GHz SSID:        WiFi-16
2.4 GHz security:    WPA2-PSK
2.4 GHz password:    ********
5 GHz:               off
...
```

Passwords are masked unless you add `--show-password`. Most WiFi commands
take `--band 2.4`, `--band 5` or `--band all` (the default).

```bash
tendactl wifi set --ssid Home --password 'correct horse battery'
tendactl wifi set --band 5 --ssid Home-5G
tendactl wifi enable --band 5
tendactl wifi hide --band 2.4             # stop broadcasting the SSID; `unhide` reverses it
tendactl wifi channel set --band 5 --channel 36 --width 80
tendactl wifi power set high
tendactl wifi wps start                   # push-button pairing
```

`wifi antijam` and `wifi beamforming` toggle anti-interference and
Beamforming+. Changing a band's channel settings restarts that radio, and
saving the power, anti-interference or Beamforming+ setting restarts the
2.4 GHz and guest radios, so WiFi clients drop for a few seconds.

### Run a guest network

```bash
tendactl guest set --ssid Visitors --password 'welcome123' --effective-time 8 --share-speed 20
tendactl guest enable
tendactl guest show --show-password
```

`--effective-time` is `4`, `8` (hours) or `always`; `--share-speed` caps all
guests together, in Mbps. The guest network needs the router in AP mode with
WiFi on.

### Block or limit a device

There are three ways to restrict a device, matching the three places in the UI:

```bash
# Cut a device off right now (the "Manage Device" blacklist)
tendactl clients block 02:00:00:00:00:05
tendactl clients blocked
tendactl clients unblock 02:00:00:00:00:05

# Cap its speed, in Mbps (Bandwidth Control)
tendactl bandwidth set 02:00:00:00:00:05 --down 10 --up 2
tendactl bandwidth rm 02:00:00:00:00:05

# Deny it at the MAC filter (Filter MAC Address)
tendactl macfilter add 02:00:00:00:00:05 --name tablet
tendactl macfilter rm 02:00:00:00:00:05
```

The MAC filter can also run as a whitelist, where only listed devices keep
access. Switching to it cuts off everything else at once, so it needs `--yes`:

```bash
tendactl macfilter mode white --yes
```

### Parental control

Allow a device online only in a daily window, and block websites by keyword:

```bash
tendactl parental set 02:00:00:00:00:04 --allow 19:00-21:00 --days all \
  --url-filter on --limit-mode blacklist --urls example,video
```

```console
$ tendactl parental show 02:00:00:00:00:04
Enabled:         on
Allowed window:  19:00-21:00
Days:            every day
URL filter:      on
Filter mode:     blacklist
Keywords (2):    example,video
```

`parental enable <mac>` blocks the device immediately, whatever the schedule
says; `parental disable <mac>` lets it back on. `parental list` shows every
device's status, and `parental rm <mac>` deletes the rule.

### Turn things off at night

```bash
# WiFi off from 23:00 to 07:00 on weeknights
tendactl wifi schedule set --time 23:00-07:00 --days mon,tue,wed,thu,fri
tendactl wifi schedule enable --yes

# Or Sleeping Mode: WiFi and LEDs off, but waits while clients are online
tendactl sleep set --time 00:00-07:00 --leds all --delay on
tendactl sleep enable --yes

# Just the LEDs
tendactl led set --mode time --time 22:00-07:00
tendactl led disable        # always off; `led enable` for always on
```

Enabling the WiFi schedule or Sleeping Mode needs `--yes`: inside the window
the only way back onto WiFi is the router's physical WiFi button (or the Tenda
app for Sleeping Mode). Both need the router in AP mode.

### Firewall, DMZ, UPnP and routes

```bash
tendactl firewall set --tcp-flood --udp-flood --icmp-flood --ignore-wan-ping
tendactl firewall set --ignore-wan-ping=false

tendactl dmz set 192.168.0.100      # every inbound WAN port goes to this host
tendactl dmz disable

tendactl upnp                       # status and active mappings
tendactl upnp disable

tendactl route add 10.0.0.0 255.255.255.0 192.168.0.254
tendactl route rm 10.0.0.0 255.255.255.0
```

`route list` also shows the router's own system routes, which are read-only.

### Dynamic DNS

```bash
tendactl ddns set --provider no-ip.com --domain myhome.ddns.net --user me --password secret
tendactl ddns enable
tendactl ddns show
```

Supported providers are `no-ip.com`, `dyndns.org`, `88ip.cn` and `oray.com`.

### VPN

The router can host a PPTP server, or dial out to a PPTP/L2TP server as a client.

```bash
# PPTP server
tendactl vpn server set --pool-start 10.0.0.100 --pool-end 10.0.0.200 --mppe on --mppe-bits 128
tendactl vpn users add alice --password 's3cret'
tendactl vpn server enable
tendactl vpn online                 # who is connected right now

# VPN client
tendactl vpn client set --type l2tp --domain vpn.example.com --user me --password secret
tendactl vpn client enable
```

```console
$ tendactl vpn users
USER NAME PASSWORD ENABLED CONNECTED
───────── ──────── ─────── ─────────
alice     ******** on      on
```

### Look after the router

Everything under the UI's System Settings lives under `system`:

```bash
tendactl system status                      # uptime, firmware, WAN and WiFi
tendactl system log                         # the system log
tendactl system log --download log.tar
tendactl system backup router.cfg           # full config backup (contains every password)
tendactl system reboot --yes

tendactl system maintenance set --time 04:00 --delay on   # daily reboot, delayed while busy
tendactl system maintenance enable

tendactl system remote set --from 198.51.100.7 --port 8080  # WAN-side admin, one source IP only
tendactl system remote enable

tendactl system time set --zone 14:00       # GMT offset plus 12h, so 14:00 is GMT+02:00
```

`system backup` refuses to overwrite an existing file.

### LAN settings

Changing the LAN address moves the router's own management address, so this
needs `--yes`, and you will reach the router at the new address afterwards:

```bash
tendactl lan show
tendactl lan set --dhcp-start 192.168.0.100 --dhcp-end 192.168.0.200 --yes
tendactl lan set --ip 192.168.1.1 --yes
```

### Scripting with JSON

Add `-o json` to any command for machine-readable output:

```bash
# Names of devices on the guest network
tendactl clients -o json | jq -r '.clients[] | select(.guest) | .name'

# Every port forwarded to 192.168.0.5
tendactl nat -o json | jq '.[] | select(.ip == "192.168.0.5") | .outPort'
```

### Calling the API directly

`api` calls any endpoint listed in [docs/router-api.md](docs/router-api.md),
including the ones with no command yet (WAN setup, AP mode, firmware upgrade and
so on). JSON replies are pretty-printed; anything else is written raw.

```console
$ tendactl api get GetDMZCfg
{
  "lanIp": "192.168.0.1",
  "lanMask": "255.255.255.0",
  "dmzEn": "0",
  "dmzIp": "192.168.0.100"
}
$ tendactl api set SetDMZCfg dmzEn=1 dmzIp=192.168.0.100
$ tendactl api get cloudv2 module=wansta opt=query
```

`api set` fails if the router answers with a non-zero `errCode`.

## Hazardous commands and `--yes`

Requests that can cut connectivity, reboot the router or erase configuration
are refused unless you pass `--yes`. There is no interactive prompt, so scripts
and people follow the same rule:

```console
$ tendactl system reboot
tendactl: refusing SysToolReboot: reboots the router; WiFi and WAN drop for about a minute; rerun with --yes
```

This covers rebooting, turning a WiFi band off, enabling the WiFi schedule or
Sleeping Mode, LAN changes, MAC whitelist mode, and (through `api`) WAN
changes, working-mode changes, config restore, factory reset, firmware
upgrades and admin password changes. The full list, with reasons, is in
[docs/router-api.md](docs/router-api.md).

## Command reference

A bare command group shows its current state (except `system`, `vpn` and
`wifi`, which only group other commands). Run `tendactl <command> --help` for
flags.

| Command | Subcommands | UI page |
|---|---|---|
| `status` | | Internet Status |
| `clients` | `list` `rename` `block` `unblock` `blocked` | Manage Device |
| `nat` | `list` `add` `rm` | Virtual Server |
| `dhcp` | `list` `add` `rm` | DHCP Reservation |
| `wifi` | `show` `set` `enable` `disable` `hide` `unhide` | WiFi Name & Password |
| `wifi channel` | `show` `set` | Channel & Bandwidth |
| `wifi power` | `show` `set` | Transmit Power |
| `wifi schedule` | `show` `set` `enable` `disable` | WiFi Schedule |
| `wifi wps` | `show` `enable` `disable` `start` | WPS |
| `wifi antijam` | `show` `set` | Anti-interference |
| `wifi beamforming` | `show` `enable` `disable` | Beamforming+ |
| `guest` | `show` `set` `enable` `disable` | Guest Network |
| `bandwidth` | `list` `set` `rm` | Bandwidth Control |
| `parental` | `list` `show` `set` `rm` `enable` `disable` | Parental Control |
| `macfilter` | `show` `add` `rm` `mode` | Filter MAC Address |
| `firewall` | `show` `set` | Firewall |
| `dmz` | `show` `set` `enable` `disable` | DMZ Host |
| `upnp` | `show` `enable` `disable` | UPnP |
| `route` | `list` `add` `rm` | Static Route |
| `ddns` | `show` `set` `enable` `disable` | DDNS |
| `sleep` | `show` `set` `enable` `disable` | Sleeping Mode |
| `led` | `show` `set` `enable` `disable` | LED Control |
| `lan` | `show` `set` | LAN Settings |
| `vpn server` | `show` `set` `enable` `disable` | PPTP Server |
| `vpn users` | `list` `add` `rm` `enable` `disable` | PPTP Server users |
| `vpn online` | | Online PPTP Users |
| `vpn client` | `show` `set` `enable` `disable` | PPTP/L2TP Client |
| `system status` | | System Status |
| `system log` | | System Log |
| `system backup` | | Backup/Restore |
| `system reboot` | | Reboot |
| `system maintenance` | `show` `set` `enable` `disable` | Automatic Maintenance |
| `system remote` | `show` `set` `enable` `disable` | Remote Management |
| `system time` | `show` `set` | Time Settings |
| `api` | `get` `set` | any endpoint |
| `completion` | `bash` `zsh` `fish` `powershell` | shell completion |

## Installation

### Download a binary

The [releases page](https://github.com/bluescorpian/tendactl/releases/latest)
has a single self-contained binary for Linux (x86-64, ARM64, and 32-bit ARM
for a Raspberry Pi), macOS (Intel and Apple Silicon) and Windows. Unpack it and
put `tendactl` (`tendactl.exe` on Windows) somewhere on your `PATH`:

```bash
tar xzf tendactl_*_linux_amd64.tar.gz
sudo install tendactl /usr/local/bin/
tendactl --version
```

Each archive also carries shell completions under `completions/`.

### Linux packages

The same page has `.deb`, `.rpm` and `.apk` packages, which install the
completions too:

```bash
sudo apt install ./tendactl_*_linux_amd64.deb      # Debian, Ubuntu, Raspberry Pi OS
sudo dnf install ./tendactl_*_linux_amd64.rpm      # Fedora, RHEL
sudo apk add --allow-untrusted ./tendactl_*_linux_amd64.apk  # Alpine
```

### Nix

```bash
nix run github:bluescorpian/tendactl -- status
nix profile install github:bluescorpian/tendactl
```

### From source

With Go 1.26 or newer:

```bash
go install github.com/bluescorpian/tendactl@latest
```

Shell completion is built in, for example:

```bash
tendactl completion bash > ~/.local/share/bash-completion/completions/tendactl
```

## Compatibility

tendactl is developed and tested against a **Tenda AC10** on firmware
`V15.03.06.50_multi`. Other Tenda models share much of the same web API and may
work in part; [docs/router-api.md](docs/router-api.md) records exactly which
endpoints have been verified.

## License

MIT. See [LICENSE](LICENSE).
