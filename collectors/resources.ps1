$os=Get-CimInstance Win32_OperatingSystem
$cpuCount=[Environment]::ProcessorCount
$before=@{}; Get-Process | ForEach-Object {if($null -ne $_.CPU){$before[$_.Id]=[double]$_.CPU}}
$started=Get-Date
# Discard first formatted counter reading to avoid startup values, then sample.
$null=Get-CimInstance Win32_PerfFormattedData_PerfOS_Processor -Filter "Name='_Total'"
Start-Sleep -Seconds ([int]$p.sample_seconds)
$cpu=Get-CimInstance Win32_PerfFormattedData_PerfOS_Processor -Filter "Name='_Total'"
if($null -eq $cpu -or $null -eq $cpu.PercentProcessorTime){throw 'CPU performance sample unavailable'}
$elapsed=((Get-Date)-$started).TotalSeconds
$processes=@(Get-Process | ForEach-Object {
 $sample=$null; if($before.ContainsKey($_.Id) -and $null -ne $_.CPU){$sample=[math]::Max(0,([double]$_.CPU-$before[$_.Id])/$elapsed/$cpuCount*100)}
 [pscustomobject]@{PID=$_.Id;Name=$_.ProcessName;MemoryMiB=[math]::Round($_.WorkingSet64/1MB,2);CPUPercent=$sample;TotalCPUSeconds=$_.CPU}
})
$data=@([pscustomobject]@{CPUPercent=[double]$cpu.PercentProcessorTime;MemoryPercent=(1-[double]$os.FreePhysicalMemory/[double]$os.TotalVisibleMemorySize)*100;TotalMemoryGiB=[double]$os.TotalVisibleMemorySize/1MB;FreeMemoryGiB=[double]$os.FreePhysicalMemory/1MB;UptimeSeconds=((Get-Date)-$os.LastBootUpTime).TotalSeconds;SampleSeconds=$elapsed;TopCPU=@($processes|Sort-Object CPUPercent -Descending|Select-Object -First 10);TopMemory=@($processes|Sort-Object MemoryMiB -Descending|Select-Object -First 10)})
