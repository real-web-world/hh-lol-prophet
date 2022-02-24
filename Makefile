.PHONY: build mode upx doc
GIT_COMMIT=`git rev-list -1 HEAD`
BUILD_TIME=`date '+%Y-%m-%d_%H:%M:%S'`
BUILD_USER=`whoami`
export GOPROXY=https://goproxy.cn,direct
build: cmd/hh-lol-prophet/main.go
	@CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -tags=jsoniter -ldflags "-s -w \
-X github.com/real-web-world/hh-lol-prophet.Commit=$(GIT_COMMIT) \
-X github.com/real-web-world/hh-lol-prophet.BuildTime=$(BUILD_TIME) \
-X github.com/real-web-world/hh-lol-prophet.BuildUser=$(BUILD_USER) \
" -o bin/hh-lol-prophet.exe cmd/hh-lol-prophet/main.go
doc: cmd/hh-lol-prophet/main.go
	swag init -g .\cmd\hh-lol-prophet\main.go
clean: bin/
	@rm -rf bin/hh-lol-prophet.exe
upx : cmd/hh-lol-prophet/main.go
	make build
	upx -9 ./bin/hh-lol-prophet.exe
