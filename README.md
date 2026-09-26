# tendactl

`tendactl` is a command-line utility for managing your Tenda router’s configuration and status. It provides subcommands for viewing connected clients, adding or removing port forwarding rules, and checking the router’s overall status.

## Features

-   View connected devices with upload/download speeds and identify guest network clients.
-   Manage port forwarding (NAT) rules to open or close specific ports.
-   Check router status including WAN IP, firmware version, and Wi-Fi configuration.
-   Manage WiFi, guest network, DHCP, firewall, VPN, parental control and most
    other router settings; run `tendactl --help` for the full command tree.
-   Call any router endpoint directly with `tendactl api`.

## Installation

1. Ensure you have Go (1.26+) installed.
2. Clone or download this repository.
3. Navigate to the project’s root folder and build the CLI:
    ```bash
    go build -o tendactl
    ```
4. Move the compiled binary to a directory in your system’s PATH (optional):
    ```bash
    mv tendactl /usr/local/bin/
    ```
5. Confirm installation:
    ```bash
    tendactl --help
    ```

## Usage

Run `tendactl --help`, or `tendactl <command> --help`, for every command and flag.

### Connecting

| Flag | Environment | Default | Meaning |
|---|---|---|---|
| `--host` | `TENDA_HOST` | `192.168.0.1` | Router address, `host[:port]` |
| | `TENDA_PASSWORD` | (prompt) | Admin password; without it `tendactl` prompts when run in a terminal |
| `-o`, `--output` | | `table` | `table` or `json` |
| `-y`, `--yes` | | off | Allow hazardous requests (reboot, turning WiFi off, LAN or WAN changes, ...) |
| `--timeout` | | `15s` | HTTP timeout per request |

The session cookie is cached per host under `$XDG_RUNTIME_DIR` (mode 0600), so
later commands need no password until the router expires the session.

### Router status

```bash
tendactl status
```

### Connected devices

```bash
tendactl clients        # also: tendactl online
```

### Port forwarding (NAT)

```bash
tendactl nat                                        # list rules
tendactl nat add <ip> <inPort> [outPort] [--proto both|tcp|udp]
tendactl nat rm <outPort>
```

`outPort` (the WAN port) defaults to `inPort` and identifies the rule; the
protocol defaults to `both`.

### Raw API access

```bash
tendactl api get <Endpoint> [key=value...]
tendactl api set <Endpoint> key=value...
```

`api` reaches every endpoint in [docs/router-api.md](docs/router-api.md),
including the ones without a command (WAN, AP mode, firmware upgrade, ...).
JSON replies are pretty-printed; other replies are written raw, so
`tendactl api get cgi-bin/DownloadCfg/RouterCfm.cfg > backup.cfg` works. `set`
fails on a non-zero `errCode`. Endpoints on the doc's "Never call casually"
list still need `--yes`.

### Everything else

Beyond the walkthroughs above, `tendactl` also manages WiFi (`wifi`, plus its
`channel`, `power`, `schedule`, `wps`, `antijam` and `beamforming`
subcommands, and `guest`), network settings (`dhcp`, `dmz`, `upnp`,
`bandwidth`, `parental`, `lan`, `firewall`, `macfilter`, `route`, `ddns`), the
router itself (`led`, `sleep`, and `system`'s `status`, `reboot`, `log`,
`maintenance`, `remote`, `time` and `backup`), and the PPTP/L2TP VPN (`vpn`'s
`server`, `users`, `online` and `client`). Run `tendactl --help` for the full
command tree, or `tendactl <command> --help` for any command's flags.

## License

See [LICENSE](LICENSE) for details.
