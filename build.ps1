[CmdletBinding()]
param(
    [string]$Name = "ghqwatch",   # match the module name in go.mod
    [switch]$Console,             # build with console enabled (do not register or launch)
    [switch]$NoRegister,          # install only; do not register on startup
    [switch]$Unregister           # remove the startup shortcut (do not build)
)

$ErrorActionPreference = "Stop"
Set-Location -Path $PSScriptRoot

$exe     = "$Name.exe"
$startup = [Environment]::GetFolderPath("Startup")
$lnk     = Join-Path $startup "$Name.lnk"

$bin = go env GOBIN
if (-not $bin) { $bin = Join-Path (go env GOPATH) "bin" }
$installed = Join-Path $bin $exe

function Stop-Running {
    Get-Process -Name $Name -ErrorAction SilentlyContinue | ForEach-Object {
        Write-Host "Stopping running $exe (PID $($_.Id))"
        $_ | Stop-Process -Force
        $_ | Wait-Process -Timeout 5 -ErrorAction SilentlyContinue
    }
}

# unregister only
if ($Unregister) {
    Stop-Running
    if (Test-Path -Path $lnk -PathType Leaf) {
        Remove-Item -Path $lnk -Force
        Write-Host "Removed: $lnk" -ForegroundColor Green
    }
    else {
        Write-Host "Not registered: $lnk"
    }
    return
}

# build fails if there are no go:embed targets
foreach ($f in "icon.ico", "icon-unchecked.ico", "icon-out-of-date.ico") {
    if (-not (Test-Path -Path $f -PathType Leaf)) {
        Write-Error "$f was not found."
    }
}

if (-not (Test-Path -Path "go.mod" -PathType Leaf)) {
    go mod init $Name
}
go mod tidy

# cannot overwrite while running
Stop-Running

# do not include -s -w because they tend to trigger false positives
$ldflags = if ($Console) { "" } else { "-H=windowsgui" }

Write-Host "Installing $exe ..."
go install -ldflags $ldflags .
if ($LASTEXITCODE -ne 0) {
    Write-Error "go install failed (exit code $LASTEXITCODE)"
    exit $LASTEXITCODE
}
Write-Host "Installed: $installed" -ForegroundColor Green

# do not register if installing only or if building the console version
if ($NoRegister -or $Console) {
    return
}

$ws = New-Object -ComObject WScript.Shell
$sc = $ws.CreateShortcut($lnk)
$sc.TargetPath       = $installed
$sc.WorkingDirectory = $bin
$sc.Save()
Write-Host "Registered: $lnk" -ForegroundColor Green

Start-Process -FilePath $installed