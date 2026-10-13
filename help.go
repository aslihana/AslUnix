package main

import (
	"fmt"
	"io"
)

const reference = `AslUnix 0.1 — portable Windows inspection
Usage: aslunix [--config PATH] [--json] [--color|--no-color] [COMMAND ...]
No command enters the interactive aslunix> shell. --version prints the version.
Global flags precede COMMAND. Quoted paths keep backslashes. No shell expansion.

Core checks
  health                         disks/resources/services/admins/raid, sequential
  disks | df                     fixed volumes, threshold assessment
  resources                      sampled CPU/memory, uptime, top processes
  services                       inventory + configured required service names
  accounts                       local SAM account inventory
  admins [--expand-domain]       direct S-1-5-32-544 membership + SID baseline
  permissions PATH               DACL/owner/SDDL, not effective permissions
  identity                       current token, groups, privileges, elevation
  events [--hours N] [--id N,N] [--log NAME] [--limit N]
         [--since RFC3339] [--until RFC3339]   recent warnings/errors
  watch [health|disks,resources,...] [--interval SECONDS] [--count N]
  report [--out DIR] [--checks disks,resources,...] [--html] [--dry-run]
  report prune [--out DIR] [--dry-run] [--yes]  explicit retention cleanup

Network (remote probes occur only when requested)
  network                        adapters, IPs, DNS, gateways, routes, neighbors,
                                 TCP/UDP endpoints and owning PIDs/processes
  ping HOST | traceroute HOST    bounded Windows inbox tools
  dns NAME [--server IP[:PORT]]   A/AAAA lookup (IPv6: [IP]:PORT)
  tcp HOST PORT                  TCP handshake
  http URL                       GET; no redirects, standard HTTPS trust
  tls HOST [PORT]                certificate chain/hostname/trust/expiry
  proxy | time | domain          local settings / explicit domain discovery
  wsus [--test]                  policy; opt-in WSUS TCP connectivity
  capture                        list installed TShark interfaces
  capture --interface ID --seconds N --out FILE.pcapng
                                 explicit bounded packet capture, no overwrite

Files (a supported subset, not GNU/POSIX compatibility)
  pwd | cd PATH | ls [PATH]       ls lists a directory, no glob expansion
  cat FILE | head FILE [-n N] | tail FILE [-n N]
  grep TEXT FILE                 literal, case-sensitive substring with line #
  find PATH [NAME_GLOB] | du PATH symlinks skipped; walks capped at 100k entries
  cp SOURCE DEST [--dry-run] [--yes]  regular files; confirm overwrites
  mv SOURCE DEST [--dry-run]      same-volume NTFS files, unused destination
  mkdir PATH [--dry-run]          one directory, no implicit parent creation
  rm PATH [--recursive] [--dry-run] [--yes]   deletion confirms by default
  follow FILE [--seconds N]      stream from end, detect rotation/truncation
  compare FILE1 FILE2            positional text-line comparison
  sha256 FILE                    streaming SHA-256
  archive list FILE.zip
  archive extract FILE.zip DEST [--dry-run]  no overwrites/traversal/symlinks
Use -- before filenames starting with --. Flags are limited to documented forms.
Archive extraction creates files only; rm refuses cwd, its ancestors and roots.

Windows/application inventory
  os | updates | drivers | tasks | hardware
  registry [PATH]                documented allowlisted registry keys only
  iis | dotnet                   sites/pools/services; runtimes/module/bundles
  signature FILE                 Authenticode inspection
  raid                           PERCCLI /call show all J, read only
  tools                          installed tool discovery
  vmware --server HOST           PowerCLI remote inventory, current credentials
  sql --server INSTANCE [--backup PATH]  integrated auth; metadata/RESTORE HEADERONLY

Shell: Tab completes commands/paths; Up/Down navigate session history on Windows.
Left/Right/Home/End/Delete edit the line. history lists session commands; exit quits.
Redirected input uses line mode. History stays in memory and is capped at 500 lines.
No pipes, command chaining, environment expansion, or arbitrary command execution.

Statuses: OK WARNING CRITICAL UNKNOWN NOT_APPLICABLE
Exit codes: 0 OK/NOT_APPLICABLE, 1 WARNING, 2 CRITICAL, 3 UNKNOWN,
4 usage/configuration/export error. Shell exit: 0. Ctrl+C stops one-shot watch/follow.
UNKNOWN means unavailable/incomplete, never healthy. Reports retain raw diagnostics.
Health does not include every inventory/probe. Read docs/COMMANDS.md for limitations.
`

func writeHelp(out io.Writer, topic string) {
	if topic != "" && !contains(commandList, topic) {
		fmt.Fprintln(out, "Unknown help topic:", safeText(topic))
	}
	fmt.Fprint(out, reference)
}
