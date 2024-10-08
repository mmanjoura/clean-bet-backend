package racing

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mmanjoura/clean-bet-backend/pkg/api/common"
	"github.com/mmanjoura/clean-bet-backend/pkg/database"
	"github.com/mmanjoura/clean-bet-backend/pkg/models"
)

type Selection struct {
	ID           int
	Name         string
	EventDate    time.Time
	NumberOfRuns int
}

// RacePicksSimulation handles the simulation of race picks and calculates win probabilities.
func DoAnalysis(c *gin.Context) {
	db := database.Database.DB

	var raceParams models.RaceParameters
	meetingsMap := make(map[string][]models.Selection)

	// Bind JSON input to optimalParams
	if err := c.ShouldBindJSON(&raceParams); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Query for today's runners
	rows, err := db.Query(`
		SELECT selection_id,
			selection_name,
			event_name,
			event_date,
			event_time,
			price,
			race_distance,
			race_category,
			track_condition,
			number_of_runners,
			race_track,
			race_class,
			selection_link,
			event_link
		FROM Meetings 
		WHERE DATE(event_date) = ? `,
		raceParams.EventDate)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var selection models.Selection

		// Use sql.NullString for nullable fields
		var selectionName, eventName, eventDate,
			eventTime, raceDistance, raceCategory,
			trackCondition,
			numberOfRunners,
			raceTrack, raceClass,
			odds, eventLink sql.NullString

		// Scan the row values into the nullable types
		if err := rows.Scan(
			&selection.ID,
			&selectionName,
			&eventName,
			&eventDate,
			&eventTime,
			&odds,
			&raceDistance,
			&raceCategory,
			&trackCondition,
			&numberOfRunners,
			&raceTrack,
			&raceClass,
			&selection.Link,
			&eventLink,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Convert sql.NullString to regular strings
		selection.Name = nullableToString(selectionName)
		selection.EventName = nullableToString(eventName)
		selection.EventDate = nullableToString(eventDate)
		selection.EventTime = nullableToString(eventTime)
		selection.RaceDistance = nullableToString(raceDistance)
		selection.RaceCategory = nullableToString(raceCategory)
		selection.TrackCondition = nullableToString(trackCondition)
		selection.NumberOfRunners = nullableToString(numberOfRunners)
		selection.RaceTrack = nullableToString(raceTrack)
		selection.RaceClass = nullableToString(raceClass)
		selection.EventLink = nullableToString(eventLink)
		selection.Odds = nullableToString(odds)

		key := selection.EventName + " " + selection.EventTime
		meetingsMap[key] = append(meetingsMap[key], selection)
	}

	mpResult := make(map[string][]models.AnalysisData)

	for key, meetings := range meetingsMap {

		fmt.Printf("Event: %s\n", key)
		for _, m := range meetings {
			fmt.Printf("  Selection: %s, Price: %s\n", m.EventName, m.EventTime)
			resultAnalysis, err := doAnalysisAndSave(raceParams, m)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			totalScore := calculateTotalScore(resultAnalysis)
			resultAnalysis.TotalScore = totalScore
			mpResult[resultAnalysis.EventTime] = append(mpResult[resultAnalysis.EventTime], resultAnalysis)
		}
	}

	// Iterate through mpResult
	for _, result := range mpResult {

		for _, r := range result {
			err = deleteAnalysis(db, r.EventDate, r.SelectionID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		for _, r := range result {

			err := insertAnalysis(db, r, raceParams.EventDate)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"simulationResults": nil})
}

func doAnalysisAndSave(raceParams models.RaceParameters, selection models.Selection) (models.AnalysisData, error) {

	db := database.Database.DB

	rows, err := db.Query(`
			SELECT
				DISTINCT
				COALESCE(selection_id, 0),
				selection_name,	
				position, 
				Age,
				Trainer,
				Sex,
				Sire,
				Dam,
				Owner,	
				race_class,	
				race_date,			
				COUNT(id) AS num_runs,
				MAX(race_date) AS last_run_date,
				MAX(race_date) - MIN(race_date) AS duration,
				COUNT(CASE WHEN position = '1' THEN 1 END) AS win_count,
				AVG(position) AS avg_position,
				AVG(rating) AS avg_rating,
				AVG(distance) AS avg_distance_furlongs,
				AVG(sp_odds) AS sp_odds,
				GROUP_CONCAT(position, ', ') AS all_positions,
				GROUP_CONCAT(distance, ', ') AS all_distances,
				GROUP_CONCAT(racecourse, ', ') AS all_racecources,
				GROUP_CONCAT(DATE(race_date), ', ') AS all_race_dates 
			FROM
				Forms 	
				WHERE selection_id = ?  order by race_date desc`,
		selection.ID)
	if err != nil {
		return models.AnalysisData{}, err
	}
	defer rows.Close()

	var data models.AnalysisData

	for rows.Next() {
		err := rows.Scan(
			&data.SelectionID,
			&data.SelectionName,
			&data.Position,
			&data.Age,
			&data.Trainer,
			&data.Sex,
			&data.Sire,
			&data.Dam,
			&data.Owner,
			&data.EventClass,
			&data.RaceDate,
			&data.NumRuns,
			&data.LastRunDate,
			&data.Duration,
			&data.WinCount,
			&data.AvgPosition,
			&data.AvgRating,
			&data.AvgDistanceFurlongs,
			&data.AvgOdds,
			&data.AllPositions,
			&data.AllDistances,
			&data.AllCources,
			&data.AllRaceDates,
		)

		if err != nil {
			continue
		}

		strRaceDate := string(data.RaceDate)[:10]
		const layout = "2006-01-02"
		// Convert the string dates to time.Time
		raceDate, err := time.Parse(layout, strRaceDate)
		if err != nil {
			return models.AnalysisData{}, err
		}
		analysisDate, err := time.Parse(layout, raceParams.EventDate)
		if err != nil {
			return models.AnalysisData{}, err
		}

		// When doing analysis only consider previous performances
		if raceDate.After(analysisDate) || raceDate.Equal(analysisDate) {
			continue
		}

		currentDistance := common.ConvertDistance(selection.RaceDistance)
		distance, err := strconv.ParseFloat(currentDistance, 64)
		if err != nil {
			return models.AnalysisData{}, err
		}
		data.CurrentDistance = distance
		data.EventName = selection.EventName
		data.EventTime = selection.EventTime
		data.EventLink = selection.EventLink
		data.SelecionLink = selection.Link

		perferedDistancd := preferredDistance(data.AllPositions, data.AllDistances, data.AllRaceDates)
		data.PreferedDistance = perferedDistancd
	}
	return data, nil
}

func filterSelectionsByID(selections []models.Selection, ids []int) []models.Selection {
	idSet := make(map[int]struct{})
	for _, id := range ids {
		idSet[id] = struct{}{}
	}

	var filteredSelections []models.Selection
	for _, selection := range selections {
		if _, exists := idSet[selection.ID]; exists {
			filteredSelections = append(filteredSelections, selection)
		}
	}

	return filteredSelections
}

func fetchConstantScore(db *sql.DB, category, item string) (float64, error) {
	var score float64
	row := db.QueryRow("SELECT score FROM score_constants WHERE category = ? AND item = ?", category, item)
	err := row.Scan(&score)
	if err != nil {
		return 0.0, err
	}
	return score, nil
}
func scoreRace(db *sql.DB, races []models.AnalysisData, raceParams models.RaceParameters) (models.ScoreBreakdown, error) {
	var raceTypeScore, courseScore, distanceScore, classScore, ageScore, positionScore float64
	var eventName, eventTime, selectionName, odds, trainer string

	return models.ScoreBreakdown{

		EventName:     eventName,
		EventTime:     eventTime,
		SelectionName: selectionName,
		Odds:          odds,
		Trainer:       trainer,
		RaceTypeScore: raceTypeScore,
		CourseScore:   courseScore,
		DistanceScore: distanceScore,
		ClassScore:    classScore,
		AgeScore:      ageScore,
		PositionScore: positionScore,
	}, nil
}

// CheckImprovement checks if the horse is improving over the distance.
func CheckImprovement(data []models.HistoricalData) string {
	sort.Slice(data, func(i, j int) bool {
		return data[i].Date.Before(data[j].Date)
	})

	improving := true
	for i := 1; i < len(data); i++ {
		if data[i].Distance < data[i-1].Distance {
			improving = false
			break
		}
	}

	if improving {
		return "Good Score"
	}
	return "Bad Score"
}

// ParseHistoricalData parses the historical data from a slice of strings.
func ParseHistoricalData(data [][]string) ([]models.HistoricalData, error) {
	var historicalData []models.HistoricalData
	for _, row := range data {

		date, err := time.Parse("2006-01-02", row[0])
		if err != nil {
			return nil, err
		}
		strDisting := common.ConvertDistance(row[1])
		distance, err := strconv.ParseFloat(strDisting, 64)
		if err != nil {
			return nil, err
		}
		historicalData = append(historicalData, models.HistoricalData{
			Date:     date,
			Position: row[0],
			Distance: distance,
		})
	}
	return historicalData, nil
}

// Helper function to parse race position
// func parsePosition(pos string) (position int, fieldSize int) {
// 	parts := strings.Split(pos, "/")
// 	position, err := strconv.Atoi(parts[0])
// 	if err != nil {
// 		return 0, 0
// 	}

// 	return position, fieldSize
// }

// Helper function to parse rating
func parseRating(rating string) float64 {
	if rating == "-" {
		return 0 // Or any neutral value for missing ratings
	}
	parsedRating, _ := strconv.ParseFloat(rating, 64)
	return parsedRating
}

// Helper function to calculate date score based on recency
func calculateDateScore(runDate string) float64 {
	const layout = "2006-01-02"
	date, _ := time.Parse(layout, runDate)
	daysAgo := time.Since(date).Hours() / 24.0

	// Closer races get higher score
	if daysAgo <= 30 {
		return 10.0
	} else if daysAgo <= 60 {
		return 8.0
	} else if daysAgo <= 90 {
		return 6.0
	} else {
		return 4.0
	}
}

// Main function to calculate scores
func calculateHorseScores(selectionRunDates, selectionPositions, selectionsRating []string) models.ScoreBreakdown {
	var dateScore, positionScore, ratingScore float64

	// Iterate through all races
	for i := range selectionRunDates {
		// Date Score
		dateScore += calculateDateScore(selectionRunDates[i])

		// Position Score
		// position, fieldSize := parsePosition(selectionPositions[i])
		// if fieldSize == 0 {
		// 	continue
		// }
		// positionScore += 10.0 - float64(position)/float64(fieldSize)

		// Rating Score
		ratingScore += parseRating(selectionsRating[i])
	}
	return models.ScoreBreakdown{

		DateScore:     dateScore,
		PositionScore: positionScore,
		RatingScore:   ratingScore,
	}
}

// New function to fetch age score based on the race distance
func fetchAgeScore(db *sql.DB, age int, distance float64) (float64, error) {
	var score float64
	var err error

	if distance > 12.0 {
		// Greater than 12 furlongs
		score, err = fetchConstantScore(db, "Age-greater-12f", strconv.Itoa(age))
	} else {
		// Less than or equal to 12 furlongs
		score, err = fetchConstantScore(db, "Age-bellow-12f", strconv.Itoa(age))
	}

	return score, err
}

// Function to filter SelectionForms based on selection_id
func filterSelectionFormsByID(forms []models.SelectionForm, selectionID int) []models.SelectionForm {
	var filteredForms []models.SelectionForm

	for _, form := range forms {
		if form.SelectionID == selectionID {
			filteredForms = append(filteredForms, form)
		}
	}

	return filteredForms
}

func stringInSlice(target string, slice []string) bool {
	for _, item := range slice {
		item := strings.Split(item, "-")[0]
		if item == target {
			return true
		}
	}
	return false
}

// Scoring Function
func ScoreSelection(selection models.AnalysisData, params models.RaceParameters, limit int) float64 {

	var score float64
	config := database.Database.Config

	deltaStr := config["delta"]
	delta, err := strconv.ParseFloat(deltaStr, 64)
	if err != nil {
		fmt.Println("Error converting to float64:", err)
	}

	if math.Abs(selection.CurrentDistance-selection.PreferedDistance) <= delta {

		// Assign a random score between 5 and 20
		score += 5 + 15*rand.Float64()

	}

	// 0. gelding Score
	if selection.Sex == "Gelding" {
		score += 2 + float64(selection.Duration)
	} else {
		score += float64(selection.Duration)
	}

	totalRuns, err := strconv.Atoi(config["total_runs"])
	if err != nil {
		fmt.Println("Error converting to integer:", err)
	}

	// 3. Number of runs so far
	if selection.NumRuns < totalRuns {
		score += float64(selection.NumRuns)
	}

	// Distance Analysis
	distances := strings.Split(selection.AllDistances, ",")
	if len(distances) > limit {
		distances = distances[:limit]
	}

	var totalDistance float64
	for _, distance := range distances {
		distance = strings.TrimSpace(distance)
		convertedDistance := common.ConvertDistance(distance)
		fd, err := strconv.ParseFloat(convertedDistance, 64)
		if err != nil {
			continue
		}
		totalDistance += fd
	}

	avgDistance := totalDistance / float64(len(distances))

	// Distance and Age-Based Scoring
	distanceDiff := math.Abs(avgDistance - selection.CurrentDistance)
	switch {
	case avgDistance <= 12:

		score += calculateDistanceScore(distanceDiff, []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9}, []float64{15, 13, 11, 9, 7, 5, 3, 1, 0})
	default:

		score += calculateDistanceScore(distanceDiff, []float64{0.5, 1.0, 1.5, 2.0, 2.5, 3.0, 3.5, 4.0, 4.5}, []float64{9, 8, 7, 6, 5, 4, 3, 2, 1})
	}

	// Position Analysis
	positions := strings.Split(selection.AllPositions, ",")
	score += calculatePositionScore(positions, limit)

	return score
}

func safeDivide(numerator, denominator float64) float64 {
	if denominator == 0.0 {
		// Handle division by zero case, maybe return 0 or some other default value.
		fmt.Println("Warning: Division by zero detected.")
		return math.Inf(1) // Or return a different value that makes sense in your context.
	}
	return numerator / denominator
}

// Function to get the highest score selection for each event time
func getHighestScoreByTime(sortedResults []models.SelectionResult) map[string]models.SelectionResult {
	// Group the results by EventTime
	groupedResults := make(map[string]models.SelectionResult)

	for _, result := range sortedResults {
		currentBest, exists := groupedResults[result.EventTime]
		if !exists || result.TotalScore > currentBest.TotalScore {
			groupedResults[result.EventTime] = result
		}
	}

	return groupedResults
}

// Function to get the second highest score selection for each event time
func getSecondHighestScoreByTime(sortedResults []models.SelectionResult) map[string]models.SelectionResult {
	// Group the results by EventTime
	groupedResults := make(map[string][]models.SelectionResult)

	for _, result := range sortedResults {
		groupedResults[result.EventTime] = append(groupedResults[result.EventTime], result)
	}

	// Prepare the map to store the second highest score selections
	secondHighestResults := make(map[string]models.SelectionResult)

	for time, results := range groupedResults {
		// Sort the results by TotalScore in descending order
		sort.Slice(results, func(i, j int) bool {
			return results[i].TotalScore > results[j].TotalScore
		})

		// Check if there are at least two results to get the second highest
		if len(results) > 1 {
			secondHighestResults[time] = results[1] // Index 1 is the second highest
		}
	}

	return secondHighestResults
}

// Function to check if any year from the input string exists in the dates array
func yearExistsInDates(yearsStr string, dates []string) bool {
	// Split the years string by comma and trim spaces
	years := strings.Split(yearsStr, ",")
	for i, year := range years {
		years[i] = strings.TrimSpace(year)
	}

	// Iterate over the dates array to check for each year
	for _, date := range dates {
		dateYear := strings.Split(date, "-")[0] // Extract year from the date

		// Check if extracted year exists in the list of years
		for _, year := range years {
			if dateYear == year {
				return true
			}
		}
	}

	return false
}

// Function to check if any position from the input string exists in the positions array
func positionExistsInArray(positionsStr string, positionsArray []string) bool {
	// Split the positions string by comma and trim spaces
	positions := strings.Split(positionsStr, ",")
	for i, pos := range positions {
		positions[i] = strings.TrimSpace(pos)
	}

	// Iterate over the positionsArray to check for each position
	for _, pos := range positionsArray {
		// Split the position by '/' and extract the second part
		parts := strings.Split(pos, "/")
		if len(parts) < 2 {
			continue // Skip if the format is invalid
		}
		horsePosition := parts[1]

		// Check if the extracted horse position exists in the list of positions
		for _, position := range positions {
			if horsePosition == position {
				return true
			}
		}
	}

	return false
}

// Function to check if the age exists in the given string of ages
func ageExistsInString(agesStr string, ageVariable string) bool {
	// Split the ages string by comma and trim spaces
	ages := strings.Split(agesStr, ",")
	for i, age := range ages {
		ages[i] = strings.TrimSpace(age)
	}

	// Use a regular expression to extract the age from the age variable string
	re := regexp.MustCompile(`\d+`)
	matchedAge := re.FindString(ageVariable)

	// Check if the extracted age exists in the list of ages
	for _, age := range ages {
		if matchedAge == age {
			return true
		}
	}

	return false
}

func getTop3ScoresByTime(sortedResults []models.SelectionResult) map[string][]models.SelectionResult {
	// Group the results by EventTime
	groupedResults := make(map[string][]models.SelectionResult)

	for _, result := range sortedResults {
		// Get the current list of results for the EventTime
		currentResults, exists := groupedResults[result.EventTime]

		if !exists {
			// If there are no results yet for this EventTime, add the current result
			groupedResults[result.EventTime] = []models.SelectionResult{result}
		} else {
			// Append the current result to the existing list
			currentResults = append(currentResults, result)

			// Sort the results by TotalScore in descending order
			sort.Slice(currentResults, func(i, j int) bool {
				return currentResults[i].TotalScore > currentResults[j].TotalScore
			})

			// Keep only the top 3 scores
			if len(currentResults) > 3 {
				currentResults = currentResults[:3]
			}

			groupedResults[result.EventTime] = currentResults
		}
	}

	return groupedResults
}

// func calculateAveragePosition(positionsString string, n int) float64 {
// 	// Split the input string into an array of positions
// 	positionsArray := strings.Split(positionsString, ", ")

// 	// Check if n is greater than the length of the positions array
// 	if n > len(positionsArray) {
// 		n = len(positionsArray)
// 	}

// 	// Variable to store the total of the numerators
// 	totalPosition := 0

// 	// Iterate over the first n elements to calculate the sum of positions
// 	for i := 0; i < n; i++ {
// 		// Split each position into numerator and denominator
// 		positionParts := strings.Split(positionsArray[i], "/")

// 		// Convert the numerator to an integer
// 		numerator, err := strconv.Atoi(positionParts[0])
// 		if err != nil {
// 			fmt.Println("Error converting to integer:", err)
// 			continue
// 		}

// 		// Add the numerator to the total
// 		totalPosition += numerator
// 	}

// 	// Calculate the average position
// 	averagePosition := float64(totalPosition) / float64(n)
// 	return averagePosition
// }

// Function to check if the age exists in the given string of ages
func ageInString(agesStr string) string {
	// Split the ages string by comma and trim spaces
	ages := strings.Split(agesStr, ",")
	for i, age := range ages {
		ages[i] = strings.TrimSpace(age)
	}

	// Use a regular expression to extract the age from the age variable string
	re := regexp.MustCompile(`\d+`)
	matchedAge := re.FindString(agesStr)

	return matchedAge
}

// Convert sql.NullFloat64 to a float64
func nullableToFloat(nf sql.NullFloat64) float64 {
	if nf.Valid {
		return nf.Float64
	}
	return 0.0 // Return a default value (e.g., 0.0) if NULL
}

// Convert sql.NullString to a regular string
func nullableToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return "" // Return empty string if NULL
}

