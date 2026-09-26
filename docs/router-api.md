# Tenda AC10 web API

The router has no published API. This reference is reverse-engineered from
the web UI's JavaScript (mirrored from the router) and checked against live
responses. It is the source for adding `tendactl` commands.

- Hardware: Tenda AC10 V1.0, Realtek RTL8197F. The router reports firmware
  `V15.03.06.50_multi`; the UI's `js/macro_config.js` still reports
  `ac10_V2.0.0.0(8254)_cn`. Field names match either way.
- Build flags in `js/macro_config.js` decide what the UI shows. With
  `CONFIG_USB_MODULES="n"`, every USB feature (printer, Samba, DLNA, USB
  status, Xunlei) is compiled out, and its goform handlers return the stock
  `Form <Name> is not defined` error page.

## Conventions

**Auth.** `POST /login/Auth` with body `username=admin&password=<md5 hex of
password>` sets a `password=<token>` cookie. A body of `1` means the password
was wrong. `tenda/client.go` implements this.

**Session expiry.** Once the session expires, pages and goform endpoints
answer `302` to `/login.html`. The UI also treats any response containing
`<!DOCTYPE` as a logged-out session. An idle session lasts at least a few
minutes. Nothing here measured the exact timeout.

**GET.** Every getter takes no parameters unless its section says otherwise.
The UI appends a random cache-buster (`?<random>` or `?_=<ts>`), but the
router ignores it.

**POST.** The body is `application/x-www-form-urlencoded`. Sending
`--data-raw` with the literal separators shown below works. List-valued
fields pack rows and columns with these separators:

| Endpoint | Field | Row sep | Col sep | Row shape |
|---|---|---|---|---|
| SetVirtualServerCfg | `list` | `~` | `,` | `ip,inPort,outPort,protocol` |
| SetStaticRouteCfg | `list` | `~` | `,` | `network,mask,gateway,ifname` (user routes only) |
| setPptpUserList | `list` | `~` | `;` | `user;pass;enable;0;;;` |
| SetNetControlList | `list` | `\n` | `\r` | `devName\rmac\rlimitUp\rlimitDown` (KB/s, ×128 of Mbps) |
| SetIpMacBind | `list` (+`bindnum`) | `\n` | `\r` | `devName\rmac\rip` |
| setMacFilterCfg | `deviceList` | `\n` | `\r` | `devName\rMAC` |
| SetDlnaCfg (unsupported) | `scanList` | `\t` | — | path |

Every list setter replaces the whole list. To add or delete one row, GET the
list, edit it, and POST the full list back.

**Result.** Setters return `{"errCode":0}`. It is a JSON number: the JS
compares it loosely and the router sends it unquoted. Most write endpoints
do this. The exceptions:
- `cloudv2` returns `err_code`, and has its own code table (see
  `cloudv2?module=olupgrade&opt=queryversion`).
- `SysToolChangePwd`, `SysToolReboot`, `SysToolRestoreSet`, `cgi-bin/UploadCfg`
  and `cgi-bin/upgrade` are native form posts. They answer with a
  redirect or page, not JSON. Errors come back as a query string on the
  redirect target (e.g. `system_password.html?1`).
- `parentControlEn` and `fast_setting_internet_set` have no response the UI
  reads.

**Status codes.** `connectStatus`/`wanStatus` are 7-digit strings. The digits
mean: can-disconnect, colour, connected, mode, WAN type, then a 2–3-digit
message key. The message table is `statusTxtObj` in the UI's `js/main.js`.
`209` means "Connected. You can access the internet now."

## Verified live

Resubmitting the current values through each of these setters returned
`errCode` 0 and left the matching GET unchanged. That confirms the body
formats below:

SetLEDCfg, SetFirewallCfg, SetDMZCfg, SetRemoteWebCfg, SetUpnpCfg,
SetVirtualServerCfg, SetIpMacBind, SetSysTimeCfg, SetSysAutoRebbotCfg,
SetPptpServerCfg, SetPptpClientCfg (disabled), PowerSaveSet, openSchedWifi,
WifiWpsSet, WifiPowerSet, WifiBeamformingSet, WifiAntijamSet,
SetDDNSCfg (disabled), `cloudv2 manage setbasic`.

These were verified by adding a dummy row, reading it back and removing it:
SetVirtualServerCfg, SetStaticRouteCfg, SetIpMacBind, setMacFilterCfg
(blacklist), `setBlackRule`/`delBlackRule`, SetNetControlList (a cap set on
one device, then removed), `saveParentControlInfo` + `delParentalRule`.

Saving WifiPowerSet, WifiBeamformingSet or WifiAntijamSet restarts the
2.4 GHz radio and the guest radio, even when the values are unchanged.
The router log shows "2.4G Main WiFi DOWN" and then UP about 4 seconds
later, once per save, so WiFi clients drop briefly. WifiWpsSet was saved in
the same run, so it may do the same.

Every getter was captured live. The other setters are documented from the
UI code only. Some were left untested because they restart a radio, drop the
WAN, or reboot the router: the WiFi radio/SSID/guest setters, WAN, LAN, MAC
clone, IPTV, AP/WISP mode, reboot, reset, and upgrade.

## Never call casually

These endpoints can cut connectivity or destroy config. Put them behind a
confirmation flag in the CLI:

- **Reboot:** `SysToolReboot`. Note that a plain `GET` reboots the router
  too, the same way `iptv.js` and `wisp.js` call it. Also `setApModeCfg` and
  `WifiExtraSet` (both change the working mode, then reboot).
  `SetIPTVCfg` is followed by a reboot in the UI.
- **Factory reset:** `SysToolRestoreSet`.
- **Config replace or flash:** `cgi-bin/UploadCfg`, `cgi-bin/upgrade`, and
  `cloudv2?module=olupgrade&opt=queryupgrade`. The first call of
  `queryupgrade` starts an online upgrade.
- **WAN:** `WanParameterSetting`, `fast_setting_internet_set`,
  `AdvSetMacMtuWan`.
- **LAN or lockout:** `AdvSetLanip`, `setMacFilterCfg` with
  `macFilterType=white`, and `SysToolChangePwd`.
- **WiFi off:** `WifiBasicSet` (`wrlEn=0`), `openSchedWifi` and
  `PowerSaveSet` (both schedule WiFi off).

## Coverage in tendactl

Run `tendactl --help`; each command's endpoints are in `tenda/<feature>.go`.

## UI names

English is the UI's source language. The router serves no English
translation file (`lang/en/translate.json` returns 400), so the English text
in the page markup and JS is exactly what renders. The labels below come
from that markup. Spot checks in the live UI matched on every page opened:
WiFi Name & Password, Channel & Bandwidth, Transmit Power, WiFi Schedule,
Wireless Repeating, Virtual Server, Firewall, Bandwidth Control, Sleeping
Mode, WAN Settings, DHCP Reservation, Automatic Maintenance, LAN Settings,
Remote Management, Guest Network and Parental Control.

Form controls often have a different `id`/`name` from the field sent on the
wire, for example `hostIp` → `dmzIp`, `bfEn` → `beamformingEn` and
`apSwitch` → `apModeEn`. The tables below map each **UI label** to the
**wire field**, which is what a command must send. Values are shown as
`wire value` = UI text where the two differ.

### Where each feature lives

| Dashboard tab | UI name | Endpoints |
|---|---|---|
| Internet Status | (landing page) | GetRouterStatus |
| Internet Status | Manage Device (click "Online") | getOnlineList, setBlackRule, getBlackRuleList, delBlackRule, SetOnlineDevName |
| Internet Settings | Internet Settings | getWanParameters, WanParameterSetting |
| WiFi Settings | (tile status) | GetWrlStatus |
| WiFi Settings | WiFi Name & Password | WifiBasicGet/Set |
| WiFi Settings | WiFi Schedule | initSchedWifi, openSchedWifi |
| WiFi Settings | Wireless Repeating | WifiExtraGet/Set, WifiApScan |
| WiFi Settings | Channel & Bandwidth | WifiRadioGet/Set |
| WiFi Settings | Transmit Power | WifiPowerGet/Set |
| WiFi Settings | WPS | WifiWpsGet/Set, WifiWpsStart |
| WiFi Settings | Beamforming+ | WifiBeamformingGet/Set |
| WiFi Settings | AP Mode | getApModeCfg, setApModeCfg |
| WiFi Settings | Anti-interference | WifiAntijamGet/Set |
| Guest Network | Guest Network | WifiGuestGet/Set |
| Parental Control | Parental Control (list) | GetParentCtrlList, parentControlEn, delParentalRule |
| Parental Control | +New / Edit popup | GetParentControlInfo, saveParentControlInfo, getParentalRuleList, SetOnlineDevName |
| VPN | (tile status) | GetVpnStatus |
| VPN | PPTP Server | GetPptpServerCfg, SetPptpServerCfg, setPptpUserList |
| VPN | Online PPTP Users | getPptpOnlineClient |
| VPN | PPTP/L2TP Client | GetPptpClientCfg/Set |
| Advanced Settings | (tile status) | GetAdvanceStatus |
| Advanced Settings | Bandwidth Control | GetNetControlList/Set, SetOnlineDevName |
| Advanced Settings | Tenda App | cloudv2 `module=manage` |
| Advanced Settings | Sleeping Mode | PowerSaveGet/Set |
| Advanced Settings | LED Control | GetLEDCfg/SetLEDCfg |
| Advanced Settings | Filter MAC Address | getMacFilterCfg, setMacFilterCfg |
| Advanced Settings | Firewall | GetFirewallCfg/Set |
| Advanced Settings | IPTV | GetIPTVCfg/Set (then SysToolReboot) |
| Advanced Settings | Static Route | GetStaticRouteCfg/Set |
| Advanced Settings | DDNS | GetDDNSCfg/Set |
| Advanced Settings | Virtual Server | GetVirtualServerCfg/Set |
| Advanced Settings | DMZ Host | GetDMZCfg/Set |
| Advanced Settings | UPnP | GetUpnpCfg/Set |
| System Settings | (tile status) | GetSysStatus |
| System Settings | LAN Settings | AdvGetLanIp, AdvSetLanip |
| System Settings | DHCP Reservation | GetIpMacBind, SetIpMacBind |
| System Settings | WAN Settings (tile subtitle "MTU/MAC/Speed") | AdvGetMacMtuWan, AdvSetMacMtuWan |
| System Settings | Time Settings | GetSysTimeCfg/Set |
| System Settings | Login Password | SysToolpassword, SysToolChangePwd |
| System Settings | Reboot and Reset | SysToolReboot, SysToolRestoreSet |
| System Settings | Firmware Upgrade | SysToolGetUpgrade, cgi-bin/upgrade, cloudv2 `module=olupgrade` |
| System Settings | Backup/Restore | cgi-bin/DownloadCfg, cgi-bin/UploadCfg |
| System Settings | Remote Management | GetRemoteWebCfg/Set |
| System Settings | System Status | GetSystemStatus |
| System Settings | System Log | GetSySLogCfg, cgi-bin/DownloadLog |
| System Settings | Automatic Maintenance | GetSysAutoRebbotCfg/Set |

Some names differ between the UI and the API:
- "DHCP Reservation" is IP-MAC binding (`IpMacBind`).
- "WAN Settings" is MAC clone, MTU and port speed (`MacMtuWan`). The WAN
  connection itself is under "Internet Settings".
- "Wireless Repeating" is `WifiExtra`/WISP.
- "Sleeping Mode" is `PowerSave`.
- "Automatic Maintenance" is the scheduled reboot.
- "Tenda App" is `cloudv2 manage`.
- The "Blacklist" in Manage Device (`setBlackRule`) is a different list from
  "Filter MAC Address".

The dashboard tile ids (`adv_*`, `sys_*`, `wrl_*`) don't match their tabs.
`adv_remoteweb` sits under System Settings, for instance.

### Field labels

**Internet Status** (GetRouterStatus): "Current Speed" is `wanInfo[].wanUploadSpeed` / `wanDownloadSpeed` (KB/s), shown with the WAN IP (`wanInfo[].wanIp`), "Firmware Version" is `onlineUpgradeInfo.curVersion`, "Online" is `clientNum`.

**Manage Device** (getOnlineList): Device Name `devName` · Upload/Download Speed `uploadSpeed`/`downloadSpeed` · Access Type `line` (`0`=Wired, `1`=2.4 GHz, `2`=5 GHz) · Blacklist "Add" → `setBlackRule mac=`. Blacklist tab: Device Name, MAC Address `deviceId`, Remove from Blacklist → `delBlackRule`.

**Internet Settings** (WanParameterSetting)

| UI label | Wire field | Values |
|---|---|---|
| Connection Type | `wanType` (form sends `netWanType`, renamed before POST) | `0` Dynamic IP Address, `1` Static IP Address, `2` PPPoE, `3`/`4`/`5` Russia PPTP/L2TP/PPPoE |
| ISP User Name / ISP Password | `adslUser` / `adslPwd` | |
| Server IP Address/Domain Name, User Name, Password | `vpnServer`, `vpnUser`, `vpnPwd` | Russia modes only |
| Address Type | `vpnWanType` | `1` Dynamic IP Address, `0` Static IP Address |
| IP Address / Subnet Mask / Default Gateway | `staticIp` / `mask` / `gateway` | |
| DNS Settings | `dnsAuto` | `1` Automatic, `0` Manual |
| Primary / Secondary DNS Server | `dns1` / `dns2` | |
| Connection Status / Connection Duration | `connectStatus` / `connectTime` (read-only) | |

**WiFi Name & Password** (WifiBasicSet)

| UI label | Wire field | Values |
|---|---|---|
| 2.4 GHz Network / 5 GHz Network (toggle) | `wrlEn` / `wrlEn_5g` | `1` on |
| WiFi Name | `ssid` / `ssid_5g` | |
| Hide (checkbox) | `hideSsid` / `hideSsid_5g` | `1` hidden |
| Encryption Mode | `security` / `security_5g` | `none` None, `wpapsk` WPA-PSK, `wpa2psk` WPA2-PSK, `wpawpa2psk` WPA/WPA2-PSK (recommended) |
| WiFi Password | `wrlPwd` / `wrlPwd_5g` | |

**Channel & Bandwidth** (WifiRadioSet). The labels are the same for both bands; the 5 GHz fields take a `_5g` suffix.

| UI label | Wire field | Values |
|---|---|---|
| Network Mode | `adv_mode` | `bgn` 11b/g/n mixed, `bg` 11b/g mixed, `n only` 11n |
| Network Mode (5 GHz) | `adv_mode_5g` | `ac` 11a/n/ac mixed, `ac only` 11ac |
| WiFi Channel | `adv_channel` | `0` Auto, else "Channel N" |
| WiFi Bandwidth | `adv_band` | `20`, `40`, `auto` shown as "20/40" (5 GHz: `80`, `auto` = "20/40/80") |

**Transmit Power** (WifiPowerSet): "2.4 GHz Signal Strength" → `power` (`low` Low, `middle` Medium, `high` High). "5 GHz Signal Strength" → `power_5g`: **`low` is labelled "Medium"**, `high` High.

**WiFi Schedule** (openSchedWifi): WiFi Schedule → `schedWifiEnable`. "Turn Off During" → `schedStartTime`–`schedEndTime`. "In" (Every Day / Specified Day) → `timeType` `0`/`1`. The Mon.…Sun. checkboxes → `day`, which is **Monday first**.

**Wireless Repeating** (WifiExtraSet)

| UI label | Wire field | Values |
|---|---|---|
| Wireless Repeating (toggle) + Repeating Mode | `wl_mode` | off = `ap`; `wisp` WISP, `apclient` Client+AP |
| Upstream WiFi Name (scan list) / WiFi Name (manual entry) | `ssid` | manual entry also sends `handset=1` |
| Frequency Band | `wifi_chkHz` | `0` 2.4 GHz, `1` 5 GHz |
| Encryption Mode | `wpapsk_type` (+ `security` `none`/`wpapsk`) | `none` NONE, `wpa` WPA-PSK, `wpa2` WPA2-PSK, `wpa&wpa2` WPA-PSK/WPA2-PSK |
| Encryption Algorithm | `wpapsk_crypto` | `aes` AES, `tkip` TKIP, `tkip&aes` TKIP&AES |
| Upstream WiFi Password | `wpapsk_key` | |

**Single toggles.** Each of these pages is just one switch or radio group:

| UI label | Wire field | Values |
|---|---|---|
| WPS | `wpsEn` | `1` on |
| Beamforming+ | `beamformingEn` | `1` on |
| AP Mode | `apModeEn` | `true` / `false` |
| Anti-interference | `WifiAntijamEn` | `auto` Auto, `true` Enable, `false` Disable |
| UPnP | `upnpEn` | `1` on |

**Guest Network** (WifiGuestSet)

| UI label | Wire field | Values |
|---|---|---|
| Guest Network (toggle) | `guestEn` and `guestEn_5g` (always equal) | |
| 2.4 GHz WiFi Name / 5 GHz WiFi Name | `guestSsid` / `guestSsid_5g` | |
| Guest Network Password | `guestWrlPwd` and `guestWrlPwd_5g` (one field, both sent) | |
| Validity | `effectiveTime` | `4` 4 hours, `8` 8 hours, `0` Always |
| Shared Bandwidth for Guests | `shareSpeed` | Mbps × 128; menu offers Unlimited (`0`), 2, 4, 8, Custom |

**Parental Control** (saveParentControlInfo; list via GetParentCtrlList)

| UI label | Wire field | Values |
|---|---|---|
| Device Name | `deviceName` | |
| MAC Address | `deviceId` | |
| Parental Control (toggle) | `enable` | |
| Internet Accessible At | `time` `HH:MM-HH:MM` | the window when access is **allowed** |
| In: Every Day / Specified Day + Sun.…Sat. | `day` | 7 flags, **Sunday first** (Every Day = all `1`) |
| Website Access Limit | `url_enable` | |
| Access Control Mode | `limit_type` | `0` Blacklist, `1` Whitelist |
| Blocked Websites | `urls` | comma-separated keywords |

In the list, the enable/disable icon → `parentControlEn isControled=`, and the delete icon → `delParentalRule`.

**PPTP Server** (SetPptpServerCfg, setPptpUserList): PPTP Server → `serverEn` · IP Address Pool → `startIp`/`endIp` · MPPE Encryption → `mppe` · Number of MPPE Encryption Bits → `mppeOp` (`40`/`128`) · user table User Name / Password / Connection Status → `list` rows / `connsta`. **Online PPTP Users** columns: User Name, Dial-In IP Address `dialIP`, Assigned IP Address `clientIP`, Uptime `onlineTime`.

**PPTP/L2TP Client** (SetPptpClientCfg)

| UI label | Wire field | Values |
|---|---|---|
| PPTP/L2TP Client (toggle) | `clientEn` | |
| Client Type | `clientType` | `pptp` PPTP, `l2tp` L2TP |
| MPPE Encryption | `clientMppe` | |
| Number of MPPE Encryption Bits | `clientMppeOp` | `40`, `128` |
| Server IP Address/Domain Name | `domain` | |
| User Name / Password | `userName` / `password` | |
| Status / Obtained PPTP Client IP Address | `pptpStatus`/`l2tpStatus`, `pptpIp`/`l2tpIp` (read-only) | `0` Disconnected, `1` Connected, `2` Connecting |

**Bandwidth Control** (SetNetControlList): columns Device Name, Upload Speed and Download Speed (shown in KB/s), then Upload Limit and Download Limit → `limitUp`/`limitDown`. The limit menus offer Unlimited (`0`), 0.5, 1.0, 2.0 Mbps or Manual. The wire value is Mbps × 128 (KB/s).

**Tenda App** (cloudv2 manage): "Manage with Tenda App" → `setbasic enable` · ID → `querybasic sn` (read-only) · Cloud Account → `setaccount list`. The page also has a Password field, but it's hidden and never sent.

**Sleeping Mode** (PowerSaveSet): Sleeping Mode → `powerSavingEn` · Sleeping Time → `time` · Indicator → `ledCloseType` (`allClose` All off, `unpowerClose` All off except power) · Delay checkbox ("Delay enabling the Sleep mode when there is an online user.") → `powerSaveDelay`.

**LED Control** (SetLEDCfg): LED Control → `ledType` (`open` Always on, `close` Always off, `time` Schedule) · Indicator → `ledCloseType` (as above) · Turn Off During → `time`.

**Filter MAC Address** (setMacFilterCfg): MAC Address Filter Mode → `macFilterType`. `black` is "Blacklist (To disallow listed devices to access the internet)" and `white` is "Whitelist (To allow only the listed devices …)". The Whitelisted/Blacklisted Device and MAC Address columns → `deviceList` rows.

**Firewall** (SetFirewallCfg): four toggles, in this order, give the four characters of `firewallEn`: ICMP Flood Attack Defense, TCP Flood Attack Defense, UDP Flood Attack Defense, Ignore Ping Packet From WAN Port.

**IPTV** (SetIPTVCfg): Multicast → `igmpEn` · IPTV → `stbEn` · VLAN → `iptvType` (`none` Default, `shanghai` Shanghai VLAN, `manual` Custom VLAN). The Shanghai area radio (`51`/`85`) → `vlanId`. The custom table "VLAN for Uplink Packets" → `vlanId` + `list`.

**Static Route** columns → row fields: Destination Network `network`, Subnet Mask `mask`, Gateway `gateway`, WAN `ifname`.

**DDNS** (SetDDNSCfg): DDNS → `ddnsEn` · Service Provider → `serverName` (`dyn.com/dns/` is shown as "dyndns.org") · User Name `ddnsUser` · Password `ddnsPwd` · Domain Name `ddnsDomain` · Connection Status `ddnsStatus` (read-only).

**Virtual Server** columns → `list` row fields: Internal IP Address `ip`, **LAN Port `inPort`**, **WAN Port `outPort`**, Protocol. The protocol dropdown shows TCP / UDP / TCP&UDP, which go on the wire as `1` / `2` / `0`.

**DMZ Host** (SetDMZCfg): DMZ Host → `dmzEn` · DMZ Host IP Address → `dmzIp` (the UI edits only the last octet).

**LAN Settings** (AdvSetLanip): LAN IP Address `lanIp` · Subnet Mask `lanMask` · DHCP Server `dhcpEn` · IP Address Range `startIp`–`endIp` · Lease Time `leaseTime` (`604800` 7 days, `172800` 2 days, `86400` 1 day, `21600` 6 hours, `3600` 1 hour) · DNS Settings `lanDnsAuto` · Primary/Secondary DNS Server `lanDns1`/`lanDns2`.

**DHCP Reservation** columns: Device Name, MAC Address, IP Address, Status, which map to the `list` row fields `devName`, `mac`, `ip`.

**WAN Settings** (AdvSetMacMtuWan)

| UI label | Wire field | Values |
|---|---|---|
| MTU | `wanMTU` | |
| Speed | `wanSpeed` | `0` 1000 Mbps auto-negotiation, `1` 10 Mbps FDX, `2` 10 Mbps HDX, `3` 100 Mbps FDX, `4` 100 Mbps HDX |
| MAC Address | `cloneType` + `mac` | `0` Default, `1` Clone local MAC address, `2` Set MAC address |
| Service Name / Server Name | `serviceName` / `serverName` | the Default/Custom select exists only in the UI; Default sends an empty value. Shown for PPPoE only |

**Time Settings** (SetSysTimeCfg): Select Time Zone → `timeZone`. **Its value is the GMT offset plus 12 hours**, e.g. `12:00` = GMT, `14:00` = GMT+02:00, `17:30` = GMT+05:30, and the range runs `0:00` (GMT−12) to `25:00` (GMT+13). The `:10` variants (`14:10`…`24:10`) are Russia/Ukraine duplicates of the same offsets. Current Time → `time` (read-only).

**Login Password** (SysToolChangePwd): Old / New / Confirm Password → `SYSOPS` / `SYSPS` / `SYSPS2`, each MD5-hashed.

**Automatic Maintenance** (SetSysAutoRebbotCfg): System Reboot Schedule → `autoRebootEn` · Reboot At → `rebootTime` (minutes in 5-min steps) · Delay checkbox ("…higher than 3 KB/s") → `delayRebootEn` `true`/`false`.

**Remote Management** (SetRemoteWebCfg): Remote Management `remoteWebEn` · Remote IP Address `remoteIp` (`0.0.0.0` = any) · Port `remotePort`.

**Firmware Upgrade**: Upgrade Type (Online Upgrade / Local Upgrade) is UI-only · Current Version `cur_fw_ver` · Select Upgrade File `upgradeFile`. **Backup/Restore**: the restore file field is `filename`.

**System Status** (GetSystemStatus, read-only)

