.PHONY: default build mode upx doc

GIT_COMMIT=`git rev-list -1 HEAD`
BUILD_TIME=`date '+%Y-%m-%d_%H:%M:%S%z'`
BUILD_USER?=`whoami`
GOPROXY?=https://goproxy.cn,direct
default: build
build: cmd/hh-lol-prophet/main.go
	@CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build -tags=sonic -ldflags "-s -w \
-X github.com/real-web-world/hh-lol-prophet.Commit=$(GIT_COMMIT) \
-X github.com/real-web-world/hh-lol-prophet.BuildTime=$(BUILD_TIME) \
-X github.com/real-web-world/hh-lol-prophet.BuildUser=$(BUILD_USER) \
" -o bin/hh-lol-prophet.exe cmd/hh-lol-prophet/main.go
doc: cmd/hh-lol-prophet/main.go
	swag init -g .\cmd\hh-lol-prophet\main.go
clean: bin/
	@rm -rf bin/hh-lol-prophet.exe
upx : cmd/hh-lol-prophet/main.go
	upx -9 ./bin/hh-lol-prophet.exe
