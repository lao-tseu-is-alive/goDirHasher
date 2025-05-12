#!/bin/bash
cd /home/cgil/cgdev/golang/goDirHasher/releases/
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o goDirHasher-linux-amd64 ./cmd/goDirHasher/main.go 
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o goDirHasher-linux-arm64 ./cmd/goDirHasher/main.go 
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o goDirHasher-windows-amd64.exe ./cmd/goDirHasher/main.go
GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o goDirHasher-windows-arm64.exe ./cmd/goDirHasher/main.go
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o goDirHasher-darwin-amd64 ./cmd/goDirHasher/main.go
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o goDirHasher-darwin-arm64 ./cmd/goDirHasher/main.go 
tar -czvf goDirHasher-linux-amd64.tar.gz goDirHasher-linux-amd64
tar -czvf goDirHasher-linux-arm64.tar.gz goDirHasher-linux-arm64
tar -czvf goDirHasher-darwin-amd64.tar.gz goDirHasher-darwin-amd64
tar -czvf goDirHasher-darwin-arm64.tar.gz goDirHasher-darwin-arm64
zip goDirHasher-windows-amd64.zip goDirHasher-windows-amd64.exe
zip goDirHasher-windows-arm64.zip goDirHasher-windows-arm64.exe
