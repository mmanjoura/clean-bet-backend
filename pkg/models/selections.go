package models

import "time"

type Selection struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Link            string `json:"link"`
	EventLink       string `json:"event_link"`
	EventName       string `json:"event_name"`
	MeetingName     string `json:"meeting_name"`
	EventDate       string `json:"event_date"`
	EventTime       string `json:"event_time"`
	Odds            string `json:"odds"`
	Position        string `json:"position"`
	RaceCategory    string `json:"race_category"`
	RaceDistance    string `json:"race_distance"`
	TrackCondition  string `json:"track_condition"`
	NumberOfRunners string `json:"number_of_runners"`
	RaceTrack       string `json:"race_track"`
	RaceClass       string `json:"race_class"`
}



type DaySince struct {
	RaceDate    time.Time `json:"race_date"`
	SelectionID int       `json:"selection_id"`
}

type Diff struct {
	DaySince
	DateDiffInDays int `json:"date_diff_in_days"`
}