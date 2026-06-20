param(
  [string]$ProjectRoot = "C:\Dev\GO\Veltryx-Tauri",
  [string]$KeyFile = "C:\Dev\GO\Veltryx-Tauri\web\~\tauri\veltryx.key",
  [string]$KeyPassword = "",
  [switch]$BuildBackend = $true
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

function Step([string]$Message) {
  Write-Host ""
  Write-Host "==> $Message" -ForegroundColor Cyan
}

function Assert-PathExists([string]$PathToCheck, [string]$Label) {
  if (-not (Test-Path -LiteralPath $PathToCheck)) {
    throw "$Label not found: $PathToCheck"
  }
}

function Get-LatestSingle([string]$Folder, [string]$Filter, [string]$Label) {
  Assert-PathExists -PathToCheck $Folder -Label "Folder"
  $item = Get-ChildItem -Path $Folder -Filter $Filter -File | Sort-Object LastWriteTime -Descending | Select-Object -First 1
  if (-not $item) {
    throw "No $Label found in $Folder with filter $Filter"
  }
  return $item
}

function Compress-SingleFile([string]$InputFile, [string]$ZipFile) {
  if (Test-Path -LiteralPath $ZipFile) {
    Remove-Item -LiteralPath $ZipFile -Force
  }
  Compress-Archive -LiteralPath $InputFile -DestinationPath $ZipFile -Force
}

function Sign-Zip([string]$WebDir, [string]$ZipFile, [string]$SigningKey, [string]$Password) {
  if ($Password) {
    & npx tauri signer sign -f $SigningKey -p $Password $ZipFile
  } else {
    & npx tauri signer sign -f $SigningKey $ZipFile
  }

  if ($LASTEXITCODE -ne 0) {
    throw "Signing failed for $ZipFile"
  }
}

function Read-SignatureBase64([string]$SigFile) {
  Assert-PathExists -PathToCheck $SigFile -Label "Signature file"
  return (Get-Content -LiteralPath $SigFile -Raw).Trim()
}

$webDir = Join-Path $ProjectRoot "web"
$sidecarDir = Join-Path $webDir "src-tauri\bin"
$sidecarExe = Join-Path $sidecarDir "api.exe"
$bundleDir = Join-Path $webDir "src-tauri\target\release\bundle"
$nsisDir = Join-Path $bundleDir "nsis"
$msiDir = Join-Path $bundleDir "msi"

Assert-PathExists -PathToCheck $ProjectRoot -Label "Project root"
Assert-PathExists -PathToCheck $webDir -Label "Web directory"
Assert-PathExists -PathToCheck $KeyFile -Label "Signing key"

if ($BuildBackend) {
  Step "Building embedded backend sidecar (Go)"
  if (-not (Test-Path -LiteralPath $sidecarDir)) {
    New-Item -ItemType Directory -Path $sidecarDir | Out-Null
  }
  Push-Location $ProjectRoot
  & go build -o $sidecarExe ".\cmd\api"
  if ($LASTEXITCODE -ne 0) {
    Pop-Location
    throw "Go backend build failed"
  }
  Pop-Location
}

Step "Building Tauri release bundles"
Push-Location $webDir
& npm run tauri build
if ($LASTEXITCODE -ne 0) {
  Pop-Location
  throw "Tauri build failed"
}
Pop-Location

$nsisExe = Get-LatestSingle -Folder $nsisDir -Filter "*-setup.exe" -Label "NSIS setup executable"
$msiFile = Get-LatestSingle -Folder $msiDir -Filter "*.msi" -Label "MSI installer"

$nsisZip = [System.IO.Path]::Combine($nsisDir, ([System.IO.Path]::GetFileNameWithoutExtension($nsisExe.Name) + ".nsis.zip"))
$msiZip = [System.IO.Path]::Combine($msiDir, ($msiFile.Name + ".zip"))

Step "Creating updater zip artifacts"
Compress-SingleFile -InputFile $nsisExe.FullName -ZipFile $nsisZip
Compress-SingleFile -InputFile $msiFile.FullName -ZipFile $msiZip

Step "Signing updater zips"
Push-Location $webDir
Sign-Zip -WebDir $webDir -ZipFile $nsisZip -SigningKey $KeyFile -Password $KeyPassword
Sign-Zip -WebDir $webDir -ZipFile $msiZip -SigningKey $KeyFile -Password $KeyPassword
Pop-Location

$nsisSig = "$nsisZip.sig"
$msiSig = "$msiZip.sig"
$nsisSignatureBase64 = Read-SignatureBase64 -SigFile $nsisSig
$msiSignatureBase64 = Read-SignatureBase64 -SigFile $msiSig

Write-Host ""
Write-Host "Build and signing completed." -ForegroundColor Green
Write-Host ""
Write-Host "NSIS updater artifacts:"
Write-Host "  Zip: $nsisZip"
Write-Host "  Sig: $nsisSig"
Write-Host "  Signature (base64):"
Write-Host $nsisSignatureBase64
Write-Host ""
Write-Host "MSI updater artifacts:"
Write-Host "  Zip: $msiZip"
Write-Host "  Sig: $msiSig"
Write-Host "  Signature (base64):"
Write-Host $msiSignatureBase64
