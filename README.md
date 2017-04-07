# gdax-bookmap - orderbook depths graph

![image](http://i.imgur.com/TAqyzeg.png)

## pre-compiled for macOS
[Alpha Releases](https://github.com/lian/gdax-bookmap/releases)

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

## current controls

```
q/esc to quit
up/down to change the price steps (aka price zoom) (PriceSteps)
w/s to change the graph price position (PriceScrollPosition)
j/k to change the volume chunks brightness (MaxSizeHisto)
left/right to change the column withd of volume chunks (ColumnWidth)
a/d to change how many seconds a chunk contains (aka time zoom) (ViewportStep)
c to try to center the graph by last price/trade (buggy)
```
