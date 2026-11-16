Unicode true

####
## Wails v3 NSIS 安装脚本（docs/48 阶段 4）
## 由 v2 的 build/windows/installer/project.nsi 迁移到 v3 官方路径
## build/windows/nsis/；wails_tools.nsh 由 wails3 生成（v3 宏名与 v2 一致）。
##
## 相对 v3 官方模板的定制点：
##   1. SetCompressor /SOLID lzma —— 物料 JSON 压缩率高，安装器体积显著缩小
##   2. 安装 Section 只随装一份种子库 data/namer.db（ADR-010「A + 单库」）：
##      零 JSON 分发、零首跑播种（此前首次启动约 80 秒无窗口），库即唯一真源
##   3. 卸载 Section 选择性删除：保留 namer.db / *.db-wal / *.db-shm / logs/
##      ——防止卸载抹掉用户收藏、历史与自学习库
##   4. INFO_PRODUCTNAME / PRODUCT_EXECUTABLE 固定——避免中文+空格进安装路径/快捷方式
##   5. 简体中文为第一安装界面语言；文件须带 UTF-8 BOM（NSIS 3.05 中文坑）
##
## 手工构建（CWD 必须为本目录）：
##   makensis -DWAILS_INSTALL_SCOPE=user -DREQUEST_EXECUTION_LEVEL=user ^
##            -DARG_WAILS_AMD64_BINARY=..\..\..\bin\namer-desktop.exe project.nsi
## 产物：..\..\..\bin\namer-amd64-installer.exe
##
## 定义先于 wails_tools.nsh：wails_tools 对已定义项不覆盖
####
!define INFO_PROJECTNAME "namer"
!define INFO_PRODUCTNAME "NameMaster"
!define INFO_COMPANYNAME "bliubiu"
!define INFO_COPYRIGHT "Copyright (c) 2026"
!define INFO_PRODUCTVERSION "1.0.0"
!define PRODUCT_EXECUTABLE "namer-desktop.exe"

!include "wails_tools.nsh"

# 固实 LZMA 压缩：种子库约 22MB，固实压缩后安装器体积可控
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
!define MUI_FINISHPAGE_NOAUTOCLOSE # Wait on the INSTFILES page so the user can take a look into the details of the installation steps
!define MUI_ABORTWARNING # This will warn the user if they exit from the installer.

!insertmacro MUI_PAGE_WELCOME # Welcome to the installer page.
!insertmacro MUI_PAGE_DIRECTORY # In which folder install page.
!insertmacro MUI_PAGE_INSTFILES # Installing page.
!insertmacro MUI_PAGE_FINISH # Finished installation page.

!insertmacro MUI_UNPAGE_INSTFILES # Uninstalling page

# 简体中文优先（第一语言为系统语言不匹配时的兜底），英文随其后
!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_LANGUAGE "English"

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe" # Name of the installer's file.
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

    # --- 种子库随装（ADR-010 单库路线）---
    # 只装一份 data/namer.db（约 22MB），不再随装 54.9MB JSON 物料：
    # 启动检测到库非空会跳过播种，首次启动不再有约 80 秒无窗口等待。
    # 不加 /nonfatal：种子库缺失是打包配置错误，必须让安装构建失败。
    SetOutPath "$INSTDIR\data"
    File "..\..\..\..\data\namer.db"
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

    # --- 选择性卸载：namer.db 同时是种子与用户数据，绝不能删 ---
    # 程序本体删除；旧版安装残留的 JSON / materials.lock / raw 顺带清理。
    # namer.db、*.db-wal/-shm、logs/ 原样保留，重装后收藏/历史无缝延续。
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
