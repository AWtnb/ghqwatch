[CmdletBinding()]
param(
    [string]$Name = "ghqwatch",
    [switch]$Console,      # Build with a console (do not register or start)
    [switch]$NoRegister,   # Build only. Do not register to startup
    [switch]$Unregister    # Remove from startup (do not build)
)

$ErrorActionPreference = "Stop"
Set-Location -Path $PSScriptRoot

$exe     = "$Name.exe"
$startup = [Environment]::GetFolderPath("Startup")
$dest    = Join-Path $startup $exe

function Stop-Running {
    Get-Process -Name $Name -ErrorAction SilentlyContinue | ForEach-Object {
        Write-Host "Stopping running $exe (PID $($_.Id))"
        $_ | Stop-Process -Force
        $_ | Wait-Process -Timeout 5 -ErrorAction SilentlyContinue
    }
}

# Unregister only
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

# go:embed fails without icon.ico
if (-not (Test-Path -Path "icon.ico" -PathType Leaf)) {
    Write-Error "icon.ico not found."
}

if (-not (Test-Path -Path "go.mod" -PathType Leaf)) {
    go mod init $Name
}
go mod tidy

# Cannot overwrite while running
Stop-Running

$ldflags = if ($Console) { "" } else { "-H=windowsgui" }

Write-Host "Building $exe ..."
go build -ldflags $ldflags -o $exe .
if ($LASTEXITCODE -ne 0) {
    Write-Error "go build failed (exit code $LASTEXITCODE)"
    exit $LASTEXITCODE
}
Write-Host "Built: $(Join-Path $PSScriptRoot $exe)" -ForegroundColor Green

# Do not register if build only or if this is a debug build
if ($NoRegister -or $Console) {
    return
}

Copy-Item -Path (Join-Path $PSScriptRoot $exe) -Destination $dest -Force
Write-Host "Registered: $dest" -ForegroundColor Green
Start-Process -FilePath $dest