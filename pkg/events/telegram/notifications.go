package telegram

import (
	"ToDoBot1/pkg/e"
)

var ctrlPoints = [][]uint64{
	{5, 10800},  // 3h
	{4, 43200},  // 12 h
	{3, 86400},  // 1 day
	{2, 172800}, // 2 days
	{1, 259200}, // 3 days
	{0, 432000}, // 5 days
}

func (p *Processor) SendNotifications() error {
	curTime, err := getCurMskTime()
	if err != nil {
		return err
	}

	for _, v := range ctrlPoints {
		notifCnt := v[0]
		unixTime := v[1]

		tempTasks, err := p.storage.TasksForNotif(curTime, unixTime, notifCnt)
		if err != nil {
			return e.Wrap("can't SendNotications", err)
		}

		for _, t := range tempTasks {
			if err := p.storage.SetNotifCount(t.TaskId, notifCnt+1); err != nil {
				return e.Wrap("can't SendNotications", err)
			}

			if err := p.tg.SendMessage(t.UserId, getNotifMsg(t, curTime)); err != nil {
				return e.Wrap("can't SendNotications", err)
			}

		}

	}

	return nil
}
