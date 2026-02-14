param(
  [string]$ServerUrl = "http://127.0.0.1:8080",
  [string]$UserLogin = "mock_user",
  [string]$UserPassword = "mock_user_password_123!",
  [string]$MasterPassword = "mock_master_password_123!"
)

$ErrorActionPreference = 'Stop'

function Assert-Ok($what) {
  if ($LASTEXITCODE -ne 0) {
    throw "FAILED: $what (exit code $LASTEXITCODE)"
  }
}

# --- Paths (use repo-local temp to avoid clobbering real state) ---
$RepoRoot = (Resolve-Path ".").Path
$WorkDir = Join-Path $RepoRoot "tmp\cli-mock-test"
$null = New-Item -ItemType Directory -Force -Path $WorkDir

$StatePath = Join-Path $WorkDir "state.json"
$VaultPath = Join-Path $WorkDir "vault.json"
$ClientExe = Join-Path $WorkDir "client.exe"

# --- Build client once ---
Write-Host "== build client =="
go build -o $ClientExe .\cmd\client
Assert-Ok "go build client"

function Invoke-Client([Parameter(ValueFromRemainingArguments=$true)][string[]]$Args) {
  & $ClientExe --server $ServerUrl --state $StatePath --vault $VaultPath @Args
  return $LASTEXITCODE
}

# --- Stable IDs so you can copy/paste ---
$ID_LOGIN = "a4f9c4d0-2f49-4a7e-8f35-6cc8f0a77c6b"
$ID_TEXT  = "7bafad08-1ac0-4b2f-a1a6-9bcbd6b82b7f"
$ID_CARD  = "e13dce30-7b26-4d03-a5c4-204b6f3e25d4"
$ID_BIN   = "d7b2c9b1-3f5f-4f30-9a5a-1c2c2d2d2d2d"

# --- Create a binary file (real bytes) ---
$BinIn  = Join-Path $WorkDir "in.bin"
$BinOut = Join-Path $WorkDir "out.bin"
[byte[]]$bytes = 0..255
[System.IO.File]::WriteAllBytes($BinIn, $bytes)

Write-Host "== register (ok if already exists) =="
Invoke-Client register --login $UserLogin --password $UserPassword | Out-Host
if ($LASTEXITCODE -ne 0) {
  Write-Host "register failed (maybe already exists) - continue"
}

Write-Host "== login =="
Invoke-Client login --login $UserLogin --password $UserPassword | Out-Host
Assert-Ok "login"

Write-Host "== create items (4 types + meta/meta-text) =="
Invoke-Client set --password $MasterPassword --type "login/pass" --id $ID_LOGIN --data '{"login":"alice","password":"p@ssw0rd","url":"https://example.com"}' --meta env=dev --meta app=gdk --meta-text "login/pass demo" | Out-Host
Assert-Ok "set login/pass"

Invoke-Client set --password $MasterPassword --type "text" --id $ID_TEXT --data "hello text #1" --meta env=dev --meta kind=text --meta-text "text demo" | Out-Host
Assert-Ok "set text"

Invoke-Client set --password $MasterPassword --type "card" --id $ID_CARD --data '{"name":"ALICE","pan":"4111111111111111","exp":"12/30","cvc":"123"}' --meta env=dev --meta kind=card --meta-text "card demo" | Out-Host
Assert-Ok "set card"

Invoke-Client set --password $MasterPassword --type "binary" --id $ID_BIN --file $BinIn --meta env=dev --meta kind=binary --meta-text "binary demo" | Out-Host
Assert-Ok "set binary"

Write-Host "== list cases =="
Write-Host "-- list (default: non-deleted)"
Invoke-Client list | Out-Host
Assert-Ok "list default"

Write-Host "-- list --type text"
Invoke-Client list --type "text" | Out-Host
Assert-Ok "list --type text"

Write-Host "-- list --deleted (should be empty right now)"
Invoke-Client list --deleted | Out-Host
Assert-Ok "list --deleted"

Write-Host "-- list --all (all, including deleted)"
Invoke-Client list --all | Out-Host
Assert-Ok "list --all"

Write-Host "-- list --all --deleted (note: --all wins)"
Invoke-Client list --all --deleted | Out-Host
Assert-Ok "list --all --deleted"

Write-Host "== get cases =="
Write-Host "-- get text"
Invoke-Client get --password $MasterPassword --id $ID_TEXT | Out-Host
Assert-Ok "get text"

Write-Host "-- get login/pass"
Invoke-Client get --password $MasterPassword --id $ID_LOGIN | Out-Host
Assert-Ok "get login/pass"

Write-Host "-- get card"
Invoke-Client get --password $MasterPassword --id $ID_CARD | Out-Host
Assert-Ok "get card"

Write-Host "-- get binary (writes out.bin)"
if (Test-Path $BinOut) { Remove-Item -Force $BinOut }
Invoke-Client get --password $MasterPassword --id $ID_BIN --out $BinOut | Out-Host
Assert-Ok "get binary"

$h1 = (Get-FileHash -Algorithm SHA256 $BinIn).Hash
$h2 = (Get-FileHash -Algorithm SHA256 $BinOut).Hash
if ($h1 -ne $h2) { throw "binary mismatch: $h1 != $h2" }
Write-Host "binary OK (sha256 matches)"

Write-Host "== update case (set same ID, different data/meta) =="
Invoke-Client set --password $MasterPassword --type "text" --id $ID_TEXT --data "hello text #2 (updated)" --meta env=dev --meta kind=text --meta updated=true --meta-text "text demo updated" | Out-Host
Assert-Ok "update text via set --id"

Write-Host "-- get updated text"
Invoke-Client get --password $MasterPassword --id $ID_TEXT | Out-Host
Assert-Ok "get updated text"

Write-Host "== delete case =="
Invoke-Client delete --id $ID_CARD | Out-Host
Assert-Ok "delete card"

Write-Host "-- list --deleted (should include card)"
Invoke-Client list --deleted | Out-Host
Assert-Ok "list --deleted after delete"

Write-Host "== negative cases (expected failures) =="
Write-Host "-- get binary WITHOUT --out (should fail)"
Invoke-Client get --password $MasterPassword --id $ID_BIN | Out-Host
if ($LASTEXITCODE -eq 0) { throw "expected failure: get binary without --out" }

Write-Host "-- get deleted item (should fail)"
Invoke-Client get --password $MasterPassword --id $ID_CARD | Out-Host
if ($LASTEXITCODE -eq 0) { throw "expected failure: get deleted item" }

Write-Host "-- list with unsupported type (should fail)"
Invoke-Client list --type "nope" | Out-Host
if ($LASTEXITCODE -eq 0) { throw "expected failure: list --type nope" }

Write-Host "-- get with wrong master password (should fail decrypt)"
Invoke-Client get --password "WRONG_PASSWORD" --id $ID_TEXT | Out-Host
if ($LASTEXITCODE -eq 0) { throw "expected failure: get with wrong master password" }

Write-Host ""
Write-Host "ALL DONE. Temp dir:"
Write-Host "  $WorkDir"
