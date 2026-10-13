# Optional integrations

AslUnix never downloads vendor software, installs modules, changes hardware, stores credentials, or asserts redistribution rights. Keep approved offline installers separately. Use configuration `tools` keys for full executable paths, for example:

```json
{
  "tools": {
    "perccli": "C:\\Tools\\Dell\\perccli64.exe",
    "tshark": "C:\\Program Files\\Wireshark\\tshark.exe",
    "sysinternals": "C:\\Tools\\Sysinternals\\sigcheck64.exe",
    "sqlcmd": "C:\\Tools\\SQL\\sqlcmd.exe"
  }
}
```

Merge this snippet with the supplied config; do not redistribute the tools without checking vendor terms. Executables are otherwise looked up by known filenames on PATH. Protect that PATH and configured directories against untrusted modification. `tools` reports detected/missing tools and enabling instructions; missing optional tools are UNKNOWN individually.

## Dell PERCCLI

`raid` calls only `/call show all J`. Raw JSON is preserved, including controller, virtual disk, physical disk and rebuild states exposed by the vendor. AslUnix interprets `Controller Status` and `State` fields and requires controller / `VD LIST` / `PD LIST` sections for an OK result. Optimal/Optl/Onln/Online are OK; Rbld/Rebuild/Rebuilding/UGood/GHS/DHS are conservatively WARNING; Dgrd/Degraded/Offln/Offline/Failed/F/UBad/PDgd are CRITICAL; unknown words/absent sections are UNKNOWN. Rebuild progress details remain in raw/data output when provided. These schemas/vocabularies are provisional until tested on actual controllers; physical controller health does not follow from CIM logical disks.

VMs without a physical controller are not automatically labeled NOT_APPLICABLE. No hardware detection currently establishes that safely. No set, start, initialize, clear, delete, import or repair operation exists.

## VMware PowerCLI

`vmware --server vcenter.example.local` checks VMware.VimAutomation.Core, connects explicitly, inventories VMs/hosts/datastores/port groups/snapshots, and disconnects the connection it created. It uses existing Windows authentication where accepted. Because the collector runs noninteractively, endpoints requiring an interactive credential prompt fail UNKNOWN. No credentials can be entered in AslUnix commands; no insecure certificate override is set. This first adapter has not been validated against PowerCLI or vCenter. All inventory requires intentional network access and sufficient permissions.

## SQL Server

`sql --server HOST\INSTANCE` uses the SqlServer PowerShell module with integrated authentication, a 5-second connection timeout and 15-second query timeout, additionally bounded by the collector deadline. It queries `sys.databases` and `sys.master_files` for status and physical locations. `sql --server HOST\INSTANCE --backup "D:\Backups\app.bak"` sends only `RESTORE HEADERONLY` and `RESTORE FILELISTONLY`. The path is escaped as a SQL string and refers to the server's filesystem, not the AslUnix client's. No database modification/restore is performed. Backup headers/file lists do not verify media integrity or restore success. Authentication/module/version errors remain UNKNOWN. `sqlcmd` is detected by `tools` but not invoked in v0.1.

## TShark / Windows capture

`capture` lists TShark interfaces through `-D`. An explicit `capture --interface 1 --seconds 10 --out sample.pcapng` writes a bounded capture using `-a duration:N`. Existing files are never deliberately overwritten. Capture stops at the earlier of duration and configured deadline; a deadline stop is UNKNOWN and can leave a partial output. TShark/Npcap and suitable rights must be installed already. Npcap installation, adapter permissions and the capture invocation need Windows validation. No netsh/pktmon capture fallback is implemented in this release.

## Sysinternals

`tools` detects Sigcheck only. Native `signature` uses PowerShell Authenticode. Sysinternals command adapters beyond discovery are deferred, avoiding license acceptance or opaque changes from invoking vendor tools automatically.
