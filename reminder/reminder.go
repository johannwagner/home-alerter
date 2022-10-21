package reminder

import (
	"context"
	"fmt"
	"time"
)

type ReminderManager struct {
	reminders []*Reminder
}

type Reminder struct {
	DailyTimes []*time.Time
	Timezone   *time.Location

	ExecutionFunction func() error
}

func (r *Reminder) GetNextExecution() *time.Duration {
	currentDate := time.Now()
	possibleExecutions := []time.Time{}

	for _, t := range r.DailyTimes {
		possibleExecutions = append(
			possibleExecutions,
			time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), t.Hour(), t.Minute(), 0, 0, r.Timezone),
			time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day()+1, t.Hour(), t.Minute(), 0, 0, r.Timezone),
		)
	}

	var minDiff *time.Duration
	for _, possibleExecution := range possibleExecutions {
		if possibleExecution.Before(currentDate) {
			continue
		}

		curDiff := possibleExecution.Sub(currentDate)
		if minDiff == nil || curDiff <= *minDiff {
			minDiff = &curDiff
		}
	}

	return minDiff
}

func NewReminderManager(reminders []*Reminder) *ReminderManager {

	reminder := ReminderManager{
		reminders,
	}

	return &reminder
}

func (r *ReminderManager) GetNextEvent() (*time.Duration, *Reminder) {
	var minDiff *time.Duration
	var minReminder *Reminder
	for _, reminder := range r.reminders {
		reminderOther := reminder
		curDiff := reminder.GetNextExecution()

		if minDiff == nil || *curDiff <= *minDiff {
			minDiff = curDiff
			minReminder = reminderOther
		}
	}

	return minDiff, minReminder
}

func (r *ReminderManager) Start(ctx context.Context) error {
	for {
		durationUntilNextEvent, nextReminder := r.GetNextEvent()

		fmt.Printf("Next scheduled Reminder is in %v, waiting...\n", durationUntilNextEvent.String())
		timer := time.NewTimer(*durationUntilNextEvent)
		select {
		case <-timer.C:
			err := nextReminder.ExecutionFunction()
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}