// Calculate Distance Score
func calculateDistanceScore(distanceDiff float64, thresholds, scores []float64) float64 {
	var score float64
	for i, threshold := range thresholds {
		if distanceDiff <= threshold {
			score += scores[i]
			break
		}
	}
	return score
}

// Calculate Position Score
func calculatePositionScore(positions []string, limit int) float64 {
	var score float64
	if len(positions) > limit {
		positions = positions[:limit]
	}

	for _, pos := range positions {
		pos = strings.TrimSpace(pos)
		if strings.Contains(pos, "F") || strings.Contains(pos, "PU") || strings.Contains(pos, "U") || strings.Contains(pos, "R") {
			score -= 5
		}
		if strings.Contains(pos, "/") {
			p := strings.Split(pos, "/")
			if len(p) != 2 {
				continue
			}
			numerator, err1 := strconv.Atoi(strings.TrimSpace(p[0]))
			denominator, err2 := strconv.Atoi(strings.TrimSpace(p[1]))
			if err1 != nil || err2 != nil || denominator == 0 {
				score -= 1
				continue
			}
			score += math.Round(safeDivide(float64(denominator), float64(numerator))) * 10
		}
	}
	return score
}

func extractNumber(s string) (string, error) {
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return "", fmt.Errorf("no number found")
	}
	number, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", err
	}
	return strconv.Itoa(number), nil
}

