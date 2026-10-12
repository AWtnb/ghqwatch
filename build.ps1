[CmdletBinding()]
param(
    [string]$Name = "ghqwatch",
    [switch]$Debug,   # Build with console (for viewing logs)
    [switch]$Run      # Run after build
)

$ErrorActionPreference = "Stop"
Set-Location -Path $PSScriptRoot

$exe = "$Name.exe"

# Check for icon.ico first because go:embed will fail without it
if (-not (Test-Path -Path "icon.ico" -PathType Leaf)) {
    Write-Error "icon.ico was not found."
}

# Initialize go.mod if it does not exist
if (-not (Test-Path -Path "go.mod" -PathType Leaf)) {
    go mod init $Name
}
go mod tidy

# Stop running processes first, otherwise the executable cannot be overwritten
Get-Process -Name $Name -ErrorAction SilentlyContinue | ForEach-Object {
    Write-Host "Stopping running $exe (PID $($_.Id))"
    $_ | Stop-Process -Force
    $_ | Wait-Process -Timeout 5 -ErrorAction SilentlyContinue
}

if ($Debug) {
    $ldflags = ""
}
else {
    $ldflags = "-H=windowsgui -s -w"
}

Write-Host "Building $exe ..."
go build -ldflags $ldflags -o $exe .
if ($LASTEXITCODE -ne 0) {
    Write-Error "go build failed (exit code $LASTEXITCODE)"
    exit $LASTEXITCODE
}
Write-Host "Done: $(Join-Path $PSScriptRoot $exe)" -ForegroundColor Green

if ($Run) {
    Start-Process -FilePath (Join-Path $PSScriptRoot $exe)
}