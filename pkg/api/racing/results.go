package racing

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gocolly/colly"
	"github.com/mmanjoura/clean-bet-backend/pkg/database"
	"github.com/mmanjoura/clean-bet-backend/pkg/models"
)

// RacePicksSimulation handles the simulation of race picks and calculates win probabilities.
func GetResults(c *gin.Context) {

	db := database.Database.DB

	params := models.GetWinnerParams{}

	// Bind JSON input to optimalParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	params.Stake = 10.0

	rows, err := db.Query(`SELECT 	selection_id,
									selection_name,
									selection_link,
									potential_return,
									current_event_price,
									current_event_position				
									FROM Analysis
									WHERE event_date = ?   AND  selection_id = ?`,
		params.EventDate, params.SelectionId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var prediction models.EventPrediction
	var potentialReturn, currentEventPrice, currentDistance sql.NullString
	for rows.Next() {

		if err := rows.Scan(
			&prediction.SelectionID,
			&prediction.SelectionName,
			&prediction.SelectionLink,
			&potentialReturn,
			&currentEventPrice,
			&currentDistance,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		prediction.CurrentEventPrice = nullableToString(currentEventPrice)

	}

	if prediction.CurrentEventPrice == "" {
		potentialReturn := 0.0
		// now get the selection form and update Analysis
		selectionForm, err := GetResult(prediction.SelectionLink, params.EventDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		prediction.CurrentEventPrice = selectionForm.SpOdds
		prediction.EventDate = params.EventDate

		prediction.CurrentEventPosition = selectionForm.Position

		if strings.Contains(selectionForm.Position, "/") {
			num := strings.Split(selectionForm.Position, "/")[0]
			den := strings.Split(selectionForm.Position, "/")[1]

			// convet the string to float
			numFloat, _ := strconv.ParseFloat(num, 64)
			denFloat, _ := strconv.ParseFloat(den, 64)
			odds := numFloat / denFloat

			// Calulate the potential return

			if num == "1" {
				potentialReturn = 10.0 * odds
				prediction.PotentialReturn = fmt.Sprintf("%.2f", potentialReturn)
			} else {
				prediction.PotentialReturn = "0.00"
			}

			err = updateAnalysis(db, prediction)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			prediction.PotentialReturn = "0.00"
			err = updateAnalysis(db, prediction)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

	}
	c.JSON(http.StatusOK, gin.H{"simulationResults": prediction})
}

func updateAnalysis(db *sql.DB, prediction models.EventPrediction) error {

	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback() // Rollback the transaction in case of error
		} else {
			tx.Commit() // Commit the transaction if all goes well
		}
	}()

	// Step 2: INSERT new record
	_, err = tx.Exec(`
						Update Analysis
						SET current_event_price = ?,
						current_event_position = ?,
						potential_return = ? where selection_id = ? and event_date = ?`,

		prediction.CurrentEventPrice,
		prediction.CurrentEventPosition,
		prediction.PotentialReturn,
		prediction.SelectionID, prediction.EventDate)
	if err != nil {
		return err
	}
	return nil
}

func GetResult(selectionLink string, eventDate string) (models.SelectionForm, error) {
	c := colly.NewCollector()

	// Slice to store all horse information
	selectionForm := models.SelectionForm{}

	// Now continue with the rest of your code to scrape other data
	c.OnHTML("table.FormTable__StyledTable-sc-1xr7jxa-1 tbody tr", func(e *colly.HTMLElement) {
		raceDate := e.ChildText("td:nth-child(1) a")
		position := e.ChildText("td:nth-child(2)")
		spOdds := e.ChildText("td:nth-child(9)")

		// Split the date by "/" and add the current year
		dateParts := strings.Split(raceDate, "/")
		raceDate = "20" + dateParts[2] + "-" + dateParts[1] + "-" + dateParts[0]

		// Convert raceDate to time.Time
		parsedRaceDate, _ := time.Parse("2006-01-02", raceDate)
		date, _ := time.Parse("2006-01-02", eventDate)

		if parsedRaceDate.Equal(date) {
			// Create a new SelectionsForm object with the scraped data
			selectionForm = models.SelectionForm{

				Position: position,
				SpOdds:   spOdds,
			}

		}

	})

	// Start scraping the URL
	c.Visit("https://www.sportinglife.com" + selectionLink)

	return selectionForm, nil
}