// DetermineBetType determines the bet type based on the odds
func DetermineBetType(odds string) string {
	// Split the odds string by "/"
	parts := strings.Split(odds, "/")
	if len(parts) == 2 {
		numerator, _ := strconv.Atoi(parts[0])
		denominator, _ := strconv.Atoi(parts[1])

		// Check for BetType based on the odds
		if float64(numerator)/float64(denominator) < 1.0 {
			return "win bet"
		} else if float64(numerator)/float64(denominator) > 4.0 {
			return "place bet"
		}
	}
	// Default to an empty BetType if criteria are not met
	return ""
}

// CalculatePotentialReturn calculates the potential return based on BetType and odds
func CalculatePotentialReturn(betType string, odds string, amount float64) float64 {
	// Split the odds string by "/"
	parts := strings.Split(odds, "/")
	if len(parts) == 2 {
		numerator, _ := strconv.ParseFloat(parts[0], 64)
		denominator, _ := strconv.ParseFloat(parts[1], 64)

		// Calculate potential return for "win bet" or "place bet"
		if betType == "win bet" {
			oddsMultiplier := (numerator / denominator)
			return amount * oddsMultiplier
		}
		if betType == "place bet" {
			oddsMultiplier := (numerator / denominator)
			return amount * oddsMultiplier
		}

	}
	// Default potential return is 0
	return 0
}

