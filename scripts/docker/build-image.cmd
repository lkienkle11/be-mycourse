@echo off
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0build-image.ps1" %*
exit /b %ERRORLEVEL%
