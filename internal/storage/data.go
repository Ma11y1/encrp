package storage

import (
	"encoding/json"
	"encrp/internal/errors"
	"sync"
)

type Data struct {
	mtx    sync.RWMutex
	parent *Node
	data   map[string][]byte
}

type storageData struct {
	Data map[string][]byte `json:"data"`
}

func NewStorageData() *Data {
	return &Data{data: make(map[string][]byte)}
}

func (d *Data) Get(key string) string {
	d.mtx.RLock()
	v := d.data[key]
	d.mtx.RUnlock()
	return string(v)
}

func (d *Data) Set(key string, value string) {
	d.mtx.Lock()
	d.data[key] = []byte(value)
	if d.parent != nil {
		d.parent.updateTsModify()
	}
	d.mtx.Unlock()
}

func (d *Data) Remove(key string) {
	d.mtx.Lock()
	if _, ok := d.data[key]; !ok {
		return
	}

	wipeBytes(d.data[key])
	delete(d.data, key)
	if d.parent != nil {
		d.parent.updateTsModify()
	}
	d.mtx.Unlock()
}

func (d *Data) Keys() []string {
	d.mtx.RLock()
	defer d.mtx.RUnlock()

	if len(d.data) == 0 {
		return []string{}
	}

	keys := make([]string, len(d.data))
	i := 0
	for k := range d.data {
		keys[i] = k
		i++
	}
	return keys
}

func (d *Data) Values() []string {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	if len(d.data) == 0 {
		return []string{}
	}
	values := make([]string, len(d.data))
	i := 0
	for _, v := range d.data {
		values[i] = string(v)
		i++
	}
	return values
}

func (d *Data) Has(key string) bool {
	d.mtx.RLock()
	_, ok := d.data[key]
	d.mtx.RUnlock()
	return ok
}

func (d *Data) Clear() {
	d.mtx.Lock()
	if len(d.data) > 0 {
		clear(d.data)
	}
	if d.parent != nil {
		d.parent.updateTsModify()
	}
	d.mtx.Unlock()
}

func (d *Data) Wipe(isUpdateTs bool) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	for k, v := range d.data {
		for i := range v {
			v[i] = 0
		}
		delete(d.data, k)
	}

	d.data = make(map[string][]byte)
	if isUpdateTs && d.parent != nil {
		d.parent.updateTsModify()
	}
}

func (d *Data) UnmarshalJSON(data []byte) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	temp := &storageData{}
	if err := json.Unmarshal(data, temp); err != nil {
		return errors.Wrap(err, "Data.UnmarshalJSON()", "Failed to unmarshal storage data")
	}
	if temp.Data == nil {
		d.data = make(map[string][]byte, 16)
	} else {
		d.data = temp.Data
	}
	return nil
}

func (d *Data) MarshalJSON() ([]byte, error) {
	d.mtx.RLock()
	data, err := json.Marshal(&storageData{Data: d.data})
	d.mtx.RUnlock()
	if err != nil {
		return nil, errors.Wrap(err, "Data.MarshalJSON()", "Failed to marshal storage data")
	}
	return data, nil
}
