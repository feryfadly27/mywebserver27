@echo off
echo Menyinkronkan domain ke hosts Windows...
echo. >> "C:\windows\System32\drivers\etc\hosts"
echo # Added by MyLokalWebserver >> "C:\windows\System32\drivers\etc\hosts"
echo 127.0.0.1  kasir.test >> "C:\windows\System32\drivers\etc\hosts"
echo 127.0.0.1  www.kasir.test >> "C:\windows\System32\drivers\etc\hosts"
echo 127.0.0.1  newproject.test >> "C:\windows\System32\drivers\etc\hosts"
echo 127.0.0.1  www.newproject.test >> "C:\windows\System32\drivers\etc\hosts"
echo 127.0.0.1  testkasir.local >> "C:\windows\System32\drivers\etc\hosts"
echo 127.0.0.1  www.testkasir.local >> "C:\windows\System32\drivers\etc\hosts"
ipconfig /flushdns >nul 2>&1
echo Selesai.
