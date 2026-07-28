Unicode true
Name "Zumbra Packaged App"
OutFile "/home/joselucasapp/projects/Zumbra-lang/code_examples/desktop_package/dist/packages/zumbra-packaged-app-1.0.0-windows-amd64-setup.exe"
InstallDir "$PROGRAMFILES64\Zumbra Packaged App"
RequestExecutionLevel admin
Icon "\home\joselucasapp\projects\Zumbra-lang\code_examples\desktop_package\assets\icon.ico"
UninstallIcon "\home\joselucasapp\projects\Zumbra-lang\code_examples\desktop_package\assets\icon.ico"

Page directory
Page instfiles
UninstPage uninstConfirm
UninstPage instfiles

Section "Install"
  SetOutPath "$INSTDIR"
  File /r "/home/joselucasapp/projects/Zumbra-lang/code_examples/desktop_package/dist/packages/zumbra-packaged-app-1.0.0-windows-amd64-portable\*"
  CreateDirectory "$SMPROGRAMS\Zumbra Packaged App"
  CreateShortcut "$SMPROGRAMS\Zumbra Packaged App\Zumbra Packaged App.lnk" "$INSTDIR\zumbra-packaged-app.exe"
  CreateShortcut "$DESKTOP\Zumbra Packaged App.lnk" "$INSTDIR\zumbra-packaged-app.exe"
  WriteUninstaller "$INSTDIR\uninstall.exe"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\dev.zumbra.packaged" "DisplayName" "Zumbra Packaged App"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\dev.zumbra.packaged" "DisplayVersion" "1.0.0"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\dev.zumbra.packaged" "Publisher" "Zumbra"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\dev.zumbra.packaged" "UninstallString" "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
  Delete "$DESKTOP\Zumbra Packaged App.lnk"
  RMDir /r "$SMPROGRAMS\Zumbra Packaged App"
  RMDir /r "$INSTDIR"
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\dev.zumbra.packaged"
SectionEnd
