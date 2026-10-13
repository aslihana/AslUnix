# ![AslUnix logo](assets/aslunix-32.png) AslUnix

A simple terminal and system administration toolkit for Windows PCs and servers.

Check disks, resources, services, accounts and network from the same terminal. Copy it to a server and run it locally. Basic checks work offline. No Cygwin, MSYS2 or WSL needed.

## Usage

Download the [Windows x64 package](release/AslUnix-0.1.0-windows-amd64.zip) and extract it.

### Running the app

Run `aslunix.exe` directly from the extracted portable folder. No installation needed. You can double-click it or run it from PowerShell:

```powershell
.\aslunix.exe
```

You can also run commands directly:

```powershell
.\aslunix.exe health
.\aslunix.exe disks
.\aslunix.exe help
```

Use Tab to complete commands and paths. Use Up/Down for command history. Type `exit` to close the terminal.

### Taskbar shortcut

Keep the portable folder in a fixed location. Create a shortcut:

```powershell
.\scripts\Create-Shortcut.ps1
```

Right-click `AslUnix.lnk` and choose **Pin to taskbar**. The icon is embedded in the executable. Windows Terminal may keep its own window/taskbar icon.


## Commands

### Health

| Command | What it does |
|---|---|
| `health` | Runs disk, resource, service, administrator and RAID checks. |
| `disks`, `df` | Shows fixed volume capacity and free-space thresholds. |
| `resources` | Shows sampled CPU, memory, uptime and top processes. |
| `services` | Lists services and checks configured required services. |
| `raid` | Reads controller, volume, disk and rebuild states through PERCCLI. |
| `events` | Lists recent Windows warnings/errors. Supports time, ID and log filters. |
| `watch` | Repeats selected checks at the chosen interval. |
| `report` | Saves results and raw details as text/JSON, optionally HTML. |
| `report prune` | Removes expired reports after confirmation. Supports a dry run. |

### Accounts and permissions

| Command | What it does |
|---|---|
| `accounts` | Lists local accounts and available account metadata. |
| `admins` | Lists direct local Administrators membership by SID. Checks an optional baseline. |
| `admins --expand-domain` | Also attempts recursive domain group resolution. Requires AD access/module. |
| `permissions PATH` | Shows owner, ACL entries, inheritance and SDDL. Does not calculate effective access. |
| `identity` | Shows current user, groups, privileges and elevation. |

Administrator membership is not a complete list of every privileged identity.

### Network

| Command | What it does |
|---|---|
| `network` | Lists adapters, IPs, gateways, DNS, routes, neighbors and TCP/UDP endpoints with owners. |
| `ping HOST` | Sends four ICMP probes using Windows ping. |
| `traceroute HOST` | Shows the route using Windows tracert. |
| `dns NAME` | Looks up A/AAAA addresses. Use `--server IP[:PORT]` for a chosen DNS server. |
| `tcp HOST PORT` | Tests a TCP connection. |
| `http URL` | Sends GET and reports status, headers and latency. Does not follow redirects. |
| `tls HOST [PORT]` | Inspects certificates, hostname, system trust and expiry. No revocation check. |
| `proxy` | Shows WinHTTP, current-user and environment proxy settings. |
| `time` | Shows local time, timezone and raw Windows time-service status. |
| `domain` | Shows domain membership and attempts domain-controller discovery. |
| `wsus` | Shows WSUS policy. Use `--test` for a TCP connectivity check. |
| `capture` | Lists installed TShark capture interfaces. |
| `capture --interface ID --seconds N --out FILE` | Writes an explicit, bounded packet capture. Requires TShark/Npcap. |

### Files and logs