// AddBetTypeAndReturnsToSelections processes the input map and adds BetType, SelectionPosition, and PotentialReturn fields
func AddBetTypeAndReturnsToSelections(selectionsMap map[string][]models.SelectionResult, amount float64, date string) map[string][]models.SelectionResult {
	for eventTime, selections := range selectionsMap {
		for i := range selections {
			// Determine the BetType for each selection
			selections[i].BetType = DetermineBetType(selections[i].Odds)

			if strings.Contains(selections[i].SelectionPosition, "/") {
				selections[i].SelectionPosition = strings.Split(selections[i].SelectionPosition, "/")[0]
			}

			// Calculate the PotentialReturn only if SelectionPosition is 1
			if selections[i].SelectionPosition == "1" {
				selections[i].PotentialReturn = CalculatePotentialReturn(selections[i].BetType, selections[i].Odds, amount)
			} else {
				selections[i].PotentialReturn = 0 // No potential return if position is not 1
			}

		}

		// Update the selections in the map
		selectionsMap[eventTime] = selections
	}
	return selectionsMap
}

// Function to get the selection with the least number of runs
func getSelectionWithLeastRuns(db *sql.DB, eventName, eventTime, eventDate string) ([]Selection, error) {
	// SQL query to find the selection ID with the least number of runs

	var selections []Selection
	rows, err := db.Query(`
					SELECT 
						SelectionsForm.selection_id,
						EventRunners.selection_name,
						EventRunners.event_date,
						COUNT(*) AS number_of_runs
					FROM 
						Forms 
						INNER JOIN EventRunners ON SelectionsForm.selection_id = EventRunners.selection_id
						WHERE SelectionsForm.racecourse = ? and EventRunners.event_time = ? and DATE(EventRunners.event_date) = ?
					GROUP BY 
						SelectionsForm.selection_id
						Order by number_of_runs`, eventName, eventTime, eventDate)

	if err != nil {
		return []Selection{}, err
	}
	defer rows.Close()

	var selection Selection

	// Get the result
	if rows.Next() {
		err := rows.Scan(
			&selection.ID,
			&selection.Name,
			&selection.EventDate,
			&selection.NumberOfRuns,
		)
		if err != nil {
			return []Selection{}, err
		}
		selections = append(selections, selection)
	}

	// Check for any error encountered during iteration
	if err := rows.Err(); err != nil {
		return []Selection{}, err
	}

	// Return the selection ID and number of runs
	return selections, nil
}

