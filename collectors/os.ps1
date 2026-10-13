$data=@(Get-CimInstance Win32_OperatingSystem|Select-Object Caption,Version,BuildNumber,OSArchitecture,LastBootUpTime,InstallDate,CSName)
