package main

import (
	"context"
	"fmt"
)

type Worker struct {
	taskQueue chan Task
}

func newWorker() *Worker {
	return &Worker{
		taskQueue: make(chan Task, 1),
	}
}

func (w *Worker) start(pool *Pool, workerIndex int) {
	go func() {
		for t := range w.taskQueue {
			if t != nil {
				result, err := w.executeTask(t, pool)
				w.handleResult(result, err, pool)
			}
			pool.Push(workerIndex)
		}
	}()
}

func (w *Worker) executeTask(t Task, pool *Pool) (result interface{}, err error) {
	for i := 0; i <= pool.retryCount; i++ {
		if pool.timeout > 0 {
			result, err = w.executeTaskWithTimeout(t, pool)
		} else {
			result, err = w.executeTaskWithoutTimeout(t)
		}
		if err == nil || i == pool.retryCount {
			return result, err
		}
	}
	return
}

func (w *Worker) executeTaskWithTimeout(t Task, pool *Pool) (result interface{}, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), pool.timeout)
	defer cancel()

	resultChan := make(chan interface{})
	errChan := make(chan error)

	go func() {
		res, err := t()
		select {
		case resultChan <- res:
		case errChan <- err:
		case <-ctx.Done():
			return
		}
	}()

	select {
	case result = <-resultChan:
		err = <-errChan
		return result, err
	case <-ctx.Done():
		return nil, fmt.Errorf("task timed out")
	}
}

func (w *Worker) executeTaskWithoutTimeout(t Task) (result interface{}, err error) {
	return t()
}

func (w *Worker) handleResult(result interface{}, err error, pool *Pool) {
	if err != nil && pool.errorCallback != nil {
		pool.errorCallback(err)
	} else if pool.resultCallback != nil {
		pool.resultCallback(result)
	}
}