func getTopScoreByTime(sortedResults []models.SelectionResult) map[string][]models.SelectionResult {
	// Create a map to store the top score by EventTime
	topScores := make(map[string][]models.SelectionResult)

	for _, result := range sortedResults {
		// Check if there's already a result for this EventTime
		currentTop, exists := topScores[result.EventTime]

		if !exists || result.TotalScore > currentTop[0].TotalScore {
			// If there's no result yet or the current result has a higher score, update it
			topScores[result.EventTime] = []models.SelectionResult{result}
		}
	}

	return topScores
}

func getTop2ScoresByTime(sortedResults []models.SelectionResult) map[string][]models.SelectionResult {
	// Create a map to store the top 2 scores by EventTime
	top2Scores := make(map[string][]models.SelectionResult)

	for _, result := range sortedResults {
		// Check if there are already results for this EventTime
		currentTopScores, exists := top2Scores[result.EventTime]

		if !exists {
			// If no results yet for this EventTime, initialize with the current result
			top2Scores[result.EventTime] = []models.SelectionResult{result}
		} else {
			// Append the current result to the existing list
			currentTopScores = append(currentTopScores, result)

			// Sort the results by TotalScore in descending order
			sort.Slice(currentTopScores, func(i, j int) bool {
				return currentTopScores[i].TotalScore > currentTopScores[j].TotalScore
			})

			// Keep only the top 2 scores
			if len(currentTopScores) > 2 {
				currentTopScores = currentTopScores[:2]
			}

			// Update the map with the top 2 results
			top2Scores[result.EventTime] = currentTopScores
		}
	}

	return top2Scores
}

