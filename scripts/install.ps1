# Perimeter installer for Windows. Run in PowerShell:
#
#   irm https://raw.githubusercontent.com/LuckyKort/perimeter/main/scripts/install.ps1 | iex
#
# To uninstall:
#
#   & ([scriptblock]::Create((irm https://raw.githubusercontent.com/LuckyKort/perimeter/main/scripts/install.ps1))) -Uninstall
#
# No admin rights needed: Perimeter lives in %LOCALAPPDATA%\Programs\Perimeter.
# No param() block: the script also runs through `irm | iex`, where one isn't allowed.
$Uninstall = [bool]($args | Where-Object { $_ -eq '-Uninstall' })
$ErrorActionPreference = 'Stop'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$dir = Join-Path $env:LOCALAPPDATA 'Programs\Perimeter'
$exe = Join-Path $dir 'perimeter.exe'
$lnk = Join-Path ([Environment]::GetFolderPath('Programs')) 'Perimeter.lnk'

Get-Process perimeter -ErrorAction SilentlyContinue | Stop-Process -Force
Start-Sleep -Milliseconds 500

if ($Uninstall) {
    Remove-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' -Name Perimeter -ErrorAction SilentlyContinue
    Remove-Item $lnk, $dir -Recurse -Force -ErrorAction SilentlyContinue
    Write-Host "Perimeter удалён. Настройки остались в $env:APPDATA\Perimeter." -ForegroundColor Green
    return
}

$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }
$base = 'https://github.com/LuckyKort/perimeter/releases/latest/download'
$name = "perimeter-windows-$arch.exe"
New-Item -ItemType Directory -Force $dir | Out-Null
Write-Host 'Скачиваю Perimeter…'
$tmp = Join-Path $env:TEMP "perimeter-$PID.exe"
Invoke-WebRequest "$base/$name" -OutFile $tmp -UseBasicParsing

# Compare with the release's SHA256SUMS.txt: a damaged or swapped file is never installed.
$sumsFile = "$tmp.sums"
Invoke-WebRequest "$base/SHA256SUMS.txt" -OutFile $sumsFile -UseBasicParsing
$line = Get-Content $sumsFile | Where-Object { $_ -match ('  ' + [regex]::Escape($name) + '$') }
$want = if ($line) { ($line -split '\s+')[0] } else { '' }
$got = (Get-FileHash $tmp -Algorithm SHA256).Hash.ToLower()
Remove-Item $sumsFile -Force
if (-not $want -or $want -ne $got) {
    Remove-Item $tmp -Force
    throw "Контрольная сумма $name не совпала - файл повреждён или подменён, ничего не установлено."
}
Move-Item $tmp $exe -Force
Unblock-File $exe

$sh = New-Object -ComObject WScript.Shell
$s = $sh.CreateShortcut($lnk)
$s.TargetPath = $exe
$s.Save()

Start-Process $exe
Write-Host 'Готово: значок Perimeter у часов (если не видно - он под стрелкой ^, его можно перетащить на панель).' -ForegroundColor Green
