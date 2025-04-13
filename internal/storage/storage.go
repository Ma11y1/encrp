package storage

import (
	"encoding/json"
	"encrp/internal/errors"
	"sync/atomic"
	"time"
)

const Version = "1.1"

type StorageType string

const (
	LocalFileStorageType StorageType = "local_file"
)

type Storage struct {
	t          StorageType
	version    string
	tsCreate   int64
	tsOpen     atomic.Int64
	tsSave     atomic.Int64
	lastDevice *Device
	data       *Node
}

type storage struct {
	Type       StorageType `json:"type"`
	Version    string      `json:"version"`
	TsCreate   int64       `json:"ts_create"`
	TsOpen     int64       `json:"ts_open"`
	TsSave     int64       `json:"ts_save"`
	LastDevice *Device     `json:"last_device"`
	Data       *Node       `json:"data"`
}

func NewStorage() *Storage {
	s := &Storage{
		t:          LocalFileStorageType,
		version:    Version,
		tsCreate:   time.Now().Unix(),
		lastDevice: &Device{},
		data:       NewStorageNode("root"),
	}
	s.tsOpen.Store(time.Now().Unix())
	return s
}

func (s *Storage) Type() StorageType   { return s.t }
func (s *Storage) Version() string     { return s.version }
func (s *Storage) TsCreate() int64     { return s.tsCreate }
func (s *Storage) TsOpen() int64       { return s.tsOpen.Load() }
func (s *Storage) TsSave() int64       { return s.tsSave.Load() }
func (s *Storage) LastDevice() *Device { return s.lastDevice }
func (s *Storage) Data() *Node         { return s.data }

func (s *Storage) UpdateTsOpen() {
	s.tsOpen.Store(time.Now().Unix())
}
func (s *Storage) UpdateTsSave() {
	s.tsSave.Store(time.Now().Unix())
}

func (s *Storage) Wipe() {
	if s.data != nil {
		s.data.Wipe()
		s.data = NewStorageNode("root")
	}
}

func (s *Storage) ToJSON() ([]byte, error)    { return json.Marshal(s) }
func (s *Storage) FromJSON(data []byte) error { return json.Unmarshal(data, s) }

func (s *Storage) UnmarshalJSON(data []byte) error {
	temp := &storage{}
	if err := json.Unmarshal(data, temp); err != nil {
		return errors.Wrap(err, "Storage.UnmarshalJSON()", "Failed to unmarshal storage")
	}
	s.t = temp.Type
	s.version = temp.Version
	s.tsCreate = temp.TsCreate
	s.tsOpen.Store(temp.TsOpen)
	s.tsSave.Store(temp.TsSave)
	s.lastDevice = temp.LastDevice
	s.data = temp.Data
	return nil
}

func (s *Storage) MarshalJSON() ([]byte, error) {
	temp := &storage{
		Type:       s.Type(),
		Version:    s.Version(),
		TsCreate:   s.TsCreate(),
		TsOpen:     s.TsOpen(),
		TsSave:     s.TsSave(),
		LastDevice: s.LastDevice(),
		Data:       s.Data(),
	}
	data, err := json.Marshal(temp)
	if err != nil {
		return nil, errors.Wrap(err, "Storage.MarshalJSON()", "Failed to marshal storage")
	}
	return data, nil
}
