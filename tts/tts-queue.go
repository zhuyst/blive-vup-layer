package tts

import (
	"context"
	"fmt"
)

type TTSQueue struct {
	tts *TTS
	ch  chan *TaskWithChannel

	ctx    context.Context
	cancel context.CancelFunc
}

func NewTTSQueue(tts *TTS) *TTSQueue {
	ctx, cancel := context.WithCancel(context.Background())
	return &TTSQueue{
		tts: tts,
		ch:  make(chan *TaskWithChannel, 64),

		ctx:    ctx,
		cancel: cancel,
	}
}

type TaskWithChannel struct {
	Task *Task
	Done chan struct{}
}

func (q *TTSQueue) Push(params *NewTaskParams) error {
	task, err := q.tts.NewTask(params)
	if err != nil {
		return fmt.Errorf("NewTask err: %w", err)
	}

	doneCh := make(chan struct{}, 1)
	q.ch <- &TaskWithChannel{
		Task: task,
		Done: doneCh,
	}
	go func() {
		task.Run()
		doneCh <- struct{}{}
	}()
	return nil
}

type TaskResult struct {
	TaskId string
	Fname  string
	Err    error
}

func (q *TTSQueue) ListenResult() <-chan *TaskResult {
	ch := make(chan *TaskResult, 64)
	go func() {
		for {
			select {
			case <-q.ctx.Done():
				{
					close(ch)
					return
				}
			case r := <-q.ch:
				{
					<-r.Done
					ch <- &TaskResult{
						TaskId: r.Task.TaskId,
						Fname:  r.Task.Fname,
						Err:    r.Task.Err,
					}
				}
			}
		}
	}()
	return ch
}

func (q *TTSQueue) Close() {
	q.cancel()
}
