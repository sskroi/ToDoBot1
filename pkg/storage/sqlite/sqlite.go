package sqlite

import (
	"ToDoBot1/pkg/e"
	"ToDoBot1/pkg/storage"
	"database/sql"
	"errors"
	"strconv"

	"github.com/mattn/go-sqlite3"
)

type SqliteStorage struct {
	db *sql.DB
}

// New устанавливает соединение с файлом БД и возвращает
// объект для взимодействия с базой данных sqlite3.
// Возвращает ошибку, если не удалось открыть файл с БД.
func New(path string) (*SqliteStorage, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, e.Wrap("can't open database", err)
	}

	if err := db.Ping(); err != nil {
		return nil, e.Wrap("can't open database", err)
	}

	return &SqliteStorage{
		db: db,
	}, nil
}

// Init инициализирует базу данных
// (создёт таблицы,если они не были созданы)
func (s *SqliteStorage) Init() error {
	queryUsers := `CREATE TABLE IF NOT EXISTS users (
		user_id INT PRIMARY KEY,
		username VARCHAR(255) DEFAULT "",
		state INT DEFAULT ` + strconv.Itoa(storage.DefState) + `,
		cur_task INT DEFAULT 0
	);`
	_, err := s.db.Exec(queryUsers)
	if err != nil {
		return e.Wrap("can't create table `users`", err)
	}

	queryTasks := `CREATE TABLE IF NOT EXISTS tasks (
		task_id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		title TEXT DEFAULT "",
		description TEXT DEFAULT "",
		create_time INTEGER DEFAULT 0,
		deadline INTEGER DEFAULT 0,
		done INTEGER NOT NULL DEFAULT 0,
		notif_count INTEGER DEFAULT 0,
		UNIQUE (user_id, title)
	);`
	if _, err := s.db.Exec(queryTasks); err != nil {
		return e.Wrap("can't create table `tasks`", err)
	}

	return nil
}

// GetState returns the state of the user or error
// if can't to get the state of user
func (s *SqliteStorage) GetState(userId uint64) (int, error) {
	err := s.checkUser(userId)
	if err != nil {
		return 0, err
	}

	qForGetUserState := `SELECT state FROM users WHERE user_id = ?;`

	var userState int

	err = s.db.QueryRow(qForGetUserState, userId).Scan(&userState)
	if err != nil {
		return 0, e.Wrap("can't get user's state", err)
	}

	return userState, nil
}

func (s *SqliteStorage) SetState(userId uint64, state int) error {
	err := s.checkUser(userId)
	if err != nil {
		return err
	}

	qForSetState := `UPDATE users SET state = ? WHERE user_id = ?;`

	_, err = s.db.Exec(qForSetState, state, userId)
	if err != nil {
		return e.Wrap("can't set state for user", err)
	}

	return nil
}

// Add insert empty task in tasks table with specified UserId
// and set the cur_task for the user with `userId` value of the newly created task.
// Task add with NULL title, description, create_time and deadline.
func (s *SqliteStorage) Add(userId uint64) error {
	err := s.checkUser(userId)
	if err != nil {
		return err
	}

	qForAddTask := `INSERT INTO tasks (user_id) VALUES (?);`

	res, err := s.db.Exec(qForAddTask, userId)
	if err != nil {
		return e.Wrap("can't add task", err)
	}

	lastInsertId, err := res.LastInsertId()
	if err != nil {
		return e.Wrap("can't get last insert id", err)
	}

	err = s.setCurTask(userId, uint64(lastInsertId))
	if err != nil {
		return err
	}

	return nil
}

// UpdTitle sets title for user's cur_task.
// if task with same title exists for this user, returns storage.ErrUnique.
func (s *SqliteStorage) UpdTitle(userId uint64, title string) error {
	taskId, err := s.getCurTask(userId)
	if err != nil {
		return err
	}

	qForUpdateTitle := `UPDATE tasks SET title = ? WHERE task_id = ?;`

	_, err = s.db.Exec(qForUpdateTitle, title, taskId)
	if err != nil {
		// проверяем, что ошибку можно преобразовать в тип ошибки sqlite3, если да, проверяем,
		// является ли эта ошибка ошибкой ErrConstraintUnique, если да, возвращаем кастомный тип ошибки ErrUnique1
		if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return storage.ErrUnique
		}
		return e.Wrap("can't add task", err)
	}

	return nil
}

