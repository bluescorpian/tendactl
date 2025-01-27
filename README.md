# tendactl

`tendactl` is a command-line utility for managing your Tenda router’s configuration and status. It provides subcommands for viewing connected clients, adding or removing port forwarding rules, and checking the router’s overall status.

## Features

-   View connected devices with upload/download speeds and identify guest network clients.
-   Manage port forwarding (NAT) rules to open or close specific ports.
-   Check detailed router status including WAN IP, firmware version, and Wi-Fi configuration.

## Installation

1. Ensure you have Go (1.18+) installed.
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

Below are the primary subcommands available under `tendactl`. Run each command with `--help` to see more details and available flags.

• Check Router Status  
Shows WAN IP, up/down speed, Wi-Fi configuration (2.4/5 GHz), number of connected clients, firmware version, and more:

```bash
tendactl status
```

• Check Online Clients  
Displays currently connected devices, upload/download speeds (in KB/s), and identifies guest network clients:

```bash
tendactl online
```

• Manage Port Forwarding (NAT) Rules  
View all existing port forwarding rules:

```bash
tendactl vs
```

Add a new forwarding rule:

```bash
tendactl vs add <ip> <inPort> [outPort] [protocol]
```

Where:
• <ip> is the target device’s IP address.  
• <inPort> is the internal port on the device.  
• [outPort] optionally specifies the corresponding external port (defaults to <inPort> if not specified).  
• [protocol] can be:  
 0 → TCP & UDP  
 1 → TCP  
 2 → UDP

Remove an existing forwarding rule:

```bash
tendactl vs delete <ip> <inPort> <outPort> <protocol>
```

## License

See [LICENSE](LICENSE) for details.