| UI label | Field |
|---|---|
| System Time / Uptime / Firmware Version / Hardware Version | `adv_sys_time` / `adv_run_time` / `adv_firm_ver` / `adv_hard_ver` |
| WAN1 Port speeds | `wanInfo[].wanUploadSpeed` / `wanDownloadSpeed` |
| Connection Type / Status / Duration | `adv_connect_type` / `adv_connect_status` / `adv_connect_time` |
| IP Address / Subnet Mask / Default Gateway / Primary DNS / Secondary DNS / MAC Address (WAN) | `adv_ip` / `adv_mask` / `adv_gateway` / `adv_dns1` / `adv_dns2` / `adv_mac` |
| LAN IP Address / Subnet Mask / MAC Address | `adv_lan_ip` / `adv_lan_mask` / `adv_lan_mac` |
| 2.4 GHz Network | `adv_wrl_en`. This is the **hidden-SSID** flag, not on/off |
| **Hotspot Name** (the SSID) / Encryption Mode / WiFi Channel / WiFi Bandwidth / MAC Address | `adv_wrl_ssid` / `adv_wrl_sec` / `adv_wrl_channel` / `adv_wrl_band` / `adv_wrl_mac` (5 GHz: `_5g`) |

---

# Dashboard, quick setup, login


Pages: `main.html` (post-login dashboard), `index.html` (quick setup wizard), the login page.
On a configured router `index.html` is a redirect stub to `main.html`, so the wizard's field names
below come from `js/index.js` selectors only.

## Endpoint summary

| Endpoint | Method | Page | Type | Risk |
|---|---|---|---|---|
| `goform/GetSysAutoRebbotCfg` | GET | main.html | get (used only as a cheap session-alive probe) | Safe |
| `goform/getHomeLink` | GET | main.html | get | Safe |
| `goform/GetRouterStatus` | GET | main.html | get (polled 5s) | Safe |
| `goform/getWanParameters` | GET | main.html | get (polled 5s) | Safe |
| `goform/WanParameterSetting` | POST | main.html | set/action | Can drop/change WAN connectivity; can force a reboot |
| `goform/GetWrlStatus` | GET | main.html | get | Safe |
| `goform/WifiGuestGet` | GET | main.html | get | Safe |
| `goform/WifiGuestSet` | POST | main.html | set | Can disconnect existing guest-WiFi clients |
| `goform/GetParentCtrlList` | GET | main.html | get (polled 5s) | Safe |
| `goform/parentControlEn` | POST | main.html | action | Blocks/unblocks a device's internet access |
| `goform/delParentalRule` | POST | main.html | action | Removes a parental-control rule (device becomes unrestricted) |
| `goform/GetUSBStatus` | GET | main.html | get | Safe |
| `goform/GetVpnStatus` | GET | main.html | get (polled 5s) | Safe |
| `goform/GetAdvanceStatus` | GET | main.html | get | Safe |
| `goform/GetSysStatus` | GET | main.html | get | Safe |
| `goform/exit` | GET (plain link) | main.html | action | Logs the session out |
| `goform/getProduct` | GET | index.html | get | Safe |
| `goform/fast_setting_get` | GET | index.html | get (polled during wizard) | Safe |
| `goform/fast_setting_internet_set` | POST | index.html | set | Rewrites WAN config; can drop internet |
| `goform/getSyncAccount` | GET | index.html | get | Safe (returns cloud-stored ISP/PPPoE credentials in cleartext) |
| `goform/getWanConnectStatus` | GET | index.html | get | Safe |
| `goform/fast_setting_wifi_set` | POST | index.html | set | Changes WiFi SSID/password and login password; disconnects WiFi clients |
| `goform/fast_setting_pppoe_set` | POST | index.html | **dead code** (call is commented out) | N/A — not invoked |
| `/login/Auth` | POST | login page | action | Safe (authentication only) |

`goform/getProduct` is also referenced from `main.js` but that call is
commented out there (dead code); the live call lives in `index.js` only, and
is documented once, under index.js.

