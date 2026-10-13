$data=@(Get-CimInstance Win32_QuickFixEngineering|Select-Object HotFixID,Description,InstalledOn,InstalledBy)
$limitations+='QFE inventory does not include every update type or establish patch compliance'
