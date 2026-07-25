package rabbitmq

import (
	"context"
	"sync"
)

type Job struct {
	Body []byte
}

type WorkerPool struct {
	workers int

	jobs chan Job

	handler Handler
}

func NewWorkerPool(
	workers int,
	handler Handler,
) *WorkerPool {
	return &WorkerPool{
		workers: workers,

		jobs: make(chan Job, 100),

		handler: handler,
	}
}

func (p *WorkerPool) Start(
	ctx context.Context,
) {
	var wg sync.WaitGroup

	for i := 0; i < p.workers; i++ {

		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {

				case <-ctx.Done():

					return

				case job := <-p.jobs:

					_ = p.handler(
						ctx,
						job.Body,
					)

				}
			}
		}()
	}

	go func() {
		<-ctx.Done()

		wg.Wait()
	}()
}

func (p *WorkerPool) Submit(
	body []byte,
) {
	p.jobs <- Job{
		Body: body,
	}
}
