package store

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Container struct {
	raw       []byte
	expiresAt time.Time
}

func (c *Container) Raw() []byte {
	return c.raw
}

func NewTTLCache(expirationTimeout time.Duration) *TTLCache {
	return &TTLCache{
		l:                 new(sync.Mutex),
		kv:                map[string]*Container{},
		expirationTimeout: expirationTimeout,
	}
}

type TTLCache struct {
	l *sync.Mutex

	kv                map[string]*Container
	expirationTimeout time.Duration
}

func (s *TTLCache) Get(key string) *Container {
	s.l.Lock()
	defer s.l.Unlock()

	container, ok := s.kv[key]
	if !ok {
		logrus.Debugf("TTLCache.Get: cache miss for key '%s'", key)
		return nil
	}

	if container.expiresAt.Before(time.Now().UTC()) {
		logrus.Debugf("TTLCache.Get: expired entry for key '%s' (expired at %s)", key, container.expiresAt)
		delete(s.kv, key)
		return nil
	}

	return container
}

func (s *TTLCache) Store(key string, value []byte) {
	s.l.Lock()
	defer s.l.Unlock()

	expiresAt := time.Now().UTC().Add(s.expirationTimeout)
	s.kv[key] = &Container{
		raw:       value,
		expiresAt: expiresAt,
	}
	logrus.Debugf("TTLCache.Store: stored key '%s' (expires at %s)", key, expiresAt)
}