func preferredDistance(performances, distances, dates string) float64 {
	// Split input strings into slices
	performanceList := strings.Split(performances, ", ")
	distanceList := strings.Split(distances, ", ")
	dateList := strings.Split(dates, ", ")

	// Check if the lengths match
	if len(performanceList) != len(distanceList) || len(distanceList) != len(dateList) {
		return 0.0
	}

	// Create a map to store total scores and count per distance
	distanceScores := make(map[string]float64)
	distanceCount := make(map[string]int)

	// Calculate scores based on performance
	for i := 0; i < len(performanceList); i++ {
		parts := strings.Split(performanceList[i], "/")
		if len(parts) != 2 {
			continue
		}

		// Parse position and total runners
		position, err1 := strconv.Atoi(parts[0])
		totalRunners, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			continue
		}

		// Calculate score (lower is better)
		score := float64(position) / float64(totalRunners)

		// Update the total score and count for the distance
		distanceScores[distanceList[i]] += score
		distanceCount[distanceList[i]]++
	}

	// Determine the preferred distance with the lowest average score
	var preferredDistance string
	lowestAverageScore := float64(1e9) // Initialize with a very high value

	for distance, totalScore := range distanceScores {
		if distanceCount[distance] > 0 {
			averageScore := totalScore / float64(distanceCount[distance])
			if averageScore < lowestAverageScore {
				lowestAverageScore = averageScore
				preferredDistance = distance
			}
		}
	}
	ret := common.ConvertDistance(preferredDistance)
	// convert to float64
	preferredDistanceFloat, err := strconv.ParseFloat(ret, 64)
	if err != nil {
		return 0.0
	}

	return preferredDistanceFloat
}

