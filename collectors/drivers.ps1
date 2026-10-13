$data=@(Get-CimInstance Win32_PnPSignedDriver|Select-Object DeviceName,DriverVersion,DriverDate,Manufacturer,IsSigned,InfName)
