param([string]$Go='go',[string]$Out='dist',[string]$Version='')
$ErrorActionPreference='Stop'
$root=Split-Path $PSScriptRoot -Parent
if(-not $Version){$Version=(Get-Content -Raw (Join-Path $root 'compatibility.json')|ConvertFrom-Json).version}
# Canonical embedded text also supports existing Windows checkouts made before
# the eol attributes were introduced. This touches repository sources only.
$embedded=@('tools/agent-bench/bench.py','tools/agent-bench/summarize.py','tools/agent-bench/resources.py','tools/agent-bench/safe_bench.py')|ForEach-Object{Join-Path $root $_}
$embedded+=Get-ChildItem -LiteralPath (Join-Path $root 'internal/app/skills') -Filter '*.md' -File|Select-Object -ExpandProperty FullName
foreach($path in $embedded){$text=[IO.File]::ReadAllText($path);[IO.File]::WriteAllText($path,$text.Replace("`r`n","`n"),[Text.UTF8Encoding]::new($false))}
npm --prefix (Join-Path $root 'web') ci --no-audit --no-fund
if($LASTEXITCODE){throw 'Frontend dependencies failed'}
npm --prefix (Join-Path $root 'web') run build
if($LASTEXITCODE){throw 'Frontend build failed'}
npm --prefix (Join-Path $root 'web') audit --omit=dev
if($LASTEXITCODE){throw 'Production dependency advisory found; review before publication'}
New-Item -ItemType Directory -Force (Join-Path $root $Out)|Out-Null
Push-Location $root
try{& $Go run golang.org/x/vuln/cmd/govulncheck@latest ./...;if($LASTEXITCODE){throw 'Reachable Go vulnerability found; do not publish'};& $Go run github.com/akavel/rsrc@v0.10.2 -ico assets/my_ai.ico -o cmd/my-ai/rsrc_windows_amd64.syso;if($LASTEXITCODE){throw 'Icon generation failed'};& $Go test ./...;if($LASTEXITCODE){throw 'Go checks failed'};& $Go build -buildvcs=false -trimpath -ldflags "-s -w -X main.version=$Version" -o (Join-Path $Out 'MyAI.exe') ./cmd/my-ai;if($LASTEXITCODE){throw 'Native build failed'};& $Go run github.com/akavel/rsrc@v0.10.2 -ico assets/ai_service.ico -o cmd/ai-service/rsrc_windows_amd64.syso;if($LASTEXITCODE){throw 'Service icon generation failed'};& $Go build -buildvcs=false -trimpath -ldflags "-s -w -X main.version=$Version" -o (Join-Path $Out 'AIService.exe') ./cmd/ai-service;if($LASTEXITCODE){throw 'AI Service build failed'}}finally{Pop-Location}
