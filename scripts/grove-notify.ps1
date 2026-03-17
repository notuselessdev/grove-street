# grove-notify.ps1 — Native Windows notification overlay for Grove Street
# Usage: powershell -File grove-notify.ps1 <sender> <phrase> <icon_path> <dismiss_seconds> <bundle_id> <project_name> <position> <slot_index> <slot_dir> [category_label]
#
# Positions: top-left, top-center, top-right, bottom-left, bottom-center, bottom-right, center

param(
    [string]$SenderName = "Carl Johnson",
    [string]$Phrase = "",
    [string]$IconPath = "",
    [double]$DismissSecs = 4,
    [string]$BundleId = "",
    [string]$ProjectName = "grove-street",
    [string]$Position = "top-right",
    [int]$SlotIndex = 0,
    [string]$SlotDir = "",
    [string]$CategoryLabel = ""
)

Add-Type -AssemblyName PresentationFramework
Add-Type -AssemblyName PresentationCore
Add-Type -AssemblyName WindowsBase
Add-Type -AssemblyName System.Drawing

$WinWidth = 360
$WinHeight = 72
$Margin = 12
$MyPid = $PID

# --- Slot management ---

function Test-SlotOccupied([int]$idx) {
    if (-not $SlotDir) { return $true }
    $path = Join-Path $SlotDir "$idx.lock"
    if (-not (Test-Path $path)) { return $false }
    try {
        $pid = [int](Get-Content $path -ErrorAction Stop).Trim()
        if ($pid -eq $MyPid) { return $true }
        $proc = Get-Process -Id $pid -ErrorAction SilentlyContinue
        return $null -ne $proc
    } catch {
        return $false
    }
}

function Move-ToSlot([int]$newIdx) {
    if (-not $SlotDir) { return }
    $oldPath = Join-Path $SlotDir "$script:SlotIndex.lock"
    $newPath = Join-Path $SlotDir "$newIdx.lock"
    try {
        $MyPid | Out-File -FilePath $newPath -Encoding ascii -NoNewline
        Remove-Item $oldPath -ErrorAction SilentlyContinue
    } catch {}
    $script:SlotIndex = $newIdx
}

function Remove-Slot {
    if (-not $SlotDir) { return }
    $path = Join-Path $SlotDir "$script:SlotIndex.lock"
    Remove-Item $path -ErrorAction SilentlyContinue
}

# --- Position calculation ---

function Get-NotificationOrigin([int]$slot) {
    $screen = [System.Windows.SystemParameters]::WorkArea
    $sw = $screen.Width
    $sh = $screen.Height
    $sx = $screen.X
    $sy = $screen.Y
    $stackOffset = $slot * ($WinHeight + $Margin)

    switch ($Position) {
        "top-left" {
            $x = $sx + $Margin
            $y = $sy + $Margin + $stackOffset
        }
        "top-center" {
            $x = $sx + ($sw - $WinWidth) / 2
            $y = $sy + $Margin + $stackOffset
        }
        "top-right" {
            $x = $sx + $sw - $WinWidth - $Margin
            $y = $sy + $Margin + $stackOffset
        }
        "bottom-left" {
            $x = $sx + $Margin
            $y = $sy + $sh - $WinHeight - $Margin - $stackOffset
        }
        "bottom-center" {
            $x = $sx + ($sw - $WinWidth) / 2
            $y = $sy + $sh - $WinHeight - $Margin - $stackOffset
        }
        "bottom-right" {
            $x = $sx + $sw - $WinWidth - $Margin
            $y = $sy + $sh - $WinHeight - $Margin - $stackOffset
        }
        "center" {
            $x = $sx + ($sw - $WinWidth) / 2
            $half = $stackOffset / 2
            if ($slot % 2 -eq 0) { $y = $sy + ($sh - $WinHeight) / 2 - $half }
            else { $y = $sy + ($sh - $WinHeight) / 2 + $half }
        }
        default {
            $x = $sx + $sw - $WinWidth - $Margin
            $y = $sy + $Margin + $stackOffset
        }
    }
    return @{ X = $x; Y = $y }
}

