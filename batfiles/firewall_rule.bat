powershell.exe /c "Import-Module NetSecurity"
powershell.exe /c "New-NetFirewallRule -DisplayName \"NIR\" -Direction Outbound -Action Block -Program \"C:\Windows\notepad.exe\""