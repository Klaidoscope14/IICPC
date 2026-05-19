Write-Host "Stopping IICPC Infrastructure..." -ForegroundColor Cyan
cd infrastructure/docker
docker compose down
cd ../..

Write-Host "Cleaning up dangling submission containers..." -ForegroundColor Cyan
# Find all containers whose name starts with "submission-"
$containers = docker ps -a -q --filter "name=^submission-"
if ($containers) {
    docker rm -f $containers
    Write-Host "Cleaned up submission containers." -ForegroundColor Green
} else {
    Write-Host "No dangling submission containers found." -ForegroundColor Yellow
}

Write-Host "All Docker containers have been stopped and cleaned up!" -ForegroundColor Green
