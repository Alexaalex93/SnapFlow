; SnapFlow Installer — Inno Setup 6
; Build with:  iscc scripts\installer.iss
; (assumes snapflow.exe is already built and placed in the repo root)

#define AppName    "SnapFlow"
#define AppVersion "1.0.0"
#define AppExe     "snapflow.exe"
#define AppURL     "https://github.com/yourusername/snapflow"

[Setup]
AppId                   = {{A1B2C3D4-E5F6-7890-ABCD-EF1234567890}
AppName                 = {#AppName}
AppVersion              = {#AppVersion}
AppVerName              = {#AppName} {#AppVersion}
AppPublisherURL         = {#AppURL}
AppSupportURL           = {#AppURL}/issues
DefaultDirName          = {localappdata}\{#AppName}
DefaultGroupName        = {#AppName}
DisableProgramGroupPage = yes
OutputDir               = dist
OutputBaseFilename      = SnapFlow-Setup
SetupIconFile           = assets\icon.ico
UninstallDisplayIcon    = {app}\{#AppExe}
Compression             = lzma2/ultra64
SolidCompression        = yes
; No admin rights needed — installs per-user to %LOCALAPPDATA%
PrivilegesRequired      = lowest
PrivilegesRequiredOverridesAllowed = dialog
WizardStyle             = modern
WizardResizable         = no
; Allow silent install:  SnapFlow-Setup.exe /VERYSILENT /SUPPRESSMSGBOXES
DisableWelcomePage      = no
CloseApplications       = yes
CloseApplicationsFilter = *\{#AppExe}
RestartApplications     = no
ArchitecturesInstallIn64BitMode = x64compatible

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "spanish"; MessagesFile: "compiler:Languages\Spanish.isl"

[Files]
Source: "{#AppExe}"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#AppName}";          Filename: "{app}\{#AppExe}"
Name: "{group}\Uninstall {#AppName}"; Filename: "{uninstallexe}"
Name: "{commondesktop}\{#AppName}";  Filename: "{app}\{#AppExe}"; Tasks: desktopicon

[Tasks]
Name: desktopicon; Description: "Create a &desktop shortcut";           Flags: unchecked
Name: autostart;   Description: "&Start SnapFlow automatically at login"; Flags: checked

[Registry]
; Autostart entry — created only when the task is selected
Root: HKCU; Subkey: "SOFTWARE\Microsoft\Windows\CurrentVersion\Run"; \
  ValueType: string; ValueName: "{#AppName}"; ValueData: """{app}\{#AppExe}"""; \
  Flags: uninsdeletevalue; Tasks: autostart

[Run]
; Launch after install
Filename: "{app}\{#AppExe}"; \
  Description: "Launch {#AppName} now"; \
  Flags: postinstall nowait skipifsilent

[UninstallRun]
; Kill before uninstalling
Filename: "taskkill.exe"; Parameters: "/F /IM {#AppExe}"; \
  Flags: runhidden; RunOnceId: "KillSnapFlow"

[Code]
// Remove the autostart key unconditionally on uninstall
// (covers the case where the user re-enabled autostart outside the installer)
procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usPostUninstall then
    RegDeleteValue(HKCU, 'SOFTWARE\Microsoft\Windows\CurrentVersion\Run', '{#AppName}');
end;
