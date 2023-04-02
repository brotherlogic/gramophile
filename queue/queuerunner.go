package queue

import (
	"context"
	"time"
)

type Queue struct{}

func (q *Queue) run() {
	for {
		ctx := context.Background()
		entry := q.getNextEntry()
		err := q.Execute(entry)

		if err != nil {
			q.delete(entry)
		} else {
			time.Sleep(entry.GetBackoff())
		}
	}
}

type (q *Queue) getNextEntry(ctx context.Context) ()
