# Command reference

Use `aslunix help` for the complete syntax, embedded in the binary and available offline. Global flags (`--config FILE`, `--json`, `--color`, `--no-color`) precede the command. Windows PowerShell/cmd already parse one-shot quoting; interactive AslUnix has its own tokenizer. No GNU compatibility, pipelines, shell escapes, subprocess execution, environment-variable expansion or tilde expansion is promised.

## Health and policy

`health` runs **disks, resources, services, admins, raid** in that order. It omits event history, network probing, WSUS, IIS, domain discovery and application endpoint checks; those are explicitly selected commands. No monitoring state is stored or installed.

`disks` / `df` query fixed Win32_LogicalDisk volumes, using total/free bytes. Displayed capacity/free values are GiB and free percent. Both percentage and absolute free-space thresholds apply; equality triggers the threshold. Zero/invalid capacity is UNKNOWN. Mount-point-only volumes not represented by Win32_LogicalDisk, thin-provisioning backing capacity, VMware datastore capacity and RAID physical capacity are not inferred.

`resources` samples system CPU over the configured interval, returns physical memory and uptime, and sorts top processes by interval CPU and working set. Process CPU is divided by logical processor count. New processes can lack a CPU delta. TotalCPUSeconds is cumulative and explicitly separate. Protected processes may hide details.

`services` gives name/display name/state/start mode/executable path/process/account. Configured `required_services` match case-insensitive service **names**, and every required name must exist and be Running. No wildcard/startup/autorepair logic exists; a missing/stopped required service is CRITICAL. Automatic services outside the policy are inventory only.

`events` defaults to the last 24 hours of System/Application level 1/2/3, max 100 records. Options: `--hours N`, `--limit N` (1..10000), `--id N,N`, `--log NAME`, `--since RFC3339`, `--until RFC3339`. Example: `events --since 2026-10-13T08:00:00+03:00 --until 2026-10-13T12:00:00+03:00 --id 7,11`. Time filters run inside Get-WinEvent, not after unbounded retrieval. No matching events is OK; query failures are UNKNOWN. Error/critical logged events yield CRITICAL, warnings WARNING; these refer to historical events, not proof of a present fault. Event sources/custom logs need explicit selection and rights.

`watch disks,resources --interval 30 --count 5` repeats selected read-only collectors with 30 seconds after each completed cycle. Defaults to health, configured interval, and infinite count. Accepted count/interval range is 1..86400. Results can repeat in streamed JSON array documents with `--json`; watch JSON is not one enclosing document. Collectors needing path/server arguments cannot currently be watched. Ctrl+C stops; a normal exit returns the last cycle's result.

`report --out DIR --checks disks,services,events --html` writes unique text/JSON/HTML files. Default selection matches health. Options accepted: `--out`, `--checks`, `--html`, `--dry-run`. Per-check options are not forwarded, so event reports use defaults. Prune is separate: `report prune --out DIR --dry-run`, then `report prune --out DIR` to confirm deletion. `--yes` bypasses confirmation explicitly. Prune targets matching AslUnix filenames only, skips symlinks/subdirectories, and checks mtime against retention days. It does not run health checks.

## Accounts and access

`accounts` lists local SAM names, domains, SIDs, disabled/lockout/password-required/password-expiry flags and available description/status. It does not include domain account inventory, password age, or every authentication route.

`admins` resolves the local group **SID S-1-5-32-544**, then returns direct member SIDs and kinds via Win32_GroupUser. Direct rows are marked `Membership=direct`. Domain groups are not silently expanded. `--expand-domain` attempts recursive AD member resolution with the installed ActiveDirectory module; returned leaf rows are `resolved_domain` and identify the parent SID. Failed/unavailable domain expansion is `UNKNOWN` with an `INCOMPLETE:` limitation. Direct membership can still be useful offline; failed Windows name/SID resolution is reported, not hidden. Domain controllers are outside scope. This is not an enumeration of every privileged identity, nested local group or delegated permission.

`approved_admin_sids` compares **direct** SIDs, flags extras as WARNING, and does not require all baseline SIDs to be present. An empty baseline means no approval comparison. Resolved domain users are separate from direct policy rows.

`permissions "C:\Program Files\DCS"` returns the ACL owner, DACL entries, inheritance/protection and SDDL. It does not calculate effective access, inspect share ACLs or include audit/SACL rights. `identity` returns current user SID, token group SIDs/names, elevated administrator role and raw `whoami /priv` output. Localized privileges output is retained.

## Network

`network` includes adapters/status/MAC/link speed, addresses, gateways, DNS servers, routes, ARP/neighbors, TCP connections/listeners and UDP endpoints with PIDs and available process names. Each section fails independently; missing sections make the result UNKNOWN. Missing owner names can occur for ended/protected processes. UDP endpoints are bindings, not remote connections.

`ping HOST` requests four probes with 1000 ms waits. `traceroute HOST` uses tracert, numeric output, 15 hops and 1000 ms waits, bounded by the general timeout; raw localized output remains. It does not interpret every hop or prove reachability from tracert exit status. A failed executable/probe is UNKNOWN.

`dns NAME [--server IP[:PORT]]` returns A/AAAA through the system resolver or a chosen numeric IPv4/IPv6 server using Go's resolver. IPv6 with a port must be bracketed. No MX/TXT/AXFR support is present. `tcp HOST PORT` makes a handshake; it does not send application data. Port range is 1..65535. Failed probes/timeouts are UNKNOWN.

