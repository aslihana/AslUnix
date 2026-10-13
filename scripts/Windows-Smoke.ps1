param([string]$Exe=(Join-Path (Split-Path -Parent $PSScriptRoot) 'aslunix.exe'),[string]$OutputDirectory=(Join-Path $env:TEMP ('AslUnix-Smoke-'+(Get-Date -Format 'yyyyMMdd-HHmmss'))))
$ErrorActionPreference='Stop'
if(-not (Test-Path -LiteralPath $Exe)){throw "Executable not found: $Exe (pass -Exe for a source checkout)"}
New-Item -ItemType Directory -Path $OutputDirectory | Out-Null
# On a source checkout, syntax-check collector files with the real Windows parser.
$collectorDirectory=Join-Path (Split-Path -Parent $PSScriptRoot) 'collectors'
if(Test-Path -LiteralPath $collectorDirectory){
 foreach($file in (Get-ChildItem -LiteralPath $collectorDirectory -Filter '*.ps1')) {
  $tokens=$null;$parseErrors=$null
  $null=[System.Management.Automation.Language.Parser]::ParseFile($file.FullName,[ref]$tokens,[ref]$parseErrors)
  if($parseErrors.Count -gt 0){throw ("Collector syntax errors in "+$file.Name+": "+($parseErrors|Out-String))}
 }
}
$states=@('OK','WARNING','CRITICAL','UNKNOWN','NOT_APPLICABLE')
$records=@()
foreach($name in @('disks','resources','services','accounts','admins','identity','network','os','events')) {
 $output=& $Exe --json $name 2>&1 | Out-String
 $code=$LASTEXITCODE
 $output | Set-Content -LiteralPath (Join-Path $OutputDirectory ($name+'.json')) -Encoding UTF8
 try {
  $data=@($output|ConvertFrom-Json)
  if($data.Count -ne 1 -or $data[0].check -ne $name -or $data[0].status -notin $states -or -not $data[0].checked_at -or -not $data[0].source){throw 'Malformed result metadata'}
  $expected=@{OK=0;NOT_APPLICABLE=0;WARNING=1;CRITICAL=2;UNKNOWN=3}[$data[0].status]
  if($code -ne $expected){throw "Exit code $code does not match status"}
  $records+=[pscustomobject]@{Command=$name;Exit=$code;Status=$data[0].status;Schema='PASS';Summary=$data[0].summary}
 } catch {$records+=[pscustomobject]@{Command=$name;Exit=$code;Status='UNKNOWN';Schema='FAIL';Summary=$_.Exception.Message}}
}
# Offline file path/quoted-space sanity check, limited to this test's fresh directory.
$spacePath=Join-Path $OutputDirectory 'path with spaces.txt'
[IO.File]::WriteAllText($spacePath,'AslUnix smoke fixture')
& $Exe sha256 $spacePath
& $Exe rm $spacePath --dry-run
if(-not (Test-Path -LiteralPath $spacePath)){throw 'rm dry-run modified fixture'}
$reportDir=Join-Path $OutputDirectory 'reports'
& $Exe report --out $reportDir --checks disks,resources,services --html
if((Get-ChildItem -LiteralPath $reportDir -File).Count -ne 3){throw 'Report files missing'}
$records|ConvertTo-Json -Depth 5|Set-Content -LiteralPath (Join-Path $OutputDirectory 'smoke-summary.json') -Encoding UTF8
$records|Format-Table -AutoSize
Write-Host "Results: $OutputDirectory"
Write-Host 'Review UNKNOWN results individually. A PASS schema does not validate Windows data accuracy or machine health.'
if(@($records|Where-Object {$_.Schema -eq 'FAIL'}).Count -gt 0){exit 1}
