# Creates a shortcut beside the portable executable. Does not pin or install anything.
param([string]$Destination)
$ErrorActionPreference='Stop'
$root=Split-Path -Parent $PSScriptRoot
$exe=Join-Path $root 'aslunix.exe'
if(-not (Test-Path -LiteralPath $exe -PathType Leaf)){throw "Executable not found: $exe. Run this script from the extracted portable package."}
if(-not $Destination){$Destination=Join-Path $root 'AslUnix.lnk'}
$Destination=[IO.Path]::GetFullPath($Destination)
if([IO.Path]::GetExtension($Destination) -ne '.lnk'){throw 'Destination must end with .lnk'}
if(Test-Path -LiteralPath $Destination){throw "Shortcut already exists: $Destination. Remove it explicitly before creating a replacement."}
$shell=New-Object -ComObject WScript.Shell
$shortcut=$shell.CreateShortcut($Destination)
$shortcut.TargetPath=$exe
$shortcut.WorkingDirectory=$root
$shortcut.IconLocation=$exe+',0'
$shortcut.Description='AslUnix portable Windows terminal'
$shortcut.WindowStyle=1
$shortcut.Save()
Write-Host "Created: $Destination"
Write-Host 'Right-click the shortcut and choose Pin to taskbar (Show more options if needed).'
Write-Host 'Keep the portable folder in the same location. If moved, recreate the shortcut.'
