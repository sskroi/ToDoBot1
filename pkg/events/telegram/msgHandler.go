package telegram

import (
	"ToDoBot1/pkg/clients/telegram"
	"ToDoBot1/pkg/e"
	"ToDoBot1/pkg/storage"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrIncorrectTimeFormat = errors.New("incorrect time format")
)

func (p *Processor) handleMsg(text string, meta Meta) error {
	text = strings.TrimSpace(text)

	log.Printf("new msg: %s | username: %s | user_id: %s | chat_id: %s\n", text, meta.Username, strconv.FormatUint(meta.UserId, 10), strconv.FormatUint(meta.ChatId, 10))

	userState, err := p.storage.GetState(meta.UserId)
	if err != nil {
		return e.Wrap("can't handle message", err)
	}

	switch userState {
	case storage.DefState:
		err = p.doCmd(text, meta)
	case storage.Adding1:
		err = p.adding1(text, meta)
	case storage.Adding2:
		err = p.adding2(text, meta)
	case storage.Adding3:
		err = p.adding3(text, meta)
	case storage.Closing1:
		err = p.closeTask(text, meta)
	case storage.Deleting1:
		err = p.deleteTask(text, meta)
	}

	if err != nil {
		return e.Wrap("can't handle message", err)
	}

	return nil
}

func (p *Processor) doCmd(text string, meta Meta) error {
	var err error

	switch text {
	case startCmd:
		err = p.doStartCmd(meta)
	case helpCmd:
		err = p.doHelpCmd(meta)
	case addTaskBtn:
		err = p.doAddCmd(meta)
	case closeTaskBtn:
		err = p.doCloseCmd(meta)
	case delTaskBtn:
		err = p.doDelCmd(meta)
	case uncomplTasksBtn:
		err = p.doUncomplCmd(meta)
	case complTasksBtn:
		err = p.doComplCmd(meta)
	case allTasksBtn:
		err = p.doAllTasksCmd(meta)
	default:
		err = p.doUnknownCmd(meta)
	}
	if err != nil {
		return e.Wrap("can't do cmd", err)
	}

	return nil
}

func (p *Processor) doUnknownCmd(meta Meta) error {
	err := p.tg.SendMessageRM(meta.ChatId, unknownCmdMsg, mainMenuBtns)
	if err != nil {
		return e.Wrap("can't do UnknownCmd", err)
	}

	return nil
}

func (p *Processor) doStartCmd(meta Meta) error {
	err := p.tg.SendMessageRM(meta.ChatId, startMsg, mainMenuBtns)
	if err != nil {
		return e.Wrap("can't do /start", err)
	}

	return nil
}

func (p *Processor) doHelpCmd(meta Meta) error {
	err := p.tg.SendMessageRM(meta.ChatId, helpMsg, mainMenuBtns)
	if err != nil {
		return e.Wrap("can't do /help", err)
	}

	return nil
}

func (p *Processor) doAddCmd(meta Meta) error {
	err := p.storage.Add(meta.UserId)
	if err != nil {
		return e.Wrap("can't do addCmd", err)
	}

	err = p.storage.SetState(meta.UserId, storage.Adding1)
	if err != nil {
		return e.Wrap("can't do addCmd", err)
	}

	err = p.tg.SendMessageRM(meta.ChatId, addingMsg+addingTitleMsg, telegram.ReplyKeyboardRemove)
	if err != nil {
		return e.Wrap("can't do addCmd", err)
	}

	return nil
}

func (p *Processor) doCloseCmd(meta Meta) error {
	err := p.storage.SetState(meta.UserId, storage.Closing1)
	if err != nil {
		return e.Wrap("can't do closeCmd", err)
	}

	err = p.tg.SendMessageRM(meta.ChatId, closingMsg+closingTitleMsg, telegram.ReplyKeyboardRemove)
	if err != nil {
		return e.Wrap("can't do closeCmd", err)
	}

	return nil
}

