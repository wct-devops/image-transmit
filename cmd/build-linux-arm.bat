SET CGO_ENABLED=0
SET GOOS=linux
SET GOARCH=arm64
SET GOARM=7
go build -ldflags "-s -w" -o image-transmit-arm
..\upx image-transmit-arm
SET GOOS=
SET GOARCH=