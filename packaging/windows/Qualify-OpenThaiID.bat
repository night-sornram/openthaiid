@echo off
setlocal
cd /d "%~dp0"
echo OpenThaiID Friday qualify
echo Never prints citizen ID, names, address, or photo bytes.
echo One JSON report: "%~dp0openthaiid-qualify.json"
echo See docs\friday-checklist.md and docs\hardware-qualification.md
echo Type pass, fail, or skip only if you pass -operator.
echo.
openthaiid.exe qualify -report "%~dp0openthaiid-qualify.json" %*
echo.
echo report: %~dp0openthaiid-qualify.json
pause
