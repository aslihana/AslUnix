# AslUnix

AslUnix is a working first version of a portable Windows terminal and administration toolkit. It provides an `aslunix>` shell, one-shot commands, read-only health collectors, explicit file operations, network probes, and text/JSON/HTML reports. It requires no Cygwin, MSYS2, WSL, Linux kernel, monitoring server, cloud account, or background service.

**Validation status:** the Go tests run on macOS and the x64 Windows executable is cross-compiled. No Windows command, console behavior, Windows PowerShell collector, Dell controller, VMware endpoint, or SQL Server has been validated on a real Windows system here. Use the included smoke test and validation checklist on a representative test machine before operational deployment. This is version 0.1.0, not a certified OT monitoring product.

## Portable quick start

Extract `release/AslUnix-0.1.0-windows-amd64.zip` into a writable directory on the Windows machine. The executable has its PowerShell collectors embedded. No installation or network access is needed for the core local checks.

```powershell
.\aslunix.exe help
.\aslunix.exe
.\aslunix.exe disks
.\aslunix.exe --json resources
Copy-Item .\aslunix.example.json .\aslunix.json
# Edit the copied config to name required services and approved administrator SIDs.
.\aslunix.exe --config .\aslunix.json health
.\aslunix.exe --config .\aslunix.json report --out .\reports --html
.\aslunix.exe permissions "C:\Program Files\DCS"
.\aslunix.exe watch disks,resources --interval 30 --count 5
```

From the interactive shell:

```text
aslunix> cd "C:\Program Files"
aslunix> ls
aslunix> health
aslunix> help
aslunix> exit
```

Configuration is loaded only when explicitly supplied with `--config`. Global flags precede the command. Disk thresholds use binary GiB even though configuration field names say `gb`. A disk becomes critical if either free percentage or free GiB meets its critical threshold, otherwise warning if either warning threshold is met. Resource thresholds apply at or above the configured CPU/memory percentages. Empty required-service and administrator-baseline lists mean those policies are not configured, not that every service or identity is approved. The example deliberately contains no guessed DCS service names.

## Target Windows platforms and prerequisites

