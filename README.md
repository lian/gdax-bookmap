# gdax-bookmap - orderbook depths graph

![image](https://i.imgur.com/M0s0o9V.png)

## pre-compiled for macOS and Windows
[Alpha Releases](https://github.com/lian/gdax-bookmap/releases)

## build from source

```
export GOPATH=$HOME/go
go get -v -d -u github.com/lian/gdax-bookmap
cd $GOPATH/src/github.com/lian/gdax-bookmap
./script/build.sh
```

## current controls

```
1/2/3 selects BTC-USD, BTC-EUR, BCH-USD
esc to quit

up/down to change the price steps (aka price zoom) (PriceSteps)
j/k to change the volume chunks brightness (MaxSizeHisto)
a/d to change how many seconds a chunk contains (aka time zoom) (ViewportStep)

left/right to change the column width of volume chunks (ColumnWidth)
c center the graph to last price
p enable auto center
w/s to change the graph price position (PriceScrollPosition)
```
