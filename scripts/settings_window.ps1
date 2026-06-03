param(
    [Parameter(Mandatory = $true)]
    [string]$ConfigPath
)

Add-Type -AssemblyName PresentationFramework
Add-Type -AssemblyName PresentationCore
Add-Type -AssemblyName WindowsBase

if (-not (Test-Path -LiteralPath $ConfigPath)) {
    throw "Config file not found: $ConfigPath"
}

$cfg = (Get-Content -LiteralPath $ConfigPath -Raw) | ConvertFrom-Json
if (-not $cfg.gesture) { $cfg | Add-Member -NotePropertyName gesture -NotePropertyValue ([pscustomobject]@{}) }

function Get-Int([object]$v, [int]$d) {
    if ($null -eq $v) { return $d }
    try { return [int]$v } catch { return $d }
}
function Get-Str([object]$v, [string]$d) {
    if ($null -eq $v) { return $d }
    $s = [string]$v
    if ([string]::IsNullOrWhiteSpace($s)) { return $d }
    return $s
}

$edge = Get-Int $cfg.gesture.edge_threshold_px 46
$corner = Get-Int $cfg.gesture.corner_threshold_px 70
$hyst = Get-Int $cfg.gesture.hysteresis_padding_px 20
$upper = Get-Int $cfg.gesture.side_third_band_upper_pct 34
$lower = Get-Int $cfg.gesture.side_third_band_lower_pct 66
$leftMode = Get-Str $cfg.gesture.left_edge_mode "dynamic"
$rightMode = Get-Str $cfg.gesture.right_edge_mode "dynamic"
$topMode = Get-Str $cfg.gesture.top_edge_mode "top_thirds"
$bottomMode = Get-Str $cfg.gesture.bottom_edge_mode "half"
$license = Get-Str $cfg.pro_license_key ""

