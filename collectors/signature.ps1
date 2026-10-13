$s=Get-AuthenticodeSignature -LiteralPath ([string]$p.path)
$data=@([pscustomobject]@{Path=$s.Path;Status=[string]$s.Status;Message=$s.StatusMessage;SignerSubject=$s.SignerCertificate.Subject;SignerExpiry=$s.SignerCertificate.NotAfter;TimestampSubject=$s.TimeStamperCertificate.Subject})
if($s.Status -ne 'Valid'){$limitations+=('Signature status '+[string]$s.Status)}
$limitations+='Offline trust evaluation can lack revocation data'
