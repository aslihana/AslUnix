$out=(& "$env:SystemRoot\System32\w32tm.exe" /query /status 2>&1|Out-String);if($LASTEXITCODE -ne 0){throw $out};$data=@([pscustomobject]@{LocalTime=(Get-Date).ToString('o');TimeZone=[string](Get-TimeZone).Id;W32Time=$out})
$limitations+='INCOMPLETE: w32tm status is raw localized output; synchronization quality is not automatically classified'
