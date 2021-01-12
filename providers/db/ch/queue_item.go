package ch

import (
	// other
	"github.com/jmoiron/sqlx"
)

// QueueItem is a queue element. It will be used in conjunction with
// sqlx's NamedExec() function.
type QueueItem struct {
	// Query is a SQL query with placeholders for NamedExec(). See sqlx's
	// documentation about NamedExec().
	Query string
	// Params should be a structure that describes parameters. Fields
	// should have proper "db" tag to be properly put in database if
	// field name and table columns differs.
	Param QueueItemParam

	// Queue item flags.
	// IsWaitForFlush signals that this item blocking execution flow
	// and true should be sent via WaitForFlush chan.
	// When this flag set query from this item won't be processed.
	// Also all queue items following this item will be re-added
	// in the beginning of queue for later processing.
	IsWaitForFlush bool
	// WaitForFlush is a channel which will receive "true" once item
	// will be processed.
	WaitForFlush chan bool
}

// QueueItemParam is an interface for wrapping passed structure.
type QueueItemParam interface {
	// IsUnique should return false if item is not unique (and therefore
	// should not be processed) and true if item is unique and should
	// be processed. When uniqueness isn't necessary you may return
	// true here.
	IsUnique(conn *sqlx.DB) bool
	// Prepare should prepare items if needed. For example it may parse
	// timestamps from JSON-only fields into database-only ones. It should
	// return true if item is ready to be processed and false if error
	// occurred and item should not be processed. If preparation isn't
	// necessary you may return true here.
	Prepare() bool
}
