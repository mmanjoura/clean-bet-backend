package racing

import (
	"net/http"
	"time"

	"github.com/mmanjoura/clean-bet-backend/pkg/database"
	"github.com/mmanjoura/clean-bet-backend/pkg/models"

	"github.com/gin-gonic/gin"
)

// GetMeeting godoc
// @Summary Get the today meeting
// @Description Get the today meeting
// @Tags Get
// @Accept  json
// @Produce  json
// @Success 200 {object} object	"ok"
// @Router /horse/events [get]
func GetEvents(c *gin.Context) {

 	db := database.Database.DB

	var eventDate string

	if len(c.Query("date")) == 10 {
		eventDate = c.Query("date")[0:10]
	} else {
		eventDate = time.Now().Format("2006-01-02")
	}
	
	// Execute the query
	rows, err := db.Query(`
							SELECT 	event_name, 
								GROUP_CONCAT(
											event_time ORDER BY event_time) AS event_times 
							FROM Meetings 
							WHERE DATE(event_date) = ? 
							GROUP BY event_name ORDER BY event_name;`,
		eventDate)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	// Create a slice to hold the events
	var events []models.Event

	// Loop through the rows and append the results to the slice
	for rows.Next() {
		var event models.Event
		if err := rows.Scan(&event.EventName, &event.EventTime); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		events = append(events, event)
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return the events
	c.JSON(http.StatusOK, events)
}
