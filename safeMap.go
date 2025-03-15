package sutils

import "sync"

type SafeMap[KEY comparable, VALUE any] struct {
	mu   sync.RWMutex
	data map[KEY]VALUE
}

func NewSafeMap[KEY comparable, VALUE any](size ...int) *SafeMap[KEY, VALUE] {
	if len(size) > 0 && size[0] > 0 {
		return &SafeMap[KEY, VALUE]{
			data: make(map[KEY]VALUE, size[0]),
		}
	}
	return &SafeMap[KEY, VALUE]{
		data: make(map[KEY]VALUE),
	}
}

func (m *SafeMap[KEY, VALUE]) Get(key KEY) (VALUE, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.data[key]
	return value, ok
}

func (m *SafeMap[KEY, VALUE]) Set(key KEY, value VALUE) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

func (m *SafeMap[KEY, VALUE]) Delete(keys ...KEY) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, key := range keys {
		delete(m.data, key)
	}
}

func (m *SafeMap[KEY, VALUE]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}

func (m *SafeMap[KEY, VALUE]) Range(f func(key KEY, value VALUE) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for k, v := range m.data {
		// 如果回傳 false，就中斷迭代
		if !f(k, v) {
			break
		}
	}
}
