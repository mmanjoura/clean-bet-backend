package common

import (
	"database/sql"
	"strconv"
	"strings"
)

func ConvertDistance(distanceStr string) string {
	// if this string contain "."
	if strings.Contains(distanceStr, ".") {
		alreadyFormated := strings.Split(distanceStr, ".")
		if len(alreadyFormated[0]) > 0 {
			return distanceStr
		}
	}

	_, err := strconv.ParseFloat(distanceStr, 64)
	if err == nil {
		return distanceStr
	}

	parts := strings.Split(distanceStr, " ")
	furlongs := 0.0
	for _, part := range parts {
		if strings.Contains(part, "m") {
			miles, err := strconv.ParseFloat(strings.TrimSuffix(part, "m"), 64)
			if err == nil {
				furlongs += miles * 8
			}
		} else if strings.Contains(part, "f") {
			f, err := strconv.ParseFloat(strings.TrimSuffix(part, "f"), 64)
			if err == nil {
				furlongs += f
			}
		} else if strings.Contains(part, "y") {
			// Assume 220 yards = 1 furlong (approximately)
			yards, err := strconv.ParseFloat(strings.TrimSuffix(part, "y"), 64)
			if err == nil {
				furlongs += yards / 220.0
			}
		}
	}
	return strconv.FormatFloat(furlongs, 'f', -1, 64)
}

// Function to remove duplicate odds patterns
func RemoveDuplicateOdds(odds string) string {
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

func NullableToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return "NULL"
}

