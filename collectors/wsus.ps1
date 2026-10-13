$path='HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate'
if(-not (Test-Path $path)){$data=@();$limitations+='WSUS policy not configured'}else{
 $policy=Get-ItemProperty $path;$au=$null;if(Test-Path ($path+'\AU')){$au=Get-ItemProperty ($path+'\AU')}
 $reachable=$null
 if($p.test -and $policy.WUServer){$uri=[uri]$policy.WUServer;$client=New-Object Net.Sockets.TcpClient;try{$task=$client.ConnectAsync($uri.DnsSafeHost,$uri.Port);if(-not $task.Wait(5000)){throw 'WSUS TCP connection timed out'};$reachable=$client.Connected}catch{$limitations+=('INCOMPLETE: WSUS connectivity: '+$_.Exception.Message);$reachable=$false}finally{$client.Dispose()}}
 $data=@([pscustomobject]@{Server=$policy.WUServer;StatusServer=$policy.WUStatusServer;UseWUServer=$au.UseWUServer;TCPReachable=$reachable})
 $limitations+='Policy inventory/TCP reachability only; does not establish update synchronization or compliance'
}
