package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type IngredientHandler struct {
	db *sql.DB
}

func NewIngredientHandler(db *sql.DB) *IngredientHandler {
	return &IngredientHandler{db: db}
}

func convertToPerHundredGrams(amount float64, servingSizeInGrams float64) float64 {
	return (amount / servingSizeInGrams) * 100
}

type CreateIngredientRequest struct {
	Name               string  `json:"name" validate:"required"`
	ServingSizeInGrams float64 `json:"serving_size_in_grams"`
	Nutrients          []struct {
		Name   string  `json:"name"`
		Amount float64 `json:"amount"`
	} `json:"nutrients" validate:"required"`
}

type CreateIngredientResponse struct {
	IngredientID int64 `json:"ingredient_id" validate:"required"`
}

func (i *IngredientHandler) CreateIngredientHandle(w http.ResponseWriter, r *http.Request) {
	var ingredientRequest *CreateIngredientRequest
	err := json.NewDecoder(r.Body).Decode(&ingredientRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// check if ingredient already exists
	// if exists, return error
	// if not exists, create ingredient
	result, err := i.db.Query("SELECT * FROM Ingredients WHERE Name = $1", ingredientRequest.Name)
	if err != nil {
		log.Println("Error while querying Ingredients table")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.Next() {
		log.Println("Ingredient already exists")
		http.Error(w, "Ingredient already exists", http.StatusBadRequest)
		return
	}
	err = result.Close()
	if err != nil {
		log.Println("Error while closing result of SELECT * FROM Ingredients")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tx, err := i.db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Println("Error while starting transaction")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// create ingredient
	res, err := tx.QueryContext(r.Context(), "INSERT INTO Ingredients (Name) VALUES ($1) RETURNING IngredientID", ingredientRequest.Name)
	if err != nil {
		log.Println("Error while inserting ingredient into Ingredients table")
		log.Println(err)
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var ingredientID int64
	if res.Next() {
		err = res.Scan(&ingredientID)
		if err != nil {
			log.Println("Error while scanning ingredientID")
			log.Println(err)
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	err = res.Close()
	if err != nil {
		log.Println("Error while closing result of INSERT INTO Ingredients")
		log.Println(err)
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Created ingredient with ID %d\n", ingredientID)

	// create nutrients
	for _, nutrient := range ingredientRequest.Nutrients {
		// check if nutrient already exists
		// if exists, use existing nutrient
		// if not exists, create nutrient
		log.Printf("Checking if nutrient %s already exists\n", nutrient.Name)

		result, err = tx.QueryContext(r.Context(), "SELECT NutrientID FROM Nutrients WHERE Name = $1", nutrient.Name)
		if err != nil {
			log.Println("Error while querying Nutrients table")
			log.Println(err)
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var nutrientID int64
		if result.Next() {
			err = result.Scan(&nutrientID)
			if err != nil {
				log.Println("Error while scanning nutrientID")
				log.Println(err)
				tx.Rollback()
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			err = result.Close()
			if err != nil {
				log.Println("Error while closing result of SELECT * FROM Nutrients")
				log.Println(err)
				tx.Rollback()
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			resultInsertNutrient, err := tx.QueryContext(r.Context(), "INSERT INTO Nutrients (Name) VALUES ($1) RETURNING NutrientID", nutrient.Name)
			if err != nil {
				log.Println("Error while inserting nutrient into Nutrients table")
				log.Println(err)
				tx.Rollback()
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if resultInsertNutrient.Next() {
				err = resultInsertNutrient.Scan(&nutrientID)
				if err != nil {
					log.Println("Error while scanning nutrientID")
					log.Println(err)
					tx.Rollback()
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			err = resultInsertNutrient.Close()
			if err != nil {
				log.Println("Error while closing result of INSERT INTO Nutrients")
				log.Println(err)
				tx.Rollback()
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// create nutrient value
		_, err = tx.ExecContext(r.Context(), "INSERT INTO Nutrient_Values (IngredientID, NutrientID, AmountPer100g) VALUES ($1, $2, $3)", ingredientID, nutrientID, convertToPerHundredGrams(nutrient.Amount, ingredientRequest.ServingSizeInGrams))
		if err != nil {
			log.Println("Error while inserting nutrient value into Nutrient_Values table")
			log.Println(err)
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	response := CreateIngredientResponse{IngredientID: ingredientID}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type Nutrient struct {
	Name          string  `json:"name"`
	AmountPer100g float64 `json:"amount_per_100g"`
}

type Ingredient struct {
	IngredientID int        `json:"ingredient_id"`
	Name         string     `json:"name"`
	Nutrients    []Nutrient `json:"nutrients"`
}

// '/api/ingredients/{id}'
func (i *IngredientHandler) GetIngredientHandle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	result, err := i.db.Query("SELECT IngredientID, Name FROM Ingredients WHERE IngredientID = $1", vars["id"])
	if err != nil {
		log.Println("Error while querying Ingredients table")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var ingredientID int
	var ingredientName string
	if result.Next() {
		err = result.Scan(&ingredientID, &ingredientName)
		if err != nil {
			log.Println("Error while scanning ingredient")
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Ingredient not found", http.StatusNotFound)
		return
	}
	err = result.Close()
	if err != nil {
		log.Println("Error while closing result of SELECT * FROM Ingredients")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// get nutrient name from Nutrients table and AmountPer100g from Nutrient_Values table
	result, err = i.db.Query("SELECT Nutrients.Name, Nutrient_Values.AmountPer100g FROM Nutrients INNER JOIN Nutrient_Values ON Nutrients.NutrientID = Nutrient_Values.NutrientID WHERE Nutrient_Values.IngredientID = $1", ingredientID)
	if err != nil {
		log.Println("Error while querying Nutrients and Nutrient_Values tables")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var nutrients []Nutrient
	for result.Next() {
		var nutrientName string
		var amountPer100g float64
		err = result.Scan(&nutrientName, &amountPer100g)
		if err != nil {
			log.Println("Error while scanning nutrient")
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		nutrients = append(nutrients, Nutrient{Name: nutrientName, AmountPer100g: amountPer100g})
	}

	err = result.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	response := Ingredient{IngredientID: ingredientID, Name: ingredientName, Nutrients: nutrients}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// PUT /api/ingredients/{id}
func (i *IngredientHandler) UpdateIngredientHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
}

type DeleteIngredientResponse struct {
	IngredientID int64  `json:"ingredient_id"`
	Name         string `json:"name"`
}

// DELETE /api/ingredients/{id}
func (i *IngredientHandler) DeleteIngredientHandle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	rows, err := i.db.QueryContext(r.Context(), "DELETE FROM Ingredients WHERE IngredientID = $1 RETURNING IngredientID, Name", vars["id"])
	defer rows.Close()
	if err != nil {
		log.Println("Error while deleting ingredient from Ingredients table")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var ingredientID int64
	var ingredientName string
	if rows.Next() {
		err = rows.Scan(&ingredientID, &ingredientName)
		if err != nil {
			log.Println("Error while scanning ingredient")
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Ingredient not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	response := DeleteIngredientResponse{IngredientID: ingredientID, Name: ingredientName}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Println("Error while encoding response")
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}
