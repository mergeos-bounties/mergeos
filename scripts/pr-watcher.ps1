param(
  [string]$Action = "status",
  [string]$Token = ""
)

$ErrorActionPreference = "Stop"
$Repo = "mergeos-bounties/mergeos"
$PR = 152
$Branch = "fix/logout-bug"

if (-not $Token) {
  # Try to get token from git credential store
  $credResult = cmd /c "echo protocol=https & echo host=github.com" 2>$null | git credential fill 2>$null
  if ($credResult -match "password=(.+)") {
    $Token = $matches[1]
  }
}

if (-not $Token) {
  Write-Host "ERROR: No GitHub token found. Set GITHUB_TOKEN env var or login via gh CLI." -ForegroundColor Red
  exit 1
}

$headers = @{
  "Authorization" = "token $Token"
  "Content-Type" = "application/json"
  "Accept" = "application/vnd.github.v3+json"
}

function Get-PRStatus {
  Write-Host "=== PR #$PR Status ===" -ForegroundColor Cyan
  $prData = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/pulls/$PR" -Headers $headers
  Write-Host "Title: $($prData.title)" -ForegroundColor White
  Write-Host "State: $($prData.state)" -ForegroundColor Yellow
  Write-Host "Mergeable: $($prData.mergeable)" -ForegroundColor Green
  Write-Host "Updated: $($prData.updated_at)" -ForegroundColor Gray

  $comments = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/issues/$PR/comments" -Headers $headers
  $reviewComments = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/pulls/$PR/comments" -Headers $headers

  Write-Host "`n=== Comments ($($comments.Count + $reviewComments.Count)) ===" -ForegroundColor Cyan
  $allComments = $comments + $reviewComments
  $allComments | ForEach-Object {
    $author = $_.user.login
    $role = $_.author_association
    $body = $_.body -replace '(?:\r\n|\n|\r)', ' '
    $truncated = if ($body.Length -gt 150) { $body.Substring(0, 150) + "..." } else { $body }
    Write-Host "[$role] @$author : $truncated" -ForegroundColor $(
      if ($role -eq 'MEMBER' -or $role -eq 'COLLABORATOR' -or $role -eq 'OWNER') { 'Red' } else { 'Gray' }
    )
  }

  # Check for maintainer change requests
  $changeRequests = $allComments | Where-Object {
    $_.user.login -ne 'davidsineri' -and
    $_.user.login -ne 'davidsineri[bot]' -and
    ($_.author_association -eq 'MEMBER' -or $_.author_association -eq 'COLLABORATOR' -or $_.author_association -eq 'OWNER') -and
    ($_.body -match 'change|fix|update|revision|request|issue|problem|error|bug|wrong|incorrect|please|need')
  }

  if ($changeRequests.Count -gt 0) {
    Write-Host "`n⚠ MAINTAINER FEEDBACK DETECTED!" -ForegroundColor Red
    $changeRequests | ForEach-Object {
      Write-Host "  @$($_.user.login): $($_.body)" -ForegroundColor Yellow
    }
    Write-Host "`nRun: .\scripts\pr-watcher.ps1 -Action reply" -ForegroundColor Green
  }
}

function Update-PRBranch {
  param([string]$Message = "")
  Write-Host "=== Updating PR branch ===" -ForegroundColor Cyan

  git add -A
  if ($Message) {
    git commit -m "$Message"
  }
  git push fork $Branch
  Write-Host "✓ Branch updated and pushed!" -ForegroundColor Green
}

function Reply-ToComment {
  param([string]$CommentId = "", [string]$Message = "Thanks for the review! I'll address this shortly and push an update.")

  if (-not $CommentId) {
    # Auto-detect last maintainer comment needing reply
    $comments = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/issues/$PR/comments" -Headers $headers
    $lastMaintainer = $comments | Where-Object {
      $_.user.login -ne 'davidsineri' -and
      ($_.author_association -eq 'MEMBER' -or $_.author_association -eq 'COLLABORATOR' -or $_.author_association -eq 'OWNER')
    } | Select-Object -First 1

    if ($lastMaintainer) {
      $body = @"
Thanks for the feedback @$($lastMaintainer.user.login)!

I've noted the requested changes and will push an update shortly.
"@
      $resp = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/issues/$PR/comments" -Method Post -Headers $headers -Body (ConvertTo-Json @{body = $body})
      Write-Host "✓ Replied to @$($lastMaintainer.user.login)" -ForegroundColor Green
    }
  }
}

switch ($Action) {
  "status" { Get-PRStatus }
  "update" { Update-PRBranch -Message $args[0] }
  "reply" { Reply-ToComment }
  default { Write-Host "Usage: .\scripts\pr-watcher.ps1 -Action [status|update|reply]" -ForegroundColor Yellow }
}
