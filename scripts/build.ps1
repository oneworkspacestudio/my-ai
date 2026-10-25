param([string]$Go='go',[string]$Out='dist',[string]$Version='0.1.0-preview.7')
$ErrorActionPreference='Stop'
$root=Split-Path $PSScriptRoot -Parent
npm --prefix (Join-Path $root 'web') ci --no-audit --no-fund
if($LASTEXITCODE){throw 'Frontend dependencies failed'}
npm --prefix (Join-Path $root 'web') run build
if($LASTEXITCODE){throw 'Frontend build failed'}
npm --prefix (Join-Path $root 'web') audit --omit=dev
if($LASTEXITCODE){throw 'Production dependency advisory found; review before publication'}
New-Item -ItemType Directory -Force (Join-Path $root $Out)|Out-Null
Push-Location $root
try{& $Go run golang.org/x/vuln/cmd/govulncheck@latest ./...;if($LASTEXITCODE){throw 'Reachable Go vulnerability found; do not publish'};& $Go run github.com/akavel/rsrc@v0.10.2 -ico assets/my_ai.ico -o cmd/my-ai/rsrc_windows_amd64.syso;if($LASTEXITCODE){throw 'Icon generation failed'};& $Go test ./...;if($LASTEXITCODE){throw 'Go checks failed'};& $Go build -buildvcs=false -trimpath -ldflags "-s -w -X main.version=$Version" -o (Join-Path $Out 'MyAI.exe') ./cmd/my-ai;if($LASTEXITCODE){throw 'Native build failed'}}finally{Pop-Location}
