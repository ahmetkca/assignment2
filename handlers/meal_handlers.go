package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type MealHandler struct {
	db *sql.DB
}

func NewMealHandler(db *sql.DB) *MealHandler {
	return &MealHandler{db: db}
}

type CreateMealRequest struct {
	Name        string    `json:"name" validate:"required"`
	DateTime    time.Time `json:"date_time" validate:"required"`
	Ingredients []struct {
		IngredientID  int64   `json:"ingredient_id"`
		AmountInGrams float64 `json:"amount_in_grams"`
	} `json:"ingredients"`
}

type CreateMealResponse struct {
	MealID int64 `json:"meal_id" validate:"required"`
}

func (m *MealHandler) CreateMealHandle(w http.ResponseWriter, r *http.Request) {
	var mealRequest *CreateMealRequest
	err := json.NewDecoder(r.Body).Decode(&mealRequest)
	if err != nil {
		log.Println("Error while decoding request body")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// create transaction
	tx, err := m.db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Println("Error while creating transaction")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// insert into Meals table
	rows, err := tx.QueryContext(r.Context(), "INSERT INTO Meals (Name, Date, Time) VALUES ($1, $2, $3) RETURNING MealID", mealRequest.Name, mealRequest.DateTime, mealRequest.DateTime)
	if err != nil {
		log.Println("Error while inserting into Meals table")
		log.Println(err)
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var mealID int64
	if rows.Next() {
		err = rows.Scan(&mealID)
		if err != nil {
			log.Println("Error while scanning mealID")
			log.Println(err)
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	err = rows.Close()
	if err != nil {
		log.Println("Error while closing result of INSERT INTO Meals")
		log.Println(err)
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// insert into MealIngredients table
	for _, ingredient := range mealRequest.Ingredients {
		_, err = tx.ExecContext(r.Context(), "INSERT INTO Meal_Ingredients (MealID, IngredientID, QuantityInGrams) VALUES ($1, $2, $3)", mealID, ingredient.IngredientID, ingredient.AmountInGrams)
		if err != nil {
			log.Println("Error while inserting into MealIngredients table")
			log.Println(err)
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Println("Error while committing transaction")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(&CreateMealResponse{MealID: mealID})
	return
}

type GetMealResponse struct {
	MealID      int64     `json:"meal_id" validate:"required"`
	Name        string    `json:"name" validate:"required"`
	Date        time.Time `json:"date" validate:"required"`
	Time        time.Time `json:"time"`
	Ingredients []struct {
		IngredientID  int64   `json:"ingredient_id"`
		AmountInGrams float64 `json:"amount_in_grams"`
		Name          string  `json:"name"`
	} `json:"ingredients"`
}

// GET /api/meals/{id}
func (m *MealHandler) GetMealHandle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	result, err := m.db.Query("SELECT MealID, Name, Date, Time FROM Meals WHERE MealID = $1", vars["id"])
	if err != nil {
		log.Println("Error while querying Meals table")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var mealID int64
	var mealName string
	var mealDate time.Time
	var mealTime time.Time
	if result.Next() {
		err = result.Scan(&mealID, &mealName, &mealDate, &mealTime)
		if err != nil {
			log.Println("Error while scanning meal")
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	err = result.Close()
	if err != nil {
		log.Println("Error while closing result for SELECT Meals query")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// get ingredients
	result, err = m.db.Query("SELECT IngredientID, QuantityInGrams FROM Meal_Ingredients WHERE MealID = $1", mealID)
	if err != nil {
		log.Println("Error while querying MealIngredients table")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var ingredients []struct {
		IngredientID  int64   `json:"ingredient_id"`
		AmountInGrams float64 `json:"amount_in_grams"`
		Name          string  `json:"name"`
	}

	for result.Next() {
		var ingredientID int64
		var amountInGrams float64
		err = result.Scan(&ingredientID, &amountInGrams)
		if err != nil {
			log.Println("Error while scanning ingredient")
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// get ingredient name
		var ingredientName string
		err = m.db.QueryRow("SELECT Name FROM Ingredients WHERE IngredientID = $1", ingredientID).Scan(&ingredientName)
		if err != nil {
			log.Println("Error while querying Ingredients table")
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ingredients = append(ingredients, struct {
			IngredientID  int64   `json:"ingredient_id"`
			AmountInGrams float64 `json:"amount_in_grams"`
			Name          string  `json:"name"`
		}{IngredientID: ingredientID, AmountInGrams: amountInGrams, Name: ingredientName})
	}

	err = result.Close()
	if err != nil {
		log.Println("Error while closing result for SELECT Meal_Ingredients query")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&GetMealResponse{MealID: mealID, Name: mealName, Date: mealDate, Time: mealTime, Ingredients: ingredients})
	return
}

type AddIngredientToMealRequest struct {
	IngredientID  int64   `json:"ingredient_id"`
	AmountInGrams float64 `json:"amount_in_grams"`
}

type AddIngredientToMealResponse struct {
	IngredientID  int64   `json:"ingredient_id"`
	MealID        int64   `json:"meal_id"`
	AmountInGrams float64 `json:"amount_in_grams"`
}

// PUT /api/meals/{id}/ingredients
func (m *MealHandler) AddIngredientToMealHandle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	mealID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		log.Println("Error while parsing mealID")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var addIngredientRequest *AddIngredientToMealRequest
	err = json.NewDecoder(r.Body).Decode(&addIngredientRequest)

	_, err = m.db.ExecContext(r.Context(), "INSERT INTO Meal_Ingredients (MealID, IngredientID, QuantityInGrams) VALUES ($1, $2, $3)", mealID, addIngredientRequest.IngredientID, addIngredientRequest.AmountInGrams)
	if err != nil {
		log.Println("Error while inserting into MealIngredients table")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&AddIngredientToMealResponse{
		MealID:        mealID,
		IngredientID:  addIngredientRequest.IngredientID,
		AmountInGrams: addIngredientRequest.AmountInGrams,
	})
	return

}

func (m *MealHandler) RemoveIngredientFromMealHandle(w http.ResponseWriter, r *http.Request) {
}

func (m *MealHandler) UpdateIngredientInMealHandle(w http.ResponseWriter, r *http.Request) {
}

func (m *MealHandler) DeleteMealHandle(w http.ResponseWriter, r *http.Request) {
}