`http URL` sends GET, does not follow redirects, reports status/headers/latency and closes without storing a body. >=500 is CRITICAL; 400..499 WARNING; other successful responses OK. HTTPS uses system trust. Embedded URL credentials are rejected. Requests use environment proxy settings; query strings can be retained in diagnostics/history, so do not embed secrets. `proxy` separately inventories WinHTTP raw settings, current-user WinINET settings and environment proxy variables.

`tls HOST [PORT]` (443 default) retrieves certificate chain, version/cipher, subject/issuer/names/serial/date, then verifies hostname, date and system trust. Invalid trust/name/date is CRITICAL, expiry within 30 days WARNING. Handshake/permission/network failures are UNKNOWN. There is no online revocation checking or application request, and only TLS 1.2+ is offered.

`time` returns local time/timezone and w32tm status, but intentionally reports UNKNOWN for synchronization quality because localized status is not parsed against a quality policy. `domain` returns workgroup membership as NOT_APPLICABLE, otherwise explicitly runs nltest DC discovery, not a secure-channel repair/test. `wsus` inventories policy or NOT_APPLICABLE when absent; `--test` additionally tests WSUS TCP connectivity. TCP reachability does not prove synchronization/compliance. `capture` requires separately installed TShark; see integrations.

## Files

`pwd`, `cd PATH`, `ls [PATH]`: current directory, explicit directory change and directory listing (no recursive ls/options/glob expansion). `cd` affects subsequent interactive commands only. `cat FILE` reads at most 8 MiB. `head FILE -n 20` / `tail FILE -n 20` default to 10 lines; line count must be 1..100000. Scanners cap individual lines at 1 MiB and scanned text at 64 MiB. Text is treated as UTF-8/byte strings; UTF-16 log decoding is not implemented. Terminal control characters are sanitized in text display; JSON retains raw values.

`grep TEXT FILE` is case-sensitive literal substring search and returns matching lines with 1-based numbers, no regex switches. `find PATH [NAME_GLOB]` recursively matches base names using Go filepath glob syntax; `du PATH` sums logical file lengths (hard links count per path, not actual allocated usage). Both skip symlinks, cap 100k entries and report errors rather than silently omitting inaccessible data.

`cp SOURCE DEST [--dry-run] [--yes]` supports regular files, rejects source/destination symlinks and same-file aliases, stages copies, and confirms replacing an existing regular file. Copying an existing file uses the OS rename operation after staging (atomic replacement is platform dependent); no owner/ACL preservation is promised. Destination parent directories must exist. New files are created exclusively. `mv SOURCE DEST [--dry-run]` requires same-volume NTFS hard-link support, regular files and a nonexistent destination; it creates an exclusive link then removes the source, rolling back the link on deletion failure. No directory or cross-volume move exists.

`mkdir PATH [--dry-run]` creates one directory, no implicit parent creation. `rm PATH [--recursive] [--dry-run] [--yes]` requires exact confirmation `yes` for deletion; directories require `--recursive`. It refuses the working directory, ancestors and filesystem roots. A symlink itself can be removed, not followed. `--yes` explicitly skips confirmation; previews always leave files unchanged. Literal filenames starting `--` must follow a `--` marker. File flags are documented subsets; there is no general POSIX option grammar.

`follow FILE [--seconds N]` prints newly appended data from the current end, checks each second for truncation/rotation, and retries missing files. Optional duration is 1..3600 seconds. It is a streaming text command and rejects global `--json`; no historical tail is included. `compare FILE1 FILE2` compares normalized text line positions rather than a minimal sequence diff. `sha256 FILE` hashes incrementally.

`archive list FILE.zip` inspects ZIP entries. `archive extract FILE.zip DEST [--dry-run]` refuses existing target files, symlink entries/components, Unix/Windows absolute paths, drive/alternate-stream separators and traversal. It limits total declared size to 1 GiB and 10k entries and verifies decompressed lengths/CRC while copying. The destination must not contain symlink/reparse paths; extraction is not hardened against hostile concurrent directory mutation. Completed files remain on partial failure, but an incomplete file is removed. No password-protected ZIP/7z/tar support, ACL/ownership restoration or implicit overwrite is present.

## Windows/application inventory

`os` reports caption/build/version/architecture/boot/install/computer. `updates` is QFE inventory and does not prove complete patch coverage. `drivers` is signed PnP driver inventory. `tasks` lists task state/actions/last and next run/results. `hardware` returns computer/BIOS/CPU/physical disk inventory; CIM disk status does not establish RAID health.

`registry [PATH]` defaults to `HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion`. Allowed additional paths: `HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate` and `HKLM:\SYSTEM\CurrentControlSet\Control\TimeZoneInformation`. This command never writes registry values.

`iis` inventories websites/bindings/pools/runtime modes and W3SVC/WAS service statuses. It needs WebAdministration and does not test application health. `dotnet` separately returns installed runtime CLI output, .NET Framework v4 release, IIS ASP.NET Core Module DLL file/version presence, and Hosting Bundle/uninstall registrations. It does not infer an operational IIS application from installed runtimes. Missing dotnet CLI is incomplete/UNKNOWN. `signature FILE` returns Authenticode status/message/signer/timestamp, with NotSigned WARNING, HashMismatch/NotTrusted CRITICAL, unsupported status UNKNOWN and Valid OK within the offline trust limitations.

For `raid`, `tools`, `vmware`, `sql` and `capture`, see [INTEGRATIONS.md](INTEGRATIONS.md).
