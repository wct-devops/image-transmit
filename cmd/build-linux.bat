SET CGO_ENABLED=0
SET GOOS=linux
SET GOARCH=amd64
go build -ldflags "-s -w" -o image-transmit
rem ..\upx image-transmit
SET GOOS=
SET GOARCH=