$id=[Security.Principal.WindowsIdentity]::GetCurrent();$principal=New-Object Security.Principal.WindowsPrincipal($id)
$groups=@($id.Groups|ForEach-Object {$name=$null;try{$name=$_.Translate([Security.Principal.NTAccount]).Value}catch{};[pscustomobject]@{SID=$_.Value;Name=$name}})
$privileges=(& "$env:SystemRoot\System32\whoami.exe" /priv /fo csv 2>&1|Out-String);if($LASTEXITCODE -ne 0){throw 'whoami privilege inspection failed'}
$data=@([pscustomobject]@{User=$id.Name;SID=$id.User.Value;Elevated=$principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator);Groups=$groups;PrivilegesRaw=$privileges})
