package models

import (
	"time"
)

type RaceConditon struct {
	RaceCategory    string `json:"race_category"`
	RaceDistance    string `json:"race_distance"`
	TrackCondition  string `json:"track_condition"`
	NumberOfRunners string `json:"number_of_runners"`
	RaceTrack       string `json:"race_track"`
	RaceClass       string `json:"race_class"`
}

type MeetingSelections struct {
	SelectionID   int    `json:"selection_id"`
	SelectionLink string `json:"selection_link"`
	EventLink     string `json:"event_link"`
	SelectionName string `json:"selection_name"`
	EventTime     string `json:"event_time"`
	EventName     string `json:"event_name"`
	Price         string `json:"price"`
	RaceConditon RaceConditon `json:"race_condition"`
	EventID      int          `json:"event_id"`
	EventDate    time.Time    `json:"event_date"`
	CreatedAt    time.Time    `json:"created_at"`
}
