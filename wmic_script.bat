cmd.exe /c wmic cpu get name
cmd.exe /c MEMPHYSICAL get MaxCapacity
cmd.exe /c wmic baseboard get product
cmd.exe /c wmic bios get SMBIOSBIOSVersion
cmd.exe /c wmic path win_32_VideoController get name
cmd.exe /c wmic path win_32_VideoController get DriveVersion
cmd.exe /c wmic path win_32_VideoController get VideoDescrtiption
cmd.exe /c wmic OS get Caption, OSArchitecture, Version
cmd.exe /c wmic DISKDRIVE get Caption

pause 
