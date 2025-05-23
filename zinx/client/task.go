package client

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type ITask interface {
	ID() uuid.UUID
	Exec() error
	Data() []interface{}
	Status() int
	Error() <- chan error
	AppendData(userData ...interface{})
}

type TaskHandler interface {
	Do() error
}

type TaskHandlerFunc func() error

func (t TaskHandlerFunc) Do() error {
	return t()
}

const (
	TaskStatusCreated = iota
	TaskStatusPending
	TaskStatusRunning
	TaskStatusFinished
	TaskStatusFailed
	TaskStatusCanceled
)

var TaskErrorCanceled = errors.New("task canceled")

type Task struct {
	id     uuid.UUID
	fn     TaskHandlerFunc
	data   []interface{}
	repeat int
	status int

	// TODO lasterror ?
	lastErr chan error

	pool *WorkerPool
}

type option func(*Task)

func WithWorkerPool(pool *WorkerPool) option {
	return func(t *Task) {
		t.pool = pool
	}
}

func WithRepeat(repeat int) option {
	return func(t *Task) {
		t.repeat = repeat
	}
}

func WithData(userData ...interface{}) option {
	return func(t *Task) {
		t.data = userData
	}
}

// NewTask 创建一个异步任务
// 在fn内部可以直接通过闭包的方式访问到Task实例。设计思路参考testing.T
func NewTask(id uuid.UUID, fn func(*Task) error, opts ...option) *Task {
	t := &Task{
		id:     id,
		status: TaskStatusCreated,
	}
	t.fn = func() error {
		t.lastErr <- fn(t)
		if t.repeat > 0 {
			t.repeat--
			t.Exec()
		}
		// else {
		// 	mu.Lock()
		// 	delete(taskMap, t.id)
		// 	mu.Unlock()
		// }
		t.status = TaskStatusFinished
		return <-t.lastErr
	}

	for _, opt := range opts {
		opt(t)
	}

	mu.Lock()
	taskMap[t.id] = t
	mu.Unlock()
	return t
}

func (t *Task) ID() uuid.UUID {
	return t.id
}

func (t *Task) Status() int {
	return t.status
}

func (t *Task) Error() <- chan error {
	return t.lastErr
}

func (t *Task) Exec() error {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf("Task %s panic: %v", t.id, err)
			t.status = TaskStatusFailed
			t.lastErr <- fmt.Errorf("task %s panic: %v", t.id, err)
			// mu.Lock()
			// delete(taskMap, t.id)
			// mu.Unlock()
		}
		close(t.lastErr)
	}()
	if t.status == TaskStatusCanceled {
		return TaskErrorCanceled
	}
	t.status = TaskStatusPending
	if t.pool != nil {
		t.pool.Post(t.fn)
		return nil
	} else {
		t.status = TaskStatusRunning
		ret := t.fn()
		return ret
	}
}

func (t *Task) Data() []interface{} {
	return t.data
}

func (t *Task) AppendData(userData ...interface{}) {
	t.data = append(t.data, userData...)
}

func (t *Task) Cancel() {
	t.status = TaskStatusCanceled
}

var (
	// TODO 是否考虑使用sync.Map？
	taskMap = make(map[uuid.UUID]ITask)
	mu      sync.Mutex
)

// GetTask 从任务表中获取任务
// 任务表更新规则：TODO
func GetTask(id uuid.UUID) ITask {
	mu.Lock()
	defer mu.Unlock()
	return taskMap[id]
}
