package racing

import (
	"database/sql"
	"net/http"
	"sort"
	"strconv"

	"github.com/mmanjoura/clean-bet-backend/pkg/database"
	"github.com/mmanjoura/clean-bet-backend/pkg/models"

	"github.com/gin-gonic/gin"
)

func GetPredictions(c *gin.Context) {
	db := database.Database.DB
	config := database.Database.Config

	params := models.GetWinnerParams{}
	float64TotalRuns := 0.0

	// Bind JSON input to optimalParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	params.Delta = config["Delta"]
	params.AvgPosition = config["average_postion"]
	params.TotalRuns = config["total_runs"]
	stake, err := strconv.Atoi(config["bet_value"])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid bet value"})
		return
	}
	params.Stake = stake

	var eventPredicitonsResponse models.EventPredictionResponse

	// Construct query based on region filter
	query := `
		SELECT id,
			selection_id,
			selection_name,
			COALESCE(odds, '') as odds,
			COALESCE(age, '') as age,
			COALESCE(clean_bet_score, '') as clean_bet_score,
			COALESCE(average_position, '') as average_position,
			COALESCE(average_rating, '') as average_rating,
			event_name,
			COALESCE(event_date, '') as event_date,
			COALESCE(race_date, '') as race_date,
			COALESCE(event_time, '') as event_time,
			COALESCE(selection_position, '') as selection_position,
			ABS(prefered_distance - current_distance) as distanceTolerence,
			COALESCE(num_runners, '') as num_runners,
			COALESCE(number_runs, '') as number_runs,
			COALESCE(prefered_distance, '') as prefered_distance,
			COALESCE(current_distance, '') as current_distance,
			COALESCE(potential_return, '') as potential_return,
			COALESCE(current_event_price, '') as current_event_price,
			COALESCE(current_event_position, '') as current_event_position,
			created_at,
			updated_at
		FROM Analysis
		WHERE event_date = ?
			AND distanceTolerence < ?
			AND average_position < ?
			AND number_runs < ?`

	// Modify query based on region parameter
	if params.Region == "Both" {
		query += ` AND event_name IN (SELECT event_name FROM events WHERE country IN ('UK', 'Ireland'))`
	} else {
		query += ` AND event_name IN (SELECT event_name FROM events WHERE country = ?)`
	}

	query += ` ORDER BY clean_bet_score DESC LIMIT 5`

	// Execute query
	var rows *sql.Rows
	if params.Region == "both" {
		rows, err = db.Query(query, params.EventDate, params.Delta, params.AvgPosition, params.TotalRuns)
	} else {
		rows, err = db.Query(query, params.EventDate, params.Delta, params.AvgPosition, params.TotalRuns, params.Region)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var predictions []models.EventPrediction

	for rows.Next() {
		racePrdiction := models.EventPrediction{}
		err := rows.Scan(
			&racePrdiction.ID,
			&racePrdiction.SelectionID,
			&racePrdiction.SelectionName,
			&racePrdiction.Odds,
			&racePrdiction.Age,
			&racePrdiction.CleanBetScore,
			&racePrdiction.AveragePosition,
			&racePrdiction.AverageRating,
			&racePrdiction.EventName,
			&racePrdiction.EventDate,
			&racePrdiction.RaceDate,
			&racePrdiction.EventTime,
			&racePrdiction.SelectionPosition,
			&racePrdiction.DistanceTolerence,
			&racePrdiction.NumRunners,
			&racePrdiction.NumbeRuns,
			&racePrdiction.PreferredDistance,
			&racePrdiction.CurrentDistance,
			&racePrdiction.PotentialReturn,
			&racePrdiction.CurrentEventPrice,
			&racePrdiction.CurrentEventPosition,
			&racePrdiction.CreatedAt,
			&racePrdiction.UpdatedAt,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		position, err := getPosition(racePrdiction.SelectionID, params.EventDate, db)

		if err != nil {
			racePrdiction.Position = "?"
		} else {
			racePrdiction.Position = position
		}

		predictions = append(predictions, racePrdiction)
	}

	eventPredicitonsResponse.TotalBet = float64(len(predictions) * params.Stake)
	eventPredicitonsResponse.Selections = predictions
	eventPredicitonsResponse.TotalReturn = float64TotalRuns

	// Sort filtered predictions by CleanBetScore if needed (descending order)
	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i].CleanBetScore > predictions[j].CleanBetScore
	})

	c.JSON(http.StatusOK, gin.H{"predictions": eventPredicitonsResponse})
}


func sumSlice(slice []float64) float64 {
	var sum float64
	for _, value := range slice {
		sum += value
	}
	return sum
}

func filterHighestBetScore(predictions []models.EventPrediction) []models.EventPrediction {
	// Create a map to store the highest CleanBetScore for each EventTime
	eventTimeMap := make(map[string]models.EventPrediction)

	// Iterate through predictions and keep only the one with the highest CleanBetScore for each EventTime
	for _, prediction := range predictions {
		existing, found := eventTimeMap[prediction.EventTime]
		if !found || prediction.CleanBetScore > existing.CleanBetScore {
			eventTimeMap[prediction.EventTime] = prediction
		}
	}

	// Convert map to a slice of EventPredictions
	filteredPredictions := make([]models.EventPrediction, 0, len(eventTimeMap))
	for _, prediction := range eventTimeMap {
		filteredPredictions = append(filteredPredictions, prediction)
	}

	return filteredPredictions
}

// Get Postion given selection Id
func getPosition(selectionId int, race_date string, db *sql.DB) (string, error) {
	var position string
	err := db.QueryRow(`
				Select position 
				from Forms 
				where DATE(race_date) = ? and selection_id = ?;`, race_date, selectionId).Scan(&position)
	if err != nil {
		return "", err
	}
	return position, nil
}
