package util

import (
	"fmt"
	"math"
	"time"

	"github.com/boltdb/bolt"
)

func PackUnixNanoKey(nano int64) []byte {
	return []byte(fmt.Sprintf("%d", nano))
}

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
	if now.Sub(p.LastDiff).Seconds() >= 1.0 {
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

func (p *BookBatchWrite) Write(db *bolt.DB, now time.Time, bucket string, buf []byte) {
	p.AddChunk(&BatchChunk{Time: now, Data: buf})

	if p.FlushBatch(now) {
		db.Update(func(tx *bolt.Tx) error {
			var err error
			var key []byte
			b := tx.Bucket([]byte(bucket))
			b.FillPercent = 0.9
			for _, chunk := range p.Batch {
				nano := chunk.Time.UnixNano()
				// windows system clock resolution https://github.com/golang/go/issues/8687
				for {
					key = PackUnixNanoKey(nano)
					if b.Get(key) == nil {
						break
					} else {
						nano += 1
					}
				}
				err = b.Put(key, chunk.Data)
				if err != nil {
					fmt.Println("HandleMessage DB Error", err)
				}
			}
			return err
		})
		//fmt.Println("flush batch chunks", len(p.Batch))
		p.Clear()
	}
}
