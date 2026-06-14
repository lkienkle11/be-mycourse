@echo off
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0compose-down.ps1" %*
exit /b %ERRORLEVEL%
