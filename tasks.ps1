param([ValidateSet('up','test','down')][string]$Task='test')
$ErrorActionPreference='Stop'
Push-Location $PSScriptRoot
try {
 switch ($Task) {
  up { docker compose up -d --wait }
  test {
   if (-not $env:DB_DSN) { $env:DB_DSN='postgres://test:test@localhost:54321/notes_test?sslmode=disable' }
   go test -v -shuffle=on -count=2 ./integration/...
  }
  down { docker compose stop }
 }
 if ($LASTEXITCODE -ne 0) { throw "Command failed: $LASTEXITCODE" }
} finally { Pop-Location }
