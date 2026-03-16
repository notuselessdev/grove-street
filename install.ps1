# Grove Street Installer for Windows
# "Ah shit, here we go again."

$ErrorActionPreference = "Stop"

$Repo = "notuselessdev/grove-street"
$InstallDir = "$env:APPDATA\grove-street"
$BinDir = "$InstallDir\bin"

function Write-CJ($msg, $color = "Cyan") {
    Write-Host "[CJ] " -ForegroundColor $color -NoNewline
    Write-Host $msg
}

function Get-LatestVersion {
    $release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
    return $release.tag_name -replace '^v', ''
}

function Install-Binary($version) {
    $arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
    $url = "https://github.com/$Repo/releases/download/v$version/grove-street_windows_$arch.exe"

    Write-CJ "Downloading grove-street v$version..."
    New-Item -ItemType Directory -Path $BinDir -Force | Out-Null

    $binPath = "$BinDir\grove-street.exe"
    Invoke-WebRequest -Uri $url -OutFile $binPath
    Write-CJ "Binary installed to $binPath" "Green"

    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -notlike "*$BinDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$currentPath;$BinDir", "User")
        Write-CJ "Added $BinDir to PATH" "Green"
    }
}

function Install-Sounds($version) {
    $soundsDir = "$InstallDir\sounds"
    $categories = @("session_start", "task_complete", "task_error", "input_required", "resource_limit", "user_spam")

    foreach ($cat in $categories) {
        New-Item -ItemType Directory -Path "$soundsDir\$cat" -Force | Out-Null
    }

    Write-CJ "Downloading sound pack..."
    $url = "https://github.com/$Repo/releases/download/v$version/sounds.zip"
    $tmp = [System.IO.Path]::GetTempFileName() + ".zip"

    try {
        Invoke-WebRequest -Uri $url -OutFile $tmp
        Expand-Archive -Path $tmp -DestinationPath $soundsDir -Force
        Remove-Item $tmp -Force
        Write-CJ "Sound pack installed." "Green"
    } catch {
        Write-CJ "Could not download sound pack. Add .wav/.mp3 files to $soundsDir\<category>\" "Yellow"
    }
}

function Write-Config {
    $configPath = "$InstallDir\config.json"
    if (-not (Test-Path $configPath)) {
        @{
            enabled = $true
            volume = 0.8
            auto_update = $true
        } | ConvertTo-Json | Set-Content $configPath
        Write-CJ "Config written to $configPath" "Green"
    }
}

function Register-Hooks {
    $settingsDir = "$env:USERPROFILE\.claude"
    $settingsPath = "$settingsDir\settings.json"
    $hookCmd = "$BinDir\grove-street.exe hook"

    New-Item -ItemType Directory -Path $settingsDir -Force | Out-Null

    $settings = @{}
    if (Test-Path $settingsPath) {
        $settings = Get-Content $settingsPath | ConvertFrom-Json -AsHashtable
    }

    $hookEntry = @(@{ matcher = ""; command = $hookCmd })
    $hooks = @{
        Stop = $hookEntry
        Notification = $hookEntry
        SubagentStop = $hookEntry
        PreCompact = $hookEntry
    }
    $settings["hooks"] = $hooks

    $settings | ConvertTo-Json -Depth 10 | Set-Content $settingsPath
    Write-CJ "Hooks registered in Claude Code." "Green"
}

function Setup-AutoUpdate {
    $taskName = "GroveStreetUpdate"
    $existing = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
    if ($existing) { return }

    $action = New-ScheduledTaskAction -Execute "$BinDir\grove-street.exe" -Argument "update"
    $trigger = New-ScheduledTaskTrigger -Daily -At "12:00"
    Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Description "Grove Street auto-update" | Out-Null
    Write-CJ "Auto-update scheduled task created (daily at noon)." "Green"
}

# Main
Write-Host ""
Write-Host ' ██████╗ ██████╗  ██████╗ ██╗   ██╗███████╗' -ForegroundColor Green
Write-Host '██╔════╝ ██╔══██╗██╔═══██╗██║   ██║██╔════╝' -ForegroundColor Green
Write-Host '██║  ███╗██████╔╝██║   ██║██║   ██║█████╗  ' -ForegroundColor Green
Write-Host '██║   ██║██╔══██╗██║   ██║╚██╗ ██╔╝██╔══╝  ' -ForegroundColor Green
Write-Host '╚██████╔╝██║  ██║╚██████╔╝ ╚████╔╝ ███████╗' -ForegroundColor Green
Write-Host ' ╚═════╝ ╚═╝  ╚═╝ ╚═════╝   ╚═══╝  ╚══════╝' -ForegroundColor Green
Write-Host '        ███████╗████████╗██████╗ ███████╗███████╗████████╗' -ForegroundColor Cyan
Write-Host '        ██╔════╝╚══██╔══╝██╔══██╗██╔════╝██╔════╝╚══██╔══╝' -ForegroundColor Cyan
Write-Host '        ███████╗   ██║   ██████╔╝█████╗  █████╗     ██║   ' -ForegroundColor Cyan
Write-Host '        ╚════██║   ██║   ██╔══██╗██╔══╝  ██╔══╝     ██║   ' -ForegroundColor Cyan
Write-Host '        ███████║   ██║   ██║  ██║███████╗███████╗   ██║   ' -ForegroundColor Cyan
Write-Host '        ╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚══════╝   ╚═╝   ' -ForegroundColor Cyan
Write-Host ""
Write-Host '  "Ah shit, here we go again." - CJ' -ForegroundColor Yellow
Write-Host ""

$version = Get-LatestVersion
Write-CJ "Latest version: v$version"

Install-Binary $version
Install-Sounds $version
Write-Config
Register-Hooks
Setup-AutoUpdate

Write-Host ""
Write-CJ "Installation complete!" "Green"
Write-Host ""
Write-Host "  Grove Street. Home. CJ is watching your terminal now." -ForegroundColor Cyan
Write-Host ""
Write-Host "  Commands:"
Write-Host "    grove-street play session_start   # Test a sound"
Write-Host "    grove-street list                  # See all sounds"
Write-Host "    grove-street update                # Check for updates"
Write-Host "    grove-street help                  # Full help"
Write-Host ""
