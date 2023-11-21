package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

const (
	// DBDriver is the driver for the database
	DBDriver = "postgres"
)

const USERS_TABLE_CREATE_SQL = `
CREATE TABLE IF NOT EXISTS Users (
    UserID SERIAL PRIMARY KEY,
    Username VARCHAR(255) NOT NULL UNIQUE,
    Email VARCHAR(255) NOT NULL UNIQUE,
    PasswordHash VARCHAR(255) NOT NULL,
    CreatedAt TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
`

type Ingredient struct {
	IngredientID int
	Name         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

const INGREDIENTS_TABLE_CREATE_SQL = `
CREATE TABLE IF NOT EXISTS Ingredients (
    IngredientID SERIAL PRIMARY KEY,
    Name VARCHAR(255) NOT NULL,
    CreatedAt TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
`

type Nutrient struct {
	NutrientID int
	Name       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

const NUTRIENTS_TABLE_CREATE_SQL = `
CREATE TABLE IF NOT EXISTS Nutrients (
    NutrientID SERIAL PRIMARY KEY,
    Name VARCHAR(255) NOT NULL,
    CreatedAt TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
`

const NUTRIENT_VALUES_TABLE_CREATE_SQL = `
CREATE TABLE IF NOT EXISTS Nutrient_Values (
    IngredientID INT NOT NULL,
    NutrientID INT NOT NULL,
    AmountPer100g NUMERIC(10,2) NOT NULL,
    PRIMARY KEY (IngredientID, NutrientID),
    FOREIGN KEY (IngredientID) REFERENCES Ingredients(IngredientID) ON UPDATE CASCADE ON DELETE CASCADE,
    FOREIGN KEY (NutrientID) REFERENCES Nutrients(NutrientID) ON UPDATE CASCADE ON DELETE CASCADE
);
`

type Meal struct {
	MealID    int
	UserID    int
	Date      time.Time
	Time      time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// const MEALS_TABLE_CREATE_SQL = `
// CREATE TABLE IF NOT EXISTS Meals (
//
//	    MealID SERIAL PRIMARY KEY,
//		Name VARCHAR(255) NOT NULL,
//	    UserID INT NOT NULL,
//	    Date DATE NOT NULL,
//	    Time TIME NOT NULL,
//	    FOREIGN KEY (UserID) REFERENCES Users(UserID) ON UPDATE CASCADE,
//	    CreatedAt TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
//	    UpdatedAt TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
//
// );
// `
const MEALS_TABLE_CREATE_SQL = `
CREATE TABLE IF NOT EXISTS Meals (
    MealID SERIAL PRIMARY KEY,
	Name VARCHAR(255) NOT NULL,
    Date DATE NOT NULL,
    Time TIME NOT NULL,
    CreatedAt TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
`

const MEAL_INGREDIENTS_TABLE_CREATE_SQL = `
CREATE TABLE IF NOT EXISTS Meal_Ingredients (
    MealID INT NOT NULL,
    IngredientID INT NOT NULL,
    QuantityInGrams NUMERIC(10,2) NOT NULL,
    PRIMARY KEY (MealID, IngredientID),
    FOREIGN KEY (MealID) REFERENCES Meals(MealID) ON UPDATE CASCADE ON DELETE CASCADE,
    FOREIGN KEY (IngredientID) REFERENCES Ingredients(IngredientID) ON UPDATE CASCADE ON DELETE CASCADE
);
`

func initDB() *sql.DB {
	db_username := os.Getenv("DB_USERNAME")
	db_password := os.Getenv("DB_PASSWORD")
	db_host := os.Getenv("DB_HOSTNAME")
	db_port := os.Getenv("DB_PORT")
	db_database := os.Getenv("DB_NAME")

	connStr := "postgres://" + db_username + ":" + db_password + "@" + db_host + ":" + db_port + "/" + db_database + "?sslmode=disable"

	db, err := sql.Open(DBDriver, connStr)
	if err != nil {
		log.Fatal(err)
	}

	// set max connections
	db.SetMaxOpenConns(20)
	// set max idle connections
	db.SetMaxIdleConns(20)
	// set max connection lifetime
	db.SetConnMaxLifetime(5)
	// set connection timeout
	db.SetConnMaxIdleTime(5)

	// check connection
	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}

	log.Println("Database is ready!")

	// create table if not exists
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	_, err = tx.Exec(USERS_TABLE_CREATE_SQL)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	_, err = tx.Exec(INGREDIENTS_TABLE_CREATE_SQL)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	_, err = tx.Exec(NUTRIENTS_TABLE_CREATE_SQL)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	_, err = tx.Exec(NUTRIENT_VALUES_TABLE_CREATE_SQL)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	_, err = tx.Exec(MEALS_TABLE_CREATE_SQL)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	_, err = tx.Exec(MEAL_INGREDIENTS_TABLE_CREATE_SQL)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	tx.Commit()

	log.Println("Database tables created!")

	return db
}
