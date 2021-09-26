package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type Cooking_details_type struct {
	Food_id int `json:"food_id"`
	Cook_id int `json:"cook_id"`
}

type Order struct {
	Order_id     int   `json:"order_id"`
	Table_id     int   `json:"table_id"`
	Waiter_id    int   `json:"waiter_id"`
	Items        []int `json:"items"`
	Priority     int   `json:"priority"`
	Max_wait     int   `json:"max_wait"`
	Pick_up_time int   `json:"pick_up_time"`
}

type Kitchen_response struct {
	Order_id        int                    `json:"order_id"`
	Table_id        int                    `json:"table_id"`
	Waiter_id       int                    `json:"waiter_id"`
	Items           []int                  `json:"items"`
	Priority        int                    `json:"priority"`
	Max_wait        int                    `json:"max_wait"`
	Pick_up_time    int                    `json:"pick_up_time"`
	Cooking_time    int                    `json:"cooking_time"`
	Cooking_details []Cooking_details_type `json:"cooking_details"`
}

func call_hall(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint Hit: Hall")
	fmt.Fprintf(w, "Welcome to the Hall!")
}

func post_dishes(w http.ResponseWriter, r *http.Request) {
	var prepared Kitchen_response
	err := json.NewDecoder(r.Body).Decode(&prepared)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Print(time.Now().Clock())
	fmt.Printf(": Dishes received. Order id: %d\n", prepared.Order_id)
}

func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/", call_hall).Methods("GET")
	myRouter.HandleFunc("/distribution", post_dishes).Methods("POST")
	log.Fatal(http.ListenAndServe(":8002", myRouter))
}

func create_orders() {
	i := 1
	max := 10
	for i <= max {
		// wait for 3-10 seconds betwwen placing orders
		preparation_time := rand.Intn(5)
		time.Sleep(time.Duration(preparation_time) * time.Second)
		build_order(i)
		i += 1
	}
}

func build_order(order_identifier int) {
	items := []int{3, 4, 4, 2}
	the_order := Order{
		order_identifier, 1, 1, items, 3, 45, int(time.Now().Unix()),
	}
	json_data, err_marshall := json.Marshal(the_order)
	if err_marshall != nil {
		log.Fatal(err_marshall)
	}

	resp, err := http.Post("http://localhost:8001/order", "application/json",
		bytes.NewBuffer(json_data))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(time.Now().Clock())
	fmt.Printf(": Order sent to kitchen. Order id: %d. Status: %d\n", order_identifier, resp.StatusCode)
}

func main() {
	rand.Seed(5431)
	go create_orders()
	handleRequests()
}
