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
1/2/3/4/5/6/7/8/9 selects BTC-USD, BTC-EUR, LTC--USD, ETH-USD, LTC-BTC, ETH-BTC, BCH-USD, BCH-BTC, BCH-EUR
q/esc to quit
c center the graph to last price
p enable auto center
up/down to change the price steps (aka price zoom) (PriceSteps)
j/k to change the volume chunks brightness (MaxSizeHisto)
w/s to change the graph price position (PriceScrollPosition)
a/d to change how many seconds a chunk contains (aka time zoom) (ViewportStep)
left/right to change the column width of volume chunks (ColumnWidth)
```
