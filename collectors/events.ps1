$filter=@{LogName=@('System','Application');Level=@(1,2,3);StartTime=(Get-Date).AddHours(-[double]$p.hours)}
if($p.since){$filter.StartTime=[datetime]::Parse([string]$p.since)}
if($p.until){$filter.EndTime=[datetime]::Parse([string]$p.until)}
if($p.id){$filter.Id=@($p.id -split ','|ForEach-Object {[int]$_})}
if($p.log){$filter.LogName=@([string]$p.log)}
try {$data=@(Get-WinEvent -FilterHashtable $filter -MaxEvents ([int]$p.limit) | ForEach-Object {[pscustomobject]@{Time=$_.TimeCreated.ToUniversalTime().ToString('o');ID=$_.Id;Level=$_.Level;Severity=$_.LevelDisplayName;Log=$_.LogName;Provider=$_.ProviderName;Message=$_.Message}})}
catch {if($_.FullyQualifiedErrorId -like 'NoMatchingEventsFound*'){$data=@()}else{throw}}