func convertDistance(preferredDistance string) {
	panic("unimplemented")
}

// findClosestHorse finds the most likely winner based on the given criteria.
func findClosestHorse(horses []models.AnalysisData) []models.AnalysisData {
	// Step 1: Filter horses based on distance preference
	currentDistance := horses[0].CurrentDistance
	var filteredHorses []models.AnalysisData
	for _, horse := range horses {
		distanceDiff := math.Abs(horse.PreferedDistance - currentDistance)
		if (currentDistance < 12.0 && distanceDiff < 4.0) || (currentDistance >= 12.0 && distanceDiff <= 2.0) {
			filteredHorses = append(filteredHorses, horse)
		}
	}

	// Step 2: If no horse matches the distance preference, return the closest by distance
	if len(filteredHorses) == 0 {
		minDistanceDiff := math.Abs(horses[0].PreferedDistance - currentDistance)
		for _, horse := range horses[1:] {
			distanceDiff := math.Abs(horse.PreferedDistance - currentDistance)
			if distanceDiff < minDistanceDiff {
				minDistanceDiff = distanceDiff
			}
		}

		return filteredHorses
	}

	// Step 3: Sort filtered horses based on Score (highest), AvgPosition (lowest), AvgRating (lowest)
	sort.Slice(filteredHorses, func(i, j int) bool {
		if filteredHorses[i].TotalScore != filteredHorses[j].TotalScore {
			return filteredHorses[i].TotalScore > filteredHorses[j].TotalScore
		}
		if filteredHorses[i].AvgPosition != filteredHorses[j].AvgPosition {
			return filteredHorses[i].AvgPosition < filteredHorses[j].AvgPosition
		}
		return filteredHorses[i].AvgRating < filteredHorses[j].AvgRating
	})

	// Return the best candidate
	return filteredHorses
}

func Contains[T comparable](slice []T, element T) bool {
	for _, v := range slice {
		if v == element {
			return true
		}
	}
	return false
}

func insertAnalysis(db *sql.DB, data models.AnalysisData, eventDate string) error {

	// Prepare the INSERT statement
	stmt, err := db.Prepare(`
		INSERT INTO Analysis (
					event_link,
					selection_link,
					event_date, race_date, selection_id,
					selection_name, odds, age,
					clean_bet_score, average_position,
					average_rating, event_name,
					event_time, selection_position, num_runners, number_runs, prefered_distance, current_distance)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	if data.RaceDate != "" {
		raceDate := string(data.RaceDate)[:10]

		ageStr := strings.Split(data.Age, " ")[0]
		ageInt, _ := strconv.Atoi(ageStr)
		numberOfRuns := strings.Split(data.AllCources, ",")

		numRunners := strings.Split(data.NumberOfRunners, " ")[0]
		intNumRunners, _ := strconv.Atoi(numRunners)

		price, err := selectionOdds(db, eventDate, data.SelectionID)

		if err != nil {
			return err
		}

		// Execute the INSERT statement
		_, err = stmt.Exec(
			data.EventLink,
			data.SelecionLink,
			eventDate,
			raceDate,
			data.SelectionID,
			data.SelectionName,
			price,
			ageInt,
			math.Round(data.TotalScore*1000)/1000,
			data.AvgPosition,
			math.Round(data.AvgRating*1000)/1000,
			data.EventName,
			data.EventTime,
			data.Position,
			intNumRunners,
			len(numberOfRuns),
			math.Round(data.PreferedDistance*1000)/1000,
			math.Round(data.CurrentDistance*1000)/1000,
		)
		if err != nil {
			return err
		}
	}

	// Return nil if no error occurred
	return nil
}
func deleteAnalysis(db *sql.DB, eventDate string, selectionId int) error {
	// Check if a record with the same event_date and selection_id exists
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM Analysis
			WHERE DATE(event_date) = ? and selection_id = ?
		)
	`, eventDate, selectionId).Scan(&exists)
	if err != nil {
		return err
	}

	// If a record exists, delete it
	if exists {
		_, err = db.Exec(`
			DELETE FROM Analysis
			WHERE DATE(event_date) = ? and selection_id = ?
		`, eventDate, selectionId)
		if err != nil {
			return err
		}
	}

	// Return nil if no error occurred
	return nil
}

