# gdax-bookmap - orderbook depths graph

![image](http://i.imgur.com/UNSFAHP.png)

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
1/2/3/4 selects BTC-USD, BTC-EUR, ETH-USD and LTC-USD
q/esc to quit
up/down to change the price steps (aka price zoom) (PriceSteps)
w/s to change the graph price position (PriceScrollPosition)
j/k to change the volume chunks brightness (MaxSizeHisto)
a/d to change how many seconds a chunk contains (aka time zoom) (ViewportStep)
left/right to change the column withd of volume chunks (ColumnWidth)
c to try to center the graph by last price/trade (buggy)
```
