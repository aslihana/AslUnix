$acl=Get-Acl -LiteralPath ([string]$p.path)
$data=@([pscustomobject]@{Path=$acl.Path;Owner=$acl.Owner;SDDL=$acl.Sddl;Protected=$acl.AreAccessRulesProtected;Entries=@($acl.Access|Select-Object IdentityReference,FileSystemRights,AccessControlType,IsInherited,InheritanceFlags,PropagationFlags)})
$limitations+='ACL entries are not an effective-access calculation; share permissions and token/group resolution are separate'