| Command | What it does |
|---|---|
| `pwd` | Shows the current directory. |
| `cd PATH` | Changes the directory in the interactive shell. |
| `ls [PATH]` | Lists directory contents. |
| `cat FILE` | Prints file contents. |
| `head FILE [-n N]` | Prints the first N lines. Defaults to 10. |
| `tail FILE [-n N]` | Prints the last N lines. Defaults to 10. |
| `grep TEXT FILE` | Finds literal, case-sensitive text and prints matching line numbers. |
| `find PATH [NAME_GLOB]` | Finds matching names recursively. Skips symlinks. |
| `du PATH` | Sums logical file sizes. Skips symlinks. |
| `cp SOURCE DEST` | Copies a regular file. Asks before overwriting. |
| `mv SOURCE DEST` | Moves a regular file on the same NTFS volume. No overwrite. |
| `mkdir PATH` | Creates one directory. |
| `rm PATH` | Removes a file after confirmation. Directories need `--recursive`. |
| `follow FILE` | Prints new log data. Handles rotation and truncation. |
| `compare FILE1 FILE2` | Compares text at each line position. |
| `sha256 FILE` | Calculates a SHA-256 checksum. |
| `archive list FILE.zip` | Lists ZIP contents. |
| `archive extract FILE.zip DEST` | Extracts ZIP files. Rejects overwrites, traversal and symlinks. |

`cp`, `mv`, `mkdir`, `rm` and archive extraction support `--dry-run`.
Use quotes for paths with spaces. These commands are a subset, not full GNU compatibility.

### Windows and applications

| Command | What it does |
|---|---|
| `os` | Shows Windows version, build, architecture and boot time. |
| `updates` | Lists installed QFE updates. Does not establish patch compliance. |
| `drivers` | Lists signed PnP driver information. |
| `tasks` | Lists scheduled tasks, actions and run results. |
| `hardware` | Shows computer, BIOS, CPU and disk inventory. |
| `registry [PATH]` | Reads documented system registry paths. Never writes values. |
| `iis` | Lists websites, bindings, application pools and IIS services. |
| `dotnet` | Lists runtimes, .NET Framework, IIS ASP.NET Core modules and Hosting Bundle registrations separately. |
| `signature FILE` | Inspects an Authenticode signature. |
| `tools` | Detects configured/PATH vendor executables and explains missing tools. |
| `vmware --server HOST` | Lists VMs, hosts, datastores, port groups and snapshots through PowerCLI. |
| `sql --server INSTANCE` | Lists database states and physical files using integrated authentication. |
| `sql --server INSTANCE --backup PATH` | Reads backup headers and file lists. Does not restore a database. |

### Shell

| Command | What it does |
|---|---|
| `help [COMMAND]` | Prints the command reference. |
| `version` | Prints the app version. |
| `history` | Lists this shell session's commands. |
| `exit`, `quit` | Closes the interactive shell. |

Global options go before the command: `--config PATH`, `--json`, `--color`, `--no-color`, `--version`.
For command options and limits, see the [full reference](docs/COMMANDS.md) or run `help`.

## Configuration

Copy the example config. Set disk thresholds, required services and approved administrator SIDs.

```powershell
Copy-Item aslunix.example.json aslunix.json
.\aslunix.exe --config aslunix.json health
```

Config is loaded only with `--config`. Use `--json` before a command for JSON output.

## Notes

- Targets 64-bit Windows 10/11 and Server 2016+. Requires Windows PowerShell 5.1.
- Health checks do not change system settings. File deletion asks for confirmation.
- `UNKNOWN` means the check could not complete. It does not mean healthy.
- Dell, VMware, SQL and TShark integrations need separately installed tools.
- 26 tests passed on macOS. The Windows build compiled. Windows and vendor hardware testing is still pending.

## Build

Requires Go 1.23+. The icon resource is included in the source. On Windows:

```powershell
.\scripts\build.ps1
go test ./...
```

[Commands](docs/COMMANDS.md) · [Setup](docs/DEPLOYMENT.md) · [Integrations](docs/INTEGRATIONS.md) · [Validation](docs/VALIDATION.md)
