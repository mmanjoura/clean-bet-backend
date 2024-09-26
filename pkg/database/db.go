package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mmanjoura/clean-bet-backend/pkg/models"
)

// DbInstance holds the database connection and configuration settings
type DbInstance struct {
	DB     *sql.DB
	Config map[string]string
}

// Global variable to store the database instance
var Database DbInstance

// ConnectDatabase initializes the database connection and retrieves configurations
func ConnectDatabase() {
	var err error

	// Open a connection to the SQLite database
	db, err := sql.Open("sqlite3", "./clean-bet.db")
	if err != nil {
		// Log fatal error and exit if database connection fails
		log.Fatalf("Failed to connect to the database: %v\n", err)
		os.Exit(2)
	}

	// Retrieve configurations from the database
	configurations, err := GetConfigs(db)
	if err != nil {
		fmt.Printf("Error retrieving configurations: %v\n", err)
		checkErr(err)
	}

	// Set the global Database variable with the database connection and configurations
	Database = DbInstance{
		DB:     db,
		Config: configurations,
	}
}

// checkErr logs fatal errors and stops execution if an error is encountered
func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// FormatLimitOffset returns a formatted string for SQL LIMIT and OFFSET clauses
func FormatLimitOffset(limit, offset int) string {
	return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
}

// GetConfigs fetches configurations from the database and returns them as a map
func GetConfigs(db *sql.DB) (map[string]string, error) {
	// Create a context for the query execution
	ctx := context.Background()

	// Execute SQL query to retrieve all configurations ordered by ID
	rows, err := db.QueryContext(ctx, `SELECT ID, key, value FROM Configurations ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // Ensure rows are closed once processing is complete

	// Create a map to store the configurations
	configurations := make(map[string]string)

	// Iterate over the rows and scan each configuration into the map
	for rows.Next() {
		configuration, err := scanConfiguration(rows)
		if err != nil {
			return nil, err
		}

		// Only add the configuration if its ID is not zero
		if configuration.ID != 0 {
			configurations[configuration.Key] = configuration.Value
		}
	}

	return configurations, nil
}

// scanConfiguration reads a single row of configuration data and returns a Configuration model
func scanConfiguration(rows *sql.Rows) (models.Configuration, error) {
	// Create an empty Configuration model
	configuration := models.Configuration{}

	// Scan the row data into the configuration model
	err := rows.Scan(&configuration.ID, &configuration.Key, &configuration.Value)

	return configuration, err
}
