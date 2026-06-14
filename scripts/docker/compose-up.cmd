@echo off
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0compose-up.ps1" %*
exit /b %ERRORLEVEL%
