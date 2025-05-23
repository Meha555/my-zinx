package client

import (
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestTask_StatusTransition(t *testing.T) {
	id := uuid.New()
	var statusHistory []int
	task := NewTask(id, func(t *Task) error {
		statusHistory = append(statusHistory, t.Status())
		return nil
	})

	if task.Status() != TaskStatusCreated {
		t.Errorf("初始状态应为TaskStatusCreated，实际为%d", task.Status())
	}

	_ = task.Exec()

	expected := []int{TaskStatusCreated, TaskStatusPending, TaskStatusRunning, TaskStatusFinished}
	for i, s := range statusHistory {
		if s != expected[i] {
			t.Errorf("状态转换第%d步错误，期望%d，实际%d", i, expected[i], s)
		}
	}
}

func TestTask_RepeatExecution(t *testing.T) {
	id := uuid.New()
	count := 0
	task := NewTask(id, func(t *Task) error {
		count++
		return nil
	}, WithRepeat(2))

	_ = task.Exec()

	if count != 3 {
		t.Errorf("重复执行次数错误，期望3次，实际%d次", count)
	}
}

func TestTask_Cancel(t *testing.T) {
	id := uuid.New()
	called := false
	task := NewTask(id, func(t *Task) error {
		called = true
		return nil
	})

	task.Cancel()
	_ = task.Exec()

	if called {
		t.Error("取消的任务不应执行fn")
	}
	if task.Status() != TaskStatusCanceled {
		t.Errorf("取消后状态应为TaskStatusCanceled，实际为%d", task.Status())
	}
}

func TestTask_AppendData(t *testing.T) {
	id := uuid.New()
	task := NewTask(id, func(t *Task) error { return nil })
	task.AppendData("data1")
	task.AppendData(2, "data3")

	expected := []interface{}{"data1", 2, "data3"}
	actual := task.Data()

	if len(actual) != len(expected) {
		t.Errorf("数据长度错误，期望%d，实际%d", len(expected), len(actual))
	}
	for i, v := range actual {
		if v != expected[i] {
			t.Errorf("第%d个数据错误，期望%v，实际%v", i, expected[i], v)
		}
	}
}

func TestTask_ConcurrencySafety(t *testing.T) {
	var wg sync.WaitGroup
	const taskCount = 100

	for i := 0; i < taskCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := uuid.New()
			task := NewTask(id, func(t *Task) error { return nil })
			retrieved := GetTask(id)
			if retrieved == nil || retrieved.ID() != id {
				t.Errorf("并发访问taskMap失败，未获取到任务%v", id)
			}
		}()
	}
	wg.Wait()
}

func TestTask_PanicRecover(t *testing.T) {
	id := uuid.New()
	panicked := false
	task := NewTask(id, func(t *Task) error {
		panic("测试panic")
	})

	// 替换为测试用的logger
	oldLogger := logger
	logger = &testLogger{}
	defer func() {
		logger = oldLogger
	}()

	_ = task.Exec()

	if !panicked {
		t.Error("未捕获到panic异常")
	}
}

// 测试用logger
type testLogger struct{}

func (l *testLogger) Errorf(format string, v ...interface{}) {
	panicked = true
}