func (p *Processor) doDelCmd(meta Meta) error {
	err := p.storage.SetState(meta.UserId, storage.Deleting1)
	if err != nil {
		return e.Wrap("can't do deleteCmd", err)
	}

	err = p.tg.SendMessageRM(meta.ChatId, deletingMsg+deletingTitleMsg, telegram.ReplyKeyboardRemove)
	if err != nil {
		return e.Wrap("can't do deleteCmd", err)
	}

	return nil
}

func (p *Processor) doUncomplCmd(meta Meta) error {
	tasks, err := p.storage.Uncompl(meta.UserId)
	if err != nil {
		return e.Wrap("can't do uncomplCmd", err)
	}

	if len(tasks) == 0 {
		p.tg.SendMessageRM(meta.ChatId, noUncomplTasksMsg, mainMenuBtns)
		if err != nil {
			return e.Wrap("can't do uncomplCmd", err)
		}

		return nil
	}

	tasksStr := makeTasksString(tasks)

	sentStr := UnComplTasksMsg + tasksStr

	p.tg.SendMessageRM(meta.ChatId, sentStr, mainMenuBtns)
	if err != nil {
		return e.Wrap("can't do uncomplCmd", err)
	}

	return nil
}

func (p *Processor) doComplCmd(meta Meta) error {
	tasks, err := p.storage.Compl(meta.UserId)
	if err != nil {
		return e.Wrap("can't do complCmd", err)
	}

	if len(tasks) == 0 {
		p.tg.SendMessageRM(meta.ChatId, noComplTasksMsg, mainMenuBtns)
		if err != nil {
			return e.Wrap("can't do complCmd", err)
		}

		return nil
	}

	tasksStr := makeTasksString(tasks)

	sentStr := ComplTasks + tasksStr

	p.tg.SendMessageRM(meta.ChatId, sentStr, mainMenuBtns)
	if err != nil {
		return e.Wrap("can't do complCmd", err)
	}

	return nil
}

func (p *Processor) doAllTasksCmd(meta Meta) error {
	tasks, err := p.storage.AllTasks(meta.UserId)
	if err != nil {
		return e.Wrap("can't do alltasksCmd", err)
	}

	if len(tasks) == 0 {
		p.tg.SendMessageRM(meta.ChatId, noTasksMsg, mainMenuBtns)
		if err != nil {
			return e.Wrap("can't do alltasksCmd", err)
		}

		return nil
	}

	tasksStr := makeTasksString(tasks)

	sentStr := allTasksMsg + tasksStr

	p.tg.SendMessageRM(meta.ChatId, sentStr, mainMenuBtns)
	if err != nil {
		return e.Wrap("can't do alltasksCmd", err)
	}

	return nil
}

func (p *Processor) adding1(text string, meta Meta) error {
	const errMsg = "can't add title to task"

	if text == "" {
		err := p.tg.SendMessage(meta.ChatId, addingMsg+incorrectTitleMsg)
		if err != nil {
			return e.Wrap(errMsg, err)
		}

		return nil
	}

	err := p.storage.UpdTitle(meta.UserId, text)
	if errors.Is(err, storage.ErrUnique) {
		if err := p.tg.SendMessage(meta.ChatId, addingMsg+taskAlreadyExistMsg); err != nil {
			return e.Wrap(errMsg, err)
		}

		return nil
	} else if err != nil {
		return e.Wrap(errMsg, err)
	}

	if err := p.storage.SetState(meta.UserId, storage.Adding2); err != nil {
		return e.Wrap(errMsg, err)
	}

	if err := p.tg.SendMessage(meta.ChatId, addingMsg+successTitleSetMsg); err != nil {
		return e.Wrap(errMsg, err)
	}

	return nil
}

func (p *Processor) adding2(text string, meta Meta) error {
	err := p.storage.UpdDescription(meta.UserId, text)
	if err != nil {
		return e.Wrap("can't add description to task", err)
	}

	if err := p.storage.SetState(meta.UserId, storage.Adding3); err != nil {
		return e.Wrap("can't add description to task", err)
	}

	if err := p.tg.SendMessage(meta.ChatId, addingMsg+successDescrSetMsg); err != nil {
		return e.Wrap("can't add description to task", err)
	}

	return nil
}

