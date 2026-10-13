function Section($name,$block){try{ @(& $block) }catch{$script:limitations+=('INCOMPLETE: '+$name+': '+$_.Exception.Message);@()}}
$adapters=Section 'adapters' {Get-NetAdapter|Select-Object Name,InterfaceDescription,Status,LinkSpeed,MacAddress,ifIndex}
$addresses=Section 'addresses' {Get-NetIPConfiguration|ForEach-Object {[pscustomobject]@{Interface=$_.InterfaceAlias;IPv4=@($_.IPv4Address.IPAddress);IPv6=@($_.IPv6Address.IPAddress);IPv4Gateway=@($_.IPv4DefaultGateway.NextHop);IPv6Gateway=@($_.IPv6DefaultGateway.NextHop);DNS=@($_.DNSServer.ServerAddresses)}}}
$routes=Section 'routes' {Get-NetRoute|Select-Object DestinationPrefix,NextHop,InterfaceIndex,RouteMetric,State}
$neighbors=Section 'neighbors' {Get-NetNeighbor|Select-Object IPAddress,LinkLayerAddress,State,InterfaceIndex}
$tcp=Section 'tcp' {Get-NetTCPConnection|ForEach-Object {$name=$null;try{$name=(Get-Process -Id $_.OwningProcess -ErrorAction Stop).ProcessName}catch{};[pscustomobject]@{LocalAddress=$_.LocalAddress;LocalPort=$_.LocalPort;RemoteAddress=$_.RemoteAddress;RemotePort=$_.RemotePort;State=[string]$_.State;PID=$_.OwningProcess;Process=$name}}}
$udp=Section 'udp' {Get-NetUDPEndpoint|ForEach-Object {$name=$null;try{$name=(Get-Process -Id $_.OwningProcess -ErrorAction Stop).ProcessName}catch{};[pscustomobject]@{LocalAddress=$_.LocalAddress;LocalPort=$_.LocalPort;PID=$_.OwningProcess;Process=$name}}}
$data=@([pscustomobject]@{Adapters=$adapters;Addresses=$addresses;Routes=$routes;Neighbors=$neighbors;TCP=$tcp;UDP=$udp})
