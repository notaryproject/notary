// +build go1.7,!go1.8

package couchdb

import (
	"encoding/json"
	"time"
)

type schedulerDoc17 struct {
	Database      string   `json:"database"`
	DocID         string   `json:"doc_id"`
	ReplicationID string   `json:"id"`
	Source        string   `json:"source"`
	Target        string   `json:"target"`
	StartTime     nullTime `json:"start_time"`
	LastUpdated   nullTime `json:"last_updated"`
	State         string   `json:"state"`
	Info          repInfo  `json:"info"`
}

type nullTime time.Time

// inspired by https://github.com/golang/go/issues/9037#issuecomment-270932787
func (t *nullTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var ts time.Time
	if err := json.Unmarshal(data, &ts); err != nil {
		return err
	}
	*t = nullTime(ts)
	return nil
}

// UnmarshalJSON handles "null" values for times, which Go 1.7 doesn't natively
// handle properly
func (d *schedulerDoc) UnmarshalJSON(data []byte) error {
	var doc schedulerDoc17
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}
	*d = schedulerDoc{
		Database:      doc.Database,
		DocID:         doc.DocID,
		ReplicationID: doc.ReplicationID,
		Source:        doc.Source,
		Target:        doc.Target,
		StartTime:     time.Time(doc.StartTime),
		LastUpdated:   time.Time(doc.LastUpdated),
		State:         doc.State,
		Info:          doc.Info,
	}
	return nil
}