# --- Build Window ---

$window = New-Object System.Windows.Window
$window.WindowStyle = "None"
$window.AllowsTransparency = $true
$window.Background = [System.Windows.Media.Brushes]::Transparent
$window.Topmost = $true
$window.ShowInTaskbar = $false
$window.Width = $WinWidth
$window.Height = $WinHeight
$window.ResizeMode = "NoResize"

# Dark rounded container
$border = New-Object System.Windows.Controls.Border
$border.CornerRadius = New-Object System.Windows.CornerRadius(14)
$border.Background = New-Object System.Windows.Media.SolidColorBrush(
    [System.Windows.Media.Color]::FromArgb(230, 30, 30, 30)
)
$border.BorderBrush = New-Object System.Windows.Media.SolidColorBrush(
    [System.Windows.Media.Color]::FromArgb(30, 255, 255, 255)
)
$border.BorderThickness = New-Object System.Windows.Thickness(1)
$border.Padding = New-Object System.Windows.Thickness(14, 10, 14, 10)

# Horizontal layout
$hStack = New-Object System.Windows.Controls.StackPanel
$hStack.Orientation = "Horizontal"

# Icon
if ($IconPath -and (Test-Path $IconPath)) {
    try {
        $bitmap = New-Object System.Windows.Media.Imaging.BitmapImage
        $bitmap.BeginInit()
        $bitmap.UriSource = New-Object System.Uri($IconPath, [System.UriKind]::Absolute)
        $bitmap.DecodePixelWidth = 40
        $bitmap.EndInit()

        $img = New-Object System.Windows.Controls.Image
        $img.Source = $bitmap
        $img.Width = 40
        $img.Height = 40
        $img.Margin = New-Object System.Windows.Thickness(0, 0, 10, 0)
        $img.VerticalAlignment = "Center"

        # Round clip
        $clip = New-Object System.Windows.Media.RectangleGeometry
        $clip.Rect = New-Object System.Windows.Rect(0, 0, 40, 40)
        $clip.RadiusX = 8
        $clip.RadiusY = 8
        $img.Clip = $clip

        $hStack.Children.Add($img) | Out-Null
    } catch {}
}

# Text column
$vStack = New-Object System.Windows.Controls.StackPanel
$vStack.Orientation = "Vertical"
$vStack.VerticalAlignment = "Center"

# Sender row: name left, category label right
$senderRow = New-Object System.Windows.Controls.Grid
$colLeft = New-Object System.Windows.Controls.ColumnDefinition
$colLeft.Width = [System.Windows.GridLength]::new(1, [System.Windows.GridUnitType]::Star)
$colRight = New-Object System.Windows.Controls.ColumnDefinition
$colRight.Width = [System.Windows.GridLength]::Auto
$senderRow.ColumnDefinitions.Add($colLeft)
$senderRow.ColumnDefinitions.Add($colRight)

$senderLabel = New-Object System.Windows.Controls.TextBlock
$senderLabel.Text = "$SenderName in $ProjectName"
$senderLabel.FontWeight = "Bold"
$senderLabel.FontSize = 13
$senderLabel.Foreground = New-Object System.Windows.Media.SolidColorBrush(
    [System.Windows.Media.Color]::FromArgb(240, 255, 255, 255)
)
$senderLabel.TextTrimming = "CharacterEllipsis"
$senderLabel.VerticalAlignment = "Center"
[System.Windows.Controls.Grid]::SetColumn($senderLabel, 0)
$senderRow.Children.Add($senderLabel) | Out-Null

