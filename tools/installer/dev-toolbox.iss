; DevToolbox Windows 安装程序脚本（Inno Setup 6）
;
; 编译：  ISCC tools\installer\dev-toolbox.iss /DAppVersion=0.1.0 /DBinGui=<gui.exe> /DBinCli=<cli.exe>
; 说明：  纯用户级安装（无需管理员）；不修改 PATH、不做任何自动配置——
;         安装/扫描/切换都是产品内的显式用户动作（产品纪律）。
;         卸载只删除本目录，不触碰用户数据（运行时目录与身份档案）。

#ifndef AppVersion
#define AppVersion "0.0.0"
#endif
#ifndef BinGui
#define BinGui "..\..\dist\dev-toolbox-gui.exe"
#endif
#ifndef BinCli
#define BinCli "..\..\dist\dev-toolbox.exe"
#endif

[Setup]
AppId={{5F3E9A1B-8C42-4B7D-9A26-D14E0F5C7B31}
AppName=DevToolbox
AppVersion={#AppVersion}
AppPublisher=xyztony999
AppPublisherURL=https://github.com/xyztony999/dev-toolbox
AppUpdatesURL=https://github.com/xyztony999/dev-toolbox/releases
PrivilegesRequired=lowest
DefaultDirName={localappdata}\Programs\DevToolbox
UsePreviousAppDir=no
DirExistsWarning=no
DisableProgramGroupPage=yes
UninstallDisplayName=DevToolbox
UninstallDisplayIcon={app}\dev-toolbox.ico
SetupIconFile=..\..\internal\desktop\icon.ico
OutputDir=..\..\dist
OutputBaseFilename=dev-toolbox-gui-{#AppVersion}-windows-amd64-setup
ArchitecturesInstallIn64BitMode=x64compatible
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
MinVersion=10.0

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "创建桌面快捷方式"; GroupDescription: "附加选项："; Flags: unchecked

[Files]
Source: "{#BinGui}"; DestDir: "{app}"; DestName: "DevToolbox.exe"; Flags: ignoreversion
Source: "{#BinCli}"; DestDir: "{app}"; DestName: "dev-toolbox.exe"; Flags: ignoreversion
Source: "..\..\internal\desktop\icon.ico"; DestDir: "{app}"; DestName: "dev-toolbox.ico"; Flags: ignoreversion
Source: "..\..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\NOTICE"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\README.md"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\DevToolbox"; Filename: "{app}\DevToolbox.exe"; Comment: "开发环境与 Git 身份管理工具箱"; IconFilename: "{app}\dev-toolbox.ico"
Name: "{autodesktop}\DevToolbox"; Filename: "{app}\DevToolbox.exe"; Comment: "开发环境与 Git 身份管理工具箱"; IconFilename: "{app}\dev-toolbox.ico"; Tasks: desktopicon

[Run]
Filename: "{app}\DevToolbox.exe"; Description: "运行 DevToolbox"; WorkingDir: "{app}"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
Type: filesandordirs; Name: "{app}"
