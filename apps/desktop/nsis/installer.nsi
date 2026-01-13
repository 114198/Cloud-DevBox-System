; Cloud DevBox NSIS Installer Script
; This script creates a modern installer with multi-language support

;--------------------------------
; Includes

!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "LogicLib.nsh"
!include "WinVer.nsh"
!include "x64.nsh"

;--------------------------------
; General Configuration

Name "Cloud DevBox"
OutFile "CloudDevBox-Setup.exe"
Unicode True
RequestExecutionLevel user
InstallDir "$LOCALAPPDATA\Cloud DevBox"
InstallDirRegKey HKCU "Software\CloudDevBox" "InstallPath"

; Version information
!define PRODUCT_NAME "Cloud DevBox"
!define PRODUCT_VERSION "${VERSION}"
!define PRODUCT_PUBLISHER "Cloud DevBox Team"
!define PRODUCT_WEB_SITE "https://devbox.io"
!define PRODUCT_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\CloudDevBox"
!define PRODUCT_UNINST_ROOT_KEY "HKCU"

; Compression
SetCompressor /SOLID lzma
SetCompressorDictSize 64

;--------------------------------
; Interface Settings

!define MUI_ABORTWARNING
!define MUI_ICON "icons\icon.ico"
!define MUI_UNICON "icons\icon.ico"
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_BITMAP "nsis\header.bmp"
!define MUI_WELCOMEFINISHPAGE_BITMAP "nsis\sidebar.bmp"
!define MUI_UNWELCOMEFINISHPAGE_BITMAP "nsis\sidebar.bmp"

; Welcome page
!define MUI_WELCOMEPAGE_TITLE "$(WelcomeTitle)"
!define MUI_WELCOMEPAGE_TEXT "$(WelcomeText)"

; Finish page
!define MUI_FINISHPAGE_RUN "$INSTDIR\Cloud DevBox.exe"
!define MUI_FINISHPAGE_RUN_TEXT "$(LaunchApp)"
!define MUI_FINISHPAGE_SHOWREADME ""
!define MUI_FINISHPAGE_SHOWREADME_NOTCHECKED
!define MUI_FINISHPAGE_SHOWREADME_TEXT "$(CreateDesktopShortcut)"
!define MUI_FINISHPAGE_SHOWREADME_FUNCTION CreateDesktopShortcut

;--------------------------------
; Pages

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\..\LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_WELCOME
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

;--------------------------------
; Languages

!insertmacro MUI_LANGUAGE "English"
!insertmacro MUI_LANGUAGE "SimpChinese"

; English strings
LangString WelcomeTitle ${LANG_ENGLISH} "Welcome to Cloud DevBox Setup"
LangString WelcomeText ${LANG_ENGLISH} "This wizard will guide you through the installation of Cloud DevBox.$\r$\n$\r$\nCloud DevBox is a powerful desktop client for managing cloud development environments.$\r$\n$\r$\nClick Next to continue."
LangString LaunchApp ${LANG_ENGLISH} "Launch Cloud DevBox"
LangString CreateDesktopShortcut ${LANG_ENGLISH} "Create Desktop Shortcut"
LangString UninstallPrevious ${LANG_ENGLISH} "A previous version of Cloud DevBox is installed. Do you want to uninstall it first?"
LangString UninstallSuccess ${LANG_ENGLISH} "Cloud DevBox has been successfully uninstalled."

; Chinese strings
LangString WelcomeTitle ${LANG_SIMPCHINESE} "欢迎使用 Cloud DevBox 安装向导"
LangString WelcomeText ${LANG_SIMPCHINESE} "此向导将引导您完成 Cloud DevBox 的安装。$\r$\n$\r$\nCloud DevBox 是一款强大的桌面客户端，用于管理云端开发环境。$\r$\n$\r$\n点击"下一步"继续。"
LangString LaunchApp ${LANG_SIMPCHINESE} "启动 Cloud DevBox"
LangString CreateDesktopShortcut ${LANG_SIMPCHINESE} "创建桌面快捷方式"
LangString UninstallPrevious ${LANG_SIMPCHINESE} "检测到已安装的旧版本 Cloud DevBox。是否先卸载？"
LangString UninstallSuccess ${LANG_SIMPCHINESE} "Cloud DevBox 已成功卸载。"

;--------------------------------
; Installer Functions

