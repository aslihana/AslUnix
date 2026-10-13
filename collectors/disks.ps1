$data = @(Get-CimInstance Win32_LogicalDisk -Filter 'DriveType=3' | ForEach-Object {
 [pscustomobject]@{Volume=$_.DeviceID;Label=$_.VolumeName;FileSystem=$_.FileSystem;SizeBytes=[double]$_.Size;FreeBytes=[double]$_.FreeSpace}
})
