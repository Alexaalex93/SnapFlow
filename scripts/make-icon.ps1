# Generates assets/icon.ico (multi-resolution) using .NET System.Drawing.
# No external tools required.  Run from repo root:  .\scripts\make-icon.ps1

param([string]$Out = "assets\icon.ico")

Add-Type -AssemblyName System.Drawing

# ── Helper: rounded rect path ─────────────────────────────────────────────────
function Add-RoundRect([System.Drawing.Drawing2D.GraphicsPath]$path,
                       [float]$x, [float]$y, [float]$w, [float]$h, [float]$r) {
    $d = $r * 2
    $path.AddArc($x,           $y,           $d, $d, 180, 90)
    $path.AddArc($x + $w - $d, $y,           $d, $d, 270, 90)
    $path.AddArc($x + $w - $d, $y + $h - $d, $d, $d,   0, 90)
    $path.AddArc($x,           $y + $h - $d, $d, $d,  90, 90)
    $path.CloseFigure()
}

# ── Draw icon at given size ───────────────────────────────────────────────────
function New-SnapFlowBitmap([int]$size) {
    $s   = [float]$size
    $bmp = New-Object System.Drawing.Bitmap($size, $size,
               [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $g   = [System.Drawing.Graphics]::FromImage($bmp)
    $g.SmoothingMode    = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
    $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic

    $cr  = $s * 0.18   # background corner radius

    # Background gradient (blue)
    $gradTop = [System.Drawing.Color]::FromArgb(255, 0, 120, 215)
    $gradBot = [System.Drawing.Color]::FromArgb(255, 0,  78, 146)
    $bgBrush = New-Object System.Drawing.Drawing2D.LinearGradientBrush(
        [System.Drawing.PointF]::new(0, 0),
        [System.Drawing.PointF]::new(0, $s),
        $gradTop, $gradBot)

    $bgPath = New-Object System.Drawing.Drawing2D.GraphicsPath
    Add-RoundRect $bgPath 0 0 $s $s $cr
    $g.FillPath($bgBrush, $bgPath)

    # Inner padding
    $pad  = $s * 0.14
    $iw   = $s - $pad * 2       # inner width/height
    $rc   = [float][Math]::Max(2, $s * 0.05)   # small rect corner radius
    $gap  = [float][Math]::Max(1, $s * 0.04)   # gap between panels
    $divX = $pad + $iw * 0.54   # vertical divider X position

    $white = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(255, 255, 255, 255))
    $light = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(170, 255, 255, 255))

    # Left panel — solid white (snapped window)
    $lp = New-Object System.Drawing.Drawing2D.GraphicsPath
    Add-RoundRect $lp $pad $pad ($divX - $pad - $gap) $iw $rc
    $g.FillPath($white, $lp)

    # Top-right panel — semi-transparent
    $midY = $pad + $iw * 0.5 + $gap * 0.5
    $trp  = New-Object System.Drawing.Drawing2D.GraphicsPath
    Add-RoundRect $trp ($divX + $gap) $pad ($s - $pad - $divX - $gap) ($midY - $pad - $gap) $rc
    $g.FillPath($light, $trp)

    # Bottom-right panel — semi-transparent
    $brp = New-Object System.Drawing.Drawing2D.GraphicsPath
    Add-RoundRect $brp ($divX + $gap) $midY ($s - $pad - $divX - $gap) ($s - $pad - $midY) $rc
    $g.FillPath($light, $brp)

    $g.Dispose(); $bgBrush.Dispose(); $white.Dispose(); $light.Dispose()
    return $bmp
}

# ── ICO writer (PNG-in-ICO, Windows Vista+) ───────────────────────────────────
function Save-Ico([System.Drawing.Bitmap[]]$bitmaps, [string]$filePath) {
    $imageData = $bitmaps | ForEach-Object {
        $ms = New-Object System.IO.MemoryStream
        $_.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
        , $ms.ToArray()
    }
    $cnt = $bitmaps.Length
    $ms  = New-Object System.IO.MemoryStream
    $bw  = New-Object System.IO.BinaryWriter($ms)

    $bw.Write([uint16]0); $bw.Write([uint16]1); $bw.Write([uint16]$cnt)

    $offset = 6 + $cnt * 16
    for ($i = 0; $i -lt $cnt; $i++) {
        $wb = if ($bitmaps[$i].Width  -eq 256) { [byte]0 } else { [byte]$bitmaps[$i].Width }
        $hb = if ($bitmaps[$i].Height -eq 256) { [byte]0 } else { [byte]$bitmaps[$i].Height }
        $bw.Write($wb); $bw.Write($hb); $bw.Write([byte]0); $bw.Write([byte]0)
        $bw.Write([uint16]1); $bw.Write([uint16]32)
        $bw.Write([uint32]$imageData[$i].Length)
        $bw.Write([uint32]$offset)
        $offset += $imageData[$i].Length
    }
    foreach ($d in $imageData) { $bw.Write($d) }
    $bw.Flush()

    $dir = Split-Path $filePath
    if ($dir -and -not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir | Out-Null }
    [System.IO.File]::WriteAllBytes($filePath, $ms.ToArray())
    $bw.Dispose(); $ms.Dispose()
}

# ── Generate and save ─────────────────────────────────────────────────────────
$sizes   = @(16, 24, 32, 48, 64, 128, 256)
$bitmaps = $sizes | ForEach-Object { New-SnapFlowBitmap $_ }

Save-Ico $bitmaps $Out

foreach ($bmp in $bitmaps) { $bmp.Dispose() }

$abs = (Resolve-Path $Out -ErrorAction SilentlyContinue).Path
if (-not $abs) { $abs = $Out }
Write-Host "Icon saved: $abs  ($($sizes -join ', ') px)" -ForegroundColor Green
