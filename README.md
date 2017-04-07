## build on linux

```
export GOPATH=$HOME/go
go get -v -u github.com/lian/gdax-bookmap
$GOPATH/bin/gdax-bookmap -pair BTC-USD
```

## build on macOS

```
export GOPATH=$HOME/go
go get -v -u github.com/lian/gdax-bookmap
cd $GOPATH/src/github.com/lian/gdax-bookmap
go build -ldflags -s -o gdax-bookmap main.go
./gdax-bookmap -pair BTC-USD
```
