package eventsystem

import (
	"fmt"
	"time"
)

type HandlerFunc func(payload interface{}) error

type EventHandler struct {
	Event   string
	Handler HandlerFunc
}

type EventSystem struct {
	handlers  []EventHandler
	datastore Datastore
	enableLog bool
}

func NewEventSystem(datastore Datastore, enableLog bool) *EventSystem {
	return &EventSystem{
		handlers:  []EventHandler{},
		datastore: datastore,
		enableLog: enableLog,
	}
}

// Register new event handler
func (e *EventSystem) Register(handler EventHandler) {
	e.log(fmt.Sprintf("EventSystem.Register handler for %s", handler.Event))
	e.handlers = append(e.handlers, handler)
}

// Publish event with payload
func (e *EventSystem) Publish(event string, payload interface{}) error {
	e.log(fmt.Sprintf("EventSystem.Publish event for %s", event))
	err := e.datastore.SaveEvent(&Event{
		ID:          "",
		Name:        event,
		Payload:     payload,
		PublishedAt: time.Now(),
	})
	if err != nil {
		return err
	}

	go e.onPublish()

	return nil
}

// onPublish will run the next event that has not started yet
func (e *EventSystem) onPublish() {
	event, err := e.datastore.GetEvent()
	if err != nil {
		e.log(fmt.Sprintf("Failed to get event; %s", err.Error()))
		return
	}
	if event == nil {
		e.log("No event available")
		return
	}
	// Do not continue if last event aborted with an error
	for event.Error != nil {
		e.log(fmt.Sprintf("Last event aborted with error; %s", err.Error()))
		return
	}

	event.StartedAt = time.Now()
	if err := e.datastore.SaveEvent(event); err != nil {
		e.log(fmt.Sprintf("Failed to save event; %s", err.Error()))
		return
	}

	for _, handler := range e.handlers {
		if handler.Event == event.Name {
			if err := handler.Handler(event.Payload); err != nil {
				e.log(fmt.Sprintf("Failed to handle event; %s", err.Error()))

				event.Error = err
				if err := e.datastore.SaveEvent(event); err != nil {
					e.log(fmt.Sprintf("Failed to save event; %s", err.Error()))
					return
				}

				return
			}
		}
	}

	event.FinishedAt = time.Now()
	if err := e.datastore.SaveEvent(event); err != nil {
		e.log(fmt.Sprintf("Failed to save event; %s", err.Error()))
		return
	}
}

// Restart all unfinished events
func (e *EventSystem) Restart() {
	e.log("EventSystem.Restart")
	event, err := e.datastore.GetEvent()
	if err != nil {
		e.log(fmt.Sprintf("Failed to get event; %s", err.Error()))
		return
	}
	if event != nil && event.Error != nil {
		e.log(fmt.Sprintf("Try to restart failed event %s", event.Name))
		event.Error = nil
		if err := e.datastore.SaveEvent(event); err != nil {
			e.log(fmt.Sprintf("Failed to save event; %s", err.Error()))
			return
		}
	}

	go e.onPublish()
}

func (e *EventSystem) log(msg string) {
	if e.enableLog {
		fmt.Println(msg)
	}
}