`goform/getWanParameters` / `goform/WanParameterSetting` are also used by
`net_set.js` (a different page/group — the WAN-settings iframe opened from
the dashboard's WAN status tile). `main.js` builds its own independent
request for these two endpoints (documented below); `net_set.js`'s request
construction is out of scope here, but note the two pages hit the exact
same goform handlers, so a change to one payload shape should be checked
against the other.

---

## main.js

### `GET goform/GetSysAutoRebbotCfg` — page: main.html (dashboard, all tabs)

- **Purpose**: Not used for data — fired on every tab switch purely as a
  cheap "is my session still valid" probe (`PageLogic.initModule`, main.js:254).
- **Request**: `GET goform/GetSysAutoRebbotCfg?<random>` (cache-buster only, no other params).
- **Response**: Real config object (auto-reboot schedule), e.g.
  `{"autoRebootEn":"1","time":"03:00-05:0","rebootTime":"03:00","delayRebootEn":"false","timeUp":"0","speed":"3"}`.
  main.js ignores all of this and only checks whether the raw response text
  contains the literal string `"<!DOCTYPE"` — if so the session has expired
  (router served the login page instead of JSON) and the page is force-reloaded.
- **Risk**: Safe.
- **Notes**: Quirk — this is the actual "auto reboot" config endpoint (also
  used for real by `system_automaintain.js`, another page), repurposed here
  purely as a lightweight keep-alive/logout-detector because it's cheap to
  compute. Fired once per menu-tab change, not on a timer.

### `GET goform/getHomeLink` — page: main.html (dashboard, System Status tab / "WiFi Extender" tile)

- **Purpose**: Fetches the vendor homepage URL used by the "WiFi Extender" / Transmit-Power help links, only when the UI language is Chinese.
- **Request**: `GET goform/getHomeLink` (no cache-buster, no params). Only called when `B.getLang() == "cn"`.
- **Response**: `{"homePageLink":"http://www.tenda.com.cn"}` — single string field. For any other language, main.js never calls the endpoint and hardcodes `G.homePage = "http://www.tendacn.com/en/product/A9.html"`.
- **Risk**: Safe.
- **Notes**: Called from `staInfo.getHomeLink()`, itself triggered by clicking the "WiFi Extender" tile or the "Transmit Power" advanced-wifi tile.

### `GET goform/GetRouterStatus` — page: main.html (Internet Status tab, UI title "Internet Status")

- **Purpose**: Populates the whole dashboard landing tab: work mode, WAN/AP/WISP status, online client count, WiFi on/off + names, firmware version, "new firmware available" banner.
- **Request**: `GET goform/GetRouterStatus` (no cache-buster). Polled every 5000ms via `setTimeout` while this tab is active (`staInfo.initValue`), stopped when the user navigates away.
- **Response** (live example, secrets redacted — this response has none):
  ```json
  {"wl5gEn":"0","wl5gName":"","wl24gEn":"1","wl24gName":"Device1","lineup":"1|1|1|0","clientNum":13,"blackNum":0,"listNum":0,"deviceName":"AC10","lanIP":"192.168.0.1","lanMAC":"aa:bb:cc:dd:ee:ff","workMode":"router","apStatus":"1310209","wanInfo":[{"wanStatus":"1310209","wanIp":"203.0.113.10","wanUploadSpeed":"7.23","wanDownloadSpeed":"521.28"}],"onlineUpgradeInfo":{"newVersionExist":"0","newVersion":"","curVersion":"V15.03.06.50_multi"}}
  ```
  Field-by-field (consumed by `staInfo.setImage` unless noted):
  - `wl24gEn` / `wl5gEn` (string `"0"`/`"1"`): whether each radio's main WiFi is on; drives the "Disable" label vs the SSID text.
  - `wl24gName` / `wl5gName` (string): current SSID for each band; only shown when the corresponding `*En` is `"1"`.
  - `clientNum` (number): count shown next to "Online" on the dashboard.
  - `deviceName` (string): device model (`"AC10"`); **not read by main.js** (dead field for this page).
  - `lineup` (string, `|`-joined flags): **not read by main.js** — likely a per-interface link-up bitfield (LAN1..LANn/WAN), inferred from the field name only; the equivalent per-WAN cable-up flag main.js actually uses comes from `getWanParameters`'s `lineUp` field instead (see below).
  - `blackNum` / `listNum` (numbers): **not read by main.js**; by naming convention likely MAC-filter blacklist count and a generic list count, not decoded further here since the code doesn't use them.
  - `lanIP` / `lanMAC` (strings): shown in AP/APClient mode (`setAPMode`/`setApclientMode`) as the router's LAN IP/MAC when it's acting as an access point rather than a router.
  - `workMode` (enum string): `"router"`, `"ap"`, `"wisp"`, or (else) treated as `"client+ap"`(AP Client). Selects which status card is rendered (`setRouterMode`/`setAPMode`/`setWispMode`/`setApclientMode`) and which nav menu items get hidden (`showWorkMode`).
  - `apStatus` (7-digit status code string, same shape as `wanInfo[].wanStatus` below): used in AP mode to color the "connected to upstream" indicator; only `"2303002"` (success) and `"2103001"` (fail) are distinguished, anything else is treated as failed.
  - `wanInfo[]` (array, one entry per WAN, only `wanInfo[0]` used in non-router modes):
    - `wanStatus`: a 7-digit status code, decoded positionally per a comment in the code (`statusTxtObj`, main.js:574-582): digit1 = can-disconnect flag (`1`=yes,`2`=no), digit2 = display color (`1`=error,`2`=trying,`3`=success), digit3 = connected flag (`0`/`1`, controls whether connect-duration is shown), digit4 = sub-mode (`0`=AP,`1`=WISP,`2`=APClient), digit5 = WAN type (`0`=DHCP,`1`=Static,`2`=PPPoE), digits 6-7 = a lookup key into `statusTxtObj` (e.g. last 4 digits `"1209"`→key `"209"` → "Connected. You can access the internet now."). The full `statusTxtObj` table (main.js:574-638) covers keys `1-9` (AP/DHCP), `101-108` (AP/static), `201-210` (AP/PPPoE), `1001-1008`/`1101-1108` (WISP), `2001-2003` (APClient).
    - `wanIp` (string, empty when not connected): shown as the WAN IP on the tile.
    - `wanUploadSpeed` / `wanDownloadSpeed` (decimal strings, presumably KB/s): summed across WANs and passed through `translateSpeed()` for display.
  - `onlineUpgradeInfo.curVersion` (string): current firmware, shown as "Firmware Version".
  - `onlineUpgradeInfo.newVersionExist` (`"0"`/`"1"`): shows/hides the "NEW" firmware badge that links to the firmware-upgrade iframe.
  - `onlineUpgradeInfo.newVersion` (string): not read by main.js in this build.
  - `usbNum`: read by `staInfo.setUSB(obj.usbNum)` but **absent from this live response** (this AC10 has no USB port; the USB tile is hidden entirely via the `CONFIG_USB_MODULES` build macro when absent).
- **Risk**: Safe (read-only, polled).
- **Notes**: Polling stops (`clearTimeout`) as soon as the user leaves the System-Status tab (`mainPageLogic.modelObj != "staInfo"` check is implicit via the `init`/`initValue` guard).

### `GET goform/getWanParameters` — page: main.html (Internet Settings tab, UI title "Internet Settings")

- **Purpose**: Loads current WAN1/WAN2 configuration to populate the connection-type form and connection-status readout.
- **Request**: `GET goform/getWanParameters?<random>` on tab open, then polled every 5000ms via `AjaxInterval` with the same URL (no cache-buster on the polled calls) while the tab stays active and the visible form still matches the on-router state.
- **Response** (live example — secrets redacted):
  ```json
  {"country":"US","wl_mode":"ap","lanIp":"192.168.0.1","lanMask":"255.255.255.0","guestIp":"192.168.10.1","guestMask":"255.255.255.0","wanInfo":[{"wanType":"2","pptpSvrIp":"","pptpSvrMask":"","connectTime":"131876","connectStatus":"1310209","downSpeedLimit":"","wanIp":"203.0.113.10","staticIp":"","mask":"","gateway":"","vpnClient":"0","vpnClientUser":"","dnsAuto":"0","dns1":"1.1.1.1","dns2":"8.8.8.8","vpnWanType":"","vpnServer":"","vpnUser":"","vpnPwd":"","adslUser":"<redacted>","adslPwd":"<redacted>"},{"wanType":"0","pptpSvrIp":"","pptpSvrMask":"","connectTime":"0","connectStatus":"2100002","downSpeedLimit":"","wanIp":"","staticIp":"","mask":"","gateway":"","vpnClient":"","vpnClientUser":"","dnsAuto":"1","dns1":"","dns2":"","vpnWanType":"","vpnServer":"","vpnUser":"","vpnPwd":"","adslUser":"","adslPwd":""}],"multiWanEn":"false","lineUp":"11"}
  ```
  - `country` (string): read but only used indirectly (locale-linked validation); not otherwise decoded here.
  - `wl_mode` (enum: `"ap"`, `"wisp"`, `"apclient"`/other = router): if `"wisp"`, the PPPoE connection-type option is removed from the dropdown (no PPPoE dial-up while repeating).
  - `lanIp` / `lanMask`: used for same-subnet validation before submitting a static-IP WAN config.
  - `guestIp` / `guestMask`: present but not read by main.js on this tab.
  - `multiWanEn` (`"true"`/`"false"`): whether dual-WAN load balancing is on; toggling it via the UI queues a reboot confirmation.
  - `lineUp` (string, one char per WAN, e.g. `"11"`): `"1"` = Ethernet cable plugged into that WAN port, else unplugged; drives the "Ethernet cable connected/disconnected" line under each WAN's port label.
  - `wanInfo[]` (array, index 0 = WAN1, index 1 = WAN2 if `multiWanEn`):
    - `wanType` (enum): `0`=Dynamic IP(DHCP), `1`=Static IP, `2`=PPPoE, `3`=Russia PPTP, `4`=Russia L2TP, `5`=Russia PPPoE (the last three options only offered when browser locale is RU/UK). Selects which sub-form is shown.
    - `connectStatus` / `connectTime`: same 7-digit status-code scheme as `GetRouterStatus.wanInfo[].wanStatus` above; `connectTime` (seconds) drives the "Connection Duration" readout when connected.
    - `downSpeedLimit` (string, Mbps or blank=unlimited): per-WAN upload bandwidth cap shown/edited via a dropdown+custom-value control.
    - `wanIp`, `staticIp`, `mask`, `gateway`, `dns1`, `dns2`, `dnsAuto` (`"1"`=auto,`"0"`=manual): static/DHCP addressing fields.
    - `vpnClient` (`"1"`/other): whether a PPTP/L2TP client is separately enabled on this WAN; if so and the user is about to switch this WAN to type 3/4 (PPTP/L2TP dual-access), the UI warns "Changing the settings will disable the VPN function."
    - `vpnClientUser`, `vpnWanType` (`"1"`=dynamic,`"0"`=static, used for the Russia PPTP/L2TP dual-access sub-form), `vpnServer`, `vpnUser`, `vpnPwd`: PPTP/L2TP dual-access fields (Russia-locale only).
    - `pptpSvrIp` / `pptpSvrMask`: used only for a same-subnet validation check against a manually entered static IP.
    - `adslUser` / `adslPwd`: PPPoE credentials — `<redacted>` above (real values are the router's live ISP login).
- **Risk**: Safe (read-only, polled).

### `POST goform/WanParameterSetting` — page: main.html (Internet Settings tab)

- **Purpose**: Connects/disconnects a WAN, or saves new WAN1/WAN2/dual-WAN settings.
- **Request**: Body is `application/x-www-form-urlencoded`, built by serializing the visible `<form>` (`#wan1SetWrap` and/or `#wan2SetWrap`) plus extra hand-appended fields; three variants depending on which button was clicked:
  - **Disconnect** (`wan_submit`/`wan_submit2` clicked while button reads "Disconnect"): body is just `module=wan1&action=disconnect` (or `module=wan2&action=disconnect`) — no other fields.
  - **Save/Connect WAN1 only**: `$("#wan1SetWrap").serialize()` (fields: `netWanType`→renamed to `wanType`, and depending on the visible sub-form any of `adslUser`, `adslPwd`, `vpnServer`, `vpnUser`, `vpnPwd`, `vpnWanType`, `dnsAuto`, `staticIp`, `mask`, `gateway`, `dns1`, `dns2` — all inputs in the form are serialized regardless of which sub-section is CSS-hidden) `+ "&module=wan1&downSpeedLimit=" + <value>`. `dnsAuto` is force-corrected to `1` if `#dns1` is empty and visible, else `0`.
  - **Save/Connect WAN2 only**: same shape with `netWanType2`→`wanType2`, field suffix `2` on every name, `&module=wan2&downSpeedLimit2=<value>`.
  - **Save all (dual-WAN toggle)**: WAN1 serialize + `&` + WAN2 serialize + `&module=wan1wan2&downSpeedLimit=<v1>&downSpeedLimit2=<v2>&multiWanEn=true|false`.
  - Example literal body for a plain "reconnect WAN1" click on this router's live PPPoE config (secrets redacted):
    `wanType=2&adslUser=<redacted>&adslPwd=<redacted>&vpnServer=&vpnUser=&vpnPwd=&vpnWanType=1&dnsAuto=0&staticIp=&mask=&gateway=&dns1=1.1.1.1&dns2=8.8.8.8&module=wan1&downSpeedLimit=`
- **Response**: JSON `{"errCode":0, "sleep_time": <n>}` (only these two keys read). `errCode == 0` → success (shows a saved message and reloads the tab's data); anything else shows an error. `sleep_time` is only used to infer `isVpn` but that computed flag (`isVpn`) is actually unused afterwards (dead variable) — the UI's post-submit wait timing (`waitTime`/`minTime`) is likewise computed but never applied to anything visible in this build.
- **Risk**: **High** — reconfiguring or disconnecting the active WAN drops internet access; toggling dual-WAN (`multiWanEn`) triggers a confirm() dialog warning the settings only take effect after a reboot, and on confirm the router is expected to reboot.
- **Notes**: Client-side validation (`checkWanData`) blocks obviously-bad submissions (same-subnet IP/gateway/DNS clashes, blank PPPoE creds, IP starting with 127/broadcast, etc.) before the POST is even sent. This same `goform/getWanParameters` / `goform/WanParameterSetting` pair is also driven by `net_set.js` (the WAN-status iframe reachable from the dashboard tile) — see the group covering that file for its request shape.

### `GET goform/GetWrlStatus` — page: main.html (WiFi Settings tab, UI title "WiFi Settings")

- **Purpose**: Populates the enable/disable status labels on each WiFi-related tile (schedule, WPS, beamforming, AP mode, repeating, anti-interference, transmit power, SSID/password).
- **Request**: `GET goform/GetWrlStatus?<random>`, fetched once per tab activation (not polled).
- **Response** (live): `{"schedWifiEn":"0","wispEn":"0","wpsEn":"0","namePwd":"1","signal":"22","beamforming":"1","apMode":"0","WifiAntijamEn":"auto"}`
  - `schedWifiEn` (`"0"`/`"1"`): WiFi Schedule on/off.
  - `beamforming` (`"1"`=on, else off).
  - `apMode` (`"1"`=on, else off): "AP Mode" tile.
  - `wispEn` (`0`=off, `2`=connected/repeating succeeded, else=on-but-not-connected): drives "Disable"/"Connected"/"Enable" label on the Wireless Repeating tile.
  - `WifiAntijamEn` (enum string `"false"`/`"true"`/anything else): anti-interference status; anything other than the literal strings `"false"`/`"true"` is shown as "Auto" (the live value here, `"auto"`, falls into that catch-all).
  - `namePwd` (`"0"`=Disable, else Enable): status for the "WiFi Name & Password" tile.
  - `signal` (2-digit enum string, digit1 = 2.4GHz power, digit2 = 5GHz power): decoded via a lookup table (`signalMsg`, main.js:2013-2020) — only the combinations `"00"`,`"02"`,`"10"`,`"12"`,`"20"`,`"22"` are mapped (2.4GHz: `0`=Low,`1`=Medium,`2`=High; 5GHz: `0`=Medium,`2`=High — no low-power option coded for 5GHz). Live value `"22"` → "2.4 GHz High/5 GHz High".
  - `wpsEn` (`"1"`=Enable, else Disable).
- **Risk**: Safe.

### `GET goform/WifiGuestGet` — page: main.html (Guest Network tab, UI title "Guest Network")

- **Purpose**: Loads the guest-WiFi form (SSID, password, validity period, shared bandwidth cap, on/off).
- **Request**: `GET goform/WifiGuestGet?<random>`, fetched once per tab activation.
- **Response** (live, secrets redacted): `{"wl_en":"1","wl_mode":"ap","guestEn":"1","guestEn_5g":"1","hideSsid":"0","guestSsid":"Device4","guestWrlPwd":"<redacted>","hideSsid_5g":"0","guestSsid_5g":"Device4","guestWrlPwd_5g":"<redacted>","effectiveTime":"0","shareSpeed":"0"}`
  - `guestEn` (`"1"`=on): drives the on/off toggle; the form is disabled (`guest_submit` button) unless `wl_mode == "ap"` and `wl_en == "1"` (guest network requires the main radio to be in AP/repeating mode and enabled).
  - `guestSsid` / `guestSsid_5g`: current 2.4GHz/5GHz guest SSIDs, copied straight into the corresponding text inputs.
  - `guestWrlPwd` / `guestWrlPwd_5g`: current guest password; only one password input is shown in the form and both bands share it.
  - `effectiveTime` (enum `"4"`/`"8"`/`"0"`): guest-network validity — 4 hours / 8 hours / Always.
  - `shareSpeed` (integer, stored server-side in units of 1/128 Mbps i.e. divide by 128 to get the displayed Mbps value, `"0"` = Unlimited): shared bandwidth cap for guest clients.
  - `wl_en`, `wl_mode`, `hideSsid`, `hideSsid_5g`: read but only `wl_en`/`wl_mode` are used (see above); `hideSsid*` not read by main.js on this tab.
- **Risk**: Safe.

### `POST goform/WifiGuestSet` — page: main.html (Guest Network tab)

- **Purpose**: Saves the guest-network configuration.
- **Request**: Custom key=value string (via `objTostring`, not a form `.serialize()`), fields:
  `guestEn`, `guestEn_5g` (both set to the same on/off toggle value), `guestSecurity`, `guestSecurity_5g`, `guestSsid`, `guestSsid_5g`, `guestWrlPwd`, `guestWrlPwd_5g`, `effectiveTime`, `shareSpeed` (submitted value × 128).
  Example (values from the live GET above, on/off state and password redacted per policy):
  `guestEn=1&guestEn_5g=1&guestSecurity=wpapsk&guestSecurity_5g=wpapsk&guestSsid=Device4&guestSsid_5g=Device4&guestWrlPwd=<redacted>&guestWrlPwd_5g=<redacted>&effectiveTime=0&shareSpeed=0`
- **Response**: JSON `{"errCode":0}`; `0` → shows saved message and reloads the tab's data.
- **Risk**: Enabling/disabling the guest network or changing its password disconnects any devices currently on the guest WiFi. Does not affect the main WiFi or WAN.
- **Notes**: Quirk — `guestSecurity`/`guestSecurity_5g` are computed as `$("#wrlPwd").val() != "" ? "wpapsk" : "none"`, but no element with id `wrlPwd` exists on this form (the actual password field is `#guestWrlPwd`). `.val()` on a nonexistent jQuery selection returns `undefined`, and `undefined != ""` is always `true`, so **`guestSecurity` is unconditionally sent as `"wpapsk"`**, even when the guest password field is left blank — an apparent copy-paste bug (likely meant to reference the guest password field to fall back to open security when blank).

### `GET goform/GetParentCtrlList` — page: main.html (Parental Control tab, UI title "Parental Control")

- **Purpose**: Lists all known LAN devices plus their parental-control rule state, for the device table.
- **Request**: `GET goform/GetParentCtrlList?<random>` on tab open, then polled every 5000ms (same URL, no cache-buster) while the tab is active.
- **Response**: a JSON array, one object per known device, e.g.:
  ```json
  {"devType":"unknown","onlineTime":210,"deviceId":"aa:bb:cc:dd:ee:ff","ip":"192.168.0.151","devName":"Device2","isControled":"0","isSet":"0","line":"1"}
  ```
  - `deviceId` (string): the device's MAC address; used as the key for `parentControlEn`/`delParentalRule`.
  - `devName` (string): shown as the device name (falls back to a generic "Unknown device" label when empty).
  - `devType` (string, e.g. `"unknown"`): passed through `translateDeviceType()` (not in this file) to pick a device icon.
  - `ip` (string, empty if none): shown under the device name, or `---` if empty.
  - `line` (`"0"`=offline,`"1"`=online): controls the row's online/offline styling and the connection-duration column.
  - `onlineTime` (integer seconds): formatted into the "Uptime" column when online.
  - `isSet` (`"0"`/`"1"`): whether a parental-control rule exists for this device — gates whether the enable/disable/delete icons are shown.
  - `isControled` (`"0"`/`"1"`): whether the existing rule is currently enabling or disabling the device's access — drives the enable/disable icon shown.
  - `block` (`0`/`1`, not present in this capture's sample rows but checked in code): rows with `block == 1` are skipped entirely from the rendered list.
  - Row ordering: sorted client-side, online-first, then unconfigured-before-configured.
  - Client-side cap: the "+New" rule button refuses to open the add-rule iframe once 30 rows with `data-set="true"` already exist ("Only a maximum of 30 rules are allowed.").
- **Risk**: Safe (read-only, polled).

### `POST goform/parentControlEn` — page: main.html (Parental Control tab)

- **Purpose**: Toggles an existing parental-control rule's enabled/disabled state from the device list (the enable/disable icon).
- **Request**: `POST goform/parentControlEn` with body `mac=<device MAC>&isControled=1` (clicking "enable" — i.e., turn access control ON, blocking the device) or `isControled=0` (turn control OFF, i.e. unblock/allow).
- **Response**: Not read at all — the `$.post` call has no success callback; the UI toggles its icon optimistically before the request is even sent, and never checks whether the server actually applied the change.
- **Risk**: Blocks/unblocks that device's internet access. Only affects the targeted device, not the whole network. Safe in the sense of not affecting router availability.
- **Notes**: Fire-and-forget — if this request fails silently, the on-screen icon state can drift from the router's actual state until the next 5s poll refresh.

### `POST goform/delParentalRule` — page: main.html (Parental Control tab)

- **Purpose**: Deletes a device's parental-control rule entirely (only available once the device is offline and has a rule set).
- **Request**: `POST goform/delParentalRule` with body `mac=<device MAC>` (plain string, not urlencoded key/value via `objTostring`). Preceded by a JS `confirm("Do you want to continue?")` prompt.
- **Response**: JSON `{"errCode":...}`; result is passed to a generic "saved" message display, then the list is refreshed via `GetParentCtrlList`.
- **Risk**: Removes access restriction from that device once it reconnects (device becomes unrestricted again). Does not affect other devices or router availability.
- **Notes**: The delete icon is only shown when `line === "0"` (offline) **and** `isSet === "1"` (has a rule) — can't delete a rule for a currently-online device from this list.

### `GET goform/GetUSBStatus` — page: main.html (USB App tab, UI title "USB App")

- **Purpose**: Populates the USB-feature tiles (Share File/Samba, DLNA, Share Printer).
- **Request**: `GET goform/GetUSBStatus?<random>`, fetched once per tab activation. This tab/nav entry is hidden entirely when the `CONFIG_USB_MODULES` build macro is `"n"`.
- **Response**: On this AC10 (which has no USB port — `CONFIG_USB_MODULES=n`), the live capture is **not JSON** but an HTML error page: `Access Error: Data follows — Form GetUSBStatus is not defined`. On hardware with a USB port, the expected fields (inferred from the consuming code) would be `printer` (`"0"`/other), `dlna` (`"0"`/other), `hasusb`(`"0"`/other) — each mapped to Enable/Disable on the corresponding tile.
- **Risk**: Safe.
- **Notes**: Quirk — the goform handler is simply absent on USB-less firmware builds; calling it returns a generic router error page rather than a per-field "not supported" response. main.js doesn't special-case this (the tab is just hidden via the build macro so the call normally never fires on this hardware).

### `GET goform/GetVpnStatus` — page: main.html (VPN tab, UI title "VPN")

- **Purpose**: Populates the PPTP Server / PPTP-L2TP Client / Online Users tiles.
- **Request**: `GET goform/GetVpnStatus?<random>` on tab open, then polled every 5000ms while the tab is active.
- **Response** (live): `{"server":"0","client":"0","workMode":"ap","wanType":"2","users":"0"}`
  - `server` (`"0"`=off, else on): PPTP Server tile.
  - `client` (`"0"`=off, else on): PPTP/L2TP Client tile.
  - `users` (integer as string): shown as "`<n>` user(s)" on the Online Users tile.
  - `wanType` (string, same WAN-type enum as `getWanParameters`): if `"3"` (Russia PPTP) or `"4"` (Russia L2TP), clicking the "PPTP/L2TP Client" tile is blocked client-side with an alert ("This function is not available at the moment.") because the WAN is already using that protocol.
  - `workMode`: read but not used on this tab.
- **Risk**: Safe (read-only, polled).

### `GET goform/GetAdvanceStatus` — page: main.html (Advanced Settings tab, UI title "Advanced Settings")

- **Purpose**: Populates every tile's enable/configured status on the Advanced Settings tab (bandwidth control, LED, Tenda App/cloud, sleep mode, DDNS, UPnP, IPTV, parental control, virtual server, MAC filter, firewall, static route, DMZ) and disables several tiles entirely in AP-client mode.
- **Request**: `GET goform/GetAdvanceStatus?<random>`, fetched once per tab activation (not polled).
- **Response** (live): `{"nopwd":"false","netControl":"0","led":"1","cloud":"0","sleepMode":"0","remoteWeb":"0","ddns":"0","upnp":"1","iptv":"0","wl_mode":"ap","dmz":"0","parentControl":"0","firewall":"1","staticRoute":"0","virtualServer":"1","macFilterType":"black"}`
  - `netControl`, `led`, `cloud`, `ddns`, `upnp`, `iptv`, `firewall` (`0`=Disable, else Enable — compared loosely, not as strings): straightforward enable/disable tiles.
  - `sleepMode` (compared to `1`): Enable/Disable tile.
  - `parentControl`, `virtualServer`, `staticRoute` (`"0"`=Not Configured, else Configured — string comparison): "configured" style tiles rather than plain on/off.
  - `dmz` (`"0"`=Disable, else Enable).
  - `macFilterType` (enum `"black"`/other): shows "Blacklist" vs "Whitelist" on the MAC-filter tile.
  - `wl_mode` (`"apclient"` = AP-Client): when true, disables (greys out, click no-ops) the Bandwidth Control, Parental Control, Remote Management, DDNS, UPnP, Virtual Server and DMZ tiles, since these features don't apply when the router is itself a wireless client.
  - `nopwd`, `remoteWeb`: read but not used on this tab in this build (`remoteWeb` is actually driven by `GetSysStatus` instead, see below).
- **Risk**: Safe.

### `GET goform/GetSysStatus` — page: main.html (System Settings tab, UI title "System Settings")

- **Purpose**: Populates the System Settings tab's tiles (LAN settings, firmware version/upgrade notice, automatic maintenance, remote management, DHCP reservation, time sync) and disables WAN/DHCP-reservation tiles in AP-client mode.
- **Request**: `GET goform/GetSysStatus?<random>`, fetched once per tab activation (not polled). Note: distinct from `GetSystemStatus` (a different, longer-named goform handler used by the separate `system_status.html` iframe reached from the "Router"/"System Status" tile — not called from this tab).
- **Response** (live): `{"firmware":"V15.03.06.50_multi","rebootEn":"1","remoteWeb":"0","wl_mode":"router","lan":"192.168.0.1","syncInternetTime":"0","apClientConnect":"0","ipMacBindEn":"1"}`
  - `lan` (string): shown directly as the LAN Settings tile's status text.
  - `firmware`: compared to the literal string `"1"` for "New version detected", else shown verbatim as the firmware string (in this response, shown as-is since it's not `"1"`) — the `"1"` case looks unreachable in practice since `firmware` is a version string, not a flag; likely dead/vestigial logic.
  - `rebootEn` (compared to `1`): Enable/Disable on the "Automatic Maintenance" tile.
  - `remoteWeb` (`0`=Disable, else Enable): Remote Management tile status. Clicking that tile passes a `"nopwd"` query flag to the iframe only when `sysInfo.data.nopwd === true` — but `nopwd` is never present in this response, so that branch is effectively dead for this endpoint (it reads a field this call doesn't return).
  - `wl_mode` (`"apclient"`) or the global `G.workMode == "ap"`: disables the WAN Settings and DHCP Reservation tiles; if additionally `apClientConnect == "1"`, also disables the LAN Settings tile.
  - `ipMacBindEn` (`"0"`=Not Configured, else Configured): DHCP Reservation tile.
  - `syncInternetTime` (`"1"`=Synchronized, else Unsynchronized): Time Settings tile.
- **Risk**: Safe.

### `GET goform/exit` — page: main.html (all tabs, top-right "Exit" link)

- **Purpose**: Plain `<a href="goform/exit">Exit</a>` link (not an AJAX call) that logs the current session out.
- **Request**: `GET goform/exit`, no params — a full page navigation, not an XHR.
- **Response**: Router-rendered page (presumably the login page); not inspected by any JS since this is a normal link navigation.
- **Risk**: Safe — logs the user out, no configuration change.

---

## index.js (quick setup wizard)

### `GET goform/getProduct` — page: index.html (quick setup wizard, step 1)

- **Purpose**: Determines whether the wizard should show wired vs. wireless-specific messaging/flows.
- **Request**: `GET goform/getProduct?<random>`, fired once on page load (`getInitData`).
- **Response** (live): `{"product":"f1203","accessType":"1"}`.
  - `accessType` (integer, `1`=wired / else wireless per the code's own comment `//1有线，2无线`): stored in `G.accessType`, later used at the end of the wizard to decide whether to show the "connected via cable, you can now browse the web" tip vs. hiding it.
  - `product` (string, e.g. `"f1203"`): read into a local variable but the code that would use it to pick a product image is commented out — effectively unused in this build.
- **Risk**: Safe.

### `GET goform/fast_setting_get` — page: index.html (quick setup wizard)

- **Purpose**: The wizard's main state/poll endpoint — returns current WAN detection state, LAN IP, and pre-fill values for the WiFi/login-password/MAC-clone steps. Called repeatedly through the flow: on load, on a 2s/5s retry loop while waiting for cable detection, and again each time the user clicks "Next".
- **Request**: `GET goform/fast_setting_get?<random>` — no other params.
- **Response** (live, secrets redacted):
  ```json
  {"line":1,"wanType":2,"net":1,"outType":1,"timeout":"0","lanIp":"192.168.0.1","lanMask":"255.255.255.0","adslUser":"<redacted>","adslPwd":"<redacted>","ssid":"Device1","wrlPassword":"<redacted>","country":"US","power":"high","cloneType":"0","mac":"aa:bb:cc:dd:ee:ff","deviceMac":"aa:bb:cc:dd:ee:ff","defMac":"aa:bb:cc:dd:ee:ff"}
  ```
  - `line` (`0`/`1`): whether an Ethernet cable is plugged into the WAN port. `0` → shows the "please connect the cable" panel and keeps polling every 2s; `1` moves the wizard on to WAN-type detection.
  - `net` (`0`/`1`): whether WAN-type auto-detection has finished; `0` → shows "detecting..." and polls again every 5s; `1` → reveals the connection-type step.
  - `wanType` (`0`/`1`/`2`, or other/unset): the auto-detected type; if it's exactly `0`/`1`/`2` (Dynamic/Static/PPPoE) it's pre-selected as the recommended type, otherwise `outType` is used as the fallback recommendation.
  - `outType` (integer, same enum as `wanType`): fallback recommended connection type, shown via the `outTypeMsg` lookup (`0`=Dynamic IP Address,`1`=Static IP Address,`2`=PPPoE,`3`=Russia PPTP,`4`=Russia L2TP,`5`=Russia PPPoE).
  - `timeout` (string `"0"`/`"1"`): `"1"` → detection timed out, shows "Detection timed out. Please select your connection type manually." instead of the auto-detected recommendation banner.
  - `lanIp` / `lanMask`: used for same-subnet validation of a manually entered static IP, and also to detect/redirect the browser to `tendawifi.com` if the address bar host doesn't match the router's LAN IP or that hostname.
  - `adslUser` / `adslPwd`: pre-fills the PPPoE username/password fields (real ISP credentials — redacted above).
  - `ssid` / `wrlPassword`: pre-fills the WiFi name/password step (real WiFi password — redacted above).
  - `power` (`"high"`/other): WiFi transmit-power radio pre-fill; only shows the power-selection control when not `"high"`.
  - `cloneType` (`"0"`=use default MAC,`"1"`=clone this PC's MAC,other=custom): MAC-clone step pre-fill.
  - `mac` / `deviceMac` / `defMac`: current WAN MAC, this browsing device's MAC, and the router's factory-default MAC, used to build the MAC-clone step's default/local/custom options.
  - `country`: read but not used by this page's own logic.
- **Risk**: Safe (read-only).

### `POST goform/fast_setting_internet_set` — page: index.html (quick setup wizard, "Next" from the connection-type step)

- **Purpose**: Saves the chosen WAN connection type and its parameters as step 2 of the wizard.
- **Request**: `application/x-www-form-urlencoded` body built from a fixed per-`netWanType` field template (index.js:169-229), always including `netWanType`, `cloneType`, `mac` (uppercased, chosen per `cloneType`: default/local/custom), plus type-specific fields:
  - `netWanType=0` (Dynamic IP): only `netWanType`, `cloneType`, `mac`.
  - `netWanType=1` (Static IP): adds `staticIp`, `mask`, `gateway`, `dns1`, `dns2`.
  - `netWanType=2` (PPPoE): adds `adslUser`, `adslPwd`.
  - `netWanType=3`/`4` (Russia PPTP/L2TP): adds `vpnServer`, `vpnUser`, `vpnPwd`, `vpnWanType`, `staticIp`, `mask`, `gateway`, `dns1`, `dns2`, `dnsAuto` (computed: `1` if both DNS fields are blank and visible, else `0`, initialised from `vpnWanType`).
  - `netWanType=5` (Russia PPPoE): same as 3/4 plus `adslUser`, `adslPwd`.
  - Example literal body for this router's live PPPoE setup (secrets redacted):
    `netWanType=2&adslUser=<redacted>&adslPwd=<redacted>&cloneType=0&mac=aa:bb:cc:dd:ee:ff`
- **Response**: Not read — `$.post` is fired with no success callback; the wizard immediately advances to the WiFi-settings step regardless of the actual result.
- **Risk**: Rewrites the WAN configuration; a bad value here (e.g. wrong PPPoE credentials) can leave the router without internet, though the wizard doesn't wait to find out before moving on.
- **Notes**: Client-side validation (same-subnet/broadcast/loopback IP checks, primary-DNS-required rules) runs before submit via the page's `$.validate` block (index.js:15-129), mirroring the equivalent checks in main.js's WAN form.

### `GET goform/getSyncAccount` — page: index.html (quick setup wizard, only when `CONFIG_SYNC_ACCOUNT` build macro is `"y"`)

- **Purpose**: "Account sync" feature — fetches ISP credentials that a companion Tenda app/cloud account previously stored, to auto-fill the PPPoE step.
- **Request**: `GET goform/getSyncAccount?<random>`. Polled: fired immediately, retried every 500ms while `status != 1`, with a separate 2s timeout-driven retry loop layered on top (`validTimeout`) — the two retry mechanisms can both be in flight simultaneously (the code does not appear to fully deduplicate them, since `getSyncData()` is called from both a timeout-expiry path and the polling `syncCallback`).
- **Response** (live, secrets redacted): `{"status":0,"username":"<redacted>","password":"<redacted>"}`
  - `status` (`0`/`1`): `1` = an account was found and credentials are populated; `0` = keep polling. Live capture shows `0` (no synced account) alongside populated `username`/`password` fields — those two fields appear to be present in the schema regardless of `status`, though they're only *used* by the page when `status == 1`.
  - `username` / `password`: PPPoE credentials, copied verbatim into the `adslUser`/`adslPwd` fields once `status == 1`, and the "Sync" button/dialog is then hidden in favor of a "synced" indicator.
- **Risk**: Safe (read-only), though it does return the router's stored ISP credentials in cleartext to any authenticated session.

### `GET goform/getWanConnectStatus` — page: index.html (quick setup wizard, WiFi step and final wait screen)

- **Purpose**: Snapshots the current WAN connect-status code immediately before submitting the WiFi/login-password step, and again while showing the "connecting..." countdown, to decide whether to show the "you're online" success screen immediately or after a wait.
- **Request**: `GET goform/getWanConnectStatus?<random>` — no other params.
- **Response** (live): `{"connectStatus":7}` — a plain integer, same encoding as the least-significant portion of the 7-digit codes used elsewhere (`7` = "Connected. You can access the internet now." per the `statusTxtObj` table in main.js). Stored as `G.wanStatus` and compared against the literal string `"7"` in one place and the number `7` in another (inconsistent type comparison, but JS loose equality/`==` isn't used here for the number case — `G.wanStatus == "7"` works either way since `==` coerces).
- **Risk**: Safe (read-only).

### `POST goform/fast_setting_wifi_set` — page: index.html (quick setup wizard, WiFi step)

- **Purpose**: Saves the WiFi SSID/password, transmit power, timezone and (optionally) a new login password — the wizard's final configuration step.
- **Request**: `application/x-www-form-urlencoded` body (via `objTostring`) with fields:
  - `ssid`: new WiFi name.
  - `wrlPassword`: new WiFi password, or `""` if the "hide/no password" checkbox is checked (open network).
  - `power`: transmit-power setting.
  - `timeZone`: computed from the browser's local UTC offset as `±HH:MM` (via `getTimeZone()`), not user-selected.
  - `loginPwd`: `hex_md5(<new login password>)` if the "change login password" and "hide password" checkboxes are both unchecked, else `""` (keep default/no change).
  - Example shape (values illustrative, not from a live capture since this POST wasn't captured with secrets):
    `ssid=MyWiFi&wrlPassword=<redacted>&power=high&timeZone=%2B02%3A00&loginPwd=<redacted-md5-hex>`
  - Also called (same endpoint, same field set minus an always-empty `wrlPassword`) from `continueSet()` when the user proceeds without setting a WiFi password on the confirmation step.
- **Response**: Handled by `handWifi()` — the raw response body isn't parsed for a success code; the function only branches on the previously-fetched `G.wanStatus` to decide whether to show the finish screen immediately or run a 10-second visual countdown before checking `getWanConnectStatus` again and showing the finish screen regardless of the new result.
- **Risk**: Changing the WiFi SSID/password or the login password disconnects existing WiFi clients and can lock the admin out if the new login password isn't recorded. This is the step that finalizes the whole quick-setup wizard.
- **Notes**: Client-side validation requires SSID non-empty and ≤29 bytes; WiFi password (if set) 8-32 ASCII chars with no leading/trailing space; login password (if not skipped) 5-32 ASCII chars with no leading/trailing space.

### `POST goform/fast_setting_pppoe_set` — dead code, not invoked

- **Purpose**: Would have submitted just a PPPoE username/password (`username=<user>&password=<pwd>`, URI-encoded) as a standalone wizard step (`#ppoe_setting`).
- **Request/Response**: See commented-out block at index.js:862-889 and the paired `handPpoe` callback at index.js:1116-1126 (also commented out).
- **Risk**: N/A.
- **Notes**: This entire code path, and the `#ppoe_setting` UI panel it targeted, is commented out in the current build. The wizard's live PPPoE flow instead goes entirely through `fast_setting_internet_set` (see above), which folds PPPoE credentials into the same generic per-type payload used for every connection type. Kept here only because the brief asked to cover every `fast_setting_*` reference in these files — do not assume this endpoint is reachable from the current UI.

---

## login.js

### `POST /login/Auth` — page: login page

- **Purpose**: Authenticates the admin session.
- **Request**: `application/x-www-form-urlencoded`-style object body: `{username: <username>, password: hex_md5(<password>)}` (password is MD5-hashed client-side before sending; username sent in plaintext). Example: `username=admin&password=<32-hex-char md5>`.
- **Response**: Plain text (not JSON). The page only distinguishes one case: response body `"1"` → "Incorrect password." shown inline; anything else is treated as success and the page does a full `location.reload(true)` (relying on the router's session cookie now being set).
- **Risk**: Safe (authentication attempt only; no configuration change). Repeated failed attempts are not rate-limited by anything visible in this script.
- **Notes**: No CSRF token or nonce is included in the request. If the page is loaded inside an iframe (`top != window`), it forces `top.location.reload(true)` instead of logging in, presumably to defeat clickjacking/nested-login attempts.

# Network and security

Pages: net_set, lan, mac_clone, dmz, virtual_server, static_route, upnp_config,
ddns_config, iptv, remote_web, firewall, net_control, ip_mac_bind, online_list,
mac_filter, parental_control.

See Conventions at the top for GET/POST encoding and `errCode`.

## Endpoint index

| Endpoint | Method | Page(s) | Type | Risk |
|---|---|---|---|---|
| `goform/getWanParameters` | GET | net_set.html | get | Safe |
| `goform/WanParameterSetting` | POST | net_set.html | set | Drops/changes WAN connectivity |
| `goform/AdvGetLanIp` | GET | lan.html | get | Safe |
| `goform/AdvSetLanip` | POST | lan.html | set | Can drop LAN access / lock out admin |
| `goform/AdvGetMacMtuWan` | GET | mac_clone.html | get | Safe |
| `goform/AdvSetMacMtuWan` | POST | mac_clone.html | set | Briefly drops/renegotiates WAN |
| `goform/GetDMZCfg` | GET | dmz.html | get | Safe |
| `goform/SetDMZCfg` | POST | dmz.html | set | Safe (opens a host to WAN — security, not connectivity, risk) |
| `goform/GetVirtualServerCfg` | GET | virtual_server.html | get | Safe |
| `goform/SetVirtualServerCfg` | POST | virtual_server.html | set (add/delete) | Safe |
| `goform/GetStaticRouteCfg` | GET | static_route.html | get | Safe |
| `goform/SetStaticRouteCfg` | POST | static_route.html | set (add/delete) | Can break routing if misconfigured |
| `goform/GetUpnpCfg` | GET | upnp_config.html | get | Safe |
| `goform/SetUpnpCfg` | POST | upnp_config.html | set | Safe |
| `goform/GetDDNSCfg` | GET | ddns_config.html | get | Safe |
| `goform/SetDDNSCfg` | POST | ddns_config.html | set | Safe |
| `goform/GetIPTVCfg` | GET | iptv.html | get | Safe |
| `goform/SetIPTVCfg` | POST | iptv.html | set | Safe by itself, but triggers a reboot (see below) |
| `goform/SysToolReboot` | GET (action) | iptv.html | action | **Reboots the router** — drops all LAN/WiFi/WAN sessions |
| `goform/GetRemoteWebCfg` | GET | remote_web.html | get | Safe |
| `goform/SetRemoteWebCfg` | POST | remote_web.html | set | Safe (exposes web admin to WAN — security risk) |
| `goform/GetFirewallCfg` | GET | firewall.html | get | Safe |
| `goform/SetFirewallCfg` | POST | firewall.html | set | Safe (weakens flood/ping defenses if disabled) |
| `goform/GetNetControlList` | GET | net_control.html | get (polled) | Safe |
| `goform/SetNetControlList` | POST | net_control.html | set | Safe (throttles bandwidth per device) |
| `goform/GetIpMacBind` | GET | ip_mac_bind.html | get | Safe |
| `goform/SetIpMacBind` | POST | ip_mac_bind.html | set | Safe (can strand a device without an IP until reconnect) |
| `goform/getOnlineList` | GET | online_list.html | get (polled) | Safe |
| `goform/setBlackRule` | POST | online_list.html | set (add one) | Immediately disconnects that device |
| `goform/getBlackRuleList` | GET | online_list.html | get | Safe |
| `goform/delBlackRule` | POST | online_list.html | set (delete one) | Safe (restores that device's access) |
| `goform/getMacFilterCfg` | GET | mac_filter.html | get | Safe |
| `goform/setMacFilterCfg` | POST | mac_filter.html | set (replace whole list) | Can cut off every non-listed device if whitelist mode misconfigured |
| `goform/GetParentControlInfo` | GET | parental_control.html | get | Safe |
| `goform/saveParentControlInfo` | POST | parental_control.html | set | Safe (restricts one device's internet access/hours) |
| `goform/getParentalRuleList` | GET | parental_control.html | get | Safe |
| `goform/SetOnlineDevName` | POST | net_control.html, ip_mac_bind.html, online_list.html, parental_control.html | set | Safe (cosmetic rename only) |

---

### `GET /goform/getWanParameters` — page: net_set.html (Internet Settings)
- **Purpose**: Fetch WAN connection status/config for one or two WAN ports, and poll connection state.
- **Request**: No params beyond jQuery's auto `?_=<ts>` cache-buster. Also polled every 5s by `netInfo.ajaxInterval` (`AjaxInterval`) while the page is open.
- **Response** (live):
  - `country` (string, e.g. `"US"`)
  - `wl_mode` (string enum: `"ap"`, `"wisp"`, `"apclient"` — repeater mode; when `"apclient"` the WAN form is hidden entirely)
  - `lanIp`, `lanMask` (strings) — current LAN IP/mask, used for same-segment validation
  - `guestIp`, `guestMask` (strings) — guest network subnet
  - `multiWanEn` (string `"true"`/`"false"`)
  - `lineUp` (string, numeric) — undocumented in the JS (not read); appears to be a line-rate value
  - `wanInfo` (array, one entry per WAN port, indexed by `wanTargetIndex`):
    - `wanType` (string enum): `"0"`=Dynamic IP (DHCP), `"1"`=Static IP, `"2"`=PPPoE, `"3"`=Russia PPTP, `"4"`=Russia L2TP, `"5"`=Russia PPPoE (the RU-only options 3–5 only render in the UI when `B.getLang()==="RU"` and this is WAN1)
    - `pptpSvrIp`, `pptpSvrMask` (strings) — PPTP/L2TP server address/mask when double-access is used
    - `connectTime` (string, seconds connected)
    - `connectStatus` (string, 7-digit code): digit1=can-disconnect(1)/cannot(2), digit2=display color (1 error/2 warning/3 success), digit3=connected(1)/not(0), digit4=work mode (0 AP/1 WISP/2 AP-Client), digits 5–7=a numeric sub-status looked up in `statusTxtObj` (e.g. `209`="Connected. You can access the internet now.", `205`="incorrect username/password", etc. — see the `statusTxtObj` map in `js/net_set.js` for the full table of ~30 codes)
    - `downSpeedLimit` (string) — hidden form field, round-tripped unchanged
    - `wanIp`, `staticIp`, `mask`, `gateway` (strings) — current/static WAN addressing
    - `vpnClient` (string `"0"`/`"1"`) — whether a PPTP/L2TP client is separately configured; if `"1"` and the user picks WAN type 3/4, a warning is shown that changing settings disables the VPN client
    - `vpnClientUser` (string, redact if present)
    - `dnsAuto` (string `"1"`=automatic, `"0"`=manual)
    - `dns1`, `dns2` (strings)
    - `vpnWanType` (string `"1"`=dynamic, `"0"`=static, used for double-access WAN types 3/4/5)
    - `vpnServer`, `vpnUser`, `vpnPwd` (strings, redact)
    - `adslUser`, `adslPwd` (strings, redact) — PPPoE credentials
- **Risk**: Safe (read-only).
- **Notes**: `wanIndex = top.staInfo.wanTargetIndex` (1 or 2) selects which `wanInfo[]` entry this page edits; that index comes from the parent frame, not this response.

### `POST /goform/WanParameterSetting` — page: net_set.html (Internet Settings)
- **Purpose**: Change the WAN connection type/parameters for one WAN port, or connect/disconnect it.
- **Request**: Body built from `#internet-form` fields via `serializeArray()`, with a WAN suffix appended to every field name (`""` for WAN1, `"2"` for WAN2), then `netWanType`→`wanType` renamed, plus `&module=wan1|wan2`. Fields depend on `netWanType`:
  - Always: `connect=1`, `lanIp`, `lanMask`, `downSpeedLimit`, `wanType[n]`
  - `wanType=2` (PPPoE) or `5` (Russia PPPoE): `adslUser[n]`, `adslPwd[n]`
  - `wanType=3/4/5` (double access): `vpnServer[n]`, `vpnUser[n]`, `vpnPwd[n]`, `vpnWanType[n]` (radio: `1`=dynamic, `0`=static)
  - DNS: `dnsAuto[n]` (`1`=auto, `0`=manual; forced to `1` if `dns1` is left blank), `dns1[n]`, `dns2[n]`
  - Static IP modes (`wanType=1`, or `3/4/5` with `vpnWanType=0`): `staticIp[n]`, `mask[n]`, `gateway[n]`
  - To disconnect, the body is instead just `action=disconnect&module=wan1|wan2` (no other fields).
  - Example (WAN1, PPPoE, auto DNS): `connect=1&lanIp=192.168.0.1&lanMask=255.255.255.0&downSpeedLimit=&wanType=2&adslUser=<redacted>&adslPwd=<redacted>&dnsAuto=1&dns1=1.1.1.1&dns2=8.8.8.8&module=wan1`
- **Response**: `{"errCode":<n>,"sleep_time":<n>}`. `errCode==0` → re-enables the Connect/Disconnect button and re-fetches `getWanParameters`. `sleep_time` estimates how long the connect/disconnect will take (used only to guess whether this is a VPN reconnect, not otherwise surfaced).
- **Risk**: Drops the current WAN connection and re-establishes it under the new settings; switching WAN type or credentials can take the router offline for tens of seconds and, on Russia double-access modes, disables a running PPTP/L2TP VPN client.
- **Notes**: Both `#netWanType` and the submit button are disabled while saving (`netInfo.saving`). Client-side validation is extensive (same-segment checks against the other WAN, PPTP server, LAN, guest network; broadcast/network-address rejection; DNS pair must differ, etc.) — see `netInfo.checkWanData` in `js/net_set.js`.

---

### `GET /goform/AdvGetLanIp` — page: lan.html (LAN Settings)
- **Purpose**: Fetch current LAN IP/mask, DHCP server config, and related cross-feature IPs used for validation.
- **Request**: None besides the auto cache-buster.
- **Response** fields actually read by the page (from the live capture, not shown here to avoid duplicating secrets — see the file): `lanIp`, `lanMask`, `dhcpEn` (`"1"`/`"0"`), `startIp`, `endIp` (full dotted IPs; only the last octet is edited in the UI), `leaseTime`, `lanDnsAuto` (`"1"`=auto DNS, `"0"`=manual — note this is inverted vs. most other `*Auto` fields, driven by the `btn-on`/`btn-off` toggle), `lanDns1`, `lanDns2`, `wl_mode`. Also present for validation only (not edited here): `wanIp`, `wanMask`, `wanIp2`, `wanMask2`, `serverIp`, `vlan2Ip`, `vlan2Mask`, `remoteIp`, `pptpSvrIp`, `pptpSvrMask`, `vpnCliIp`, `guestIp`.
- **Risk**: Safe.
- **Notes**: n/a.

### `POST /goform/AdvSetLanip` — page: lan.html (LAN Settings)
- **Purpose**: Change the router's LAN IP/mask and/or its DHCP server range, lease time, and LAN-side DNS.
- **Request**: `objTostring()`-encoded body with keys `lanIp`, `lanMask`, `dhcpEn` (`1`/`0`), `startIp`, `endIp` (full IPs, reconstructed from the LAN subnet prefix + the last-octet input), `leaseTime`, `lanDnsAuto` (`0`/`1`), `lanDns1`, `lanDns2`. If DHCP is off, `startIp`/`endIp` are re-sent unchanged from the original load rather than from the (hidden) inputs. Example: `lanIp=192.168.0.1&lanMask=255.255.255.0&dhcpEn=1&startIp=192.168.0.100&endIp=192.168.0.200&leaseTime=<n>&lanDnsAuto=1&lanDns1=&lanDns2=`
- **Response**: `{"errCode":<n>}`. `callback()` reloads the page via `top.showSaveMsg`, passing a `changeFlag` of `true` if `lanIp` was changed (this makes the reload target the new IP).
- **Risk**: Changing `lanIp`/`lanMask` immediately changes the router's own management address — the admin's browser session is redirected and DHCP clients briefly lose connectivity until they renew. Shrinking/moving the DHCP range can strand already-leased clients.
- **Notes**: Extensive same-segment validation against WAN, WAN2, PPTP server, VPN client, and connected-server IPs (see `checkData()` in `js/lan.js`) to avoid an unreachable router.

---

### `GET /goform/AdvGetMacMtuWan` — page: mac_clone.html (MAC Clone / WAN MTU)
- **Purpose**: Fetch per-WAN MTU, port speed, and MAC-clone configuration.
- **Response** (live): `wanInfo` array, one entry per WAN port:
  - `wanType` (string, same enum as net_set: `0` DHCP, `1` static, `2` PPPoE, `3`/`4` PPTP/L2TP, `5` Russia PPPoE) — only used to pick the valid MTU range (`2`→576-1492, `3`→576-1444, `4`→576-1460, else 576-1500) and to decide whether to show the PPPoE service/server-name fields
  - `wanMTU` (string, numeric)
  - `wanSpeed` (string enum): `0`=1000 Mbps auto-negotiation (label becomes "Auto-negotiation" when `top.CONFIG_1000M_ETH=='n'`), `1`=10 Mbps FDX, `2`=10 Mbps HDX, `3`=100 Mbps FDX, `4`=100 Mbps HDX
  - `cloneType` (string enum): `0`=use default MAC, `1`=clone the logged-in-device's MAC, `2`=use a manually entered MAC
  - `defMac`, `deviceMac`, `mac` (strings) — router's factory MAC, the browsing device's MAC, and the currently configured/cloned MAC
  - `serviceName`, `serverName` (strings) — PPPoE service-name/AC-name overrides, only shown (and only sent back) when `top.CONFIG_PAGE_HAVE_SERV_NAME=="y"` and WAN1's `wanType=="2"` (PPPoE)
- **Risk**: Safe.
- **Notes**: If `top.sysInfo.data.wl_mode === "wisp"`, the Save button is disabled (MAC clone/MTU isn't applicable in WISP mode).

### `POST /goform/AdvSetMacMtuWan` — page: mac_clone.html (MAC Clone / WAN MTU)
- **Purpose**: Set WAN1 (and WAN2, if present) MTU, port speed, and cloned MAC address.
- **Request**: Manually concatenated body (not urlencoded via `objTostring`): `wanMTU=<n>&wanSpeed=<0-4>&cloneType=<0-2>&mac=<XX:XX:XX:XX:XX:XX>` where `mac` is resolved client-side to `defMac`/`deviceMac`/the manual `#mac` input depending on `cloneType`. If `CONFIG_PAGE_HAVE_SERV_NAME=="y"` and current WAN type is PPPoE, appends `&serviceName=<name-or-empty>&serverName=<name-or-empty>`. If a second WAN exists, appends `&wanMTU2=...&wanSpeed2=...&cloneType2=...&mac2=...` the same way. Example: `wanMTU=1480&wanSpeed=0&cloneType=0&mac=aa:bb:cc:dd:ee:ff`
- **Response**: `{"errCode":<n>}`; on success (`0`) the page re-fetches via `top.advInfo.initValue()`.
- **Risk**: Changing the WAN MAC or MTU causes the WAN interface to re-negotiate (drop and re-establish the connection); some ISPs bind a subscriber to the first-seen MAC, so cloning the wrong MAC can cause the ISP to reject the connection until it is changed back or the modem is power-cycled.
- **Notes**: Client-side check rejects WAN1/WAN2 sharing the same effective MAC.

---

### `GET /goform/GetDMZCfg` — page: dmz.html (DMZ)
- **Purpose**: Fetch DMZ host configuration.
- **Response** (live): `lanIp`, `lanMask` (strings, for validation), `dmzEn` (`"0"`/`"1"`), `dmzIp` (string, full IP of the DMZ host; only its last octet is shown/edited, prefixed by the LAN's /24 prefix).
- **Risk**: Safe.
- **Notes**: n/a.

### `POST /goform/SetDMZCfg` — page: dmz.html (DMZ)
- **Purpose**: Enable/disable the DMZ and set the DMZ host's IP.
- **Request**: `dmzEn=<0|1>&dmzIp=<full IP>` (when disabling, `dmzIp` is re-sent from the last loaded value rather than recomputed). Example: `dmzEn=1&dmzIp=192.168.0.100`
- **Response**: `{"errCode":<n>}`. `errCode==2` is a special case meaning "DMZ host IP equals LAN IP" — shown inline without the generic save-message flow. `errCode==0` triggers `top.advInfo.initValue()` to refresh.
- **Risk**: Safe for the router itself, but exposes the target host to all inbound traffic from the WAN (all ports forwarded, bypassing NAT/firewall) — a security exposure for that host, not a connectivity risk to the router.
- **Notes**: Client validation requires the DMZ IP to be in the same subnet as the LAN and not equal to the LAN IP itself.

---

### `GET /goform/GetVirtualServerCfg` — page: virtual_server.html (Virtual Servers / Port Forwarding)
- **Purpose**: Fetch the list of configured port-forwarding rules.
- **Response** (live): `lanIp`, `lanMask` (strings, for validation); `virtualList` (array), each entry: `ip` (string, internal host), `inPort` (string, internal/LAN port), `outPort` (string, external/WAN port), `protocol` (string enum: `"0"`=TCP&UDP, `"1"`=TCP, `"2"`=UDP).
- **Risk**: Safe.
- **Notes**: n/a.

### `POST /goform/SetVirtualServerCfg` — page: virtual_server.html (Virtual Servers / Port Forwarding)
- **Purpose**: Add one rule or delete one rule — but always resubmits the *entire* resulting rule list, not a diff.
- **Request**: `list=<row>~<row>~...` where each row is `ip,inPort,outPort,protocol` (protocol re-encoded as `1`=TCP, `2`=UDP, `0`=TCP&UDP) and rows are joined with `~`. On delete, the row marked `data-target="delete"` is simply omitted from the rebuilt list; on add, the new row (from the `#ip`/`#inPort`/`#outPort`/`#protocol` inputs) is appended. Example (2 rules): `list=192.168.0.5,25565,25565,1~192.168.0.5,80,80,1`
- **Response**: `{"errCode":<n>}`. On success (`0`), the JS updates the on-page table directly (`addList()`/`delList()`) rather than reloading; there is no `top.showSaveMsg` call (commented out).
- **Risk**: Safe — opens/removes a WAN→LAN port mapping only.
- **Notes**: Max 16 rules enforced client-side; duplicate external (`outPort`) values are rejected; target IP must be in the LAN subnet and not the router's own LAN IP.

---

### `GET /goform/GetStaticRouteCfg` — page: static_route.html (Static Routing)
- **Purpose**: Fetch the routing table (system + user-added static routes).
- **Response** (live): `lanIp`, `lanMask`, `wanMask`, `wanGateway` (strings, for validation/display); `routeList` (array), each entry: `network` (string), `mask` (string), `gateway` (string, `"0.0.0.0"` if none), `ifname` (string, e.g. `"WAN1"`, `"br0"`), `operateType` (string enum: `"0"`=system route, not deletable/editable; `"1"`=user-added, deletable), `effective` (string `"1"`=currently active, `"0"`=inactive — inactive rows are shown greyed out with a "This route will not take effect." tooltip).
- **Risk**: Safe.
- **Notes**: n/a.

### `POST /goform/SetStaticRouteCfg` — page: static_route.html (Static Routing)
- **Purpose**: Add or delete a user-defined static route; resubmits the full list of user routes each time.
- **Request**: `list=<row>~<row>~...`, each row `network,mask,gateway,ifname`, rows joined by `~`. Critically, **only rows with `operateType=="1"` (user-added) are ever included** — system routes (`operateType=="0"`) are never part of this payload even though they appear in the GET response. On add, `ifname` is `WAN1` (single-WAN) or the selected `WAN1`/`WAN2` (multi-WAN); `gateway` defaults to `0.0.0.0` if left blank. Example: `list=10.0.0.0,255.255.255.0,192.168.0.254,WAN1`
- **Response**: `{"errCode":<n>}`; on success the table is updated in place client-side (no `top.showSaveMsg`, commented out).
- **Risk**: Adding an incorrect route can black-hole traffic to that destination network; deleting one removes that reachability. Does not affect the default route or system routes.
- **Notes**: Max 10 user routes enforced client-side; duplicate/overlapping destination networks (by longest-common-mask comparison) are rejected client-side (`checkIpInSameSegment` in this file, a different/bespoke implementation from the one in `js/lan.js`/`js/ip_mac_bind.js`, and it has a latent bug — the inner loop `for(var j=0,l=msk.length;i<l;i++)` compares `i` instead of `j`, so it relies on `i` being left over 4 from a previous loop; harmless in practice because `i` is always 4 there, so the loop body never executes and `index` stays at its default of 3).

---

### `GET /goform/GetUpnpCfg` — page: upnp_config.html (UPnP)
- **Purpose**: Fetch UPnP enable state and the list of currently active UPnP-negotiated port mappings.
- **Response** (live): a **JSON array**, not an object. `obj[0] = {"upnpEn": "0"|"1"}`. `obj[1..]` (if any) are live mappings, each with `remoteHost`, `outPort`, `host`, `inPort`, `protocol` — these are read-only/informational (created by LAN devices via UPnP, e.g. game consoles), not editable on this page.
- **Risk**: Safe.
- **Notes**: The live capture (live) has no active mappings, only `[{"upnpEn":"1"}]`.

### `POST /goform/SetUpnpCfg` — page: upnp_config.html (UPnP)
- **Purpose**: Enable/disable UPnP.
- **Request**: `upnpEn=<0|1>`. Submitted immediately on toggle click (no separate Save button).
- **Response**: `{"errCode":<n>}`; success reloads via `top.advInfo.initValue()`.
- **Risk**: Safe. Disabling UPnP can break applications (games, VoIP, some P2P) that rely on automatic port mapping; enabling it allows LAN devices to open inbound ports on the WAN without further admin action.
- **Notes**: n/a.

---

### `GET /goform/GetDDNSCfg` — page: ddns_config.html (DDNS)
- **Purpose**: Fetch Dynamic DNS configuration and current registration status.
- **Response** (live): `ddnsEn` (`"0"`/`"1"`), `serverName` (string enum of provider hostnames: `no-ip.com`, `dyn.com/dns/` (labelled "dyndns.org"), `88ip.cn`, `oray.com`), `ddnsUser`, `ddnsPwd` (strings, redact), `ddnsDomain` (string; hidden in the UI when the provider is `88ip.cn` or `oray.com`, which don't need a domain), `ddnsStatus` (`"0"`=Disconnected, `"1"`=Connected).
- **Risk**: Safe.
- **Notes**: n/a.

### `POST /goform/SetDDNSCfg` — page: ddns_config.html (DDNS)
- **Purpose**: Enable/disable DDNS and set provider/credentials/domain.
- **Request**: `objTostring()`-encoded: `ddnsEn`, `serverName`, `ddnsUser`, `ddnsPwd`, `ddnsDomain`. When disabling, the previous values are re-sent unchanged rather than the (hidden) form fields. Example: `ddnsEn=1&serverName=no-ip.com&ddnsUser=<redacted>&ddnsPwd=<redacted>&ddnsDomain=`
- **Response**: `{"errCode":<n>}`; success reloads via `top.advInfo.initValue()`.
- **Risk**: Safe.
- **Notes**: n/a.

---

### `GET /goform/GetIPTVCfg` — page: iptv.html (IPTV)
- **Purpose**: Fetch IPTV/multicast (set-top-box, IGMP, VLAN) configuration.
- **Response** (live): `wl_mode` (string; if not `"ap"` the page shows an error and disables Save, since IPTV needs plain AP mode), `stbEn` (`"0"`/`"1"` — "STB dedicated port"/VLAN tagging), `igmpEn` (`"0"`/`"1"` — IGMP proxy/snooping), `iptvType` (string enum: `"none"`=Default, `"shanghai"`=Shanghai VLAN preset, `"manual"`=Custom VLAN), `vlanId` (string, the active/selected VLAN ID — for `shanghai` it's one of the two preset area values `51`/`85`), `list` (string, comma-separated list of up to 8 custom VLAN IDs, e.g. `"85,51"` for the Shanghai preset which hardcodes `list="85,51"` client-side).
- **Risk**: Safe.
- **Notes**: n/a.

### `POST /goform/SetIPTVCfg` — page: iptv.html (IPTV)
- **Purpose**: Save STB/IGMP/VLAN settings for IPTV.
- **Request**: `objTostring()`-encoded: `stbEn`, `igmpEn`, `iptvType`, `vlanId`, `list`. If `stbEn=="0"`, `iptvType`/`vlanId`/`list` are re-sent unchanged from the last load. Example: `stbEn=1&igmpEn=1&iptvType=manual&vlanId=100&list=100`
- **Response**: `{"errCode":<n>}`. If any of `stbEn`, `iptvType`, `vlanId`, or `list` changed from the previous config, the page had already popped a `confirm()` dialog before submitting ("Please reboot the router after changing the IPTV settings. Do you want to reboot the router?"); cancelling it aborts the whole submit (`beforeSubmit` returns `false`, no POST is sent). If confirmed and `errCode==0`, the page immediately issues the reboot GET below instead of just refreshing.
- **Risk**: Safe by itself, but in the common case (STB/VLAN change) it is paired with an automatic reboot — see `SysToolReboot` below.
- **Notes**: Max 8 custom VLAN IDs (range 4–4094, no duplicates), enforced client-side.

### `GET /goform/SysToolReboot` — page: iptv.html (IPTV) — reboot trigger
- **Purpose**: Reboot the router, called automatically by `iptv.js`'s save callback after a successful `SetIPTVCfg` that changed STB/VLAN settings.
- **Request**: `$.get("goform/SysToolReboot?" + Math.random(), ...)` — a GET with a random cache-busting query string (no named params). No user confirmation happens at this point; the confirmation already happened before `SetIPTVCfg` was sent.
- **Response**: Not meaningfully used by `iptv.js` beyond checking `top.isTimeout(str)`; the page shows a "rebooting" progress overlay (`top.$.progress.showPro("reboot")`) rather than parsing a result.
- **Risk**: **High** — reboots the entire router. Drops all LAN/WiFi clients and the WAN connection for the duration of the reboot (typically tens of seconds).
- **Notes**: Misspelling is `SysToolReboot` (not misspelled, unlike some other firmware's `Rebbot`); note `GetSysAutoRebbotCfg` *does* use the "Rebbot" misspelling, but that's an unrelated (auto-reboot-schedule) endpoint on a page outside this group.

---

### `GET /goform/GetRemoteWebCfg` — page: remote_web.html (Remote Web Management)
- **Purpose**: Fetch remote (WAN-side) web-admin access configuration.
- **Response** (live): `syspwdflag` (string `"0"`/`"1"` — whether a login password is set; if `"0"`, enabling remote access is blocked client-side with a warning to set a password first), `lanIp`, `lanMask`, `wlGuestIp` (strings, present for potential validation though the same-segment check against them is commented out/disabled by design), `remoteWebEn` (`"0"`/`"1"`), `remoteIp` (string; `"0.0.0.0"` means "allow any WAN IP"), `remotePort` (string, numeric).
- **Risk**: Safe.
- **Notes**: If the URL hash contains `&nopwd`, the whole form is replaced with a static notice instead of loading data.

### `POST /goform/SetRemoteWebCfg` — page: remote_web.html (Remote Web Management)
- **Purpose**: Enable/disable remote web admin access and set the allowed source IP/port.
- **Request**: `objTostring()`-encoded: `remoteWebEn`, `remoteIp`, `remotePort`. When disabling, `remoteIp`/`remotePort` are re-sent unchanged. Example: `remoteWebEn=1&remoteIp=0.0.0.0&remotePort=8080`
- **Response**: `{"errCode":<n>}`; success reloads via `top.sysInfo.initValue()`.
- **Risk**: Safe for connectivity, but enabling this exposes the router's web admin UI to the WAN (optionally restricted to one source IP) — a security-sensitive change, especially combined with `remoteIp="0.0.0.0"` (any source).
- **Notes**: `remoteIp` accepts `"0.0.0.0"` as a special "allow all" value in addition to a specific IP.

---

### `GET /goform/GetFirewallCfg` — page: firewall.html (Firewall)
- **Purpose**: Fetch flood-defense / WAN-ping toggle states.
- **Response** (live): `firewallEn` — a single 4-character string where each character is `"1"`/`"0"`: position 1=ICMP flood defense, 2=TCP flood defense, 3=UDP flood defense, 4=ignore ping from WAN.
- **Risk**: Safe.
- **Notes**: n/a.

### `POST /goform/SetFirewallCfg` — page: firewall.html (Firewall)
- **Purpose**: Set the four flood-defense/ping toggles.
- **Request**: `firewallEn=<4-char string>` built by concatenating the four toggle states in the same order as the GET (ICMP, TCP, UDP, ignore-WAN-ping). Example: `firewallEn=1110`
- **Response**: `{"errCode":<n>}`; success reloads via `top.advInfo.initValue()`.
- **Risk**: Safe (no connectivity impact); disabling these reduces DoS/flood protection.
- **Notes**: n/a.

---

### `GET /goform/GetNetControlList` — page: net_control.html (Bandwidth Control)
- **Purpose**: Fetch per-device bandwidth-control state and live up/down speeds; polled continuously.
- **Request**: Initial load via `pageModel`; then repolled every 5s via `AjaxInterval` while the page is open.
- **Response** (live): a **JSON array**. `obj[0] = {"netControlEn": "0"|"1"}` (global enable flag; present in the payload but the corresponding UI toggle is currently commented out in the page, so it's not actually surfaced/editable). `obj[1..]`, one per known LAN device: `mac`, `ip`, `hostName`, `devType` (string, fed into `translateDeviceType()` for an icon — not enumerated here), `upSpeed`, `downSpeed` (strings, current throughput, Kbps-ish units consistent with `limitUp`/`limitDown`), `limitUp`, `limitDown` (strings, configured caps; `"0"`=unlimited), `isControled` (`"0"`/`"1"`), `offline` (`"0"`=online, `"1"`=offline), `isSet` (`"0"`/`"1"`, whether a custom limit has been set).
- **Risk**: Safe.
- **Notes**: n/a.

### `POST /goform/SetNetControlList` — page: net_control.html (Bandwidth Control)
- **Purpose**: Push upload/download speed caps for every known device in one call (full replace, not a diff).
- **Request**: `list=<row>\n<row>\n...` (rows separated by `\n`); each row is `<urlencoded devName>\r<mac>\r<limitUpKB>\r<limitDownKB>` (fields separated by `\r`). Limits are converted from the UI's Mbps selection to the wire unit by `×128` (i.e. KB/s). Example (2 devices, 1 row shown, name redacted-format only for illustration): `list=Device2%20%28Living%20Room%29%0Dc4%3Ad0%3Ae3%3A89%3Af7%3A01%0D0%0D0` — i.e. `devName\rmac\r0\r0` per line, unlimited in this example.
- **Response**: `{"errCode":<n>}`; success stops the polling interval and reloads via `top.advInfo.initValue()`.
- **Risk**: Safe — only throttles individual devices' throughput; does not disconnect them.
- **Notes**: `limitUp`/`limitDown` inputs are capped at 2000 (Mbps) client-side and rounded to 2 decimals; `0` means unlimited.

### `POST /goform/SetOnlineDevName` — pages: net_control.html, ip_mac_bind.html, online_list.html, parental_control.html
- **Purpose**: Rename a device (by MAC) in the router's device-name table. Identical usage across all four pages.
- **Request**: `mac=<mac address>&devName=<urlencoded name>` (in `parental_control.html`'s `editDevice()`/`checkParentData()` paths the name is *not* URL-encoded before concatenation — a minor inconsistency vs. the other three call sites, which all use `encodeURIComponent`). Example: `mac=aa:bb:cc:dd:ee:ff&devName=Living%20Room%20Laptop`
- **Response**: `{"errCode":<n>}`; `0`=success (page shows "Modification success" and updates the name in place), non-zero=failure ("Modification failure"). None of the four callers reload the whole page.
- **Risk**: Safe — cosmetic only.
- **Notes**: Device name is validated client-side by the shared `checkDevNameValidity()` helper (not in this group's files) before the request is sent; max length 20 characters (`maxlength` on the relevant inputs).

---

### `GET /goform/GetIpMacBind` — page: ip_mac_bind.html (IP & MAC Binding)
- **Purpose**: Fetch the current DHCP client list and the list of already-bound IP↔MAC pairs.
- **Response** (live): `lanIp`, `lanMask` (strings), `dhttpIP` (string; an IP the UI forbids binding to — likely a DHCP-related reserved address), `dhcpClientList` (array of currently-leased-but-unbound clients: `ipaddr`, `macaddr`, `devname`, `status` (`"0"`=offline, `"1"`=online)), `bindList` (array of already-bound entries, same shape: `ipaddr`, `macaddr`, `devname`, `status`).
- **Risk**: Safe.
- **Notes**: `dhcpClientList` entries with a UUID-looking `devname` (e.g. `"4c3e84f2-df70-..."`) reflect devices that haven't announced a DHCP hostname; the router falls back to a generated placeholder.

### `POST /goform/SetIpMacBind` — page: ip_mac_bind.html (IP & MAC Binding)
- **Purpose**: Push the full set of active IP↔MAC bindings (add, remove, or edit a binding all result in resubmitting the whole bound set).
- **Request**: `bindnum=<N>&list=<row>\n<row>\n...`; each row is `<urlencoded devName>\r<mac>\r<ip>` (fields separated by `\r`, rows by `\n`). Only rows currently in the "bound" state (operate icon class `unbind`, i.e. click-to-unbind) are included — plain unbound DHCP-client rows and rows mid-delete are excluded. Example: `bindnum=1&list=Device5%0Dec%3Ab9%3A31%3A73%3A36%3A90%0D192.168.0.30`
- **Response**: `{"errCode":<n>}`, but the page ignores it entirely and always shows a fixed success message ("The configuration is saved and will take effect as soon as your device connects to the router next time.") — see the commented-out real handling in `callback()`.
- **Risk**: Safe in the immediate sense (no reboot/disconnect); however, per the page's own message, a binding only takes effect the *next* time that device requests a DHCP lease, so misconfiguring a binding can leave a device unable to get the expected/any IP on its next reconnect until corrected.
- **Notes**: Max 32 bindings enforced client-side (`maxBindNum`); an IP cannot equal the router's own `lanIp` or the reserved `dhttpIP`; duplicate IP or MAC (re-binding the same MAC just overwrites its row) are checked client-side.

---

### `GET /goform/getOnlineList` — page: online_list.html (Online Devices / Attached Devices)
- **Purpose**: Fetch the list of currently/recently seen LAN clients (used for both the "Attached Devices" tab and to drive the MAC-filter-mode-aware "Add to blacklist" button).
- **Request**: `$.getJSON("goform/getOnlineList?" + Math.random(), cb)`; polled every 5s via a plain `setInterval` (not the shared `AjaxInterval` helper) while the page is open, and torn down in `window.onunload`.
- **Response** (live): a **JSON array**. `obj[0]`: `blackNum` (number — count of devices currently in the "quick blacklist"; the live capture shows an implausibly large value, `12492040`, on a router whose blacklist is actually empty, so this field may not be reliable/meaningful), `macFilterType` (string `"black"`/`"white"` — mirrors the mode set on `mac_filter.html`; when `"black"`, an "Add" button appears per device to quick-blacklist it), `localhostIP`, `localhostName`, `localhostMac` (strings — identifies the browsing admin's own device so it's labelled "Local Host" and excluded from the blacklist-add action), `isWirelessConnect` (present multiple times in the sample capture with different values — likely one-per-radio internally but only the last value survives in this JSON due to duplicate keys; not read by this page's JS at all). `obj[1..]`, one per device: `deviceId` (MAC), `ip`, `devName`, `line` (string enum: `"0"`=Wired, `"1"`=2.4GHz, `"2"`=5GHz), `uploadSpeed`, `downloadSpeed` (strings; only shown when `top.G.workMode` is `"wisp"` or `"router"`), `linkType` (string, device-type icon key), `black` (`0`/`1` — filtered out of the "online" view when `1`, unless `isGuestClient=="true"`), `isGuestClient` (string `"true"`/`"false"`).
- **Risk**: Safe.
- **Notes**: Devices with `black==1` (unless guest) are hidden from the normal online list; the local host device is always pinned to the top of the list if it isn't itself blacklisted.

### `POST /goform/setBlackRule` — page: online_list.html (Online Devices → "Add" button)
- **Purpose**: Quick-add a currently-online device (by MAC) to the router's blacklist, immediately cutting off its network access.
- **Request**: `mac=<mac address>` (no URL-encoding needed, MAC only). Example: `mac=aa:bb:cc:dd:ee:ff`
- **Response**: `{"errCode":<n>}`. `0`=success → the online list is refreshed and a "Adding to the blacklist..." toast is shown; `1`=failure meaning the blacklist is full (max 30, `maxBlackNum`) and the just-clicked "Add" button is re-enabled with an error shown.
- **Risk**: Immediately disconnects that specific device from the network (its future traffic is dropped by the router). Does not affect other devices or the router itself.
- **Notes**: A commented-out check would also count parental-control entries against the same 30-item cap, but it isn't currently enforced client-side (only enforced server-side via `errCode==1`).

### `GET /goform/getBlackRuleList` — page: online_list.html (Online Devices → Blacklist tab)
- **Purpose**: Fetch the current blacklist (devices added via `setBlackRule`, shown on the "Blacklist" sub-tab).
- **Request**: `$.getJSON("goform/getBlackRuleList?" + Math.random(), cb)`, loaded once when the Blacklist tab is opened.
- **Response** (live): a **JSON array** of entries, each with `deviceId` (MAC, displayed uppercased) and `devName` (string; falls back to `top.G.deviceNameSpace` — a generic "Unknown"-style label — if empty).
- **Risk**: Safe.
- **Notes**: This is a distinct blacklist mechanism from `mac_filter.html`'s `setMacFilterCfg` blacklist — the two are not the same list (one is the quick "kick this device" list surfaced here and driven by `setBlackRule`/`delBlackRule`; the other is the full allow/deny table configured on the MAC Filter page). There is also dead code in this file (`js/online_list.js`, in a comment) referencing `goform/initWifiMacFilter` — it is never actually called.

### `POST /goform/delBlackRule` — page: online_list.html (Online Devices → Blacklist tab, "Remove")
- **Purpose**: Remove a device from the quick blacklist, restoring its network access.
- **Request**: `mac=<mac address>`. Example: `mac=aa:bb:cc:dd:ee:ff`
- **Response**: `{"errCode":<n>}`; `0`=success → re-fetches `getBlackRuleList` and shows a "Removing from the blacklist..." toast.
- **Risk**: Safe (restorative — re-enables that device's access).
- **Notes**: n/a.

---

### `GET /goform/getMacFilterCfg` — page: mac_filter.html (MAC Filtering / Access Control)
- **Purpose**: Fetch the MAC filter mode and both the black- and white-list device tables, plus the online-device list used for one-click "add all online devices" to the whitelist.
- **Response** (live): `localhostIP`, `localhostName`, `localhostMac` (strings — the browsing admin's own device, pinned to the top of the whitelist and shown as "Local Host" rather than deletable), `macFilterType` (string `"black"`/`"white"` — which list is currently enforced), `blackList` (array of `{devName, devMac}`), `whiteList` (array of `{devName, devMac}`), `onlineList` (array of `{devName, devMac}` — all currently-online devices, used only to populate the "Add all online devices to the whitelist" helper).
- **Risk**: Safe.
- **Notes**: n/a.

### `POST /goform/setMacFilterCfg` — page: mac_filter.html (MAC Filtering / Access Control)
- **Purpose**: Set the filter mode (blacklist/whitelist) and replace the entire device list for whichever mode is selected.
- **Request**: `macFilterType=<black|white>&deviceList=<row>\n<row>\n...`; each row is `<urlencoded devName>\r<MAC (uppercase)>` (fields separated by `\r`, rows by `\n`). Only the table for the currently-selected radio (`whiteListTable` or `blackListTable`) is serialized — the other list's rows aren't touched/sent. Example: `macFilterType=white&deviceList=Device6%0D6C%3A4C%3ABC%3A83%3A95%3A2B`
- **Response**: `{"errCode":<n>}` (with `2` handled as a distinct silent case in `callback()`, though no message is actually shown for it — likely a leftover from copy-pasting the DMZ page's callback); `0`=success → reloads via `top.advInfo.initValue()`.
- **Risk**: Switching to `macFilterType=white` (whitelist mode) means **only** the listed MAC addresses can reach the network — any device not in the list (including future new devices) is cut off immediately. The router pre-seeds the admin's own device into the whitelist to avoid self-lockout, but any other currently-connected device not added will lose access as soon as this is saved. Blacklist mode only affects the listed devices.
- **Notes**: Max 30 entries per list enforced client-side; duplicate MACs within the same list rejected client-side.

---

### `GET /goform/GetParentControlInfo` — page: parental_control.html (Parental Controls, per-device edit view)
- **Purpose**: Fetch the existing parental-control rule (if any) for one specific device, identified by MAC.
- **Request**: **Query parameter** `mac=<device MAC, lowercase>` plus a `random=<Math.random()>` cache-buster, called as `goform/GetParentControlInfo?mac=<mac>&random=<random>` via `$.getJSON`. The MAC comes from `top.parentInfo.editObj.deviceMac` (set by the parent frame when the admin clicks "Edit" on a device, or empty string when adding a brand-new rule — in which case this GET is skipped entirely and the page just renders defaults).
- **Response**: **No live capture exists** (live); the shape is fully inferred from `initParentControl()` in `js/parental_control.js`, which merges the response over this default object:
  ```
  {"enable": 1, "mac": "", "url_enable": 1, "urls": "", "time": "19:00-21:00", "day": "1,1,1,1,1,1,1", "limit_type": 0}
  ```
  So for a device with an existing rule, the server is expected to return an object with:
  - `enable` (number/string `0`/`1`) — whether the rule is active
  - `mac` (string) — not actually re-read from the response by the page (device identity comes from the query param / `top.parentInfo`)
  - `url_enable` (`0`/`1`) — whether website filtering is on for this device
  - `urls` (string) — comma-separated keyword list (blacklist or whitelist, depending on `limit_type`)
  - `time` (string) — `"HH:MM-HH:MM"` window during which the device *may* use the internet (UI label "Internet Accessible At"); the literal value `"00:00-24:00"` is special-cased to display as `"00:00-00:00"`
  - `day` (string) — 7 comma-separated `0`/`1` flags, Sunday-first (`day0`=Sunday … `day6`=Saturday)
  - `limit_type` (`0`/`1`) — `0`=blacklist mode (blocked keywords), `1`=whitelist mode (only these keywords reachable)
  - If the device has **no** existing rule, the code checks `typeof obj.enable == "undefined"` and treats that as "not configured", resetting `enable`/`url_enable`/`urls`/`time`/`day` to the defaults above — implying the live endpoint likely returns `{}` (or an object lacking `enable`) for a device with no rule, rather than an HTTP error.
- **Risk**: Safe (read-only).
- **Notes**: Live check: for a device with no rule the router returns only `{"mac":"<mac>"}`; the populated shape above is from the code and unverified until a rule exists.

### `POST /goform/saveParentControlInfo` — page: parental_control.html (Parental Controls, per-device edit view)
- **Purpose**: Create/update (or disable) the parental-control rule for one device.
- **Request**: `objTostring()`-encoded, shape depends on state:
  - Enabled, new device (`G_current_operate=="1"`, i.e. adding): `deviceId` (MAC, lowercase), `deviceName`, `enable` (`1`), `time` (`"HH:MM-HH:MM"`), `url_enable` (`0`/`1`), `urls` (comma-separated keywords, lowercased, max 10, each 2–31 chars of `[-.a-z0-9]`), `day` (7 comma-separated `0`/`1` flags), `limit_type` (`0`=blacklist/`1`=whitelist, from the checked radio).
  - Enabled, existing device (editing): same fields except `deviceId` comes from the displayed `#device_mac` label instead of a fresh input, and `deviceName` is omitted (a rename, if the name field was in edit mode, is sent separately via `SetOnlineDevName` before this request).
  - Disabled (`#parentcontrolEnable` off): only `deviceId` and `enable=0` are sent — no time/url/day fields.
  - Example (new device, blacklist keywords, every day, 19:00–21:00): `deviceId=aa:bb:cc:dd:ee:ff&deviceName=Kids-Tablet&enable=1&time=19:00-21:00&url_enable=1&urls=example,video&day=1,1,1,1,1,1,1&limit_type=0`
- **Response**: `{"errCode":<n>}`. `0`=success → shows the save message and refreshes via `top.parentInfo.initValue()`; `1`=failure meaning the parental-control rule list is full (max 30 entries, shared cap message wording matches the blacklist's); any other value shows a generic "Configuration failed." toast.
- **Risk**: Safe for the router/network as a whole — only restricts internet access (fully or by URL keyword, on a day/time schedule) for the one targeted device.
- **Notes**: At least one weekday must be checked when "Specified Day" is selected; start/end time must differ; keyword list capped at 10 entries.

### `GET /goform/getParentalRuleList` — page: parental_control.html (Parental Controls, rule-list tab)
- **Purpose**: Fetch the summary list of all devices with a parental-control rule configured (the "Rule List" sub-tab, separate from the per-device edit view).
- **Request**: `$.getJSON("goform/getParentalRuleList?" + Math.random(), cb)`, loaded when the "Rule List" tab is opened.
- **Response** (live): array of `{devName, mac, enable}` — `enable` (`0`/`1`) drives the displayed "Enable"/"Disable" label per row. A "Delete" button is rendered per row in the UI but this file contains no corresponding delete handler/endpoint (dead UI — deletion isn't wired up in `js/parental_control.js`).
- **Risk**: Safe.
- **Notes**: n/a.

# Wireless

Pages: `wireless.html`, `wireless_ssid.html`, `wifi_power.html`, `wifi_bf.html`,
`wifi_time.html`, `wifi_wps.html`, `wifi_ap.html`, `wisp.html`,
`anti_interference.html`, `sleep_mode.html`, `system_led.html`.

See Conventions at the top for GET/POST encoding and `errCode`.

**Working-mode changers (flagged — can cut internet/WiFi access):**
- `goform/setApModeCfg` (`wifi_ap.html`) — switches the whole device between
  normal Router mode and pure AP (bridge) mode. Forces a reboot and, once in AP
  mode, disables Internet/VPN/Parental-Control/Virtual-Server/etc. entirely.
- `goform/WifiExtraSet` (`wisp.html`) — switches `wl_mode` between `ap` (normal),
  `wisp` (WISP repeater) and `apclient` (Client+AP repeater). Any transition
  to/from a non-`ap` mode forces a reboot (the page calls `goform/SysToolReboot`
  itself) and, once in `wisp`/`apclient`, the router gets its internet access by
  bridging to an upstream WiFi network instead of its WAN port — wrong
  SSID/password there means no internet after reboot.

## Endpoint summary

| Endpoint | Method | Page | Type | Risk |
|---|---|---|---|---|
| `goform/WifiRadioGet` | GET | wireless.html | get | safe |
| `goform/WifiRadioSet` | POST | wireless.html | set | wifi clients on changed band(s) reconnect |
| `goform/WifiBasicGet` | GET | wireless_ssid.html | get | safe |
| `goform/WifiBasicSet` | POST | wireless_ssid.html | set | can disable WiFi / change SSID+password, drops that band's clients |
| `goform/WifiPowerGet` | GET | wifi_power.html | get | safe |
| `goform/WifiPowerSet` | POST | wifi_power.html | set | reduces range, minor risk of dropping marginal clients |
| `goform/WifiBeamformingGet` | GET | wifi_bf.html | get | safe |
| `goform/WifiBeamformingSet` | POST | wifi_bf.html | set | brief WiFi hiccup while radio re-tunes |
| `goform/initSchedWifi` | GET | wifi_time.html | get | safe |
| `goform/openSchedWifi` | POST | wifi_time.html | set | schedules WiFi radios to turn OFF during a window — can cut WiFi access |
| `goform/WifiWpsGet` | GET | wifi_wps.html | get | safe |
| `goform/WifiWpsSet` | POST | wifi_wps.html | set | safe (enable/disable WPS only) |
| `goform/WifiWpsStart` | POST (action) | wifi_wps.html | action | safe, starts a 2-min WPS pairing window |
| `goform/getApModeCfg` | GET | wifi_ap.html | get | safe |
| `goform/setApModeCfg` | POST | wifi_ap.html | set | **WORKING-MODE CHANGE + reboot** — can cut internet/LAN access entirely |
| `goform/WifiExtraGet` | GET | wisp.html | get | safe |
| `goform/WifiExtraSet` | POST | wisp.html | set | **WORKING-MODE CHANGE (ap/wisp/apclient) + reboot** — can cut internet |
| `goform/WifiApScan` | GET | wisp.html | action | safe, momentary scan on the radio being scanned |
| `goform/SysToolReboot` | GET (action) | wisp.html | action | **reboots the router** — full outage during reboot |
| `goform/WifiAntijamGet` | GET | anti_interference.html | get | safe |
| `goform/WifiAntijamSet` | POST | anti_interference.html | set | may trigger a channel switch, brief WiFi drop |
| `goform/PowerSaveGet` | GET | sleep_mode.html | get | safe |
| `goform/PowerSaveSet` | POST | sleep_mode.html | set | schedules WiFi/LEDs/USB OFF during a window — can cut WiFi access |
| `goform/GetLEDCfg` | GET | system_led.html | get | safe |
| `goform/SetLEDCfg` | POST | system_led.html | set | safe, cosmetic (LED indicators only) |

---

### `GET /goform/WifiRadioGet` — page: wireless.html ("Channel & Bandwidth")
- **Purpose**: Fetch current 2.4 GHz and 5 GHz radio mode/channel/bandwidth settings and the valid option lists for each.
- **Request**: none (plain GET, cache-busted).
- **Response**:
  - `adv_mode` (string enum): 2.4 GHz network mode — `"bgn"` = 11b/g/n mixed, `"bg"` = 11b/g mixed, `"n only"` = 11n.
  - `adv_channel` (string): selected 2.4 GHz channel; `"0"` = Auto.
  - `adv_band` (string enum): 2.4 GHz bandwidth — `"20"`, `"40"`, or `"auto"` (20/40); only `"20"`/`"40"`/`"auto"` are offered when mode is `bgn`/`n only`, forced to `"20"` for `bg`.
  - `adv_extend_channel` (string): 2.4 GHz extension-channel setting, e.g. `"none"`. Not surfaced as a control on this page.
  - `adv_country` (string): 2.4 GHz regulatory domain, e.g. `"US"`.
  - `channel` (array): valid 2.4 GHz channel list; index 0 is a placeholder (rendered as "Auto" in the channel dropdown), the rest (`1..11` here) are real channel numbers.
  - `adv_mode_5g` (string enum): 5 GHz network mode — `"ac"` = 11a/n/ac mixed, `"ac only"` = 11ac.
  - `adv_channel_5g` (string): selected 5 GHz channel; `"0"` = Auto.
  - `adv_band_5g` (string enum): 5 GHz bandwidth — `"20"`, `"40"`, `"80"`, or `"auto"`; the option list offered depends on which keys exist in `channel_5g` (`80` > `40` > `20`).
  - `adv_extend_channel_5g` (string): 5 GHz extension-channel setting, e.g. `"none"`.
  - `adv_country_5g` (string): 5 GHz regulatory domain, e.g. `"US"`.
  - `channel_5g` (object): keyed by bandwidth (`"20"`, `"40"`, `"80"`), each value an array of valid 5 GHz channels for that bandwidth (index 0 again a placeholder for "Auto").
- **Risk**: safe, read-only.
- **Notes**: none.

### `POST /goform/WifiRadioSet` — page: wireless.html ("Channel & Bandwidth")
- **Purpose**: Save 2.4 GHz / 5 GHz network mode, channel, and bandwidth.
- **Request**: body is `$("#wireless").serialize()` — a standard urlencoded form serialization of all 6 selects in the form:
  `adv_mode`, `adv_channel`, `adv_band`, `adv_mode_5g`, `adv_channel_5g`, `adv_band_5g` (same enums as the GET response above; all fields always sent together).
  Example (built from live values above): `adv_mode=bgn&adv_channel=0&adv_band=20&adv_mode_5g=ac&adv_channel_5g=0&adv_band_5g=auto`
- **Response**: `{"errCode":<n>}`; `"0"` = success, no further payload.
- **Risk**: Not a working-mode change, but changing channel/bandwidth/mode on either band forces that radio to restart, which disconnects and reconnects any currently-associated WiFi clients on that band (including the admin's own browser session if connected over WiFi). Does not touch WAN/routing.
- **Notes**: 2.4 GHz bandwidth option list is mode-dependent (`bg` forces 20 MHz only). 5 GHz bandwidth list depends on which widths the hardware/region reports in `channel_5g`.

### `GET /goform/WifiBasicGet` — page: wireless_ssid.html ("WiFi Name & Password")
- **Purpose**: Fetch current SSID, password, encryption, enable/hide state for 2.4 GHz and 5 GHz.
- **Response**:
  - `wrlEn` / `wrlEn_5g` (string `"0"`/`"1"`): whether the 2.4 GHz / 5 GHz radio itself is broadcasting.
  - `ssid` / `ssid_5g` (string): network name, e.g. `"Device1"` / `"Device3"`.
  - `security` / `security_5g` (string enum): `"none"`, `"wpapsk"` (WPA-PSK), `"wpa2psk"` (WPA2-PSK), `"wpawpa2psk"` (WPA/WPA2-PSK, labelled "recommended" in the UI).
  - `wrlPwd` / `wrlPwd_5g` (string): WiFi password — `<redacted>`.
  - `wpapsk_type` / `wpapsk_type_5g`, `wpapsk_crypto` / `wpapsk_crypto_5g` (string, e.g. `"psk2"` / `"tkip+aes"`): present in the payload but not read by this page's JS (`initValue` only uses `security*`/`wrlPwd*`/`hideSsid*`/`wrlEn*`).
  - `hideSsid` / `hideSsid_5g` (string `"0"`/`"1"`): whether SSID broadcast is hidden (checkbox).
  - `uptime` / `uptime_5g` (string, e.g. `"0"`): unused by this page's JS.
- **Risk**: safe, read-only (password is exposed in plaintext in this response, hence must never be echoed).
- **Notes**: none.

### `POST /goform/WifiBasicSet` — page: wireless_ssid.html ("WiFi Name & Password")
- **Purpose**: Save SSID, password, encryption, enable/hide state for 2.4 GHz and 5 GHz.
- **Request**: body built via `objTostring()` from:
  `wrlEn`, `wrlEn_5g` (`"0"`/`"1"`), `security`, `security_5g` (enum as above), `ssid`, `ssid_5g` (string, max 32 bytes, validated with `.valid.ssid`), `hideSsid`, `hideSsid_5g` (`"0"`/`"1"`), `wrlPwd`, `wrlPwd_5g` (8–63 chars, or a 64-char hex PSK, required whenever `security*` is not `"none"`; validated with `.valid.ssidPwd`).
  Example (from live values, password redacted): `wrlEn=1&wrlEn_5g=0&security=wpa2psk&security_5g=wpa2psk&ssid=Device1&ssid_5g=Device3&hideSsid=1&hideSsid_5g=0&wrlPwd=<redacted>&wrlPwd_5g=<redacted>`
- **Response**: `{"errCode":<n>}`; on `"0"` the page also refreshes `top.wrlInfo` and `top.staInfo`.
- **Risk**: Disabling `wrlEn`/`wrlEn_5g` turns that WiFi band off immediately; changing `ssid*`/`wrlPwd*`/`security*` on an enabled band immediately disconnects all clients on that band (they must reconnect with the new credentials). Does not affect WAN/routing or the other band. Not a working-mode change.
- **Notes**: Client-side SSID validation caps byte length at 32 (double-byte chars count as 2). Password field accepts a 64-hex-char string as a raw PSK, otherwise 8–63 ASCII chars.

### `GET /goform/WifiPowerGet` — page: wifi_power.html ("Transmit Power")
- **Purpose**: Fetch current transmit power level for each band.
- **Response**: `power`, `power_5g` (string enum `"low"`/`"middle"`/`"high"`; the 5 GHz radio UI in this build only offers `"low"` (labelled "Medium") and `"high"`). Live values: `power":"high"`, `"power_5g":"high"`.
- **Risk**: safe, read-only.
- **Notes**: none.

### `POST /goform/WifiPowerSet` — page: wifi_power.html ("Transmit Power")
- **Purpose**: Set transmit power for 2.4 GHz and 5 GHz.
- **Request**: hand-built string `"power=" + power + "&power_5g=" + power_5g`, values from the checked radio buttons (see enum above).
  Example: `power=high&power_5g=high`
- **Response**: `{"errCode":<n>}`; on `"0"` the page refreshes `top.advInfo` and `top.wrlInfo`.
- **Risk**: Not a working-mode change; lowering power reduces WiFi range/signal strength and may drop clients that were only marginally in range. No reboot, no WAN impact.
- **Notes**: none.

### `GET /goform/WifiBeamformingGet` — page: wifi_bf.html ("Beamforming+")
- **Purpose**: Fetch whether Beamforming+ is enabled.
- **Response**: `beamformingEn` (string `"0"`/`"1"`). Live value: `"1"`.
- **Risk**: safe, read-only.
- **Notes**: none.

### `POST /goform/WifiBeamformingSet` — page: wifi_bf.html ("Beamforming+")
- **Purpose**: Toggle Beamforming+ (signal-focusing feature for client devices).
- **Request**: `beamformingEn=0` or `beamformingEn=1`, sent immediately on click (no separate Save button).
- **Response**: `{"errCode":<n>}`; on `"0"` the page waits 2 s (showing "Enabling/Disabling beamforming…"), then re-fetches `WifiBeamformingGet` and refreshes `top.wrlInfo`.
- **Risk**: Not a working-mode change. Toggling can cause a brief WiFi interruption while the radio re-tunes; low risk overall.
- **Notes**: The 2 s wait before refetch suggests the backend applies this asynchronously.

### `GET /goform/initSchedWifi` — page: wifi_time.html ("WiFi Schedule")
- **Purpose**: Fetch the current WiFi on/off schedule.
- **Response**:
  - `wifiEn` (number `1`): present but unused by this page's JS.
  - `schedWifiEnable` (string/number `0`/`1`): whether the schedule is active.
  - `schedStartTime`, `schedEndTime` (string `"HH:MM"`): window during which WiFi is turned OFF. `"0"`/`"0"` is special-cased client-side to default to `00:00`–`07:00` for display.
  - `timeType` (string `"0"`/`"1"`): `"0"` = Every Day, `"1"` = Specified Day.
  - `day` (string): 7 comma-separated flags for Mon..Sun, `1`=selected/`0`=not, meaningful only when `timeType="1"`. Live: `"1,1,1,1,1,0,0"` (Mon–Fri).
  - `powerSaveTime` (string): the currently-configured Sleeping Mode window (e.g. `"00:00-07:00"`) or `""` if Sleeping Mode is off; used only to warn the user about a time overlap between WiFi Schedule and Sleeping Mode before submit.
  - `wl_mode` (string): current router work mode (`"ap"` here). The schedule can only be edited/enabled when `wl_mode === "ap"` — the toggle and Save button are disabled otherwise, with an error shown ("Please disable Wireless Repeating on the WiFi Settings page first.").
  - `timeUp` (string `"0"`/`"1"`): whether the router's clock has synced with internet time; if not `"1"`, a warning banner is shown that the schedule won't yet take effect.
- **Risk**: safe, read-only.
- **Notes**: none.

### `POST /goform/openSchedWifi` — page: wifi_time.html ("WiFi Schedule")
- **Purpose**: Save the WiFi on/off schedule.
- **Request**: body from `objTostring()`:
  - `schedWifiEnable` (`"0"`/`"1"`)
  - `schedStartTime`, `schedEndTime` (`"HH:MM"`) — only taken from the UI when `schedWifiEnable=="1"`; otherwise the previous `initObj.schedStartTime`/`schedEndTime` are resent unchanged.
  - `timeType` (`"0"` Every Day / `"1"` Specified Day) — likewise only from UI when enabled, else resent from `initObj`.
  - `day` (comma-separated 7-flag string, Mon..Sun) — likewise.
  Example (enabling, weekdays, current window): `schedWifiEnable=1&schedStartTime=00:00&schedEndTime=07:00&timeType=1&day=1,1,1,1,1,0,0`
- **Response**: `{"errCode":<n>}`; on `"0"` the page refreshes `top.wrlInfo`.
- **Risk**: Not a mode change, but **enabling this schedules the WiFi radio(s) to turn OFF during the configured window** — a real risk of losing WiFi access during that period. The only documented recovery inside that window is pressing the physical WiFi button on the router. Blocked entirely while the router is in WISP/Client+AP mode.
- **Notes**: Before submit, if the new window overlaps the router's configured Sleeping Mode window (`powerSaveTime` from the GET), a `confirm()` warns the user and lets them cancel. Start and end time must differ (client-side validation).

### `GET /goform/WifiWpsGet` — page: wifi_wps.html ("WPS")
- **Purpose**: Fetch WPS enable state, PIN, and gating info.
- **Response**: `wpsEn` (string `"0"`/`"1"`), `pinCode` (string, router's WPS PIN — `<redacted>`), `wl_mode` (string, must be `"ap"` for WPS to be usable), `wl_en` (string `"0"`/`"1"`, whether the 2.4 GHz WiFi radio is on — WPS requires it enabled).
- **Risk**: safe, read-only (PIN is a WiFi-join credential, must never be echoed).
- **Notes**: none.

### `POST /goform/WifiWpsSet` — page: wifi_wps.html ("WPS")
- **Purpose**: Toggle WPS on/off.
- **Request**: `wpsEn=0` or `wpsEn=1`, sent immediately on click (blocked client-side unless `wl_mode=="ap"` and `wl_en!="0"`).
- **Response**: `{"errCode":<n>}`; on `"0"` the page refreshes `top.wrlInfo`, waits 2 s, then re-fetches `WifiWpsGet`.
- **Risk**: safe; does not disconnect existing clients or change work mode.
- **Notes**: none.

### `POST /goform/WifiWpsStart` — page: wifi_wps.html ("WPS")
- **Purpose**: Start a WPS pairing session (push-button method).
- **Request**: fixed body `action=wps`. Only sent if the "Click Here"/WPS button is enabled (i.e. WPS is on, `wl_mode=="ap"`, radio enabled).
- **Response**: same handler as `WifiWpsSet` — `{"errCode":<n>}`; on `"0"` refreshes `top.wrlInfo` and re-fetches `WifiWpsGet` after 2 s.
- **Risk**: safe; opens a 2-minute WPS negotiation window during which a nearby device can join without a password. No disruption to existing clients.
- **Notes**: Misleading name matches the button's dual purpose in the UI text ("press the WPS button on the router or Click Here").

### `GET /goform/getApModeCfg` — page: wifi_ap.html ("AP Mode")
- **Purpose**: Fetch whether the router currently operates as a pure Access Point (bridge) instead of a full router.
- **Response**: `apModeEn` (string `"true"`/`"false"`, note: string literals, not `"0"`/`"1"`). Live value: `"false"` (router mode, not AP mode).
- **Risk**: safe, read-only.
- **Notes**: none.

### `POST /goform/setApModeCfg` — page: wifi_ap.html ("AP Mode")
- **Purpose**: Switch the whole device between normal Router mode and standalone AP (bridge) mode.
- **Request**: `apModeEn=true` or `apModeEn=false`.
  Example (turning AP mode on): `apModeEn=true`
- **Response**: `{"errCode":<n>}`. On `"0"`, if the value actually changed from the previous state, the page does **not** poll a reboot endpoint itself — it just shows a fake local countdown ("Rebooting... Please wait.", 0→100%) for ~75 s and then does `top.jumpTo(window.location.host)`, assuming the router has rebooted into the new mode by then.
- **Risk**: **WORKING-MODE CHANGE.** A client-side `confirm()` warns "Your settings will take effect after the system reboots. Do you want to reboot the system?" before submit. Enabling AP mode disables Internet Settings, VPN, Parental Control, Bandwidth Control, and Virtual Server functionality, and changes the admin UI's hostname to `tendawifi.com`. Switching either direction reboots the router (full outage during reboot) and fundamentally changes how it gets/provides internet — calling this can cut internet/LAN access.
- **Notes**: The UI's own copy states: after enabling AP mode, the router's WAN/LAN ports become plain LAN-side uplinks to an upstream router, and the management domain changes.

### `GET /goform/WifiExtraGet` — page: wisp.html ("Wireless Repeating")
- **Purpose**: Fetch current Wireless Repeating (WISP/Client+AP) configuration and gating flags.
- **Response**:
  - `wl_mode` (string enum): `"ap"` = normal router (repeating off), `"wisp"` = WISP repeater, `"apclient"` = Client+AP repeater.
  - `wl_enable` (string `"0"`/`"1"`): whether the WiFi radio(s) are enabled at all; repeating can't be turned on if `"0"`.
  - `wifi_chkHz` (string `"0"`/`"1"`): 2.4 GHz / 5 GHz band used for the upstream bridge link.
  - `country_code` (string, e.g. `"US"`).
  - `iptvEn`, `smartSaveEn`, `guestEn`, `wpsEn`, `wifiTimerEn` (string `"0"`/`"1"`): whether IPTV / Sleeping Mode / Guest Network / WPS / WiFi Schedule are currently enabled — used only to build a (currently unused/commented-out) warning that those features get disabled when repeating is turned on.
  - When `wl_mode` is `"wisp"`/`"apclient"` (not present/empty in the captured `"ap"`-mode snapshot): `ssid` (upstream SSID), `security` (`"none"`/`"wpapsk"`), `wpapsk_type`, `wpapsk_crypto`, `wpapsk_key` (upstream WiFi password, `<redacted>`), `mac` (upstream AP's MAC), `handset` (`"0"` auto-picked from scan / `"1"` manually entered).
- **Risk**: safe, read-only (may contain the upstream WiFi password when repeating is configured).
- **Notes**: none.

### `POST /goform/WifiExtraSet` — page: wisp.html ("Wireless Repeating")
- **Purpose**: Enable/disable Wireless Repeating and configure which upstream WiFi to bridge to.
- **Request**: body from `objTostring(subObj)`, shape depends on UI state:
  - Repeating turned off: `{"wl_mode": "ap"}` → `wl_mode=ap`
  - Repeating on, re-using the previously scanned/selected network: `{"wl_mode": "wisp"|"apclient", "ssid", "security": "none"|"wpapsk", "wpapsk_type", "wpapsk_crypto", "wpapsk_key", "wifi_chkHz", "mac", "handset": "0"}`
  - Repeating on, manual entry (upstream network not in scan list): `{"wl_mode", "ssid": <handset_ssid input>, "security": "none"|"wpapsk", "wpapsk_type" (from select: none/wpa/wpa2/wpa&wpa2), "wpapsk_crypto" (aes/tkip/tkip&aes, only if type≠none), "wpapsk_key", "wifi_chkHz" (0=2.4GHz/1=5GHz), "mac": "", "handset": "1"}`
  - Repeating on, picking a scanned network: same shape as "re-using" but `ssid`/`security`/`wpapsk_type`/`wpapsk_crypto`/`wifi_chkHz`/`mac` taken from the corresponding `WifiApScan` result entry (security split on `"/"` into type/crypto), `handset="0"`.
  Example (manual WISP setup, password redacted): `wl_mode=wisp&ssid=UpstreamRouter&security=wpapsk&wpapsk_type=wpa2&wpapsk_crypto=aes&wpapsk_key=<redacted>&wifi_chkHz=0&mac=&handset=1`
- **Response**: `{"errCode":<n>}`. On `"0"`: if the submitted `wl_mode` differs from the previous `wl_mode`, OR the previous mode was already non-`"ap"` (i.e. any change while repeating, or any transition into/out of `"ap"`), the page shows a reboot/apclient progress overlay and immediately issues `GET goform/SysToolReboot`. Otherwise it just shows the save message and refreshes `top.wrlInfo`.
- **Risk**: **WORKING-MODE CHANGE.** A client-side `confirm()` ("The router must reboot to activate your settings...") gates any submission that changes to/from a non-`"ap"` mode. Switching to `wisp`/`apclient` makes the router get its internet by bridging to another WiFi network instead of via its own WAN port — a wrong upstream SSID/password means **no internet after the reboot**. This is the primary "could cut internet" endpoint in this group.
- **Notes**: WEP-secured upstream networks are explicitly rejected client-side ("WEP encryption has low security and is not supported"). The confirm dialog that would normally warn "WPS/Guest Network/WiFi Schedule/Sleeping Mode will be disabled when Wireless Repeating is enabled" is present in the code but commented out, so this happens silently.

### `GET /goform/WifiApScan` — page: wisp.html ("Wireless Repeating") — **not captured, described from code**
- **Purpose**: Trigger/return a scan of nearby WiFi networks to populate the "Upstream WiFi Name" picker.
- **Request**: `GET goform/WifiApScan?<cache-buster>`, no other params.
- **Response** (inferred from `js/wisp.js` `initScan`/`initCustomSelect`/`reCreateObj`): either `{"errCode":"999"}` (scan failed/session issue — the page forces a full reload of the top window), or a JSON array of scan results, each entry shaped like:
  - `ssid` (string): upstream network name (HTML-escaped before display).
  - `security` (string): `"none"`, `"wep"`, or `"<type>/<crypto>"` e.g. `"wpa2/aes"`, `"wpa&wpa2/tkip&aes"` — split on `/` into `wpapsk_type`/`wpapsk_crypto` when the user selects this entry.
  - `signal` (number-like string): received signal, negative dBm; used both to sort results (strongest first) and to pick a 4-bar icon (`>-60`→bar4, `-60..-70`→bar3, `-70..-80`→bar2, else bar1).
  - `wifi_chkHz` (string `"0"`/`"1"`): which band (2.4/5 GHz) this network was seen on.
  - `mac` (string): the upstream AP's BSSID.
- **Risk**: Read-only network scan. Actively scanning can cause a brief interruption to clients already associated with whichever radio (2.4 GHz or 5 GHz) performs the scan, as is typical for consumer routers, but it does not change configuration or work mode.
- **Notes**: Results are re-sorted strongest-signal-first client-side (`reCreateObj`) before being rendered; the dropdown always prepends "--Select--" and "--Enter WiFi name manually--" (values `-2`/`-3`).

### `GET /goform/SysToolReboot` — page: wisp.html ("Wireless Repeating") — action endpoint
- **Purpose**: Reboot the router. Called automatically by `wisp.js` right after a successful `WifiExtraSet` that changes the work mode (see above); also used by other pages/groups for manual reboot.
- **Request**: `GET goform/SysToolReboot?<cache-buster>`, no body.
- **Response**: not JSON-parsed by this page; the callback only checks `top.isTimeout(str)` (likely a login-page-redirect / session-expiry check) and otherwise ignores the content, since the UI is already showing a fixed-duration fake progress bar.
- **Risk**: **Reboots the router immediately** — full network/WiFi/LAN outage for the duration of the reboot (router-side timing, not controlled by the page, which just waits a fixed ~75 s before reloading the admin page).
- **Notes**: The page does not wait for/verify the reboot actually completing; it just assumes it did after the fixed countdown and reloads `window.location.host`.

### `GET /goform/WifiAntijamGet` — page: anti_interference.html ("Anti-interference")
- **Purpose**: Fetch the anti-interference (channel jam avoidance) mode.
- **Response**: `WifiAntijamEn` (string enum `"auto"`/`"true"`/`"false"` — Auto / Enable / Disable). Live value: `"auto"`.
- **Risk**: safe, read-only.
- **Notes**: none.

### `POST /goform/WifiAntijamSet` — page: anti_interference.html ("Anti-interference")
- **Purpose**: Set anti-interference mode.
- **Request**: `WifiAntijamEn=auto` / `WifiAntijamEn=true` / `WifiAntijamEn=false`, sent immediately on radio-button click.
- **Response**: `{"errCode":<n>}`; on `"0"` refreshes `top.wrlInfo`.
- **Risk**: Not a working-mode change. Enabling/Auto anti-interference lets the router dynamically switch WiFi channel to avoid interference, which can briefly disconnect currently-associated clients when a switch happens; no reboot, no WAN impact.
- **Notes**: none.

### `GET /goform/PowerSaveGet` — page: sleep_mode.html ("Sleeping Mode")
- **Purpose**: Fetch Sleeping Mode (power-saving schedule) configuration and gating info.
- **Response**:
  - `powerSavingEn` (number/string `0`/`1`): whether Sleeping Mode is enabled.
  - `powerSaveDelay` (number/string `0`/`1`): "Delay enabling the Sleep mode when there is an online user" checkbox.
  - `time` (string `"HH:MM-HH:MM"`): the sleep window, e.g. `"00:00-07:00"`.
  - `wl_mode` (string): must be `"ap"` for this page's controls to be editable; disabled with an error ("This function is not available if Wireless Repeating is enabled.") otherwise.
  - `ledTime` (string): the currently-configured LED Control schedule window (from `system_led.html`), or `""` if LED isn't on a schedule; used only to warn about overlap.
  - `wifiTime` (string): the currently-configured WiFi Schedule window (from `wifi_time.html`), or `""`; used only to warn about overlap.
  - `ledCloseType` (string enum `"allClose"`/`"unpowerClose"`): which indicators to turn off during sleep (all, or all except power LED).
  - `timeUp` (string `"0"`/`"1"`): system clock synced with internet time; gates the "takes effect only if synced" tip.
- **Risk**: safe, read-only.
- **Notes**: none.

### `POST /goform/PowerSaveSet` — page: sleep_mode.html ("Sleeping Mode")
- **Purpose**: Save the Sleeping Mode schedule.
- **Request**: body from `objTostring()`:
  - `powerSavingEn` (`"0"`/`"1"`)
  - `time` (`"HH:MM-HH:MM"`, start/end from the four hour/minute selects)
  - `ledCloseType` (`"allClose"`/`"unpowerClose"`)
  - `powerSaveDelay` (`"1"` if checkbox checked else `"0"`)
  Example (from live values, enabling): `powerSavingEn=1&time=00:00-07:00&ledCloseType=allClose&powerSaveDelay=1`
- **Response**: `{"errCode":<n>}`; on `"0"` refreshes `top.advInfo`.
- **Risk**: Not a mode change, but **enabling this turns off WiFi (and, per the page's own help text, LEDs and — on USB-capable models — the USB port) during the configured window**, which is a real risk of losing WiFi access during that time. The documented way to wake it early is pressing the WiFi button or using the Tenda App. Editing is blocked entirely while `wl_mode != "ap"` (Wireless Repeating enabled).
- **Notes**: Client-side validation rejects identical start/end time. Before submit, a `confirm()` warns if the new sleep window overlaps the configured LED Control schedule (`ledTime`) or WiFi Schedule (`wifiTime`), and lets the user cancel.

### `GET /goform/GetLEDCfg` — page: system_led.html ("LED Control")
- **Purpose**: Fetch the LED indicator control mode.
- **Response**:
  - `ledType` (string enum): `"open"` = always on, `"close"` = always off, `"time"` = scheduled. Live: `"close"`.
  - `time` (string `"HH:MM-HH:MM"`): the off-schedule window, meaningful when `ledType="time"`.
  - `ledCloseType` (string enum `"allClose"`/`"unpowerClose"`): only shown/used when the `CONFIG_LED_CLOSE_TYPE` build macro is `"y"`.
  - `powerSaveTime` (string): the currently-configured Sleeping Mode window, or `""`; used only to warn about overlap when `ledType="time"`.
  - (no `timeUp` observed to gate this page beyond the generic `#timeUpTip` show/hide, which still checks an implied clock-sync flag pattern shared with the other schedule pages.)
- **Risk**: safe, read-only.
- **Notes**: none.

### `POST /goform/SetLEDCfg` — page: system_led.html ("LED Control")
- **Purpose**: Save the LED indicator control mode/schedule.
- **Request**: hand-built string:
  `ledType=<open|close|time>&time=<HH:MM-HH:MM>&ledCloseType=<allClose|unpowerClose>`
  — `time` is only taken from the UI selects when `ledType="time"`; otherwise the previously-fetched `initObj.time` is resent unchanged.
  Example (from live values): `ledType=close&time=00:00-07:00&ledCloseType=allClose`
- **Response**: `{"errCode":<n>}`; on `"0"` refreshes `top.advInfo`.
- **Risk**: Cosmetic only (controls the router's status LEDs) — no effect on WiFi, WAN, LAN, or working mode. Safe.
- **Notes**: Client-side validation rejects identical start/end time when `ledType="time"`; a `confirm()` warns if the LED schedule window overlaps the router's configured Sleeping Mode window (`powerSaveTime`), but still allows saving either way.

# System, cloud, VPN, USB

**Model gating (`js/macro_config.js`)**: `CONFIG_USB_MODULES = "n"` on this SKU. In `js/main.js` (line ~276
and ~532) the entire "USB App" nav item (`#nav-usb`, `.usb-line`, `.status-usb`, the whole `usb-setting`
menu section that hosts Share File/DLNA/Share Printer/Xunlei) is removed from the DOM whenever
`CONFIG_USB_MODULES != "y"`. Since it's `"n"` here, **`printer.html`, `samba.html`, `dlna.html`,
`status_usb.html` and `xunleiDownload.html` are all unreachable from the UI** on this router — there is no
menu path that opens them, regardless of the `CONFIG_PRINTER_SERVER`/`CONFIG_DLNA_SERVER`/`CONFIG_FILE_SHARE`
sub-flags (those are only read inside the `CONFIG_USB_MODULES=="y"` branch). This is confirmed live:
every getter for these pages returns the router's stock CGI error body
(`Form <Name> is not defined`), meaning the goform handlers themselves are compiled out. `xunleiDownload`'s
markup is additionally hard-commented-out in `html/main.html` (`<!-- <div ... id="usb_xunlei"> -->`), so it
was disabled independently of the USB flag.

`system_config.html`'s only menu entry point in `js/main.js` (`case "sys_config"`) is also commented out, so
that standalone page is unreachable too — the "Reset" (factory-restore) action is only reachable in this UI
via the button embedded in `system_reboot.html`. Both pages POST the same `SysToolRestoreSet` endpoint, which
is live.

`wan_status.html`'s only caller, `mainPageLogic.showWanStatusPicIframe()`, has its entire body commented out
in `js/main.js`, so this page is also currently unreachable from the UI, though the file/behavior is intact
if that function were re-enabled.

Body encodings for the `objTostring()`-built setters on this page were confirmed by live round-trips
where listed under Verified live at the top; the rest follow the same `key=value&...` form.

## Endpoint summary

| Endpoint | Method | Page(s) | Type | Risk |
|---|---|---|---|---|
| `goform/SysToolpassword` | GET | system_password.html | get | Safe |
| `goform/SysToolChangePwd` | POST | system_password.html | set | Safe (logs out current session) |
| `goform/SysToolReboot` | POST | system_reboot.html | action | **Reboots router (~45s outage)** |
| `goform/SysToolRestoreSet` | POST | system_reboot.html, system_config.html (unreachable) | action | **Factory reset — wipes all config** |
| `cgi-bin/DownloadCfg/RouterCfm.cfg` | GET | system_backup.html | get (download) | Safe (file contains all secrets) |
| `cgi-bin/UploadCfg` | POST | system_backup.html | action | **Replaces entire config, likely reboots** |
| `goform/GetSystemStatus` | GET | system_status.html | get | Safe |
| `goform/GetSysTimeCfg` | GET | system_time.html | get | Safe |
| `goform/SetSysTimeCfg` | POST | system_time.html | set | Safe |
| `goform/GetSySLogCfg` | GET | system_log.html | get | Safe |
| `cgi-bin/DownloadLog/syslog.tar` | GET | system_log.html | get (download) | Safe |
| `goform/GetSysAutoRebbotCfg` | GET | system_automaintain.html | get | Safe |
| `goform/SetSysAutoRebbotCfg` | POST | system_automaintain.html | set | Safe (schedules future reboot) |
| `goform/SysToolGetUpgrade` | GET | system_upgrade.html | get | Safe |
| `cgi-bin/upgrade` | POST | system_upgrade.html (local upgrade) | action | **Flashes firmware, reboots** |
| `goform/cloudv2?module=wansta&opt=query` | GET | system_upgrade.html, cloud_managment.html | get | Safe |
| `goform/cloudv2?module=olupgrade&opt=queryversion` | GET | system_upgrade.html | get | Safe |
| `goform/cloudv2?module=olupgrade&opt=queryupgrade` | GET | system_upgrade.html | get/action | **Triggers/polls online download+flash, reboots at completion** |
| `goform/setNotUpgrade` | POST | system_upgrade.html | set | Safe |
| `goform/cloudv2?module=manage&opt=querybasic` | GET | cloud_managment.html | get | Safe |
| `goform/cloudv2?module=manage&opt=queryaccount` | GET | cloud_managment.html | get | Safe |
| `goform/cloudv2?module=manage&opt=setaccount` | POST | cloud_managment.html | set | Safe |
| `goform/cloudv2?module=manage&opt=setbasic` | POST | cloud_managment.html | set | Safe (enables remote cloud control) |
| `goform/GetPptpClientCfg` | GET | pptp_client.html | get | Safe |
| `goform/SetPptpClientCfg` | POST | pptp_client.html | set/action | Can disrupt WAN if misconfigured; drops existing PPTP/L2TP tunnel |
| `goform/GetPptpServerCfg` | GET | pptp_server.html | get | Safe |
| `goform/SetPptpServerCfg` | POST | pptp_server.html | set | Safe |
| `goform/setPptpUserList` | POST | pptp_server.html | set | Safe (disconnects that user if disabled/deleted) |
| `goform/getPptpOnlineClient` | GET | pptp_user.html | get | Safe |
| `goform/GetPrinterCfg` | GET | printer.html | get | **Unsupported on this model — 404** |
| `goform/SetPrinterCfg` | POST | printer.html | set | **Unsupported on this model — unreachable** |
| `goform/GetSambaCfg` | GET | samba.html | get | **Unsupported on this model — 404** |
| `goform/SetSambaCfg` | POST | samba.html | set | **Unsupported on this model — unreachable** |
| `goform/GetDlnaCfg` | GET | dlna.html | get | **Unsupported on this model — 404** |
| `goform/SetDlnaCfg` | POST | dlna.html | set | **Unsupported on this model — unreachable** |
| `goform/expandDlnaFile` | POST | dlna.html | get(action) | **Unsupported on this model — unreachable** |
| `goform/refreshDLNA` | POST | dlna.html | action | **Unsupported on this model — unreachable** |
| `goform/GetUsbCfg` | GET | status_usb.html | get | **Unsupported on this model — 404** |
| `goform/setUsbUnload` | POST | status_usb.html | action | **Unsupported on this model — unreachable** |
| `goform/getThundercfg` | GET | xunleiDownload.html | get | **Unsupported on this model — 404** |
| `goform/setThundercfg` | POST | xunleiDownload.html | set | **Unsupported on this model — unreachable** |
| *(none)* | — | ap_diagnosis.html | static | Safe — no endpoint, static troubleshooting text |
| `goform/getHomeLink` (indirect, via main.js) | GET | status_extender.html | get | Safe — page itself makes no call, only reads `top.G.homePage` |
| *(none, dead code)* | — | wan_status.html | static | Safe — page makes no call; its only caller is commented out in main.js |

---

### `GET goform/SysToolpassword` — page: system_password.html (UI title: "Login Password")
- **Purpose**: Fetches whether an admin password is currently set and whether remote (WAN-side) web management is enabled, to drive client-side field visibility/validation before a password change.
- **Request**: No params other than the standard cache-buster: `goform/SysToolpassword?<Math.random()>`.
- **Response**: `{"ispwd":1,"remoteEn":0}`
  - `ispwd` (0/1): whether a login password is currently configured. If `1`, the "Old Password" field is shown/required; if `0` it's hidden (fresh/no-password router).
  - `remoteEn` (0/1): whether Web-based Remote Management is enabled. Used only client-side to block saving an empty new password while remote management is on (see Risk).
- **Risk**: Safe (read-only).
- **Notes**: Polled once on page load via `initPwd()` in `js/system.js`.

### `POST goform/SysToolChangePwd` — page: system_password.html (UI title: "Login Password")
- **Purpose**: Changes the router admin login password.
- **Request** (`html/system_password.html` form `frmSetup`, full native form submit — not AJAX): fields `GO`, `SYSOPS`, `SYSPS`, `SYSPS2` (all hidden inputs populated by JS right before submit).
  - `GO` = `"system_password.html"` (fixed, tells the backend which page to redirect back to on error).
  - `SYSOPS` = old password, **MD5-hashed client-side** (`hex_md5()` from `js/libs/md5.js`) unless the "Old Password" field is hidden (`#old_pwd` has class `none`, i.e. `ispwd==0`) or left blank, in which case it's sent as `""`.
  - `SYSPS` / `SYSPS2` = new password / confirm, each MD5-hashed the same way, or `""` if left blank.
  - Client-side validation (`pwdview.checkData()` in `js/system.js`) before submit: new ≠ old; no non-ASCII chars; no leading/trailing space; length ≥5 if non-empty; new == confirm; and — if new password is blank while `remoteEn=="1"` (from `SysToolpassword`) — refuses to submit at all ("security risk" message), since an empty password with remote management on would allow unauthenticated WAN access.
  - Example body (values shown are MD5 digests, not the plaintext password): `GO=system_password.html&SYSOPS=&SYSPS=<redacted-md5>&SYSPS2=<redacted-md5>` (example: router currently has no password set, `ispwd=0`, so `SYSOPS` is empty).
- **Response**: Full page reload/redirect (native form POST, not JSON). On failure the backend redirects to `system_password.html?1`, which `initPwd()` detects (`location.search == "1"`) and shows "Incorrect old password." Success presumably redirects back to `system_password.html` (or `?0`/no query) and re-shows the form; changing the password immediately invalidates the current session (subsequent requests will be treated as logged out, per `top.loginOut()`/session-cookie behavior elsewhere in the app), forcing re-login with the new password.
- **Risk**: Safe to call for a legitimate change, but a wrong old-password / bad new-password combination just re-shows an error — no destructive side effect. The real risk is operator error: setting a new password without remote management disabled and blank is specifically blocked, but setting a password you don't record locks you out of the UI (LAN access to the router itself is unaffected).
- **Notes**: Passwords are hashed client-side with MD5 before transmission (weak by modern standards, but note the hash itself functions as the credential over the wire — do not log/quote it as if it were harmless).

### `POST goform/SysToolReboot` — page: system_reboot.html (UI title: "Reboot and Reset")
- **Purpose**: Reboots the router.
- **Request**: Native form submit (`html/system_reboot.html`, `frmSetup`, first form on the page), single hidden field: `action=0` (fixed constant, not user-editable). Example body: `action=0`.
- **Response**: Full page reload; UI text states "The router will disconnect from the internet for about 45 seconds when it reboots."
- **Risk**: **Reboots the router** — drops WAN, LAN and WiFi connectivity for roughly 45 seconds.
- **Notes**: Triggered purely by the "Reboot" button (`#sys_reboot`) calling `document.forms[0].submit()` in `js/system.js`; no confirmation dialog.

### `POST goform/SysToolRestoreSet` — pages: system_reboot.html (UI title: "Reboot and Reset"), system_config.html (unreachable from UI, see note above)
- **Purpose**: Restores factory default settings.
- **Request**: Native form submit, single hidden field `action=0` (fixed constant). On `system_reboot.html` it's the second `<form>` on the page (`document.forms[1]`), submitted by the "Reset" button (`#sys_config`) with no confirmation dialog beyond the static warning text ("Restoring the factory settings deletes all current settings..."). Example body: `action=0`.
- **Response**: Full page reload (native form POST). Not observed live; expected behavior per UI copy is a factory reset followed by reboot.
- **Risk**: **Destroys all current configuration** — WiFi SSID/password, WAN/PPPoE credentials, port forwarding, VPN, everything reverts to factory defaults. Router will need full reconfiguration to regain internet access. Effectively also a reboot.
- **Notes**: Same endpoint is embedded on two pages; `system_config.html` is dead code in the current build (its only menu entry, `case "sys_config"` in `js/main.js`, is commented out), so in practice this action is only reachable via the "Reset" button on `system_reboot.html`.

### `GET cgi-bin/DownloadCfg/RouterCfm.cfg` — page: system_backup.html (UI title: "Backup/Restore")
- **Purpose**: Downloads the full router configuration as a `.cfg` file to the local machine.
- **Request**: Plain browser navigation (`window.location = "cgi-bin/DownloadCfg/RouterCfm.cfg"`), triggered by the "Backup" button after a `confirm()` dialog ("Do you want to back up your configuration to your local host?"). No parameters.
- **Response**: Binary/opaque `RouterCfm.cfg` file download (not JSON; content format not inspected here).
- **Risk**: Safe to call (read-only, no state change) — but the resulting file itself is highly sensitive: it embeds the router's full configuration, including WiFi keys and PPPoE/VPN credentials. Treat the downloaded file as a secret.
- **Notes**: n/a.

### `POST cgi-bin/UploadCfg` — page: system_backup.html (UI title: "Backup/Restore")
- **Purpose**: Restores a previously downloaded `.cfg` file, replacing the entire router configuration.
- **Request**: `multipart/form-data` POST of `html/system_backup.html`'s form (`enctype="multipart/form-data"`), single file field `filename`. Client-side validation before submit (`js/system.js`): the field must be non-empty and the filename must end in `.cfg`. Example body (multipart, schematic): `filename=<selected .cfg file>`.
- **Response**: Full page reload (native form POST). On failure it redirects to `system_backup.html?1`, detected by `initBackup()` which shows "Restoration failure." Success is expected to apply the restored configuration and reboot the router (not directly observed in this capture).
- **Risk**: **Fully replaces the running configuration** with whatever the uploaded file contains, and very likely reboots — equivalent in impact to a factory reset except the new state is whatever the .cfg encodes, which could itself drop WAN/WiFi/LAN connectivity depending on its contents.
- **Notes**: There is no server-side round-trip confirmation before applying — the browser fires the multipart POST directly on `<input type="file">`'s `change` event once a `.cfg`-named file is picked, no separate "confirm restore" click.

### `GET goform/GetSystemStatus` — page: system_status.html (UI title: "System Status")
- **Purpose**: Populates the read-only System/WAN/LAN/WiFi status dashboard.
- **Request**: `goform/GetSystemStatus` with jQuery's automatic cache-buster (`?_=<ts>`), no other params.
- **Response** (live capture; WAN IP/gateway/MAC shown as captured, not classified as a credential by this task but do treat as identifying):
  - `adv_sys_time` (string, `"2000-01-08 07:07:10"`): current system clock. The 2000-XX-XX value is the router's own placeholder for "not yet synced to internet time" (see system_log.html's note about the same convention).
  - `adv_run_time` (int, seconds, `630448`): uptime, rendered via `formatSeconds()`.
  - `adv_firm_ver` (string, `"V15.03.06.50_multi"`), `adv_hard_ver` (string, `"V1.0"`): firmware/hardware version strings, displayed verbatim.
  - `wanInfo` (array, one object per WAN port — this model has 1): each entry has `adv_connect_status` (int enum 0–3: `0`=Ethernet cable disconnected, `1`=Disconnected, `2`=Connecting…, `3`=Connected — decoded live value `3`=Connected), `adv_connect_time` (seconds as string, `"131877"`, rendered via `formatSeconds()`), `adv_ip`/`adv_mask`/`adv_gateway` (WAN-side IP config), `wanUploadSpeed`/`wanDownloadSpeed` (string decimal KB/s, `"0.72"`/`"0.82"`, rendered via `translateSpeed()`), `adv_connect_type` (string numeral 0–5, indexed into `[Dynamic IP Address, Static IP Address, PPPoE, Russia PPTP, Russia L2TP, Russia PPPoE]` — live value `"2"`=PPPoE), `adv_dns1`/`adv_dns2` (DNS servers), `adv_mac` (WAN MAC, upper-cased for display).
  - `adv_lan_ip`, `adv_lan_mask`, `adv_lan_mac`: LAN-side IP/mask/MAC, displayed verbatim.
  - 2.4GHz WiFi: `adv_wrl_en` (string "0"/"1" — **inverted naming**: despite the name, `1` means the network is **hidden** ("Network invisible"), not that it's enabled; `0` shows "Visible". Actual radio-on/off state is a separate field, `wifi_enable`, not present in this particular live capture's top-level JSON — the page only renders it when present), `adv_wrl_ssid` (SSID string), `adv_wrl_sec` (string enum `none`/`wpapsk`/`wpa2psk`/`wpawpa2psk`, decoded to `None `/`WPA-PSK`/`WPA2-PSK`/`WPA/WPA2-PSK`; live value `wpa2psk`), `adv_wrl_channel` (channel number string), `adv_wrl_band` (`"20"` or `"40"`, or `"auto"` which is displayed as "20/40"), `adv_wrl_mac`.
  - 5GHz WiFi: `wifi_enable_5g` (string "0"/"1" — `0` = 5GHz radio off, hiding the whole 5GHz block; live value `"0"`), plus the 5GHz counterparts of the above fields (`adv_wrl_en_5g`, `adv_wrl_ssid_5g`, etc.) when the radio is on — not populated in this capture since 5GHz is off.
- **Risk**: Safe (read-only).
- **Notes**: The page also auto-refreshes the WAN block every 5s while it's the active tab (`refreshTimer` in `js/system_status.js`). Dual-WAN models (`top.G.wanNum==2`) get a second WAN status block with `2`-suffixed field names (`adv_connect_type2`, etc.) — not applicable to this single-WAN AC10.

### `GET goform/GetSysTimeCfg` — page: system_time.html (UI title: "Time Settings")
- **Purpose**: Fetches current time zone / NTP configuration and current system time.
- **Request**: `goform/GetSysTimeCfg` with cache-buster.
- **Response**: `{"timeType":"sync","timeZone":"14:00","timePeriod":"86400","ntpServer":"time.windows.com","time":"2000-01-08 07:07:10","isSyncInternetTime":"false"}`
  - `timeType` (string, `"sync"`): mode; the UI only implements the "sync with NTP" mode (a "manual set" block exists in the markup but is commented out).
  - `timeZone` (string `"H:MM"`, e.g. `"14:00"`): selected time zone, matched against `<option value=...>` in `html/system_time.html`'s `#timeZone` `<select>` (each option is `GMT offset : minutes`, e.g. `14:00` = GMT+02:00 Israel/Egypt/Bucharest; the `:10`-suffixed duplicate options are Russia/Ukraine-only alternates for the same offsets, shown only when the browser/UI language is `ru`/`uk`).
  - `timePeriod` (string, seconds, `"86400"`): NTP re-sync interval. Not user-editable in this build — the corresponding UI control is commented out in the HTML; the value is simply read back and resubmitted unchanged.
  - `ntpServer` (string, `"time.windows.com"`): NTP server. Also not user-editable here (its input is commented out); resubmitted unchanged.
  - `time` (string, current system time) and `isSyncInternetTime` (string "true"/"false" — whether the clock has actually synced): rendered as-is / drive a "(synchronized with internet time)" vs "(unsynchronized...)" hint.
- **Risk**: Safe.
- **Notes**: n/a.

### `POST goform/SetSysTimeCfg` — page: system_time.html (UI title: "Time Settings")
- **Purpose**: Saves the selected time zone.
- **Request**: AJAX POST, body built by `objTostring()` (urlencoded `key=value&...`, exact escaping unconfirmed — see top-of-doc caveat) over `{timePeriod, ntpServer, timeZone}`. Only `timeZone` actually reflects a user choice (from the `#timeZone` `<select>`); `timePeriod` and `ntpServer` are always resent verbatim from the last `GetSysTimeCfg` fetch (`initObj`), since their controls are disabled/hidden. There is also dead code (`ruTimeZoneList` loop, `for(var i=0;i++;i<...)`) that never executes due to a malformed for-loop condition (`i++` as the middle clause always short-circuits after the first iteration), so the Russia-specific `:10`-suffix remapping it's meant to do silently never runs.
  - Example body (from live values): `timePeriod=86400&ntpServer=time.windows.com&timeZone=14:00`.
- **Response**: JSON string parsed for `errCode` (`afterSubmit: callback` in `js/system_time.js`); `0` = success, shown via `top.showSaveMsg()`. On success the page also refreshes the top-level status widget (`top.advInfo.initValue()`).
- **Risk**: Safe.
- **Notes**: n/a.

### `GET goform/GetSySLogCfg` — page: system_log.html (UI title: "System Log")
- **Purpose**: Fetches the system event log for on-page display.
- **Request**: `goform/GetSySLogCfg` with cache-buster.
- **Response**: JSON array of log entries, each `{"index":<int>,"time":"YYYY-MM-DD HH:MM:SS","type":"system"|"wan","log":"<free text>"}`. Live capture has 219 entries spanning boot events, PPPoE dial sequence (PADI/PADO/PADR/PADS/LCP), WAN up/down, and LAN port link up/down. Per the page's own help text: "If the router is not connected to the internet, the default logging time is 2000-X-X XX:XX:XX" — i.e. timestamps before the clock has synced are relative-from-boot, not wall-clock.
- **Risk**: Safe (read-only).
- **Notes**: Client re-sorts entries newest-first and re-numbers `index` before paginating locally (10 rows/page via a `Pagination` helper) — the raw response order/index is not what's displayed.

### `GET cgi-bin/DownloadLog/syslog.tar` — page: system_log.html (UI title: "System Log")
- **Purpose**: Exports the full system log as a `.tar` archive.
- **Request**: Plain browser navigation (`window.location = "cgi-bin/DownloadLog/syslog.tar"`), triggered by the "Export" button. No parameters.
- **Response**: Binary `.tar` file download.
- **Risk**: Safe (read-only).
- **Notes**: The "Export" button is client-side disabled for 5 seconds after click ("takes about 5 seconds" per the code comment) to avoid double-triggering the download.

### `system_log.html` dead code note
`js/system_log.js`'s `moduleModel.getSubmitData()` returns `"wpsEn=" + $("#wpsEn").val()`, but `pageModel.setUrl` is `""` (no save endpoint configured) and there is no `#wpsEn` element anywhere in `html/system_log.html` — this is unreachable leftover code (likely copy-pasted from another page's module) with no live endpoint or effect.

### `GET goform/GetSysAutoRebbotCfg` — page: system_automaintain.html (UI title: "Automatic Maintenance")
- **Purpose**: Fetches the scheduled-reboot configuration.
- **Request**: `goform/GetSysAutoRebbotCfg` with cache-buster.
- **Response**: `{"autoRebootEn":"1","time":"03:00-05:0","rebootTime":"03:00","delayRebootEn":"false","timeUp":"0","speed":"3"}`
  - `autoRebootEn` (string "0"/"1"): whether scheduled auto-reboot is on (live: enabled).
  - `rebootTime` (string `"HH:MM"`): scheduled reboot time; split into hour/minute `<select>`s.
  - `delayRebootEn` (string "true"/"false"): if true, defers the scheduled reboot while the router is passing traffic above the threshold noted in the UI copy ("higher than 3 KB/s" — matches the unused `speed` field, `"3"`, which is not otherwise read by the JS).
  - `timeUp` (string "0"/"1"): whether the system clock is synced to internet time; `0` shows the warning "Automatic maintenance takes effect only if the system time is synchronized with the internet time."
  - `time` (string, informational range e.g. `"03:00-05:0"`): present in the response but not read anywhere in `js/system_automaintain.js`.
- **Risk**: Safe.
- **Notes**: n/a.

### `POST goform/SetSysAutoRebbotCfg` — page: system_automaintain.html (UI title: "Automatic Maintenance")
- **Purpose**: Saves the scheduled-reboot configuration.
- **Request**: AJAX POST, body built by manual string concatenation (not `objTostring`): `autoRebootEn=<0|1>&delayRebootEn=<true|false>&rebootTime=<HH>:<MM>`, where `HH`/`MM` come from the `#rebootHour`/`#rebootMin` `<select>`s (hour 00–23, minute in 5-minute steps 00–55). Example body (from live values): `autoRebootEn=1&delayRebootEn=false&rebootTime=03:00`.
- **Response**: JSON `errCode` (`0`=success), shown via `top.showSaveMsg()`.
- **Risk**: Safe to call now — it only schedules a *future* reboot at the chosen time, it does not reboot immediately. (Endpoint name is misspelled "Rebbot" throughout, matching `GetSysAutoRebbotCfg`.)
- **Notes**: Misspelling "Rebbot" is consistent between the get/set endpoint names.

### `GET goform/SysToolGetUpgrade` — page: system_upgrade.html (UI title: "Firmware Upgrade")
- **Purpose**: Fetches the currently running firmware version for display.
- **Request**: `goform/SysToolGetUpgrade?<Math.random()>`.
- **Response**: `{"cur_fw_ver":"V15.03.06.50_multi"}` — shown in the "Current Version" field.
- **Risk**: Safe.
- **Notes**: Called from `js/directupgrade.js`'s `initEvent()`, independent of which upgrade mode (online/local) is selected.

### `POST cgi-bin/upgrade` — page: system_upgrade.html, "Local Upgrade" mode (UI title: "Firmware Upgrade")
- **Purpose**: Uploads and flashes a local firmware image.
- **Request**: `multipart/form-data` POST of `html/system_upgrade.html`'s form, fields: hidden `action=0` and file field `upgradeFile`. Client-side validation only checks that a file was selected before allowing submit. Example body (multipart, schematic): `action=0&upgradeFile=<selected firmware file>`.
- **Response**: Full page reload (native form submit, not AJAX). On error the backend redirects with a numeric query code that `initUpgrade()` in `js/directupgrade.js` decodes: `1001`=Format error, `1002`=CRC check failure, `1003`=File size error, `1004`=Fail to upgrade, `1005`=Internal memory not enough (reboot before upgrading). Success is expected to flash and reboot the router (not directly observed here).
- **Risk**: **Flashes new firmware and reboots the router** — a bad/incompatible image can brick the device. No confirmation dialog beyond the static "Do not power off the router during the upgrade" warning text.
- **Notes**: The button is disabled immediately on click to prevent double-submit.

### `GET goform/cloudv2?module=wansta&opt=query` — pages: system_upgrade.html (online upgrade), cloud_managment.html
- **Purpose**: Checks whether the router currently has WAN/internet connectivity, as a precondition gate before contacting Tenda's cloud servers (online upgrade check, or Tenda-App cloud binding).
- **Request**: `goform/cloudv2?module=wansta&opt=query&rand=<Math.random()>`.
- **Response**: `{"wan_sta":1}` — `wan_sta` (0/1): `0`=no internet access (blocks the online-upgrade / cloud-account flow with a "Failed to access the internet" message), `1`=has access (live capture: `1`).
- **Risk**: Safe.
- **Notes**: Same `cloudv2` CGI endpoint is shared by both pages, disambiguated by the `module=`/`opt=` query params; see the endpoint-naming pattern repeated for `manage`/`olupgrade` below.

### `GET goform/cloudv2?module=olupgrade&opt=queryversion` — page: system_upgrade.html, "Online Upgrade" mode
- **Purpose**: Asks Tenda's cloud service whether a newer firmware version is available.
- **Request**: `goform/cloudv2?module=olupgrade&opt=queryversion&rand=<new Date().toTimeString()>`.
- **Response**: `{"ver_info":{"err_code":0,"resp_type":1}}` (live capture)
  - `err_code` (int enum, shared across all `cloudv2` calls — see full decode table below in Notes): `0`=no error.
  - `resp_type` (int): `0`=a newer version is available (page then shows `ver_info.detail.newest_ver`, `update_date`, and a `description`/`description_en`/`description_zh_tw` release-note array, language-selected), `1`=no newer version available (live capture: `1`, i.e. router is already up to date, or Tenda's OTA has nothing newer registered for this hardware/version).
  - Special case `err_code==19` ("Connecting to the server..."): the page keeps re-polling this same endpoint every 2s until a different code/result comes back.
- **Risk**: Safe (read-only query against Tenda's cloud).
- **Notes**: `cloudv2` shared error-code table (`onlineErrCode()` in both `js/directupgrade.js` and `js/cloud_management.js`): `0`=OK, `1`=Unknown error, `2`=JSON too long, `3`=Out of memory, `4`=Connection failed, `5`/`6`=Socket connect failed, `7`/`8`=Command execution failed, `9`=Invalid command, `10`/`11`/`12`=Data parse/package/detect failure, `13`/`14`/`17`=Server connection failed, `15`=Authentication failure, `16`=Tenda App feature disabled, `18`=Cloud server busy, `19`=Connecting (retry).

### `GET goform/cloudv2?module=olupgrade&opt=queryupgrade` — page: system_upgrade.html, "Online Upgrade" mode
- **Purpose**: Starts and polls the online-download-and-flash process once the user clicks the online "Upgrade" button.
- **Request**: `goform/cloudv2?module=olupgrade&opt=queryupgrade&rand=<new Date().toTimeString()>`. First call fires right after `dowloadSoft()` confirms `wan_sta==1`; subsequent polls happen on a self-rescheduling timer (`checkingStatus()`), interval varies by state: 5000ms while queued/preparing, 2000ms while queuing/waiting/downloading/at `err_code==19`.
- **Response**: `{"up_info":{"err_code":19}}` (live capture — router was idle/not mid-upgrade, so it reports "connecting"). Full state machine per `up_info.resp_type` when `err_code==0`: `0`=preparing to download (shows progress UI, keeps polling), `1`=out of memory (shows "reboot before downloading" error, stops), `2`=queued on Tenda's server (`up_info.detail.{pos,time}` = queue position/estimated wait seconds, keeps polling), `3`=actively downloading firmware (`up_info.detail.{recved,fw_size,sec_left}` drive a progress bar/remaining-time display, keeps polling), `4`=writing/flashing firmware (switches to a purely client-side simulated progress bar via `onlineProgress()` that counts 0→100% over ~145s and then calls `top.jumpTo(window.location.host)` to reload the UI, assuming the router has rebooted by then).
- **Risk**: **Once triggered (resp_type reaching 3/4), this downloads and flashes new firmware from Tenda's cloud and reboots the router** — same real-world impact as the local upgrade path, just server-initiated. The GET-with-side-effects design means simply polling this URL for status is safe, but the very first call in the sequence (from `dowloadSoft()`) is what kicks the process off.
- **Notes**: If the raw response comes back as an HTML document (`obj.indexOf("!DOCTYPE html") >= 0`) instead of JSON — e.g. because the router reset the session/page mid-upgrade — the client treats that as a hard failure ("Upgrade error. Please check the internet connection status.").

### `POST goform/setNotUpgrade` — page: system_upgrade.html, "Online Upgrade" mode
- **Purpose**: Records that the user opted not to install a detected new version, so it isn't nagged again for that version.
- **Request**: `dataStr = "action=1&newVersion=" + $("#new_fw_ver").html()`, sent via `$.GetSetData.setData`. Fires only when the user closes the upgrade dialog (`noUpgradeFirm()`) with the "don't ask again" checkbox (`#noUpdateTips`) checked. Example body: `action=1&newVersion=<latest detected version string>`.
- **Response**: Not inspected by the caller — it just calls `top.closeIframe()` in the callback regardless.
- **Risk**: Safe.
- **Notes**: n/a.

### `GET goform/cloudv2?module=manage&opt=querybasic` — page: cloud_managment.html (UI title: "Tenda App")
- **Purpose**: Fetches whether Tenda-App cloud management is enabled and the router's cloud-binding serial number (SN).
- **Request**: `goform/cloudv2?module=manage&opt=querybasic&rand=<Math.random()>`.
- **Response**: `{"err_code":0,"enable":0,"sn":"<sn>"}` (live capture; `enable:0` — Tenda App management is currently off on this router)
  - `err_code`: shared `cloudv2` error code (see decode table above).
  - `enable` (0/1): whether remote management via the Tenda App is on.
  - `sn` (string): the router's cloud device ID, shown in the "ID" field and used by the app to pair. If empty while `enable=="1"`, the page keeps re-polling this same endpoint every 3–5s until an SN is assigned.
- **Risk**: Safe (read-only).
- **Notes**: n/a.

### `GET goform/cloudv2?module=manage&opt=queryaccount` — page: cloud_managment.html
- **Purpose**: Fetches the Tenda cloud account (email/phone) currently bound for app-based remote management.
- **Request**: `goform/cloudv2?module=manage&opt=queryaccount&rand=<Math.random()>`.
- **Response**: `{"err_code":0,"list":""}` (live capture — no account currently bound). `list` (string): the bound email/phone, pre-filled into the `#account` field when non-empty.
- **Risk**: Safe.
- **Notes**: Polled with retry-on-`err_code==19` (connecting) every 3s while cloud management is enabled.

### `POST goform/cloudv2?module=manage&opt=setaccount` — page: cloud_managment.html
- **Purpose**: Binds a Tenda cloud account (email or, for `cn` locale only, a phone number) to the router for app-based remote management.
- **Request**: `goform/cloudv2?module=manage&opt=setaccount&rand=<Math.random()>` with body `list=<account>` (POST body, not query — via `$.post`). Client validates the account is a syntactically valid email (regex check in `verifyEmail()`), or for `B.getLang()=="cn"` also accepts a Chinese mobile number pattern, before allowing submit. Only sent when the "Manage with Tenda App" toggle is being turned on. Example body: `list=user@example.com` (redacted placeholder — no account is bound live, so there is no real value to quote).
- **Response**: JSON `{"err_code": ...}`. `err_code==19` (connecting) triggers a 3s retry of the whole `preSubmit()` flow; `err_code==0` proceeds to also call `setbasic` (below) to persist `enable=1`; any other code shows the decoded `onlineErrCode()` message and aborts.
- **Risk**: Safe functionally, but enables remote (internet-facing, via Tenda's cloud relay) control of the router from whatever account is bound — a security-relevant change, not a connectivity risk.
- **Notes**: n/a.

### `POST goform/cloudv2?module=manage&opt=setbasic` — page: cloud_managment.html
- **Purpose**: Persists the Tenda-App cloud management on/off toggle.
- **Request**: `goform/cloudv2?module=manage&opt=setbasic&rand=<Math.random()>` with body `enable=<0|1>` (via `$.post`). Called directly when turning the toggle off (no account needed), or after `setaccount` succeeds when turning it on. Example body: `enable=0`.
- **Response**: JSON `{"err_code": ...}` (note: `js/cloud_management.js` does an extra `$.parseJSON(obj)` on the response in `saveEnable()`'s callback, implying the raw response is a JSON-encoded string, not a plain object, unlike most other endpoints on this router). `err_code==0` = success, refreshes the top-level status widget (`top.advInfo.initValue()`); otherwise shows the decoded error.
- **Risk**: Safe; toggling this off immediately disables remote app-based management (does not affect LAN access).
- **Notes**: n/a.

### `GET goform/GetPptpClientCfg` — page: pptp_client.html (UI title: "PPTP/L2TP Client")
- **Purpose**: Fetches the PPTP/L2TP client configuration and live tunnel status.
- **Request**: `goform/GetPptpClientCfg?<Math.random()>` (initial load), then polled every 2s while the page is open (`refreshStatus()`).
- **Response**: `{"clientEn":"0","clientType":"pptp","domain":"","clientMppe":"0","clientMppeOp":"128","clientWanid":"1","userName":"","password":"","clientIp":"","clientMask":"","pptpStatus":"0","pptpIp":"0.0.0.0","l2tpStatus":"0","l2tpIp":"","wanConnType":"2","wanUser":"<redacted>","wanIp":"203.0.113.10"}`
  - `clientEn` (0/1): PPTP/L2TP client feature on/off (live: off).
  - `clientType` (string enum `"pptp"`/`"l2tp"`): which client mode is configured.
  - `domain` (string): PPTP/L2TP server IP or domain name to dial. Client-side validated to differ from the router's own WAN IP.
  - `clientMppe` (0/1), `clientMppeOp` (string `"40"`/`"128"`): MPPE encryption on/off and bit strength; PPTP-only (ignored/passed through unchanged for L2TP).
  - `clientWanid` (string, `"1"`): which WAN port this client binds to (present in response, not surfaced in the single-WAN UI).
  - `userName`/`password` (strings): PPTP/L2TP auth credentials — **treat as secrets**; live values are empty (feature not configured).
  - `clientIp`/`clientMask`: present in response but their corresponding UI inputs are commented out in `html/pptp_client.html` — not currently editable.
  - `pptpStatus`/`l2tpStatus` (string enum 0–2, indexed into `[Disconnected, Connected, Connecting… ]`): live tunnel connection state for each protocol, independent of which `clientType` is currently selected.
  - `pptpIp`/`l2tpIp` (string, obtained tunnel IP or `"0"`/`""`/`"0.0.0.0"` when not connected): shown as "Obtained PPTP/L2TP Client IP Address" only when the corresponding status is `"1"` (Connected) and the IP isn't `"0"`/empty.
  - `wanConnType` (string numeral, same enum as `GetSystemStatus`'s `adv_connect_type`; live `"2"`=PPPoE): the underlying WAN connection type, used for a validation check.
  - `wanUser` (string): the router's own PPPoE username on its WAN uplink — **redacted here; a real PPPoE credential**.
  - `wanIp` (string): current WAN IP, used to reject a client `domain` value equal to it.
- **Risk**: Safe (read-only).
- **Notes**: n/a.

### `POST goform/SetPptpClientCfg` — page: pptp_client.html (UI title: "PPTP/L2TP Client")
- **Purpose**: Enables/disables and configures the router as a PPTP or L2TP VPN client (dialing out to an upstream VPN server).
- **Request**: AJAX POST, body via `objTostring()`.
  - When enabling (`clientEn=="1"`): `{clientEn, clientType, clientMppe, clientMppeOp, domain, userName, password}`, where `clientMppe`/`clientMppeOp` are only taken from the form when `clientType=="pptp"` — for `"l2tp"` they're resent unchanged from the last fetch (`initObj`), since L2TP MPPE isn't user-configurable in this UI.
  - When disabling (`clientEn=="0"`): all other fields are resent verbatim from `initObj` (the last `GetPptpClientCfg` fetch) — i.e. disabling doesn't clear the stored server/credentials, only the enable flag.
  - Client-side validation before submit (only when enabling): `domain`, `userName`, `password` must all be non-empty; `domain` must not equal the router's own `wanIp`; `userName`/`password` are restricted by a custom `ppoe` validator to exclude `~;'&"%`, whitespace, and non-ASCII characters.
  - Example body (from live/disabled state): `clientEn=0&clientType=pptp&clientMppe=0&clientMppeOp=128&domain=&userName=&password=`.
- **Response**: JSON `{"errCode":<n>}`. If `clientEn=="1"`, success just shows a save toast and restarts the 2s status poll (the tunnel connects asynchronously in the background); if `clientEn=="0"`, it also refreshes the top-level VPN status widget (`top.vpnInfo.initValue()`).
- **Risk**: Enabling this reconfigures the router's default route to potentially tunnel traffic through the specified VPN server; a wrong/unreachable server can leave the client stuck "Connecting…" but does not itself break local WAN/LAN/WiFi. Disabling it drops any existing PPTP/L2TP client tunnel.
- **Notes**: The enable toggle button's initial CSS class is deliberately set to the *opposite* of the fetched state and then immediately toggled via `changeClientEn()` (`js/pptp_client.js` `initValue()`) — this is an initialization trick to reuse `changeClientEn()`'s side effects (updating the Save/Connect button label and show/hiding `#client_set`), not a bug; the net displayed state does match `clientEn`.

### `GET goform/GetPptpServerCfg` — page: pptp_server.html (UI title: "PPTP Server")
- **Purpose**: Fetches the PPTP server configuration, IP pool, and configured user accounts.
- **Request**: `goform/GetPptpServerCfg?<Math.random()>` (initial load), then polled every 5s (`checkServerStatus`) purely to refresh per-user connection status.
- **Response**: JSON array; element `[0]` is the server-wide config, elements `[1..]` (absent in this live capture — no users configured) are one object per configured PPTP user account (`{userName, password, enable, connsta}` per `addList()`/`updateConnectStatus()` usage in the JS).
  - Live capture, `[0]`: `{"serverEn":"0","wanid":"1","mppe":"0","mppeOp":"128","startIp":"10.0.0.100","endIp":"10.0.0.200","lanIp":"192.168.0.1","lanMask":"255.255.255.0","guestIp":"192.168.10.1","guestMask":"255.255.255.0","serverIp":"","vlan2Ip":"","vlan2Mask":"","wanIp":"203.0.113.10","wanMask":"255.255.255.255","pptpSvrIp":"10.0.0.1","pptpSvrMask":"255.255.255.0"}`
    - `serverEn` (0/1): PPTP server on/off (live: off).
    - `wanid` (string, `"1"`): bound WAN port index.
    - `mppe`/`mppeOp`: MPPE encryption on/off and bit strength (40/128) for the server.
    - `startIp`/`endIp`: PPTP client IP address pool bounds (last octet only is user-editable in the UI; the /24 prefix is derived from `endIp`).
    - `lanIp`/`lanMask`, `guestIp`/`guestMask`, `serverIp`, `vlan2Ip`/`vlan2Mask`, `wanIp`/`wanMask`: various local subnets, read purely for client-side validation that the chosen IP pool doesn't overlap LAN/guest/VLAN2/WAN segments.
    - `pptpSvrIp`/`pptpSvrMask`: the server's own tunnel-interface IP/mask (`10.0.0.1/24` by default), not directly surfaced in the UI.
  - Per-user rows (when present): `userName`/`password` (strings — **credentials, treat as secrets**), `enable` (0/1), `connsta` (0/1, live connection status for that account).
- **Risk**: Safe (read-only).
- **Notes**: n/a.

### `POST goform/SetPptpServerCfg` — page: pptp_server.html (UI title: "PPTP Server")
- **Purpose**: Saves PPTP server-wide settings (on/off, IP pool, MPPE).
- **Request**: AJAX POST, body via manual concatenation (not `objTostring`).
  - Enabling (`serverEn=="1"`): `serverEn=1&startIp=<ip>&endIp=<3rd-octet-prefix><last-octet>&mppe=<0|1>&mppeOp=<40|128>`.
  - Disabling (`serverEn=="0"`): same keys, all values resent from `initObj[0]` unchanged except `serverEn=0`.
  - Client-side validation (only when enabling) rejects a pool whose start IP collides with the LAN, WAN(s), a connected-server IP, or VLAN2 subnet; requires end ≥ start, start's last octet ≠ 1, and at least 8 addresses in the pool.
  - Example body (from live/disabled state): `serverEn=0&startIp=10.0.0.100&endIp=10.0.0.200&mppe=0&mppeOp=128`.
- **Response**: JSON `{"errCode":<n>}`, shown via `top.showSaveMsg()`; on success also refreshes `top.vpnInfo.initValue()` and clears the 5s connection-status poll.
- **Risk**: Safe — this endpoint does not touch the user-account list (that's a separate call, below), so it can't disconnect existing PPTP clients on its own beyond an outright `serverEn=0` disable.
- **Notes**: This endpoint's request body never actually includes the user list, despite `getPptpServerList()` (which builds a `list=` param) existing in the same file — that function is only invoked by the separate `setPptpUserList` action below, and the (commented-out) `data += "&list=..."` block in `SetPptpServerCfg`'s own `getSubmitData` was disabled.

### `POST goform/setPptpUserList` — page: pptp_server.html (UI title: "PPTP Server")
- **Purpose**: Adds, enables, disables, or deletes a PPTP server user account. One endpoint handles all four operations by resubmitting the *entire* current table state.
- **Request**: AJAX POST, single param `list=<rows>`. Rows are joined by `~`; within a row, fields are joined by `;` in this fixed order: `username;password;enable;netEn;serverIp;serverMask;remark` (the client always sends `netEn=0` and empty `serverIp`/`serverMask`/`remark` — those sub-fields are UI-disabled/commented out). `username`/`password` are `encodeURIComponent`-escaped. `enable` is `"1"` for on, `"0"` for off. On **add**, the new row's data (from the "+New" row's `#userName`/`#password` inputs) is appended after all existing rows; the row being added always starts `enable=1`. On **delete**, the row marked `data-target="delete"` is simply omitted from the resubmitted list. On **enable/disable**, that row's `enable` value is flipped in place. Max 8 rows total (client-enforced).
  - Example body for adding a first user (router currently has zero users configured, so there's no real captured example — built from the code's format): `list=alice;<redacted-password>;1;0;;;`
- **Response**: JSON `{"errCode":<n>}`; on `0`, the corresponding client-side table mutation (`handlerList()`/`delList()`) is applied to the DOM.
- **Risk**: Deleting or disabling a user immediately terminates that user's active PPTP session (their `connsta` will flip to disconnected). Not otherwise disruptive to the router itself.
- **Notes**: Username uniqueness and basic username/password validity (`$.validate.valid.pwd`) are enforced client-side only, before this call is made.

### `GET goform/getPptpOnlineClient` — page: pptp_user.html (UI title: "Online PPTP Users")
- **Purpose**: Lists currently-connected PPTP server clients (this router acting as PPTP server) with session details.
- **Request**: `goform/getPptpOnlineClient?<Math.random()>`, polled every 5s continuously while the page is open.
- **Response**: `{"clientList":[]}` (live capture — no clients connected). Each entry (when present) is expected to be `{"username":..., "dialIP":..., "clientIP":..., "onlineTime":<minutes, int>}` per `initUserList()`'s rendering code (`username`, dial-in IP, assigned client IP, and online duration rendered via `formatSeconds()`).
- **Risk**: Safe (read-only).
- **Notes**: This is a display-only page — no set/action endpoints exist for it; it just visualizes `getPptpOnlineClient` in a table, showing "The online users list is empty." when `clientList` has zero entries.

### `GET goform/GetPrinterCfg` — page: printer.html (UI title: "Share Printer") — **unsupported on this model**
- **Purpose** (as coded): Fetches USB-printer-sharing status/name.
- **Request**: `goform/GetPrinterCfg?<Math.random()>`.
- **Response** (live capture): router's stock CGI error page, `Access Error: Data follows — Form GetPrinterCfg is not defined`. The goform handler does not exist in this firmware build.
- **Risk**: N/A — unreachable. `printer.html` is never opened by any live UI path on this SKU (see model-gating note at top of doc): `CONFIG_USB_MODULES=="n"` removes the entire USB-App menu section that would host it.
- **Notes**: n/a.

### `POST goform/SetPrinterCfg` — page: printer.html (UI title: "Share Printer") — **unsupported on this model**
- **Purpose** (as coded): Toggles USB printer sharing on/off (`printerEn=<0|1>`, sent automatically on every toggle click, no explicit Save button).
- **Request/Response**: Not tested live (page unreachable); expected to also 404 like `GetPrinterCfg`, since printer sharing depends on the same disabled USB stack.
- **Risk**: N/A — unreachable.
- **Notes**: n/a.

### `GET goform/GetSambaCfg` — page: samba.html (UI title: "Share File") — **unsupported on this model**
- **Purpose** (as coded): Fetches FTP/Samba file-sharing configuration (encoding, internet-access permission/port, guest account, LAN/WAN access links).
- **Request**: `goform/GetSambaCfg?<Math.random()>`.
- **Response** (live capture): `Access Error: Data follows — Form GetSambaCfg is not defined`.
- **Risk**: N/A — unreachable (same USB-module gating as above; the page also has its own client-side fallback for a "no USB device" state via a `&nousb` URL flag, separate from this model-level gating).
- **Notes**: n/a.

### `POST goform/SetSambaCfg` — page: samba.html (UI title: "Share File") — **unsupported on this model**
- **Purpose** (as coded): Saves FTP/Samba settings: `fileCode` (encoding: `UTF-8`/`GBK`/`Big5`), `password` (admin FTP/Samba password), `premitEn` (allow-internet-access 0/1), `guestpwd`/`guestuser`/`guestaccess` (guest account, `r`/`rw`), `internetPort` (WAN-facing port, only sent when `premitEn==1`).
- **Request/Response**: Not tested live (page unreachable).
- **Risk**: N/A — unreachable.
- **Notes**: n/a.

### `GET goform/GetDlnaCfg` — page: dlna.html (UI title: "DLNA") — **unsupported on this model**
- **Purpose** (as coded): Fetches DLNA media-server on/off state, device name, scan status, and the list of USB storage devices/folders available to share.
- **Request**: `goform/GetDlnaCfg?<Math.random()>` (also polled every 5s while the page is open, purely for `dlnaScanStatus`).
- **Response** (live capture): `Access Error: Data follows — Form GetDlnaCfg is not defined`.
- **Risk**: N/A — unreachable.
- **Notes**: n/a.

### `POST goform/SetDlnaCfg` — page: dlna.html (UI title: "DLNA") — **unsupported on this model**
- **Purpose** (as coded): Saves `dlnaEn` (0/1), `deviceName` (only sent when enabling; otherwise resent from the last fetch), and `scanList` — the set of folders to share, built from the currently-selected folder chips' `data-path` attributes, joined by a **tab character** (`\t`) and then `encodeURIComponent`-escaped as a whole. Max 10 folders (client-enforced).
- **Request/Response**: Not tested live (page unreachable).
- **Risk**: N/A — unreachable.
- **Notes**: n/a.

### `POST goform/expandDlnaFile` — page: dlna.html (UI title: "DLNA") — **unsupported on this model**
- **Purpose** (as coded): Expands one level of the USB storage folder tree (used when the user clicks to open a first- or second-level folder in the folder picker).
- **Request**: Params `folderGrade` (`"primary"`|`"secondary"`) and `filePath` (the parent path being expanded), sent as an object via `$.GetSetData.setData` (exact wire encoding unconfirmed — `js/libs/public.js` not present in this mirror).
- **Response** (as coded): JSON-encoded string with a `subfileList` array of `{fileName, hasChildFile}` entries for that folder level.
- **Risk**: N/A — unreachable.
- **Notes**: n/a.

### `POST goform/refreshDLNA` — page: dlna.html (UI title: "DLNA") — **unsupported on this model**
- **Purpose** (as coded): Forces a re-scan of USB media for the DLNA server.
- **Request**: `action=1` (fixed constant). Note: the button that would call this (`#refresh`) is commented out in `html/dlna.html`, so `refreshDLNA()` in the JS is itself dead code independent of the USB-module gating.
- **Response** (as coded): JSON `{"errCode":<n>}`, result not used by the caller.
- **Risk**: N/A — unreachable (doubly so: no USB support, and no UI element to trigger it even if there were).
- **Notes**: n/a.

### `GET goform/GetUsbCfg` — page: status_usb.html (UI title: "USB App") — **unsupported on this model**
- **Purpose** (as coded): Lists connected USB storage devices, their partitions, and used/free space, for the USB-App overview popup.
- **Request**: `goform/GetUsbCfg?<Math.random()>`, polled every 2s while the page is open.
- **Response** (live capture): `Access Error: Data follows — Form GetUsbCfg is not defined`.
- **Risk**: N/A — unreachable (see model-gating note at top of doc).
- **Notes**: A related but distinct endpoint, `goform/GetUSBStatus`, is polled from `js/main.js` (not from this page) for the top-level nav badge; it 404s identically on this router (live) — out of scope of this page's own JS but confirms the USB subsystem is absent end-to-end.

### `POST goform/setUsbUnload` — page: status_usb.html (UI title: "USB App") — **unsupported on this model**
- **Purpose** (as coded): Safely unmounts/ejects a specific USB storage device.
- **Request**: `deviceName=<encodeURIComponent(device id)>`, triggered by an "unlink"/eject button per device row.
- **Response**: Not tested live; unreachable.
- **Risk**: N/A — unreachable. (If it worked, this would be a "safe eject" action — not itself connectivity-affecting, only affects attached USB storage availability.)
- **Notes**: n/a.

### `GET goform/getThundercfg` — page: xunleiDownload.html (UI title: "Offline Download with Xunlei") — **unsupported on this model**
- **Purpose** (as coded): Fetches Xunlei offline-download binding status and the device's activation code.
- **Request**: `goform/getThundercfg` (no cache-buster in this one, unlike most other GETs on this router).
- **Response** (live capture): `Access Error: Data follows — Form getThundercfg is not defined`.
- **Risk**: N/A — unreachable. Beyond the USB-module gating, this feature's own entry point is separately hard-disabled: the `#usb_xunlei` menu tile that would open this page is commented out in `html/main.html`.
- **Notes**: n/a.

### `POST goform/setThundercfg` — page: xunleiDownload.html (UI title: "Offline Download with Xunlei") — **unsupported on this model**
- **Purpose** (as coded): Toggles the Xunlei offline-download feature (`thunderEn=true|false`) or unbinds the device from a Xunlei account (`action=unbind`).
- **Request/Response**: Not tested live; unreachable for the same reasons as `getThundercfg`.
- **Risk**: N/A — unreachable.
- **Notes**: n/a.

### `ap_diagnosis.html` (UI title: "Diagnose") — no endpoint
- Purely static troubleshooting copy ("Verify that the connection between this router and its upstream router is normal...") shown in a popup when the user clicks a "show info"/diagnose link from the WAN status area in extender/AP-related modes. Contains no `<script>` beyond the standard `lang/b28n_async.js` include; makes no network calls of its own.
- **Risk**: Safe (static content).

### `status_extender.html` (UI title: "Extend WiFi Signal") — no page-owned endpoint
- Static marketing content promoting Tenda's WiFi extender product, opened from the System Status page. Its one inline `<script>` sets the "Details" link's `href` to `top.G.homePage` — a value populated elsewhere (`main.js`'s `staInfo.getHomeLink()`, via `goform/getHomeLink`, itself gated to `cn` locale only, otherwise hardcoded to `http://www.tendacn.com/en/product/A9.html`). The page itself issues no request.
- **Risk**: Safe (static content; the only "action" is an outbound link to Tenda's own marketing site).

### `wan_status.html` — no endpoint, currently unreachable
- Reads `wanStatus=<N>` from its own query string and shows/hides one of a small set of static per-status messages (only status `0`, "no Ethernet cable on Internet port," is implemented in the markup). Makes no `goform`/`cgi-bin` calls itself.
- Its only intended caller, `mainPageLogic.showWanStatusPicIframe()` in `js/main.js`, has its entire function body commented out, so nothing in the current UI actually opens this page.
- **Risk**: Safe (static content; also currently unreachable).
