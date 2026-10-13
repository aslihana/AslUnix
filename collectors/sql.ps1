if(-not (Get-Module -ListAvailable SqlServer)){throw 'SQL Server PowerShell module missing; install separately through approved process'}
Import-Module SqlServer
if(-not $p.server){throw 'Use sql --server INSTANCE; integrated authentication only'}
$query='SELECT name,state_desc,recovery_model_desc FROM sys.databases; SELECT DB_NAME(database_id) AS database_name,name,type_desc,physical_name,size FROM sys.master_files;'
if($p.backup){$escaped=([string]$p.backup).Replace("'","''");$query="RESTORE HEADERONLY FROM DISK = N'$escaped'; RESTORE FILELISTONLY FROM DISK = N'$escaped';"}
$data=@(Invoke-Sqlcmd -ServerInstance ([string]$p.server) -Query $query -QueryTimeout 15 -ConnectionTimeout 5 -ErrorAction Stop|Select-Object * -ExcludeProperty RowError,RowState,Table,ItemArray,HasErrors)
$limitations+='SQL metadata only, no restore operation; backup paths refer to the SQL server filesystem; no backup validity guarantee'
