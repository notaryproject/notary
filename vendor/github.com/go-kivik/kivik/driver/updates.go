package driver

// DBUpdate represents a database update event.
type DBUpdate struct {
	DBName string `json:"db_name"`
	Type   string `json:"type"`
	Seq    string `json:"seq"`
}

// DBUpdates is a DBUpdates iterator.
type DBUpdates interface {
	// Next is called to populate DBUpdate with the values of the next update in
	// the feed.
	//
	// Next should return io.EOF when the feed is closed normally.
	Next(*DBUpdate) error
	// Close closes the iterator.
	Close() error
}

// DBUpdater is an optional interface that may be implemented by a Client to
// provide access to the DB Updates feed.
type DBUpdater interface {
	// DBUpdates must return a channel on which *DBUpdate events are sent,
	// and a function to close the connection.
	DBUpdates() (DBUpdates, error)
}
