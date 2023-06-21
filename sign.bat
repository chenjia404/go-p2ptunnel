@echo off & setlocal enabledelayedexpansion
for /r .\dist  %%i in (*.gz) do ( gpg --armor --detach-sign  %%i )
for /r .\dist  %%i in (*.zip) do ( gpg --armor --detach-sign  %%i )
pause