Function .onInit
  ; Check Windows version (require Windows 10 or later)
  ${IfNot} ${AtLeastWin10}
    MessageBox MB_OK|MB_ICONSTOP "Cloud DevBox requires Windows 10 or later."
    Abort
  ${EndIf}
  
  ; Check for previous installation
  ReadRegStr $0 HKCU "Software\CloudDevBox" "InstallPath"
  ${If} $0 != ""
    MessageBox MB_YESNO|MB_ICONQUESTION "$(UninstallPrevious)" IDYES uninst IDNO done
    uninst:
      ExecWait '"$0\Uninstall.exe" /S'
    done:
  ${EndIf}
  
  ; Language selection
  !insertmacro MUI_LANGDLL_DISPLAY
FunctionEnd

Function CreateDesktopShortcut
  CreateShortCut "$DESKTOP\Cloud DevBox.lnk" "$INSTDIR\Cloud DevBox.exe"
FunctionEnd

;--------------------------------
; Installer Section

Section "Install"
  SetOutPath "$INSTDIR"
  
  ; Main application files
  File /r "${MAINBINARYPATH}\*.*"
  
  ; Create Start Menu shortcuts
  CreateDirectory "$SMPROGRAMS\Cloud DevBox"
  CreateShortCut "$SMPROGRAMS\Cloud DevBox\Cloud DevBox.lnk" "$INSTDIR\Cloud DevBox.exe"
  CreateShortCut "$SMPROGRAMS\Cloud DevBox\Uninstall.lnk" "$INSTDIR\Uninstall.exe"
  
  ; Write registry keys
  WriteRegStr HKCU "Software\CloudDevBox" "InstallPath" "$INSTDIR"
  WriteRegStr HKCU "Software\CloudDevBox" "Version" "${PRODUCT_VERSION}"
  
  ; Protocol handler: devbox://
  WriteRegStr HKCU "Software\Classes\devbox" "" "URL:DevBox Protocol"
  WriteRegStr HKCU "Software\Classes\devbox" "URL Protocol" ""
  WriteRegStr HKCU "Software\Classes\devbox\shell\open\command" "" '"$INSTDIR\Cloud DevBox.exe" "%1"'
  
  ; File association: .devbox
  WriteRegStr HKCU "Software\Classes\.devbox" "" "CloudDevBox.Environment"
  WriteRegStr HKCU "Software\Classes\CloudDevBox.Environment" "" "DevBox Environment Configuration"
  WriteRegStr HKCU "Software\Classes\CloudDevBox.Environment\DefaultIcon" "" "$INSTDIR\Cloud DevBox.exe,0"
  WriteRegStr HKCU "Software\Classes\CloudDevBox.Environment\shell\open\command" "" '"$INSTDIR\Cloud DevBox.exe" "%1"'
  
  ; Uninstaller
  WriteUninstaller "$INSTDIR\Uninstall.exe"
  
  ; Add/Remove Programs entry
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "DisplayName" "${PRODUCT_NAME}"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "UninstallString" "$INSTDIR\Uninstall.exe"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "DisplayIcon" "$INSTDIR\Cloud DevBox.exe"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "DisplayVersion" "${PRODUCT_VERSION}"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "URLInfoAbout" "${PRODUCT_WEB_SITE}"
  WriteRegDWORD ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "NoModify" 1
  WriteRegDWORD ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "NoRepair" 1
  
  ; Calculate installed size
  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  IntFmt $0 "0x%08X" $0
  WriteRegDWORD ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "EstimatedSize" "$0"
SectionEnd

;--------------------------------
; Uninstaller Section

Section "Uninstall"
  ; Kill running process
  nsExec::ExecToLog 'taskkill /F /IM "Cloud DevBox.exe"'
  Sleep 1000
  
  ; Remove files
  RMDir /r "$INSTDIR"
  
  ; Remove shortcuts
  Delete "$DESKTOP\Cloud DevBox.lnk"
  RMDir /r "$SMPROGRAMS\Cloud DevBox"
  
  ; Remove registry keys
  DeleteRegKey HKCU "Software\CloudDevBox"
  DeleteRegKey HKCU "Software\Classes\devbox"
  DeleteRegKey HKCU "Software\Classes\.devbox"
  DeleteRegKey HKCU "Software\Classes\CloudDevBox.Environment"
  DeleteRegKey ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}"
  
  MessageBox MB_OK "$(UninstallSuccess)"
SectionEnd
