rem rsrc.exe -manifest main.manifest -o main.syso -ico main.ico
go build -ldflags "-s -w -H windowsgui" -o image-transmit.exe
rem go build -ldflags "-s -w" -o image-transmit.exe
rem go build
rem ..\upx image-transmit.exe