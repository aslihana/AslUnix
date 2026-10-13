# Validation record — 13 October 2026

## Executed here

Development host: macOS / arm64. Toolchain: Go 1.26.3. No Windows VM, PowerShell host, Dell PERC, SQL Server or VMware endpoint was available.

* `go test ./...`: passes. Fixtures test thresholds (including equality/absolute GiB), missing CPU values/permissions/timeouts, required services, direct administrator baseline scoping, RAID unknown vocabulary/missing sections, command parsing, and Windows quoted paths.
* Filesystem tests execute on temporary files: rm confirmation/preview/protected directories, copy overwrite behavior, ZIP traversal/preview/no overwrite/symlink-parent rejection, copy cancellation, SHA/result infrastructure.
* Local loopback integration tests execute HTTP status/redirect handling, normal HTTPS rejection of an untrusted certificate, TLS inspection with preserved untrusted chain and CRITICAL validation, and TCP connections/port validation.
* Tests exercise HTML escaping, raw diagnostic preservation in JSON, retention filename/age scoping, config rejection and bounded output buffers/process deadlines.
* `go vet ./...`: passes. Windows amd64 compilation succeeds, including console/process code; this validates types/build only, not runtime behavior.
* Host CLI smoke checks exercise help, the redirected shell, quoted paths, SHA-256 and report generation with explicit UNKNOWN Windows collectors on macOS. No mock data are presented as live diagnostics.

Loopback tests require permission to bind local ports. The development sandbox's existing Go cache was stale, so verification uses a separate temporary `GOCACHE`; this is a host workaround, not a target prerequisite. No Go dependencies were fetched.

## Not executed on Windows

**None of the Windows collectors or vendor integrations has been runtime-tested on Windows in this delivery.** Cross-compilation and Go fixtures do not establish data accuracy, permissions behavior, CIM completeness or script syntax/runtime on Windows PowerShell 5.1. The release is an unsigned portable executable; obtain organizational signing/allowlisting through your normal process if required. AslUnix does not change execution policy to run itself.

Use `scripts/Windows-Smoke.ps1 -Exe .\release\aslunix.exe` from the source checkout, or the bundled script's default executable path from the portable release. It creates fresh test files/reports in TEMP, executes read-only collectors and rm preview, and records status/schema/exit-code results for review. It contacts no vendor endpoints. Do not interpret a schema PASS as proof of healthy/accurate Windows data.

## Required target validation

| Area | Concrete validation |
|---|---|
| OS matrix | Run on x64 Windows 10/11 and each intended Server release/Server Core image; record build/version |
| Console | CMD, Windows PowerShell console and Windows Terminal: prompt, quoted trailing backslashes, Tab path completion, arrows, Ctrl+C, redirected input, optional colors, non-English names |
| PowerShell | Parse and execute all scripts with inbox Windows PowerShell 5.1; restricted/constrained-language policy must return UNKNOWN without policy changes |
| Disks | Compare bytes/free percent to local authoritative values; force threshold config only, not disk usage; validate offline/missing volume/provider cases |
| Resources | Compare intervals/top-process sampling and memory/uptime against Task Manager/counters; inaccessible/protected process behavior |
| Services | Existing Running/Stopped and deliberately nonexistent required names; localized display names must not affect matching |
| Accounts/admins | Normal/elevated, localized Administrators name, direct domain groups, orphaned SIDs, no domain connectivity, optional AD module/recursion, baseline SID extras; domain controllers UNKNOWN |
| Events | Empty logs, access denied, bounded results, RFC3339 timezone range, event IDs/log names and provider errors |
| Network | Multiple/NIC-disabled/v6 networks, DNS chosen server and timeouts, owning process races, proxy layers, w32tm failure, workgroup and unavailable domain |
| IIS/.NET | IIS absent/present, stopped services, ASP.NET Core Module present/absent and Hosting Bundle separately from runtimes |
| Files | NTFS/ACL behavior, copy replacement and same-volume mv, deletion default/preview, files over limits, ZIP traversal/reparse points, log rotation/truncation and UTF-8 limitations |
| Reports | Exit status, timestamps, complete raw JSON, offline HTML, failed/disk-full export, explicit retention cleanup in an isolated test directory |
| Deadlines | Slow collector/vendor process, child-process cancellation, output caps; confirm no lingering processes; inaccessible providers never OK |
| Dell RAID | Real PERCCLI versions/controllers: command JSON schema, controller/VD/PD sections, optimal/degraded/offline/rebuild/unconfigured states and progress |
| VMware | Approved vCenter/ESXi, compatible PowerCLI, Windows authentication, certificate failure, permissions, snapshot/network coverage and disconnect |
| SQL | Compatible SqlServer module, integrated auth, instance connection failure, database/file inventory, HEADERONLY/FILELISTONLY permissions/path escaping; no restore/repair |
| TShark | Installed Npcap/rights/interface selection, explicit duration/deadline, output reservation, partial files and no accidental overwrite |

Deferred: complete privileged-identity graph, effective permissions, domain-controller coverage, GNU/POSIX compatibility, cross-volume/directory mv, recursive cp, other archive formats, TLS revocation, credential management, SSH/remote shell, automatic repairs, netsh/pktmon capture fallback, Sysinternals command execution beyond discovery, vendor auto installation and continuous/background monitoring.


## Icon update

The Windows x64 executable now contains a multi-resolution RT_ICON/RT_GROUP_ICON resource, version metadata and an asInvoker manifest. Resource contents are checked against the source ICO after the cross-build. The shortcut and legacy console branding code compile but have not run on Windows here. Validate Explorer/file icons, shortcut icons, Windows taskbar pinning, high-DPI sizes, shared-console preservation and Windows Terminal host behavior on a Windows test machine. No new CLI command was added.
