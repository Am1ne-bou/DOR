# start-dor.ps1
# Lance tout le projet DOR et ouvre 4 terminaux interactifs

Write-Host "=== Demarrage du projet DOR ===" -ForegroundColor Cyan

# Se placer dans le dossier du script
Set-Location $PSScriptRoot

# Lancer compose en arriere-plan
Write-Host "[1/3] docker compose up -d..." -ForegroundColor Yellow
docker compose up -d

if ($LASTEXITCODE -ne 0) {
    Write-Host "Erreur lors du demarrage de compose. Abandon." -ForegroundColor Red
    exit 1
}

# Attendre que tous les noeuds soient enregistres
Write-Host "[2/3] Attente que les noeuds s'enregistrent (~30s)..." -ForegroundColor Yellow
$maxWait = 90
$waited = 0
while ($waited -lt $maxWait) {
    $logs = docker compose logs directory 2>&1 | Out-String
    $registered = ([regex]::Matches($logs, "registered")).Count
    if ($registered -ge 3) {
        Write-Host "  -> 3 noeuds enregistres !" -ForegroundColor Green
        break
    }
    Start-Sleep -Seconds 2
    $waited += 2
    Write-Host "  ($registered/3 noeuds enregistres, $waited s)" -ForegroundColor Gray
}

if ($registered -lt 3) {
    Write-Host "Attention : seulement $registered/3 noeuds enregistres apres $maxWait s" -ForegroundColor Yellow
    Write-Host "Les terminaux vont quand meme s'ouvrir." -ForegroundColor Yellow
}

# Ouvrir 4 terminaux PowerShell, chacun attache a un conteneur
Write-Host "[3/3] Ouverture de 4 terminaux..." -ForegroundColor Yellow

$containers = @("directory", "node1", "node2", "node3")
foreach ($c in $containers) {
    $title = "DOR - $c"
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "`$Host.UI.RawUI.WindowTitle='$title'; Write-Host '=== Attached to $c ===' -ForegroundColor Cyan; Write-Host 'Pour detacher SANS tuer le conteneur : Ctrl+P puis Ctrl+Q' -ForegroundColor Yellow; Write-Host ''; docker attach $c"
    Start-Sleep -Milliseconds 500
}

Write-Host ""
Write-Host "=== Pret ===" -ForegroundColor Green
Write-Host "4 terminaux ouverts (directory + 3 noeuds)."
Write-Host ""
Write-Host "Pour tout arreter : docker compose down"
