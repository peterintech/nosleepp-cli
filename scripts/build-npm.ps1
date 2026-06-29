param(
    [string]$Version = "0.2.0"
)

$ErrorActionPreference = "Stop"

$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$npmDir = Join-Path $root "npm"
$binDir = Join-Path $npmDir "bin"
$cacheDir = Join-Path $root ".gocache"

New-Item -ItemType Directory -Force -Path $binDir | Out-Null
New-Item -ItemType Directory -Force -Path $cacheDir | Out-Null

function Build-nosleepp {
    param(
        [string]$Goos,
        [string]$Goarch,
        [string]$Output
    )

    Write-Host "Building $Goos/$Goarch -> $Output"
    $env:GOOS = $Goos
    $env:GOARCH = $Goarch
    $env:CGO_ENABLED = "0"
    $env:GOCACHE = $cacheDir
    go build -ldflags "-X nosleepp/cmd.version=$Version" -o (Join-Path $binDir $Output) .
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed for $Goos/$Goarch"
    }
}

Push-Location $root
try {
    Build-nosleepp -Goos "windows" -Goarch "amd64" -Output "nosleepp-win32-x64.exe"
    Build-nosleepp -Goos "darwin" -Goarch "arm64" -Output "nosleepp-darwin-arm64"
    Build-nosleepp -Goos "darwin" -Goarch "amd64" -Output "nosleepp-darwin-x64"
}
finally {
    Pop-Location
    Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
    Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue
    Remove-Item Env:\GOCACHE -ErrorAction SilentlyContinue
}

Write-Host "Built npm binaries in $binDir"
