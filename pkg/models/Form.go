package models

import (
	"time"
)

type SelectionForm struct {
	ID               int       `json:"id"`
	SelectionName    string    `json:"selection_name"`
	SelectionID      int       `json:"selection_id"`
	RaceDate         time.Time `json:"race_date"`
	Position         string    `json:"position"`
	Rating           string    `json:"rating"`
	RaceType         string    `json:"race_type"`
	Racecourse       string    `json:"racecourse"`
	Distance         string    `json:"distance"`
	Going            string    `json:"going"`
	Class            string    `json:"class"`
	SpOdds           string    `json:"sp_odds"`
	Age              string    `json:"age"`
	Trainer          string    `json:"trainer"`
	Sex              string    `json:"sex"`
	Sire             string    `json:"sire"`
	Dam              string    `json:"dam"`
	Owner            string    `json:"owner"`
	AVGPosition      float64   `json:"avg_position"`
	AVGRating        float64   `json:"avg_rating"`
	CurrentEventName string    `json:"current_event_name"`
	CurrentEventDate string    `json:"current_event_date"`
	CurrentEventTime string    `json:"current_event_time"`
	Score            string    `json:"score"`
	RaceCategory     string    `json:"race_category"`
	RaceDistance     string    `json:"race_distance"`
	TrackCondition   string    `json:"track_condition"`
	NumberOfRunners  string    `json:"number_of_runners"`
	RaceTrack        string    `json:"race_track"`
	RaceClass        string    `json:"race_class"`
	RaceURL          string    `json:"race_url"`
	EventDate        time.Time `json:"event_date"`
	CreatedAt        time.Time `json:"created_at"`
}




