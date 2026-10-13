$group=Get-CimInstance Win32_Group -Filter "LocalAccount=True AND SID='S-1-5-32-544'"
if($null -eq $group){throw 'Local Administrators SID not found; this collector targets member servers and PCs, not domain controller privileged groups'}
$members=@(Get-CimAssociatedInstance -InputObject $group -Association Win32_GroupUser)
$data=@($members | ForEach-Object {
 $kind='Unresolved';if($_.CimClass.CimClassName -eq 'Win32_Group'){$kind='Group'}elseif($_.CimClass.CimClassName -eq 'Win32_UserAccount'){$kind='User'}
 [pscustomobject]@{Name=($_.Domain+'\'+$_.Name);SID=$_.SID;Kind=$kind;Membership='direct';Resolution='CIM';ParentSID='S-1-5-32-544'}
})
if($p.expand_domain){
 foreach($member in $members){
  if($member.CimClass.CimClassName -eq 'Win32_Group' -and -not $member.LocalAccount){
   try {
    Import-Module ActiveDirectory -ErrorAction Stop
    $resolved=@(Get-ADGroupMember -Identity $member.SID -Recursive -ErrorAction Stop)
    $data+=@($resolved | ForEach-Object {[pscustomobject]@{Name=$_.Name;SID=[string]$_.SID;Kind=$_.objectClass;Membership='resolved_domain';Resolution='AD recursive';ParentSID=$member.SID}})
   } catch {$limitations+=('INCOMPLETE: Unresolved domain group '+$member.SID+': '+$_.Exception.Message)}
  }
 }
}
$limitations+= 'CIM membership resolution can require domain access; unresolved SIDs or provider failure make this inspection incomplete. This does not enumerate all privileged identities or delegated rights.'
