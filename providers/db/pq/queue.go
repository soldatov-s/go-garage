package pq

import (
	"sync"
	"time"
)

func (c *Enity) AppendToQueue(queueItem *QueueItem) {
	defer c.queueMutex.Unlock()
	c.queueMutex.Lock()
	c.queue = append(c.queue, queueItem)
}

func (c *Enity) recreateQueue() {
	defer c.queueMutex.Unlock()
	c.queueMutex.Lock()
	c.queue = make([]*QueueItem, 0, 10240)
}

// Queue worker goroutine entry point.
func (c *Enity) startQueueWorker() {
	c.log.Info().Msg("starting queue worker")

	ticker := time.NewTicker(c.cfg.QueueWorkerTimeout)
	c.recreateQueue()

	c.queueMutex = &sync.Mutex{}

	for range ticker.C {
		if c.workWithQueue() {
			break
		}
	}

	ticker.Stop()
	c.log.Info().Msg("queue worker stopped")
	c.queueWorkerStopped = true
}

func (c *Enity) workWithQueue() bool {
	// We should stop on shutdown.
	if c.weAreShuttingDown {
		return true
	}

	// Do nothing if connection wasn't yet established.
	if c.Conn == nil {
		c.log.Debug().Msg("connection to database wasn't established, do nothing")
		return false
	}

	c.queueMutex.Lock()
	var queriesToProcess []*QueueItem
	queriesToProcess = append(queriesToProcess, c.queue...)
	c.queueMutex.Unlock()
	c.recreateQueue()

	if c.weAreShuttingDown {
		if len(queriesToProcess) == 0 {
			return true
		}

		c.log.Warn().Int("items in queue", len(queriesToProcess)).Msg("still has items in queue to process, delaying shutdown")
	}

	c.log.Debug().Int("items to process", len(queriesToProcess)).Msg("got queries to process")

	if len(queriesToProcess) == 0 {
		c.log.Debug().Msg("nothing to process, skipping iteration")
		return false
	}

	tx, err := c.Conn.Beginx()
	if err != nil {
		c.log.Error().
			Err(err).
			Msg("failed to start sql transaction; items will be pushed back to queue")
		c.queueMutex.Lock()
		c.queue = append(c.queue, queriesToProcess...)
		c.queueMutex.Unlock()

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
		if item.Param.IsUnique(c.Conn) {
			c.log.Debug().Msgf("parameters that will be passed to sqlx: %+v", item.Param)

			_, err1 := tx.NamedExec(item.Query, item.Param)
			if err1 != nil {
				// Maybe write problematic queries somewhere.
				c.log.Error().Err(err1).Msg("failed to execute query!")
			}
		} else {
			c.log.Warn().Msgf("this item already present in database: %+v", item.Param)
		}
	}

	err1 := tx.Commit()
	if err1 != nil {
		// What to do with items?
		c.log.Error().Err(err1).Msg("failed to commit transaction to database, rolling back...")

		err2 := tx.Rollback()
		if err2 != nil {
			c.log.Error().Err(err2).Msg("failed to rollback failed transaction, expect database inconsistency!")
		}
	} else {
		c.log.Info().Msg("sql transaction committed")
	}

	if waitForFlush {
		waitForFlushChan <- true

		c.queueMutex.Lock()
		c.queue = append(queriesToProcess[reAddRangeStart:], c.queue...)
		c.queueMutex.Unlock()
	}

	return false
}
