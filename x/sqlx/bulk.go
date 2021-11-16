package sqlx

import (
	"context"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var ErrItemAlreadyPresent = errors.New("item already present in database")

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

type Queue struct {
	queue      []*QueueItem
	queueMutex *sync.Mutex
	db         *sqlx.DB
}

func NewQueue(db *sqlx.DB) *Queue {
	return &Queue{}
}

func (q *Queue) AppendToQueue(queueItem *QueueItem) {
	q.queueMutex.Lock()
	defer q.queueMutex.Unlock()
	q.queue = append(q.queue, queueItem)
}

func (q *Queue) RecreateQueue() {
	q.queueMutex.Lock()
	defer q.queueMutex.Unlock()
	q.queue = make([]*QueueItem, 0, 10240)
}

// nolint:funlen // long function
func (q *Queue) ProcessQueue(ctx context.Context) error {
	q.queueMutex.Lock()
	var queriesToProcess []*QueueItem
	queriesToProcess = append(queriesToProcess, q.queue...)
	q.queueMutex.Unlock()
	q.RecreateQueue()

	if len(queriesToProcess) == 0 {
		return nil
	}

	tx, err := q.db.Beginx()
	if err != nil {
		q.queueMutex.Lock()
		q.queue = append(q.queue, queriesToProcess...)
		q.queueMutex.Unlock()
		return errors.Wrap(err, "start sql transaction")
	}

	var (
		waitForFlush     bool
		waitForFlushChan chan bool
		reAddRangeStart  int
	)

	for idx, item := range queriesToProcess {
		// If wait-for-flush item here - stop processing queries and
		// flush data.
		if item.IsWaitForFlush {
			waitForFlush = true
			waitForFlushChan = item.WaitForFlush
			reAddRangeStart = idx

			break
		}
		// Every structure should have "Prepare" function, even if
		// it's empty.
		item.Param.Prepare()
		// And IsUnique function that will check if item is unique
		// for us.
		if item.Param.IsUnique(q.db) {
			_, err := tx.NamedExec(item.Query, item.Param)
			if err != nil {
				return errors.Wrap(err, "execute query")
			}
		} else {
			return errors.Wrapf(ErrItemAlreadyPresent, "%+v", item.Param)
		}
	}

	if err := tx.Commit(); err != nil {
		if errBack := tx.Rollback(); errBack != nil {
			return errors.Wrap(err, "rollback failed transaction")
		}
		return errors.Wrap(err, "commit transaction to database")
	}

	if waitForFlush {
		waitForFlushChan <- true

		q.queueMutex.Lock()
		q.queue = append(queriesToProcess[reAddRangeStart:], q.queue...)
		q.queueMutex.Unlock()
	}

	return nil
}
