; SnapFlow Installer — Inno Setup 6
; Build from repo root:  iscc scripts\installer.iss
; Requires snapflow.exe to be in ..\dist\ relative to this script.

#define AppName    "SnapFlow"
#define AppVersion "1.0.0"
#define AppExe     "snapflow.exe"
#define AppURL     "https://github.com/Alexaalex93/SnapFlow"

; Source path relative to this .iss file (scripts/)
; ..\dist\snapflow.exe  →  repo-root\dist\snapflow.exe
#define SourceExe "..\dist\" + AppExe

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
OutputDir               = ..\dist
OutputBaseFilename      = SnapFlow-Setup
SetupIconFile           = ..\assets\icon.ico
UninstallDisplayIcon    = {app}\{#AppExe}
Compression             = lzma2/ultra64
SolidCompression        = yes
PrivilegesRequired      = lowest
PrivilegesRequiredOverridesAllowed = dialog
WizardStyle             = modern
WizardResizable         = no
CloseApplications       = yes
CloseApplicationsFilter = *\{#AppExe}
RestartApplications     = no
ArchitecturesInstallIn64BitMode = x64compatible

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "spanish"; MessagesFile: "compiler:Languages\Spanish.isl"

[Files]
Source: "{#SourceExe}"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#AppName}";           Filename: "{app}\{#AppExe}"
Name: "{group}\Uninstall {#AppName}"; Filename: "{uninstallexe}"
Name: "{commondesktop}\{#AppName}";   Filename: "{app}\{#AppExe}"; Tasks: desktopicon

[Tasks]
Name: desktopicon; Description: "Create a &desktop shortcut";            Flags: unchecked
Name: autostart;   Description: "&Start SnapFlow automatically at login"

[Registry]
Root: HKCU; Subkey: "SOFTWARE\Microsoft\Windows\CurrentVersion\Run"; \
  ValueType: string; ValueName: "{#AppName}"; ValueData: """{app}\{#AppExe}"""; \
  Flags: uninsdeletevalue; Tasks: autostart

[Run]
Filename: "{app}\{#AppExe}"; \
  Description: "Launch {#AppName} now"; \
  Flags: postinstall nowait skipifsilent

[UninstallRun]
Filename: "taskkill.exe"; Parameters: "/F /IM {#AppExe}"; \
  Flags: runhidden; RunOnceId: "KillSnapFlow"

[Code]
procedure CurStepChanged(CurStep: TSetupStep);
var
  ResultCode: Integer;
begin
  if CurStep = ssInstall then
    Exec('taskkill.exe', '/F /IM {#AppExe}', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usPostUninstall then
    RegDeleteValue(HKCU, 'SOFTWARE\Microsoft\Windows\CurrentVersion\Run', '{#AppName}');
end;
