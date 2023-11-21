// main.go

package main

import (
	"assignment2/handlers"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	// Initialize database
	db := initDB()
	defer db.Close()

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080" // Default port if not specified
	}

	r := mux.NewRouter()
	r.HandleFunc("/health", healthHandler).Methods("GET")

	mealHandler := handlers.NewMealHandler(db)
	r.HandleFunc("/api/meals", mealHandler.CreateMealHandle).Methods("POST")
	r.HandleFunc("/api/meals/{id}", mealHandler.GetMealHandle).Methods("GET")
	r.HandleFunc("/api/meals/{id}/ingredients", mealHandler.AddIngredientToMealHandle).Methods("PUT")
	r.HandleFunc("/api/meals/{id}/ingredients/{ingredient_id}", mealHandler.RemoveIngredientFromMealHandle).Methods("DELETE")
	r.HandleFunc("/api/meals/{id}/ingredients", mealHandler.UpdateIngredientInMealHandle).Methods("PUT")
	r.HandleFunc("/api/meals/{id}", mealHandler.DeleteMealHandle).Methods("DELETE")

	ingredientHandler := handlers.NewIngredientHandler(db)
	r.HandleFunc("/api/ingredients", ingredientHandler.CreateIngredientHandle).Methods("POST")
	r.HandleFunc("/api/ingredients/{id}", ingredientHandler.GetIngredientHandle).Methods("GET")
	r.HandleFunc("/api/ingredients/{id}", ingredientHandler.UpdateIngredientHandle).Methods("PUT")
	r.HandleFunc("/api/ingredients/{id}", ingredientHandler.DeleteIngredientHandle).Methods("DELETE")

	srv := &http.Server{
		Handler:      r,
		Addr:         ":" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	go func() {
		log.Println("Starting the HTTP server on port 8080")
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("shutting down")
	os.Exit(0)
}
