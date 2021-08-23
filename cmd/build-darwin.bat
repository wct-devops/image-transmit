SET CGO_ENABLED=0
SET GOOS=darwin
SET GOARCH=amd64
SET GOARM=7
go build -ldflags "-s -w" -o image-transmit-darwin
rem ..\upx image-transmit-darwin
SET GOOS=
SET GOARCH=