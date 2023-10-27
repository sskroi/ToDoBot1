package storage

import (
	"errors"
)

type Storage interface {
	GetState(userId uint64) (int, error)
	SetState(userId uint64, state int) error
	Add(userId uint64) error
	UpdTitle(userId uint64, title string) error
	UpdDescription(userId uint64, description string) error
	UpdDeadline(userId uint64, deadline, createTime uint64) error
	Delete(userId uint64, title string) error
	CloseTask(userId uint64, title string) error
	Uncompl(userId uint64) ([]Task, error)
	Compl(userId uint64) ([]Task, error)
	AllTasks(userId uint64) ([]Task, error)

	TasksForNotif(curTime uint64, timeDiff uint64, notifCount uint64) ([]Task, error)
}

// Types of state
const (
	DefState  int = 10
	Adding1   int = 21
	Adding2   int = 22
	Adding3   int = 23
	Deleting1 int = 31
	Closing1  int = 41
)

var (
	ErrUnique       = errors.New("unique error")
	ErrNotExist     = errors.New("requested data does not exist")
	ErrAlreayClosed = errors.New("task alreay closed")
)

type Task struct {
	TaskId      uint64 `db:"task_id"`
	UserId      uint64 `db:"user_id"`
	Title       string `db:"title"`
	Description string `db:"description"`
	CreateTime  uint64 `db:"create_time"`
	Deadline    uint64 `db:"deadline"`
	Done        bool   `db:"done"`
	NotifCount  uint64 `db:"notif_count"`
}

type User struct {
	UserId   uint64 `db:"user_id"`
	Username string `db:"username"`
	State    int    `db:"state"`
	CurTask  uint   `db:"cur_task"`
}
