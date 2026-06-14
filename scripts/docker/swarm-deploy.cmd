@echo off
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0swarm-deploy.ps1" %*
exit /b %ERRORLEVEL%
