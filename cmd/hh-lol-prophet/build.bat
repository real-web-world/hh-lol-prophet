go install github.com/akavel/rsrc@latest
rsrc -manifest .\hh-lol-prophet.manifest -ico icon.ico -o hh-lol-prophet.syso
go build -ldflags "-s -w"