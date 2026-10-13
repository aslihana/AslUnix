$allowed=@('HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate','HKLM:\SYSTEM\CurrentControlSet\Control\TimeZoneInformation','HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion')
$path=[string]$p.path;if(-not $path){$path=$allowed[2]}
if($path -notin $allowed){throw 'Registry inspection is restricted to documented read-only system paths'}
$data=@(Get-ItemProperty -LiteralPath $path | Select-Object * -ExcludeProperty PSPath,PSParentPath,PSChildName,PSDrive,PSProvider)
