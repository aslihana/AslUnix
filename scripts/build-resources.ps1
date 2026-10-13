# Optional: regenerate the checked-in Windows x64 .syso after changing icon/version.
# Needs either LLVM (llvm-rc + llvm-cvtres) or Windows SDK tools (rc + cvtres).
# Regular scripts/build.ps1 uses the checked-in object and needs only Go.
$ErrorActionPreference='Stop'
$root=Split-Path -Parent $PSScriptRoot
$temp=Join-Path ([IO.Path]::GetTempPath()) ('AslUnix-Resources-'+[guid]::NewGuid().ToString('N')+'.res')
Push-Location $root
try {
 $resourceCompiler=Get-Command llvm-rc -ErrorAction SilentlyContinue
 $converter=Get-Command llvm-cvtres -ErrorAction SilentlyContinue
 if($resourceCompiler -and $converter){
  & $resourceCompiler.Source /no-preprocess /FO $temp assets/aslunix.rc
 }else{
  $resourceCompiler=Get-Command rc -ErrorAction SilentlyContinue
  $converter=Get-Command cvtres -ErrorAction SilentlyContinue
  if(-not $resourceCompiler -or -not $converter){throw 'Use an LLVM or Windows SDK developer environment with resource tools on PATH. No tools will be downloaded.'}
  & $resourceCompiler.Source /nologo /fo $temp assets/aslunix.rc
 }
 if($LASTEXITCODE -ne 0){throw 'Resource compilation failed'}
 & $converter.Source /MACHINE:X64 /OUT:aslunix_windows_amd64.syso $temp
 if($LASTEXITCODE -ne 0){throw 'Resource conversion failed'}
 Write-Host 'Updated aslunix_windows_amd64.syso. Run scripts/build.ps1 to rebuild the release.'
}finally{if(Test-Path -LiteralPath $temp){Remove-Item -LiteralPath $temp};Pop-Location}
