package ordermap

import (
	"encoding/json"
	"sort"
)

type OrderMap struct {
	m    map[string]interface{} // origin map
	keys []string               // ordered keys
}

func New() *OrderMap {
	return &OrderMap{
		m:    make(map[string]interface{}),
		keys: make([]string, 0),
	}
}

func (o *OrderMap) Get(key string) interface{} {
	return o.m[key]
}

func (o *OrderMap) Set(key string, value interface{}) {
	if key == "" || value == nil {
		return
	}
	if _, ok := o.m[key]; ok {
		o.m[key] = value
	} else {
		o.m[key] = value
		o.keys = append(o.keys, key)
	}
}

func (o *OrderMap) Del(key string) {
	delete(o.m, key)
	for idx, val := range o.keys {
		if val == key {
			o.keys = append(o.keys[:idx], o.keys[idx+1:]...) // remove from slice
		}
	}
}

// Keys return the ordered keys of params map
func (o *OrderMap) Keys() []string {
	sort.Sort(o)
	return o.keys
}

// MarshalJSON implement json.Marshaler
func (o *OrderMap) MarshalJSON() ([]byte, error) {
	type Param struct {
		Key string      `json:"key"`
		Val interface{} `json:"val"`
	}

	var OrderMap = []*Param{}

	for _, key := range o.Keys() {
		OrderMap = append(OrderMap, &Param{key, o.Get(key)})
	}

	return json.Marshal(OrderMap)
}

// implement sort.Interface
func (o *OrderMap) Len() int {
	return len(o.keys)
}

func (o *OrderMap) Less(i int, j int) bool {
	return o.keys[i] < o.keys[j]
}

func (o *OrderMap) Swap(i int, j int) {
	o.keys[i], o.keys[j] = o.keys[j], o.keys[i]
}