if ($CategoryLabel) {
    $catLabel = New-Object System.Windows.Controls.TextBlock
    $catLabel.Text = $CategoryLabel
    $catLabel.FontSize = 10
    $catLabel.Foreground = New-Object System.Windows.Media.SolidColorBrush(
        [System.Windows.Media.Color]::FromArgb(153, 255, 255, 255)
    )
    $catLabel.VerticalAlignment = "Center"
    $catLabel.Margin = New-Object System.Windows.Thickness(6, 0, 0, 0)
    [System.Windows.Controls.Grid]::SetColumn($catLabel, 1)
    $senderRow.Children.Add($catLabel) | Out-Null
}

$vStack.Children.Add($senderRow) | Out-Null

if ($Phrase) {
    $phraseLabel = New-Object System.Windows.Controls.TextBlock
    $phraseLabel.Text = $Phrase
    $phraseLabel.FontSize = 12
    $phraseLabel.Foreground = New-Object System.Windows.Media.SolidColorBrush(
        [System.Windows.Media.Color]::FromArgb(130, 255, 255, 255)
    )
    $phraseLabel.TextTrimming = "CharacterEllipsis"
    $phraseLabel.Margin = New-Object System.Windows.Thickness(0, 2, 0, 0)
    $vStack.Children.Add($phraseLabel) | Out-Null
}

$hStack.Children.Add($vStack) | Out-Null
$border.Child = $hStack
$window.Content = $border

# Position
$origin = Get-NotificationOrigin $SlotIndex
$window.Left = $origin.X
$window.Top = $origin.Y

# Capture foreground window at launch for click-to-focus
Add-Type @"
using System;
using System.Runtime.InteropServices;
public class WinAPI {
    [DllImport("user32.dll")]
    public static extern bool SetForegroundWindow(IntPtr hWnd);
    [DllImport("user32.dll")]
    public static extern IntPtr GetForegroundWindow();
}
"@

$foregroundAtLaunch = [WinAPI]::GetForegroundWindow()

$window.Add_MouseLeftButtonDown({
    # Restore the window that was active when the notification launched
    if ($foregroundAtLaunch -ne [IntPtr]::Zero) {
        [WinAPI]::SetForegroundWindow($foregroundAtLaunch) | Out-Null
    }
    Remove-Slot
    $window.Close()
})

# Fade in
$window.Opacity = 0
$window.Add_Loaded({
    $fadeIn = New-Object System.Windows.Media.Animation.DoubleAnimation(0, 1, (New-Object System.Windows.Duration([TimeSpan]::FromMilliseconds(200))))
    $window.BeginAnimation([System.Windows.Window]::OpacityProperty, $fadeIn)
})

# Reflow timer
if ($SlotDir) {
    $reflowTimer = New-Object System.Windows.Threading.DispatcherTimer
    $reflowTimer.Interval = [TimeSpan]::FromMilliseconds(500)
    $reflowTimer.Add_Tick({
        for ($s = 0; $s -lt $script:SlotIndex; $s++) {
            if (-not (Test-SlotOccupied $s)) {
                Move-ToSlot $s
                $newOrigin = Get-NotificationOrigin $script:SlotIndex
                $window.Left = $newOrigin.X
                $window.Top = $newOrigin.Y
                break
            }
        }
    })
    $reflowTimer.Start()
}

# Auto-dismiss timer
if ($DismissSecs -gt 0) {
    $dismissTimer = New-Object System.Windows.Threading.DispatcherTimer
    $dismissTimer.Interval = [TimeSpan]::FromSeconds($DismissSecs)
    $dismissTimer.Add_Tick({
        $fadeOut = New-Object System.Windows.Media.Animation.DoubleAnimation(1, 0, (New-Object System.Windows.Duration([TimeSpan]::FromMilliseconds(300))))
        $fadeOut.Add_Completed({
            Remove-Slot
            $window.Close()
        })
        $window.BeginAnimation([System.Windows.Window]::OpacityProperty, $fadeOut)
        $dismissTimer.Stop()
    })
    $dismissTimer.Start()
}

$app = New-Object System.Windows.Application
$app.Run($window) | Out-Null