The release targets **64-bit Windows 10/11 and Windows Server 2016/2019/2022/2025**, including Server Core with the required management components. This follows the [Go runtime minimum of Windows 10 / Windows Server 2016](https://go.dev/wiki/MinimumRequirements); these are intended targets, not a claim of completed testing or of OS vendor support. Windows 7/8 and Server 2008/2012, Nano Server, 32-bit Windows, and domain-controller privileged-group coverage are outside this release's scope.

* Inbox **Windows PowerShell 5.1**, CIM/WMI providers, and the relevant management modules must be present and permitted by local policy. AslUnix invokes the System32 PowerShell directly with no profile and does **not** bypass execution policy, application control, or constrained-language restrictions.
* No Go, Python, .NET SDK, PowerShell 7, or vendor tooling is required to run core commands. Go is needed only to build from source. Windows PowerShell itself relies on Windows components already installed on the machine.
* Start as a normal user. Elevation can make additional service/process, event, ACL, identity, and vendor data accessible. Restricted access produces `UNKNOWN` or an incomplete result rather than a healthy result. AslUnix never elevates itself.
* Accounts/admins target local SAM on member servers and workstations. Domain controllers and missing local Administrators groups report a limitation/error. Domain expansion is optional and needs the ActiveDirectory module and domain access.
* IIS commands need WebAdministration; VMware needs VMware.VimAutomation.Core; SQL inventory needs the SqlServer module and integrated authentication. TShark capture needs installed TShark/Npcap and suitable rights. PERCCLI is optional and needs compatible Dell controllers.

Local collectors use local CIM/WMI, not remote WinRM. Domain names in memberships may still trigger Windows resolution and fail when disconnected; that is recorded. No tool is installed or downloaded automatically. Optional probes and integrations intentionally contact the requested endpoint. Core health never repairs, restarts, changes registry/configuration, updates RAID, or enables remote access.

## Architecture

The Go standard library provides a single executable, filesystem operations, TLS/HTTP/DNS/TCP, bounded process execution, output formatting, and the interactive console editor. There are **no third-party Go modules**, avoiding runtime package installation. Windows-specific console code uses Win32 console input for history, Tab completion, and cursor editing. Redirected input remains line-oriented. Session history is bounded to 500 lines and never written to disk.

Embedded PowerShell scripts collect structured Windows data through supported CIM and management cmdlets. Collector parameters are passed as JSON through a child environment variable, never concatenated into executable PowerShell source. Scripts return data and limitations; Go interprets thresholds/policies. Raw JSON/diagnostics are retained alongside interpreted results. The `Runner` interface supports deterministic failure and policy tests without fabricating live Windows results. New collectors can be added independently; see [architecture](ARCHITECTURE.md).

Each result has a check name, UTC check time, source, summary, status, interpreted data, raw output, and limitations. `OK` on an inventory means collection succeeded within its documented scope, not that the application/system is comprehensively healthy. `health` combines disks, resources, services, direct administrator membership, and RAID. RAID missing on an ordinary VM remains `UNKNOWN` because AslUnix cannot establish that it is not applicable from a missing tool.

## Statuses, limits, and exit codes

| Status | Meaning | Exit |
|---|---|---:|
| `OK` | Check succeeded in the stated scope | 0 |
| `NOT_APPLICABLE` | Explicitly inapplicable/cancelled operation | 0 |
| `WARNING` | Warning threshold, unexpected membership, or relevant warning | 1 |
| `CRITICAL` | Critical threshold/required service failure/relevant error | 2 |
| `UNKNOWN` | Permission/tool/provider failure, timeout, unsupported state or incomplete check | 3 |

Exit 4 is a command-dispatch/options, configuration, or report export error. File/probe validation errors are returned as `UNKNOWN` (exit 3). Composite severity is CRITICAL > WARNING > UNKNOWN > OK > NOT_APPLICABLE; a critical result never hides the individual unknown results in reports. Interactive shell exit is 0; inspect each command's printed status. Watch returns the last cycle's status; Ctrl+C stops a one-shot watch/follow. In the interactive shell Ctrl+C during a running command exits the shell too.

Each PowerShell/vendor invocation has a configurable deadline (default 30 seconds); a combined health check can take up to five such deadlines. Windows timeout cancellation attempts process-tree termination with inbox taskkill, then kills the parent. Captured stdout is capped at 8 MiB, stderr at 1 MiB; saturation is an incomplete result. Resource sampling defaults to one second. Watch waits the interval **after** each completed cycle, so the cycle period also includes execution time. Walks are capped at 100,000 entries, text scans at 64 MiB, cat/compare at 8 MiB per file, and ZIP extraction at 10,000 entries / 1 GiB total. Hash/copy/walk loops honor interruption but OS-level filesystem calls can still block. File operations have no guaranteed deadline for an unresponsive filesystem.

Reports are explicit writes, use unique filenames, and never automatically prune existing files. `report prune --dry-run` previews removal of matching generated report names older than the configured retention based on file modification time; actual pruning confirms unless `--yes` is supplied. Zero retention days disables pruning. HTML is self-contained and escapes collected content. Reports can contain sensitive account, process, network, registry, file path, and log data; save them to an access-controlled directory. Go file modes do not substitute for Windows ACL policy.

## Build and test

Go 1.23 or newer is required by the source module; the supplied release was built with Go 1.26.3. Build on a development machine, then transfer the executable and documents offline. No compilation is needed on the target server.

```powershell
# Windows developer machine; creates release\aslunix.exe and portable ZIP
.\scripts\build.ps1
# Pure Go unit/integration tests; no live Windows configuration changes
 go test ./...
 go vet ./...
```

```sh
# Cross-compile on macOS/Linux; no C compiler or network modules needed
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags '-s -w' -o release/aslunix.exe .
go test ./...
go vet ./...
```

Run `scripts/Windows-Smoke.ps1` on an approved Windows test machine. It checks collector JSON/status metadata and records failures; it does not assert that the machine is healthy. See [validation record and checklist](VALIDATION.md), [command reference](COMMANDS.md), and [integration notes](INTEGRATIONS.md).


## Icon and taskbar shortcut

The x64 executable embeds a multi-resolution icon (16, 20, 24, 32, 40, 48, 64, 128 and 256 px), version information, and an asInvoker manifest. It continues to be a console application. No elevation is requested by the manifest.

After extracting the portable package into a stable location, run `scripts/Create-Shortcut.ps1`. It writes `AslUnix.lnk` beside the executable and sets its icon to `aslunix.exe,0`. Pin that shortcut through the Windows UI. The script does not install anything or change taskbar settings. Moving the portable folder breaks absolute shortcut targets; recreate the shortcut after moving it. Existing shortcuts are not overwritten.

An independently launched visible legacy console also receives the embedded icon and AslUnix title. Existing shared shells are left alone. Windows Terminal owns its visible window and may keep its own taskbar icon; AslUnix cannot change that through the pseudoconsole window. See [Microsoft's GetConsoleWindow documentation](https://learn.microsoft.com/en-us/windows/console/getconsolewindow).

Source artwork is `assets/aslunix.svg`. PNG/ICO regeneration uses `scripts/make-icon.py` with optional Pillow on a development machine. Ordinary Go builds use checked-in `aslunix_windows_amd64.syso` and need no icon compiler. After changing icon/version/manifest, regenerate the resource with `scripts/build-resources.ps1` (LLVM or Windows SDK resource tools), then build normally. On macOS/Linux with LLVM, run from the repository root:

```sh
llvm-rc /no-preprocess /FO /tmp/aslunix.res assets/aslunix.rc
llvm-cvtres /MACHINE:X64 /TIMESTAMP:0 /OUT:aslunix_windows_amd64.syso /tmp/aslunix.res
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags '-s -w' -o release/aslunix.exe .
```

The resource object is specific to Windows amd64; a future architecture needs its own matching object. Icon loading uses [LoadImageW](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-loadimagew) and the bounded [WM_SETICON](https://learn.microsoft.com/en-us/windows/win32/winmsg/wm-seticon) message. Windows display, shortcut creation and pinning still require real Windows validation.