func (p *Processor) adding3(text string, meta Meta) error {
	deadlineUnixTime, err := parseTime(text)
	if err == ErrIncorrectTimeFormat {
		if err := p.tg.SendMessage(meta.ChatId, addingMsg+incorrectDeadlineMsg); err != nil {
			return e.Wrap("can't set deadline", err)
		}

		return nil
	}

	notifCnt, err := getCorrectNotifCount(deadlineUnixTime)
	if err != nil {
		return err
	}

	err = p.storage.UpdDeadline(meta.UserId, deadlineUnixTime, meta.Date, notifCnt)
	if err != nil {
		return e.Wrap("can't set deadline", err)
	}

	if err := p.storage.SetState(meta.UserId, storage.DefState); err != nil {
		return e.Wrap("can't set deadline", err)
	}

	if err := p.tg.SendMessageRM(meta.UserId, successDeadlineMsg, mainMenuBtns); err != nil {
		return e.Wrap("can't set deadline", err)
	}

	return nil
}

func parseTime(text string) (uint64, error) {
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return 0, e.Wrap("can't find location", err)
	}

	text = strings.TrimSpace(text)

	if utf8.RuneCount([]byte(text)) <= 10 {
		text = text + " 23:59"
	}

	parsedTime, err := time.ParseInLocation(dateTimeFormat, text, location)
	if err != nil {
		return 0, ErrIncorrectTimeFormat
	}

	res := parsedTime.Unix()

	if res <= time.Now().Unix() {
		return 0, ErrIncorrectTimeFormat
	}

	return uint64(res), nil
}

func (p *Processor) closeTask(text string, meta Meta) error {
	err := p.storage.SetState(meta.UserId, storage.DefState)
	if err != nil {
		return e.Wrap("can't close task", err)
	}

	err = p.storage.CloseTask(meta.UserId, text)
	if err == storage.ErrNotExist || err == storage.ErrAlreayClosed {
		if err == storage.ErrNotExist {
			if err := p.tg.SendMessageRM(meta.ChatId, taskNotExistMsg, mainMenuBtns); err != nil {
				return e.Wrap("can't close task", err)
			}
		} else if err == storage.ErrAlreayClosed {
			if err := p.tg.SendMessageRM(meta.ChatId, closingAlreadyClosedMsg, mainMenuBtns); err != nil {
				return e.Wrap("can't close task", err)
			}
		}

		return nil
	} else if err != nil {
		return e.Wrap("can't close task", err)
	}

	if err := p.tg.SendMessageRM(meta.ChatId, closingSuccessClosed, mainMenuBtns); err != nil {
		return e.Wrap("can't close task", err)
	}

	return nil
}

func (p *Processor) deleteTask(text string, meta Meta) error {
	err := p.storage.SetState(meta.UserId, storage.DefState)
	if err != nil {
		return e.Wrap("can't delete task", err)
	}

	err = p.storage.Delete(meta.UserId, text)
	if err == storage.ErrNotExist {
		if err := p.tg.SendMessageRM(meta.ChatId, taskNotExistMsg, mainMenuBtns); err != nil {
			return e.Wrap("can't delete task", err)
		}

		return nil
	} else if err != nil {
		return e.Wrap("can't delete task", err)
	}

	if err := p.tg.SendMessageRM(meta.ChatId, deletingSuccessDelete, mainMenuBtns); err != nil {
		return e.Wrap("can't delete task", err)
	}

	return nil
}

func getCorrectNotifCount(time uint64) (uint64, error) {
	curTime, err := getCurMskTime()
	if err != nil {
		return 0, err
	}

	var res uint64 = 0

	for _, v := range ctrlPoints {
		notifCnt := v[0]
		unixTime := v[1]
		if time-curTime <= unixTime {
			res = notifCnt + 1
			break
		}
	}

	return res, nil
}
