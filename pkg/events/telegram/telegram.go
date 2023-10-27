package telegram

import (
	"ToDoBot1/pkg/clients/telegram"
	"ToDoBot1/pkg/e"
	"ToDoBot1/pkg/events"
	"ToDoBot1/pkg/storage"
	"errors"
	"time"
)

type Processor struct {
	tg      *telegram.Client
	storage storage.Storage
	offset  int
}

type Meta struct {
	UserId   uint64
	ChatId   uint64
	Username string
	Date     uint64
}

var (
	ErrUnknownEventType = errors.New("unknown event type")
	ErrUnknownMetaType  = errors.New("unknown meta type")
)

func New(tgClient *telegram.Client, storage storage.Storage) *Processor {
	return &Processor{
		tg:      tgClient,
		storage: storage,
	}
}

func (p *Processor) Fetch(limit int) ([]events.Event, error) {
	updates, err := p.tg.Updates(p.offset, limit)
	if err != nil {
		return nil, e.Wrap("can't get updates", err)
	}

	if len(updates) == 0 {
		return nil, nil
	}

	result := make([]events.Event, 0, len(updates))

	for _, v := range updates {
		result = append(result, event(v))
	}

	p.offset = updates[len(updates)-1].UpdateId + 1

	return result, nil
}

func (p *Processor) Process(event events.Event) error {
	var err error

	switch event.Type {
	case events.Message:
		err = p.processMessage(event)
	default:
		return e.Wrap("can't process events", ErrUnknownEventType)
	}

	if err != nil {
		return e.Wrap("can't process event", err)
	}

	return nil
}

func (p *Processor) processMessage(event events.Event) error {
	meta, err := getMeta(event)
	if err != nil {
		return e.Wrap("can't process message", err)
	}

	err = p.handleMsg(event.Text, meta)
	if err != nil {
		return e.Wrap("can't process message", err)
	}

	return nil
}

func getMeta(event events.Event) (Meta, error) {
	meta, ok := event.Meta.(Meta)
	if !ok {
		return Meta{}, e.Wrap("can't get meta", ErrUnknownMetaType)
	}

	return meta, nil
}

func event(upd telegram.Update) events.Event {
	updType := fetchType(upd)

	res := events.Event{
		Type: updType,
		Text: fetchText(upd),
	}

	if updType == events.Message {
		res.Meta = Meta{
			UserId:   upd.Message.From.UserId,
			ChatId:   upd.Message.Chat.ChatId,
			Username: upd.Message.From.Username,
			Date:     upd.Message.Date,
		}
	}

	return res
}

func fetchText(upd telegram.Update) string {
	if upd.Message == nil {
		return ""
	}

	return upd.Message.Text
}

func fetchType(upd telegram.Update) events.EvType {
	if upd.Message == nil {
		return events.Unknown
	}

	return events.Message
}

func getCurMskTime() (uint64, error) {
	currentTime := time.Now().Unix()

	// loc, err := time.LoadLocation("Europe/Moscow")
	// if err != nil {
	// 	return 0, e.Wrap("Ошибка при загрузке часового пояса Москвы:", err)
	// }

	// // Преобразуем текущее время в московское время
	// moscowTime := time.Unix(currentTime, 0).In(loc)

	// return moscowTime, nil

	return uint64(currentTime), nil
}
