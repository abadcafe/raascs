package cmds

import (
	"sync"
	"time"
)

type expirableMap struct {
	*sync.Map
}

type value struct {
	payload interface{}
	expiration *time.Time
}

var globalMap = expirableMap{Map: &sync.Map{}}

func (m *expirableMap) Load(key interface{}) (interface{}, bool) {
	v, ok := m.Map.Load(key)
	if !ok {
		return nil, false
	}

	vv := v.(*value)
	if vv.expiration != nil && time.Now().After(*vv.expiration) {
		m.Delete(key)
		return nil, false
	}

	return v.(*value).payload, true
}

func (m *expirableMap) StoreWithTtl(key interface{}, payload interface{}, ttl time.Duration) {
	e := time.Now().Add(ttl)
	value := &value{
		payload:    payload,
		expiration: &e,
	}

	m.Map.Store(key, value)
}

func (m *expirableMap) Store(key interface{}, payload interface{}) {
	value := &value{
		payload:    payload,
		expiration: nil,
	}

	m.Map.Store(key, value)
}
