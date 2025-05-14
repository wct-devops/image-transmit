rem rsrc.exe -manifest main.manifest -o main.syso -ico main.ico
SET GOOS=windows
SET GOARCH=386
REM SET GOROOT=C:\gox86
REM C:\gox86\bin\go build -ldflags "-s -w -H windowsgui" -o image-transmit-x86.exe
go build -ldflags "-s -w -H windowsgui" -o image-transmit-windows-x86.exe
rem go build -ldflags "-s -w" -o image-transmit.exe
rem go build
rem ..\upx image-transmit.exe
SET GOOS=
SET GOARCH=
SET GOROOT=