[xml]$xaml = @"
<Window xmlns="http://schemas.microsoft.com/winfx/2006/xaml/presentation"
        xmlns:x="http://schemas.microsoft.com/winfx/2006/xaml"
        Title="SnapFlow Settings"
        Width="980"
        Height="660"
        MinWidth="920"
        MinHeight="620"
        WindowStartupLocation="CenterScreen"
        Background="#0F172A"
        FontFamily="Segoe UI">
  <Grid Margin="18">
    <Border CornerRadius="22" Background="#F8FAFC" BorderBrush="#DCE3EC" BorderThickness="1">
      <Grid>
        <Grid.RowDefinitions>
          <RowDefinition Height="Auto"/>
          <RowDefinition Height="*"/>
          <RowDefinition Height="Auto"/>
        </Grid.RowDefinitions>

        <Border Background="#FFFFFF" BorderBrush="#E2E8F0" BorderThickness="0,0,0,1" Padding="20,16">
          <StackPanel>
            <TextBlock Text="SnapFlow Settings" FontSize="24" FontWeight="SemiBold" Foreground="#0F172A"/>
            <TextBlock Text="Adjust Snap Areas and gesture behavior" Margin="0,4,0,0" Foreground="#64748B" FontSize="13"/>
          </StackPanel>
        </Border>

        <Grid Grid.Row="1" Margin="18">
          <Grid.ColumnDefinitions>
            <ColumnDefinition Width="*"/>
            <ColumnDefinition Width="300"/>
          </Grid.ColumnDefinitions>

          <TabControl x:Name="MainTabs" Background="Transparent" BorderBrush="#E2E8F0">
            <TabItem Header="Snap Areas">
              <Grid Background="#F8FAFC" Margin="8">
                <Border Background="White" CornerRadius="14" BorderBrush="#E2E8F0" BorderThickness="1" Padding="18">
                  <Grid>
                    <Grid.RowDefinitions>
                      <RowDefinition Height="Auto"/>
                      <RowDefinition Height="Auto"/>
                      <RowDefinition Height="Auto"/>
                    </Grid.RowDefinitions>

                    <UniformGrid Grid.Row="0" Rows="2" Columns="2" Margin="0,0,0,18">
                      <StackPanel Margin="0,0,12,12">
                        <TextBlock Text="Left edge" Foreground="#475569" Margin="0,0,0,6"/>
                        <ComboBox x:Name="CbLeft" Height="34"/>
                      </StackPanel>
                      <StackPanel Margin="12,0,0,12">
                        <TextBlock Text="Right edge" Foreground="#475569" Margin="0,0,0,6"/>
                        <ComboBox x:Name="CbRight" Height="34"/>
                      </StackPanel>
                      <StackPanel Margin="0,12,12,0">
                        <TextBlock Text="Top edge" Foreground="#475569" Margin="0,0,0,6"/>
                        <ComboBox x:Name="CbTop" Height="34"/>
                      </StackPanel>
                      <StackPanel Margin="12,12,0,0">
                        <TextBlock Text="Bottom edge" Foreground="#475569" Margin="0,0,0,6"/>
                        <ComboBox x:Name="CbBottom" Height="34"/>
                      </StackPanel>
                    </UniformGrid>

                    <UniformGrid Grid.Row="1" Rows="1" Columns="5" Margin="0,0,0,12">
                      <StackPanel Margin="0,0,12,0">
                        <TextBlock Text="Edge px" Foreground="#475569" Margin="0,0,0,6"/>
                        <TextBox x:Name="TbEdge" Height="34"/>
                      </StackPanel>
                      <StackPanel Margin="0,0,12,0">
                        <TextBlock Text="Corner px" Foreground="#475569" Margin="0,0,0,6"/>
                        <TextBox x:Name="TbCorner" Height="34"/>
                      </StackPanel>
                      <StackPanel Margin="0,0,12,0">
                        <TextBlock Text="Hysteresis px" Foreground="#475569" Margin="0,0,0,6"/>
                        <TextBox x:Name="TbHyst" Height="34"/>
                      </StackPanel>
                      <StackPanel Margin="0,0,12,0">
                        <TextBlock Text="Upper %" Foreground="#475569" Margin="0,0,0,6"/>
                        <TextBox x:Name="TbUpper" Height="34"/>
                      </StackPanel>
                      <StackPanel>
                        <TextBlock Text="Lower %" Foreground="#475569" Margin="0,0,0,6"/>
                        <TextBox x:Name="TbLower" Height="34"/>
                      </StackPanel>
                    </UniformGrid>

                    <TextBlock Grid.Row="2" Foreground="#64748B" FontSize="12"
                               Text="Tip: increase Corner px if corners are hard to trigger. Dynamic modes require Pro."/>
                  </Grid>
                </Border>
              </Grid>
            </TabItem>
            <TabItem Header="General">
              <Grid Background="#F8FAFC" Margin="8">
                <Border Background="White" CornerRadius="14" BorderBrush="#E2E8F0" BorderThickness="1" Padding="18">
                  <StackPanel>
                    <TextBlock Text="Pro license key" Foreground="#475569" Margin="0,0,0,8"/>
                    <TextBox x:Name="TbLicense" Height="34"/>
                  </StackPanel>
                </Border>
              </Grid>
            </TabItem>
          </TabControl>

          <Border Grid.Column="1" Margin="16,0,0,0" Background="White" CornerRadius="14" BorderBrush="#E2E8F0" BorderThickness="1" Padding="14">
            <StackPanel>
              <TextBlock Text="Preview" FontWeight="SemiBold" Foreground="#0F172A"/>
              <Border Margin="0,10,0,0" Height="180" CornerRadius="12">
                <Border.Background>
                  <LinearGradientBrush StartPoint="0,0" EndPoint="1,1">
                    <GradientStop Color="#0B3B66" Offset="0"/>
                    <GradientStop Color="#0EA5E9" Offset="1"/>
                  </LinearGradientBrush>
                </Border.Background>
                <Grid>
                  <Border HorizontalAlignment="Left" VerticalAlignment="Top" Margin="10"
                          CornerRadius="10" Background="#AA0F172A" BorderBrush="#88BAE6FD" BorderThickness="1"
                          Padding="8,4">
                    <TextBlock Text="SNAPFLOW PREVIEW" Foreground="#DBEAFE" FontSize="11"/>
                  </Border>
                </Grid>
              </Border>
              <TextBlock Margin="0,12,0,0" Foreground="#64748B" FontSize="12" TextWrapping="Wrap"
                         Text="This preview watermark confirms SnapFlow is handling drag snapping, not the default Windows snap UI."/>
            </StackPanel>
          </Border>
        </Grid>

        <Border Grid.Row="2" BorderBrush="#E2E8F0" BorderThickness="0,1,0,0" Background="#FFFFFF" Padding="18,12">
          <Grid>
            <Grid.ColumnDefinitions>
              <ColumnDefinition Width="*"/>
              <ColumnDefinition Width="Auto"/>
              <ColumnDefinition Width="Auto"/>
            </Grid.ColumnDefinitions>
            <TextBlock x:Name="StatusText" VerticalAlignment="Center" Foreground="#334155"/>
            <Button x:Name="BtnClose" Grid.Column="1" Width="92" Height="34" Margin="0,0,10,0"
                    Content="Close" Background="#E2E8F0" BorderBrush="#CBD5E1"/>
            <Button x:Name="BtnSave" Grid.Column="2" Width="120" Height="34"
                    Content="Save Settings" Foreground="White" BorderBrush="#0284C7">
              <Button.Background>
                <LinearGradientBrush StartPoint="0,0" EndPoint="1,1">
                  <GradientStop Color="#0284C7" Offset="0"/>
                  <GradientStop Color="#0EA5E9" Offset="1"/>
                </LinearGradientBrush>
              </Button.Background>
            </Button>
          </Grid>
        </Border>
      </Grid>
    </Border>
  </Grid>
</Window>
"@

