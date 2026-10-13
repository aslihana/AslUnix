$sys=Get-CimInstance Win32_ComputerSystem
if(-not $sys.PartOfDomain){$data=@([pscustomobject]@{PartOfDomain=$false;Domain=$sys.Domain});$limitations+='Workgroup machine: domain connectivity not applicable'}else{
 $out=(& "$env:SystemRoot\System32\nltest.exe" (('/dsgetdc:')+$sys.Domain) 2>&1|Out-String);if($LASTEXITCODE -ne 0){throw $out};$data=@([pscustomobject]@{PartOfDomain=$true;Domain=$sys.Domain;Discovery=$out})
 $limitations+='Domain controller discovery only; does not prove all application authentication or secure-channel health'
}
