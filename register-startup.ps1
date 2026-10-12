[CmdletBinding()]
param(
    [string]$Name = "ghqwatch",
    [switch]$Unregister,   # remove from startup
    [switch]$Run           # launch after registration
)

$ErrorActionPreference = "Stop"

$startup = [Environment]::GetFolderPath("Startup")
$src     = Join-Path $PSScriptRoot "$Name.exe"
$dest    = Join-Path $startup "$Name.exe"

# Stop running processes first, since they cannot be overwritten or deleted while running
function Stop-Running {
    Get-Process -Name $Name -ErrorAction SilentlyContinue | ForEach-Object {
        Write-Host "Stopping running $Name (PID $($_.Id))"
        $_ | Stop-Process -Force
        $_ | Wait-Process -Timeout 5 -ErrorAction SilentlyContinue
    }
}

if ($Unregister) {
    Stop-Running
    if (Test-Path -Path $dest -PathType Leaf) {
        Remove-Item -Path $dest -Force
        Write-Host "Removed: $dest" -ForegroundColor Green
    }
    else {
        Write-Host "Not registered: $dest"
    }
    return
}

if (-not (Test-Path -Path $src -PathType Leaf)) {
    Write-Error "$src was not found. Please run build.ps1 first."
}

Stop-Running
Copy-Item -Path $src -Destination $dest -Force
Write-Host "Registered: $dest" -ForegroundColor Green

if ($Run) {
    Start-Process -FilePath $dest
}