package ch

import (
	// stdlib
	"sync"
	"time"
)

func (conn *Enity) AppendToQueue(queueItem *QueueItem) {
	defer conn.queueMutex.Unlock()
	conn.queueMutex.Lock()
	conn.queue = append(conn.queue, queueItem)
}

func (conn *Enity) recreateQueue() {
	conn.queue = make([]*QueueItem, 0, 10240)
}

// Queue worker goroutine entry point.
func (conn *Enity) startQueueWorker() {
	conn.log.Info().Msg("Starting queue worker")

	ticker := time.NewTicker(conn.options.QueueWorkerTimeout)
	conn.recreateQueue()

	conn.queueMutex = &sync.Mutex{}

	for range ticker.C {
		if conn.workWithQueue() {
			break
		}
	}

	ticker.Stop()
	conn.log.Info().Msg("Queue worker stopped")
	conn.queueWorkerStopped = true
}

func (conn *Enity) workWithQueue() bool {
	// We should stop on shutdown.
	if conn.weAreShuttingDown {
		return true
	}

	// Do nothing if connection wasn't yet established.
	if conn.Conn == nil {
		conn.log.Debug().Msg("Connection to database wasn't established, do nothing")
		return false
	}

	conn.queueMutex.Lock()
	var queriesToProcess []*QueueItem
	queriesToProcess = append(queriesToProcess, conn.queue...)
	conn.recreateQueue()
	conn.queueMutex.Unlock()

	if conn.weAreShuttingDown {
		if len(queriesToProcess) == 0 {
			return true
		}

		conn.log.Warn().Int("items in queue", len(queriesToProcess)).Msg("Still has items in queue to process, delaying shutdown")
	}

	conn.log.Debug().Int("items to process", len(queriesToProcess)).Msg("Got queries to process")

	if len(queriesToProcess) == 0 {
		conn.log.Debug().Msg("Nothing to process, skipping iteration")
		return false
	}

	tx, err := conn.Conn.Beginx()
	if err != nil {
		conn.log.Error().
			Err(err).
			Msg("Failed to start SQL transaction. Items prepared to be pushed to ClickHouse will be pushed back to queue")
		conn.queueMutex.Lock()
		conn.queue = append(conn.queue, queriesToProcess...)
		conn.queueMutex.Unlock()

		return false
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
		if item.Param.IsUnique(conn.Conn) {
			conn.log.Debug().Msgf("Parameters that will be passed to sqlx: %+v", item.Param)

			_, err1 := tx.NamedExec(item.Query, item.Param)
			if err1 != nil {
				// Maybe write problematic queries somewhere.
				conn.log.Error().Err(err1).Msg("Failed to execute query!")
			}
		} else {
			conn.log.Warn().Msgf("This item already present in database: %+v", item.Param)
		}
	}

	err1 := tx.Commit()
	if err1 != nil {
		// What to do with items?
		conn.log.Error().Err(err1).Msg("Failed to commit transaction to ClickHouse, rolling back...")

		err2 := tx.Rollback()
		if err2 != nil {
			conn.log.Error().Err(err2).Msg("Failed to rollback failed transaction! Expect database inconsistency!")
		}
	} else {
		conn.log.Info().Msg("SQL transaction committed")
	}

	if waitForFlush {
		waitForFlushChan <- true

		conn.queueMutex.Lock()
		conn.queue = append(queriesToProcess[reAddRangeStart:], conn.queue...)
		conn.queueMutex.Unlock()
	}

	return false
}
