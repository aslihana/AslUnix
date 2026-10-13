# Developer build; Go must be installed. No third-party modules are downloaded.
$ErrorActionPreference='Stop'
$root=Split-Path -Parent $PSScriptRoot
$stage=Join-Path ([IO.Path]::GetTempPath()) ('AslUnix-Build-'+[guid]::NewGuid().ToString('N'))
$oldGOOS=$env:GOOS;$oldGOARCH=$env:GOARCH;$oldCGO=$env:CGO_ENABLED
Push-Location $root
try {
 $env:CGO_ENABLED='0';$env:GOOS='windows';$env:GOARCH='amd64'
 New-Item -ItemType Directory -Force release | Out-Null
 & go build -trimpath -ldflags '-s -w' -o release/aslunix.exe .
 if($LASTEXITCODE -ne 0){throw 'Go build failed'}
 New-Item -ItemType Directory $stage | Out-Null
 Copy-Item release/aslunix.exe,aslunix.example.json,README.md $stage
 Copy-Item docs (Join-Path $stage 'docs') -Recurse
 New-Item -ItemType Directory (Join-Path $stage 'scripts') | Out-Null
 Copy-Item scripts/Windows-Smoke.ps1,scripts/Create-Shortcut.ps1 (Join-Path $stage 'scripts')
 New-Item -ItemType Directory (Join-Path $stage 'assets') | Out-Null
 Copy-Item assets/aslunix.ico,assets/aslunix.png,assets/aslunix-32.png (Join-Path $stage 'assets')
 $archive=Join-Path $root 'release/AslUnix-0.1.0-windows-amd64.zip'
 Compress-Archive -Path "$stage/*" -DestinationPath $archive -Force
 $hashes=Get-FileHash release/aslunix.exe,$archive -Algorithm SHA256
 $hashes | ForEach-Object {($_.Hash.ToLower()+'  '+[IO.Path]::GetFileName($_.Path))} | Set-Content release/SHA256SUMS.txt -Encoding ASCII
 $hashes | Format-Table
} finally {
 if(Test-Path -LiteralPath $stage){Remove-Item -LiteralPath $stage -Recurse -Force}
 $env:GOOS=$oldGOOS;$env:GOARCH=$oldGOARCH;$env:CGO_ENABLED=$oldCGO
 Pop-Location
}
