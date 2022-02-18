.PHONY: build mode upx doc
GIT_COMMIT=`git rev-list -1 HEAD`
BUILD_TIME=`date '+%Y-%m-%d_%H:%M:%S'`
BUILD_USER=`whoami`
export GOPROXY=https://goproxy.cn,direct
build: cmd/hh-lol-prophet/main.go
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags=jsoniter -ldflags "-s -w \
-X hh_lol_prophet.Commit=$(GIT_COMMIT) \
-X hh_lol_prophet.BuildTime=$(BUILD_TIME) \
-X hh_lol_prophet.BuildUser=$(BUILD_USER) \
" -o bin/hh-lol-prophet.exe cmd/hh-lol-prophet/main.go
doc: cmd/hh-lol-prophet/main.go
	swag init -g .\cmd\hh-lol-prophet\main.go
clean: bin/
	@rm -rf bin/*
upx : cmd/hh-lol-prophet/main.go
	make build
	upx -9 ./bin/hh-lol-prophet.exe