$reader = New-Object System.Xml.XmlNodeReader $xaml
$window = [Windows.Markup.XamlReader]::Load($reader)

$cbLeft = $window.FindName("CbLeft")
$cbRight = $window.FindName("CbRight")
$cbTop = $window.FindName("CbTop")
$cbBottom = $window.FindName("CbBottom")
$tbEdge = $window.FindName("TbEdge")
$tbCorner = $window.FindName("TbCorner")
$tbHyst = $window.FindName("TbHyst")
$tbUpper = $window.FindName("TbUpper")
$tbLower = $window.FindName("TbLower")
$tbLicense = $window.FindName("TbLicense")
$statusText = $window.FindName("StatusText")
$btnSave = $window.FindName("BtnSave")
$btnClose = $window.FindName("BtnClose")

function Add-ModeItems($combo, $items) {
    foreach ($entry in $items) {
        $item = New-Object System.Windows.Controls.ComboBoxItem
        $item.Content = $entry.label
        $item.Tag = $entry.value
        [void]$combo.Items.Add($item)
    }
}
function Set-ComboByValue($combo, [string]$value) {
    foreach ($item in $combo.Items) {
        if ($item.Tag -eq $value) {
            $combo.SelectedItem = $item
            return
        }
    }
    if ($combo.Items.Count -gt 0) { $combo.SelectedIndex = 0 }
}
function Get-ComboValue($combo) {
    if ($null -eq $combo.SelectedItem) { return "" }
    return [string]$combo.SelectedItem.Tag
}
function Parse-IntOrDefault([string]$text, [int]$min, [int]$max, [int]$defaultValue) {
    $out = 0
    if (-not [int]::TryParse($text, [ref]$out)) { return $defaultValue }
    if ($out -lt $min) { return $min }
    if ($out -gt $max) { return $max }
    return $out
}

Add-ModeItems $cbLeft @(
    @{label="Dynamic (third <-> two-thirds)"; value="dynamic"},
    @{label="Half"; value="half"},
    @{label="Third"; value="third"},
    @{label="Two-thirds"; value="two_thirds"}
)
Add-ModeItems $cbRight @(
    @{label="Dynamic (third <-> two-thirds)"; value="dynamic"},
    @{label="Half"; value="half"},
    @{label="Third"; value="third"},
    @{label="Two-thirds"; value="two_thirds"}
)
Add-ModeItems $cbTop @(
    @{label="Top thirds dynamic (1/3 <-> 2/3)"; value="top_thirds"},
    @{label="Half"; value="half"},
    @{label="Top third"; value="third"},
    @{label="Top two-thirds"; value="two_thirds"}
)
Add-ModeItems $cbBottom @(
    @{label="Half"; value="half"},
    @{label="Bottom third"; value="third"},
    @{label="Bottom two-thirds"; value="two_thirds"}
)

Set-ComboByValue $cbLeft $leftMode
Set-ComboByValue $cbRight $rightMode
Set-ComboByValue $cbTop $topMode
Set-ComboByValue $cbBottom $bottomMode
$tbEdge.Text = [string]$edge
$tbCorner.Text = [string]$corner
$tbHyst.Text = [string]$hyst
$tbUpper.Text = [string]$upper
$tbLower.Text = [string]$lower
$tbLicense.Text = $license

$btnClose.Add_Click({ $window.Close() })
$btnSave.Add_Click({
    $edgeValue = Parse-IntOrDefault $tbEdge.Text 12 180 46
    $cornerValue = Parse-IntOrDefault $tbCorner.Text 16 260 70
    $hystValue = Parse-IntOrDefault $tbHyst.Text 0 120 20
    $upperValue = Parse-IntOrDefault $tbUpper.Text 10 49 34
    $lowerValue = Parse-IntOrDefault $tbLower.Text 51 90 66
    if ($upperValue -ge $lowerValue) {
        $statusText.Text = "Upper % must be lower than Lower %."
        return
    }

    $cfg.version = 2
    $cfg.pro_license_key = $tbLicense.Text.Trim()
    $cfg.gesture.edge_threshold_px = $edgeValue
    $cfg.gesture.corner_threshold_px = $cornerValue
    $cfg.gesture.hysteresis_padding_px = $hystValue
    $cfg.gesture.side_third_band_upper_pct = $upperValue
    $cfg.gesture.side_third_band_lower_pct = $lowerValue
    $cfg.gesture.left_edge_mode = Get-ComboValue $cbLeft
    $cfg.gesture.right_edge_mode = Get-ComboValue $cbRight
    $cfg.gesture.top_edge_mode = Get-ComboValue $cbTop
    $cfg.gesture.bottom_edge_mode = Get-ComboValue $cbBottom

    $json = $cfg | ConvertTo-Json -Depth 8
    Set-Content -LiteralPath $ConfigPath -Value $json -Encoding UTF8
    $statusText.Text = "Settings saved. Restart SnapFlow to apply all changes."
})

[void]$window.ShowDialog()
