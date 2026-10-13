if(-not (Get-Module -ListAvailable WebAdministration)){throw 'IIS WebAdministration module missing; install IIS management scripting tools through your approved process'}
Import-Module WebAdministration
$data=@([pscustomobject]@{Sites=@(Get-Website|Select-Object Name,State,PhysicalPath,@{n='Bindings';e={@($_.Bindings.Collection.bindingInformation)}});Pools=@(Get-ChildItem IIS:\AppPools|Select-Object Name,State,managedRuntimeVersion,managedPipelineMode);Services=@(Get-Service W3SVC,WAS|Select-Object Name,@{n='Status';e={[string]$_.Status}})})
$limitations+='Inventory only; does not probe application responses'