// UpdDescription set description for user's cur_task
func (s *SqliteStorage) UpdDescription(userId uint64, description string) error {
	taskId, err := s.getCurTask(userId)
	if err != nil {
		return err
	}

	qForUpdateDescr := `UPDATE tasks SET description = ? WHERE task_id = ?;`

	_, err = s.db.Exec(qForUpdateDescr, description, taskId)
	if err != nil {
		return e.Wrap("can't update description in `tasks`", err)
	}

	return nil
}

// UpdDeadline sets deadline and create_time for user's cur_task
func (s *SqliteStorage) UpdDeadline(userId uint64, deadline, createTime uint64) error {
	taskId, err := s.getCurTask(userId)
	if err != nil {
		return err
	}

	qForUpdDeadlineAndCreateTime :=
		`UPDATE tasks SET create_time = ?, deadline = ? WHERE task_id = ?;`

	_, err = s.db.Exec(qForUpdDeadlineAndCreateTime, createTime, deadline, taskId)
	if err != nil {
		return e.Wrap("can't update create time and deadline in `tasks`", err)
	}

	return nil
}

// Delete deletes the user's task with specified title and
// returns ErrNotExist if task not exist
func (s *SqliteStorage) Delete(userId uint64, title string) error {
	err := s.isTaskExist(userId, title)
	if err == storage.ErrNotExist {
		return err
	} else if err != nil {
		return err
	}

	qForDelTask := `DELETE FROM tasks WHERE user_id = ? AND title = ?;`

	_, err = s.db.Exec(qForDelTask, userId, title)
	if err != nil {
		return e.Wrap("can't delete task", err)
	}

	return nil
}