func selectionOdds(db *sql.DB, eventDate string, selectionId int) (string, error) {
	// Check if a record with the same event_date and selection_id exists
	var price string
	err := db.QueryRow(`
	SELECT price  FROM Meetings where DATE(event_date) = ? and selection_id = ?`, eventDate, selectionId).Scan(&price)
	if err != nil {
		return "", err
	}

	// Return nil if no error occurred
	return price, nil
}

// Parse the position string (e.g., "5/9") to get the finishing position
func parsePosition(position string) int {
	parts := strings.Split(position, "/")
	if len(parts) > 0 {
		pos, err := strconv.Atoi(parts[0])
		if err == nil {
			return pos
		}
	}
	return -1 // return -1 for invalid positions
}

// Calculate the average of the last run positions
func calculateAveragePosition(positions string) float64 {
	posList := strings.Split(positions, ",")
	total := 0
	count := 0

	for _, pos := range posList {
		position := parsePosition(strings.TrimSpace(pos))
		if position > 0 {
			total += position
			count++
		}
	}

	if count > 0 {
		return float64(total) / float64(count)
	}
	return 0.0
}

// Score based on last race positions
func scoreLastRunPositions(averagePosition float64) float64 {
	// Example scoring based on average position:
	// Lower positions (better finishes) get higher scores
	if averagePosition <= 1.5 {
		return 10
	} else if averagePosition <= 3 {
		return 8
	} else if averagePosition <= 5 {
		return 6
	} else {
		return 3
	}
}

// Score based on days since last run
func scoreDaysSinceLastRun(days int) float64 {
	// Ideal rest time is between 10 and 21 days
	if days < 10 {
		return 3 // too soon
	} else if days <= 21 {
		return 10 // ideal range
	} else if days <= 35 {
		return 7 // still good
	} else {
		return 4 // long break, might be under-prepared
	}
}

// Score based on distance suitability
func scoreDistanceSuitability(currentDistance, preferredDistance float64) float64 {
	// If the current race distance matches the preferred distance, give a higher score
	if currentDistance == preferredDistance {
		return 15
	} else if currentDistance < preferredDistance {
		return 10 // slightly shorter distance
	} else {
		return 5 // longer distance
	}
}

// Score based on race class
func scoreRaceClass(eventClass int) float64 {
	switch eventClass {
	case 1:
		return 10
	case 2:
		return 8
	case 3:
		return 7
	case 4:
		return 6
	default:
		return 4
	}
}

// Score based on win count
func scoreWinCount(winCount int, previousRuns int) float64 {
	// Higher win percentage should get more points
	if previousRuns == 0 {
		return 0 // no history
	}

	winRate := float64(winCount) / float64(previousRuns)
	if winRate >= 0.5 {
		return 10
	} else if winRate >= 0.25 {
		return 7
	} else {
		return 4
	}
}

// Score based on odds
func scoreOdds(averageOdds float64) float64 {
	// Lower odds suggest a higher likelihood of winning
	if averageOdds < 3 {
		return 9
	} else if averageOdds < 5 {
		return 7
	} else if averageOdds < 10 {
		return 5
	} else {
		return 3
	}
}

// Total score calculation based on all factors
func calculateTotalScore(data models.AnalysisData) float64 {
	totalScore := 0.0

	// Calculate individual component scores
	averagePosition := calculateAveragePosition(data.AllPositions)
	totalScore += scoreLastRunPositions(averagePosition)
	totalScore += scoreDaysSinceLastRun(data.RecoveryDays)
	totalScore += scoreDistanceSuitability(data.CurrentDistance, data.PreferedDistance)
	// totalScore += scoreRaceClass(data.EventClass)
	totalScore += scoreWinCount(data.WinCount, data.NumRuns)
	totalScore += scoreOdds(data.AvgOdds)

	// Add any additional factors as needed

	return totalScore
}

