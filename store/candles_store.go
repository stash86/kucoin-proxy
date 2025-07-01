package store

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stash86/kucoin-proxy/model"
)

var candlePool = sync.Pool{
	New: func() interface{} { return new(model.Candle) },
}

type Store struct {
	l           *sync.RWMutex
	mappedLists map[string]*candlesLinkedList
	cacheSize   int
	logCache    sync.Map // for log rate limiting, now thread-safe
}

func NewStore(cacheSize int) *Store {
	return &Store{
		l:           new(sync.RWMutex),
		mappedLists: map[string]*candlesLinkedList{},
		cacheSize:   cacheSize,
	}
}

func (s *Store) Store(key string, period time.Duration, candles ...*model.Candle) {
	if len(candles) == 0 {
		return
	}

	s.l.Lock()
	bucket := s.mappedLists[key]
	if bucket == nil {
		logrus.Infof("creating new bucket for key '%s'", key)
		bucket = newCandlesLinkedList()
		s.mappedLists[key] = bucket
	}
	s.l.Unlock()

	for _, c := range candles {
		if c == nil {
			logrus.Warnf("skipping nil candle for key '%s'", key)
			continue
		}

		// Only lock for writing if we are modifying the bucket
		s.l.Lock()
		if bucket.first != nil {
			steps := c.Ts.Sub(bucket.first.value.Ts) / period
			if steps > 1 {
				painted := candlePool.Get().(*model.Candle)
				*painted = *bucket.first.value // copy fields
				for i := 1; i < int(steps); i++ {
					painted.Ts = painted.Ts.Add(period)
					painted.Volume = 0
					painted.Amount = 0
					// Log at most once per minute per key (thread-safe)
					if lastVal, ok := s.logCache.Load(key); !ok || time.Since(lastVal.(time.Time)) > time.Minute {
						logrus.Warnf("saving painted candle: ts '%s' for '%s'...", painted.Ts, key)
						s.logCache.Store(key, time.Now())
					}
					s.store(bucket, painted)
				}
				candlePool.Put(painted)
			}
		}
		s.store(bucket, c)
		if bucket.size() > s.cacheSize {
			logrus.Infof("trimming bucket for key '%s' to cache size %d", key, s.cacheSize)
			bucket.remove(s.cacheSize)
		}
		s.l.Unlock()
	}
}

func (s *Store) store(bucket *candlesLinkedList, candle *model.Candle) {
	first, ok := bucket.get(0)
	if ok && first.Ts.Equal(candle.Ts) {
		logrus.Tracef("%s %s - update first", first.Ts.String(), candle.Ts.String())
		bucket.set(0, candle)

		return
	}

	if bucket.size() == s.cacheSize {
		bucket.remove(s.cacheSize - 1)
	}

	if ok && first.Ts.Before(candle.Ts) {
		logrus.Tracef("%s %s - prepend", first.Ts.String(), candle.Ts.String())
		bucket.prepend(candle)
	} else {
		if first != nil {
			logrus.Tracef("%s %s - append", first.Ts.String(), candle.Ts.String())
		}

		bucket.append(candle)
	}
}

func (s *Store) Get(key string, from time.Time, to time.Time) []*model.Candle {
	s.l.RLock()
	bucket := s.mappedLists[key]
	s.l.RUnlock()
	if bucket == nil {
		logrus.Debugf("Get: no bucket found for key '%s'", key)
		return nil
	}

	// No need to lock for reading the bucket if selectFn is safe
	candles := bucket.selectFn(
		func(candle *model.Candle) bool { return candle.Ts.Equal(from) || candle.Ts.Before(from) },
		func(candle *model.Candle) bool { return candle.Ts.Equal(to) || candle.Ts.Before(to) },
	)

	return candles
}
