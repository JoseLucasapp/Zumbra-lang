@echo off
echo Building for Windows (vaja.exe)...
go build -o vaja.exe

echo Building for Linux (vaja-linux)...
set GOOS=linux
set GOARCH=amd64
go build -o vaja-linux

echo Building for macOS (vaja-macos)...
set GOOS=darwin
set GOARCH=amd64
go build -o vaja-macos

echo Build complete!
