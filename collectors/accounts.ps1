$data=@(Get-CimInstance Win32_UserAccount -Filter 'LocalAccount=True' | Select-Object Name,Domain,SID,Disabled,Lockout,PasswordRequired,PasswordExpires,Description,Status)
$limitations+= 'Local SAM inventory only; domain controllers do not provide a normal local account database'
