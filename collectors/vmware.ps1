if(-not (Get-Module -ListAvailable VMware.VimAutomation.Core)){throw 'VMware PowerCLI missing; install separately under vendor licensing and approved offline change procedure'}
Import-Module VMware.VimAutomation.Core
if(-not $p.server){throw 'Use vmware --server HOST; credentials are prompted separately by PowerCLI, never accepted in command history'}
# No credential persistence or certificate validation bypass. Uses current Windows credentials where accepted.
$connection=Connect-VIServer -Server ([string]$p.server) -ErrorAction Stop
try{$data=@([pscustomobject]@{VMs=@(Get-VM -Server $connection|Select-Object Name,PowerState,NumCpu,MemoryGB);Hosts=@(Get-VMHost -Server $connection|Select-Object Name,ConnectionState,PowerState,Version);Datastores=@(Get-Datastore -Server $connection|Select-Object Name,CapacityGB,FreeSpaceGB);Networks=@(Get-VirtualPortGroup -Server $connection|Select-Object Name,VLanId);Snapshots=@(Get-VM -Server $connection|Get-Snapshot|Select-Object Name,VM,Created,SizeGB)})}finally{Disconnect-VIServer -Server $connection -Confirm:$false}
$limitations+='Remote inventory requires authorized server access and current Windows authentication; noninteractive credential prompts are unsupported'
