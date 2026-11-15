Unicode true

####
## 本文件由 Wails 模板（v2.17.0）定制而来，wails build -nsis 不会覆盖本文件。
## 相对 wails 官方模板的定制点（S4，docs/47）：
##   1. SetCompressor /SOLID lzma —— 物料 JSON 压缩率高，安装器体积显著缩小
##   2. 安装 Section 追加「物料随装」：仓库根 data/ 顶层物料 + 康熙笔画表
##      打进安装目录 data/，首跑播种进 SQLite，此后运行期零 JSON 依赖（ADR-008）
##   3. 卸载 Section 选择性删除：保留 namer.db / *.db-wal / *.db-shm / logs/
##      ——防止卸载抹掉用户收藏、历史与自学习库
##   4. INFO_PRODUCTNAME 固定 "NameMaster"——避免中文+空格进安装路径/快捷方式
##   5. 简体中文为第一安装界面语言
##
## 手工构建（脱离 wails，CWD 必须为本目录）：
##   makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\namer-desktop.exe project.nsi
####
## 定义先于 wails_tools.nsh：wails_tools 对已定义项不覆盖
####
!define INFO_PRODUCTNAME "NameMaster"

!include "wails_tools.nsh"

# 固实 LZMA 压缩：物料 JSON（~63MB）压缩后安装器体积可控
SetCompressor /SOLID lzma

# The version information for this two must consist of 4 parts
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} 安装程序"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# Enable HiDPI support. https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
# !define MUI_WELCOMEFINISHPAGE_BITMAP "resources\leftimage.bmp" #Include this to add a bitmap on the left side of the Welcome Page. Must be a size of 164x314
!define MUI_FINISHPAGE_NOAUTOCLOSE # Wait on the INSTFILES page so the user can take a look into the details of the installation steps
!define MUI_ABORTWARNING # This will warn the user if they exit from the installer.

!insertmacro MUI_PAGE_WELCOME # Welcome to the installer page.
# !insertmacro MUI_PAGE_LICENSE "resources\eula.txt" # Adds a EULA page to the installer
!insertmacro MUI_PAGE_DIRECTORY # In which folder install page.
!insertmacro MUI_PAGE_INSTFILES # Installing page.
!insertmacro MUI_PAGE_FINISH # Finished installation page.

!insertmacro MUI_UNPAGE_INSTFILES # Uinstalling page

# 简体中文优先（第一语言为系统语言不匹配时的兜底），英文随其后
!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_LANGUAGE "English"

## The following two statements can be used to sign the installer and the uninstaller. The path to the binaries are provided in %1
#!uninstfinalize 'signtool --file "%1"'
#!finalize 'signtool --file "%1"'

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe" # Name of the installer's file.
!ifdef WAILS_INSTALL_SCOPE
  !if "${WAILS_INSTALL_SCOPE}" == "user"
    InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
  !else
    InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
  !endif
!else
  InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
!endif # Default installing folder ($PROGRAMFILES is Program Files folder).
ShowInstDetails show # This will always show the installation details.

Function .onInit
   !insertmacro wails.checkArchitecture
FunctionEnd

Section
    !insertmacro wails.setShellContext

    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR

    !insertmacro wails.files

    # --- 物料随装（S4）---
    # 仓库根 data/ 顶层物料 + 康熙笔画表打进安装目录 data/。
    # /nonfatal：构建机缺少物料时仅告警不阻断（CI 场景物料须提前就位）。
    SetOutPath "$INSTDIR\data"
    File /nonfatal "..\..\..\..\data\*.json"
    File /nonfatal "..\..\..\..\data\materials.lock"
    SetOutPath "$INSTDIR\data\raw"
    File /nonfatal "..\..\..\..\data\raw\kangxi-strokecount.csv"
    SetOutPath $INSTDIR

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols

    !insertmacro wails.writeUninstaller
SectionEnd

Section "uninstall"
    !insertmacro wails.setShellContext

    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}" # Remove the WebView2 DataPath

    # --- 选择性卸载（S4）：保留用户数据，防止卸载抹掉收藏/历史/自学习库 ---
    # 程序本体与物料全部删除；namer.db、*.db-wal/-shm、logs/ 原样保留，
    # 重装后首跑播种自动跳过（库非空），历史数据无缝延续。
    Delete "$INSTDIR\${PRODUCT_EXECUTABLE}"
    Delete "$INSTDIR\data\*.json"
    Delete "$INSTDIR\data\materials.lock"
    RMDir /r "$INSTDIR\data\raw"
    RMDir "$INSTDIR\data"
    RMDir $INSTDIR # 仅当目录为空（无用户数据残留）时生效

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    !insertmacro wails.deleteUninstaller
SectionEnd
