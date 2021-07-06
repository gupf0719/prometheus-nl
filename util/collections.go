package util

import (
	"sync"
)

type BeeMap struct {
	lock *sync.RWMutex

	bm map[interface{}]interface{}
}

func NewBeeMap() *BeeMap {

	return &BeeMap{

		lock: new(sync.RWMutex),

		bm: make(map[interface{}]interface{}),
	}

}

func (m *BeeMap) Len() int {
	return len(m.bm)
}

func (m *BeeMap) GetAll() map[interface{}]interface{} {
	m.lock.RLock()
	defer m.lock.RUnlock()

	rs := make(map[interface{}]interface{})
	for k, v := range m.bm {
		rs[k] = v
	}

	return rs
}

//Get from maps return the k's value

func (m *BeeMap) Get(k interface{}) interface{} {

	m.lock.RLock()

	defer m.lock.RUnlock()

	if val, ok := m.bm[k]; ok {

		return val

	}

	return nil

}

// Maps the given key and value. Returns false

// if the key is already in the map and changes nothing.

func (m *BeeMap) Set(k interface{}, v interface{}) bool {

	m.lock.Lock()

	defer m.lock.Unlock()

	if val, ok := m.bm[k]; !ok { //Not exists, then append a new pair of key and value

		m.bm[k] = v

	} else if val != v { //exists and holds different value, then rewrite

		m.bm[k] = v

	} else { //holds the same key and value, then nothing to do.

		return false

	}

	return true

}

// Returns true if k is exist in the map.

func (m *BeeMap) Check(k interface{}) bool {

	m.lock.RLock()

	defer m.lock.RUnlock()

	if _, ok := m.bm[k]; !ok {

		return false

	}

	return true

}

func (m *BeeMap) Delete(k interface{}) {

	m.lock.Lock()

	defer m.lock.Unlock()

	delete(m.bm, k)

}

type BeeSlice struct {
	lock *sync.RWMutex

	bm []interface{}
}

func NewBeeSlice() *BeeSlice {

	return &BeeSlice{

		lock: new(sync.RWMutex),

		bm: make([]interface{}, 0),
	}

}

func (s *BeeSlice) Append(slice []interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.bm = append(s.bm, slice...)

}
