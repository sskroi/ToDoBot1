package processorloop

import (
	"ToDoBot1/pkg/events"
	"log"
	"time"
)

type ProcessorLoop struct {
	processor events.Processor
	batchSize int
}

func New(processor events.Processor, batchSize int) ProcessorLoop {
	return ProcessorLoop{
		processor: processor,
		batchSize: batchSize,
	}
}

func (p *ProcessorLoop) Start() error {
	ticker := time.NewTicker(30 * time.Second)

	for {
		gotEvents, err := p.processor.Fetch(p.batchSize)
		if err != nil {
			log.Printf("__ERR ProcessorLoop: %s", err.Error())

			continue
		}

		select {
		case <-ticker.C:
			err := p.processor.SendNotifications()
			if err != nil {
				log.Printf("__ERR__ ProcessorLoop: %s", err.Error())
			}
		default:
			// empty block
		}

		if len(gotEvents) == 0 {
			time.Sleep(time.Millisecond * 50)

			continue
		}

		err = p.handleEvents(gotEvents)
		if err != nil {
			log.Print(err)

			continue
		}

		time.Sleep(time.Millisecond * 50)
	}
}

func (p *ProcessorLoop) handleEvents(events []events.Event) error {
	for _, event := range events {
		// log.Printf("got new event: %s", event.Text)

		err := p.processor.Process(event)
		if err != nil {
			log.Printf("can't handle event: %s", err.Error())

			continue
		}
	}

	return nil
}
