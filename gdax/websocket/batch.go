package websocket

import (
	"math"
	"time"
)

type BatchChunk struct {
	Time time.Time
	Data []byte
}

type BookBatchWrite struct {
	BatchTime   time.Time
	LastSync    time.Time
	LastDiff    time.Time
	LastDiffSeq uint64
	Count       int
	Batch       []*BatchChunk
}

func (p *BookBatchWrite) NextSync(now time.Time) bool {
	return math.Mod(float64(p.Count), 600) == 0
	/*
		if now.Sub(p.LastSync).Seconds() >= 60.0 {
			p.LastSync = now
		}
	*/
}

func (p *BookBatchWrite) NextDiff(now time.Time) bool {
	if now.Sub(p.LastDiff).Seconds() >= 0.5 {
		p.LastDiff = now
		return true
	}
	return false
}

func (p *BookBatchWrite) FlushBatch(now time.Time) bool {
	if now.Sub(p.BatchTime).Seconds() >= 0.5 {
		p.BatchTime = now
		return true
	}
	return false
}

func (p *BookBatchWrite) AddChunk(chunk *BatchChunk) {
	p.Count = p.Count + 1
	p.Batch = append(p.Batch, chunk)
}

func (p *BookBatchWrite) Clear() {
	p.Batch = []*BatchChunk{}
}