// CloseTask sets the done field to 1 for the task and
// returns storage.ErrAlreayClosed if task already closed
// returns ErrNotExist if task not exist
func (s *SqliteStorage) CloseTask(userId uint64, title string) error {
	err := s.isTaskExist(userId, title)
	if err == storage.ErrNotExist {
		return err
	} else if err != nil {
		return err
	}

	qForUpdateDone := `UPDATE tasks SET done = 1 WHERE user_id = ? AND title = ? AND done != 1;`

	res, err := s.db.Exec(qForUpdateDone, userId, title)
	if err != nil {
		return e.Wrap("can't set done status", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return e.Wrap("can't get rowsAffected info", err)
	}

	if rowsAffected == 0 {
		return storage.ErrAlreayClosed
	}

	return nil
}

// Uncompl returns slice of uncompleted tasks for specified user.
func (s *SqliteStorage) Uncompl(userId uint64) ([]storage.Task, error) {
	tasks, err := s.getTasks(userId, 0)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

// Compl returns slice of uncompleted tasks for specified user.
func (s *SqliteStorage) Compl(userId uint64) ([]storage.Task, error) {
	tasks, err := s.getTasks(userId, 1)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

// AllTasks returns slice of all user's tasks ordered by create time
func (s *SqliteStorage) AllTasks(userId uint64) ([]storage.Task, error) {
	tasks, err := s.getTasks(userId, 2)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

func (s *SqliteStorage) SetNotifCount(taskId uint64, notifCount uint64) error {
	qForSetNotifCount := `UPDATE tasks SET notif_count = ? WHERE task_id = ?;`

	_, err := s.db.Exec(qForSetNotifCount, notifCount, taskId)
	if err != nil {
		return e.Wrap("can't set notif count", err)
	}

	return nil
}

func (s *SqliteStorage) TasksForNotif(curTime uint64, timeDiff uint64, notifCount uint64) ([]storage.Task, error) {
	qForGetTasks := `SELECT * FROM tasks WHERE ((? - deadline) <= ?) AND notif_count = ?;`

	rows, err := s.db.Query(qForGetTasks, curTime, timeDiff, notifCount)
	if err != nil {
		return nil, e.Wrap("can't get tasks for notification", err)
	}

	tasks, err := fetchTasksFromRows(rows)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

// getTask returns slice of uncompleted tasks for user if qFilter = 0
// and slice of completed tasks if qFilter = 1.
func (s *SqliteStorage) getTasks(userId uint64, qFilter int) ([]storage.Task, error) {
	var qForGetTasks string

	switch qFilter {
	case 0:
		qForGetTasks =
			`SELECT * FROM tasks WHERE user_id = ? AND done = 0;`
	case 1:
		qForGetTasks =
			`SELECT * FROM tasks WHERE user_id = ? AND done = 1;`
	case 2:
		qForGetTasks =
			`SELECT * FROM tasks WHERE user_id = ? ORDER BY create_time DESC`
	default:
		return nil, errors.New("unknown qFilter")
	}

	rows, err := s.db.Query(qForGetTasks, userId)
	if err != nil {
		return nil, e.Wrap("can't select uncomp tasks", err)
	}

	tasks, err := fetchTasksFromRows(rows)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

func fetchTasksFromRows(rows *sql.Rows) ([]storage.Task, error) {
	defer rows.Close()

	tasks := make([]storage.Task, 0)

	for rows.Next() {
		var newT = storage.Task{}
		err := rows.Scan(&newT.TaskId, &newT.UserId, &newT.Title,
			&newT.Description, &newT.CreateTime, &newT.Deadline, &newT.Done, &newT.NotifCount)
		if err != nil {
			return nil, e.Wrap("can't scan tasks", err)
		}
		tasks = append(tasks, newT)
	}

	return tasks, nil
}

// isTaskExist checks if user has a task with title return nil if yes and storage.ErrNotExist if not.
func (s *SqliteStorage) isTaskExist(userId uint64, title string) error {
	qForCheckExist := `SELECT task_id FROM tasks WHERE user_id = ? AND title = ?;`

	var checkExistRes int

	err := s.db.QueryRow(qForCheckExist, userId, title).Scan(&checkExistRes)
	if err == sql.ErrNoRows {
		return storage.ErrNotExist
	} else if err != nil {
		return e.Wrap("can't delete task", err)
	}

	return nil
}

// checkUser check if the user with `UserId` exists, if not,
// create user.
func (s *SqliteStorage) checkUser(userId uint64) error {
	qForCheckUser := `SELECT user_id FROM users WHERE user_id = ?;`

	var checkUserRes int

	err := s.db.QueryRow(qForCheckUser, userId).Scan(&checkUserRes)
	if err == sql.ErrNoRows {
		qForAddUser := `INSERT INTO users (user_id) VALUES (?);`
		_, err = s.db.Exec(qForAddUser, userId)
		if err != nil {
			return e.Wrap("can't create user", err)
		}
	} else if err != nil {
		return e.Wrap("can't check user", err)
	}

	return nil
}

// setCurTask set the cur_task for the user with the specified `UserId`
func (s *SqliteStorage) setCurTask(userId uint64, taskId uint64) error {
	qForSetCurTask := `UPDATE users SET cur_task = ? WHERE user_id = ?;`

	_, err := s.db.Exec(qForSetCurTask, taskId, userId)
	if err != nil {
		return e.Wrap("can't update cur_task in users", err)
	}

	return nil
}

// getCurTask returns user's cur_task
func (s *SqliteStorage) getCurTask(userId uint64) (uint64, error) {
	qForGetCurTask := `SELECT cur_task FROM users WHERE user_id = ?;`

	var curTask uint64

	err := s.db.QueryRow(qForGetCurTask, userId).Scan(&curTask)
	if err != nil {
		return 0, e.Wrap("can't get cur_task from users", err)
	}

	return curTask, nil
}
