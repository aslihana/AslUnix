$winhttp=(& "$env:SystemRoot\System32\netsh.exe" winhttp show proxy 2>&1|Out-String);if($LASTEXITCODE -ne 0){throw 'netsh proxy inspection failed'}
$user=Get-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings'
$data=@([pscustomobject]@{WinHTTP=$winhttp;UserProxyEnabled=$user.ProxyEnable;UserProxy=$user.ProxyServer;AutoConfigURL=$user.AutoConfigURL;EnvironmentHTTP=$env:HTTP_PROXY;EnvironmentHTTPS=$env:HTTPS_PROXY;EnvironmentNoProxy=$env:NO_PROXY})
