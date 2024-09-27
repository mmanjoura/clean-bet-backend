package racing

import (
	"net/http"
	"strings"
	"time"

	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/gocolly/colly"
	"github.com/mmanjoura/clean-bet-backend/pkg/database"
	"github.com/mmanjoura/clean-bet-backend/pkg/models"
)

func GetForms(c *gin.Context) {
	db := database.Database.DB

	var raceDate EventDate

	// Bind JSON input to optimalParams
	if err := c.ShouldBindJSON(&raceDate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}


	todayRunners, err := TodayRunners(db, c, raceDate.Date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, todayRunner := range todayRunners {

		form, err := GetSelectionForm(todayRunner.SelectionLink)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, fr := range form {
			
			lastRunDate, err := getLastRunDate(db, todayRunner.SelectionID)			
			if err != nil {
				if err.Error() == "sql: no rows in result set" {
					err = SaveSelectionForm(db, fr, c, todayRunner.SelectionName, todayRunner.SelectionID)
					if err != nil {

						c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
						return
					}
					continue
				}
			}		

			parsedLastRunDate, _ := time.Parse("2006-01-02", lastRunDate[:10])
			if err != nil {

				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			_ = lastRunDate

			if fr.EventDate.After(parsedLastRunDate){
				err = SaveSelectionForm(db, fr, c, todayRunner.SelectionName, todayRunner.SelectionID)
				if err != nil {

					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}				
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Horse information saved successfully"})

}



// Function to remove duplicate odds patterns
func removeDuplicateOdds(odds string) string {
	// Count the number of '/' characters in the string
	count := strings.Count(odds, "/")

	// If there are exactly two '/' characters, return the first half of the string
	if count == 2 {
		mid := len(odds) / 2
		return odds[:mid]
	}

	// If there is not exactly two '/', return the original string
	return odds
}

func TodayRunners(db *sql.DB, c *gin.Context, date string) ([]models.MeetingSelections, error) {

	var todayRunners []models.MeetingSelections
	var todayRunner models.MeetingSelections

	rows, err := db.Query(`select selection_link,
	selection_id,
	event_link,
	selection_name,
	event_time,
	event_name,
	price,
	event_date

	from Meetings where DATE(event_date) = ?`, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return todayRunners, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&todayRunner.SelectionLink,
			&todayRunner.SelectionID, &todayRunner.EventLink, &todayRunner.SelectionName,
			&todayRunner.EventTime, &todayRunner.EventName, &todayRunner.Price,
			&todayRunner.EventDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return todayRunners, err
		}
		todayRunners = append(todayRunners, todayRunner)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return todayRunners, err
	}

	return todayRunners, nil
}


func GetSelectionForm(selectionLink string) ([]models.SelectionForm, error) {
	c := colly.NewCollector()
	config := database.Database.Config

	// Slice to store all horse information
	selectionsForm := []models.SelectionForm{}

	var age, trainer, sex, sire, dam, owner string

	c.OnHTML("table.Header__DataTable-xeaizz-1", func(e *colly.HTMLElement) {
		age = e.ChildText("tr:nth-child(1) td.Header__DataValue-xeaizz-4")
		trainer = e.ChildText("tr:nth-child(2) td.Header__DataValue-xeaizz-4 a")
		sex = e.ChildText("tr:nth-child(3) td.Header__DataValue-xeaizz-4")
		sire = e.ChildText("tr:nth-child(4) td.Header__DataValue-xeaizz-4")
		dam = e.ChildText("tr:nth-child(5) td.Header__DataValue-xeaizz-4")
		owner = e.ChildText("tr:nth-child(6) td.Header__DataValue-xeaizz-4")
	})

	// Now continue with the rest of your code to scrape other data
	c.OnHTML("table.FormTable__StyledTable-sc-1xr7jxa-1 tbody tr", func(e *colly.HTMLElement) {
		raceDate := e.ChildText("td:nth-child(1) a")
		raceLink := e.ChildAttr("td:nth-child(1) a", "href")
		position := e.ChildText("td:nth-child(2)")
		rating := e.ChildText("td:nth-child(3)")
		raceType := e.ChildText("td:nth-child(4)")
		racecourse := e.ChildText("td:nth-child(5)")
		distance := e.ChildText("td:nth-child(6)")
		going := e.ChildText("td:nth-child(7)")
		class := e.ChildText("td:nth-child(8)")
		spOdds := e.ChildText("td:nth-child(9)")

		// Split the date by "/" and add the current year
		dateParts := strings.Split(raceDate, "/")
		raceDate = "20" + dateParts[2] + "-" + dateParts[1] + "-" + dateParts[0]

		// Convert raceDate to time.Time
		parsedRaceDate, _ := time.Parse("2006-01-02", raceDate)
		// parsedEventDate, _ := time.Parse("2006-01-02", eventDate)

		// Create a new SelectionsForm object with the scraped data
		selectionForm := models.SelectionForm{
			RaceDate:   parsedRaceDate,
			Position:   position,
			Rating:     rating,
			RaceType:   raceType,
			Racecourse: racecourse,
			Distance:   distance,
			Going:      going,
			RaceClass:  class,
			SpOdds:     spOdds,
			RaceURL:    raceLink,
			EventDate:  parsedRaceDate,
			Age:        age,
			Trainer:    trainer,
			Sex:        sex,
			Sire:       sire,
			Dam:        dam,
			Owner:      owner,
			CreatedAt:  time.Now(),
		}

		// Append the selection form to the slice
		selectionsForm = append(selectionsForm, selectionForm)

	})

	// Start scraping the URL
	c.Visit(config["DataLink"] + selectionLink)

	return selectionsForm, nil
}

func getLastRunDate(db *sql.DB, selectionId int) (string, error) {

	var lastRunDate string
	err := db.QueryRow(`select race_date  from Forms where selection_id = ? order by race_date desc limit 1;`, selectionId).Scan(&lastRunDate)
	if err != nil {
		return "", err
	}
	return lastRunDate, nil

}

func SaveSelectionForm(db *sql.DB, selectionForm models.SelectionForm, c *gin.Context, selectionName string, selectionID int) error {

	// Start a transaction
	tx, err := db.BeginTx(c, nil)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(c, `
        INSERT INTO Forms (
			selection_name,
			selection_id,
			race_class,
			race_date,
			position,
			rating,
			race_type,
			racecourse,
			distance,
			going,
			sp_odds,
			Age,
			Trainer,
			Sex,
			Sire,
			Dam,
			Owner,
			created_at,
			updated_at
        )
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		selectionName, selectionID, selectionForm.RaceClass, selectionForm.RaceDate, selectionForm.Position,
		selectionForm.Rating, selectionForm.RaceType, selectionForm.Racecourse,
		selectionForm.Distance, selectionForm.Going,
		selectionForm.SpOdds, selectionForm.Age, selectionForm.Trainer,
		selectionForm.Sex, selectionForm.Sire, selectionForm.Dam, selectionForm.Owner,
		time.Now(),
		time.Now())

	if err != nil {
		tx.Rollback()
		return err
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}