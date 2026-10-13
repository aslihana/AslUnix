$data=@(Get-CimInstance Win32_Service | Sort-Object Name | Select-Object Name,DisplayName,State,StartMode,PathName,ProcessId,StartName)
