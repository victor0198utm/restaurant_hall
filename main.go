package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/victor0198utm/restaurant_hall/app_data"
	"github.com/victor0198utm/restaurant_hall/models"
)

var w sync.WaitGroup

var m sync.Mutex

// Global vars

var tables = []models.Table{}
var order_id = 1
var orders_done = []models.Kitchen_response{}
var reputation = []int{}

// Builds table instances stored globally
func build_tables(n int) {

	for i := 0; i < n; i++ {
		table := models.Table{
			i,
			"free",
			0,
			check_dish,
		}
		tables = append(tables, table)
	}
}

func check_dish(table *models.Table, dish models.Kitchen_response) bool {
	if table.My_order_id == dish.Order_id {
		return true
	} else {
		return false
	}
}

// Make threads to occupy tables
// after the table was free for 1.5-3 min
func table_occupation(n_tables int) {
	// make an occupation thread for each table
	for i := 0; i < n_tables; i++ {
		w.Add(1)
		go occupy(n_tables)
	}
}

func occupy(n_tables int) {
	for {
		// wait 1.5-2.5 min before adding client
		time.Sleep(time.Duration((rand.Intn(1000) + 1500)) * time.Millisecond)

		m.Lock()

		// add to random table
		i := rand.Intn(n_tables)
		if tables[i].State == "free" {
			tables[i].State = "WO"
			fmt.Println("Tables:", tables, "\nNew client to table", i, "\n")
		}

		m.Unlock()
	}
	w.Done()
}

// helping function to take out the dish from kitchen distribution
func RemoveDish(s []models.Kitchen_response, index int) []models.Kitchen_response {
	return append(s[:index], s[index+1:]...)
}

// helping function to remove dishes from waiter's memory
func RemoveCoordinate(s []models.Table_Order, index int) []models.Table_Order {
	return append(s[:index], s[index+1:]...)
}

// Make waiter threads
func waiter(waiter_id int) {
	// coordinates := []models.Table_Order{}
	for {
		new_order_id := 0
		approached_table_id := -1
		found_kitchen_response := false

		m.Lock()

		// serve (offer dishes) to clients
		for j := 0; j < len(orders_done); j++ {
			if orders_done[j].Waiter_id == waiter_id {
				tableId := orders_done[j].Table_id
				this_table := tables[tableId]
				accepted := this_table.Receive_dishes(
					&this_table,
					orders_done[j],
				)

				if accepted {
					fmt.Println(
						"Client accepted dishes with order:",
						orders_done[j].Order_id,
						"| Cooking time:",
						orders_done[j].Cooking_time,
						"\n")

					orders_done = RemoveDish(orders_done, j)
					tables[tableId].State = "free"
					tables[tableId].My_order_id = 0

				} else {
					fmt.Println(
						"Client refused dishes with order:",
						orders_done[j].Order_id,
						"\n")
				}
			}
			// for k := 0; k < len(coordinates); k++ {
			// 	if coordinates[k].Order_id == orders_done[j].Order_id {
			// 		found_kitchen_response = true

			// 		this_table := tables[coordinates[k].Table_id]
			// 		accepted := this_table.Receive_dishes(
			// 			&this_table,
			// 			orders_done[j],
			// 		)

			// 		if accepted {
			// 			fmt.Println(
			// 				"Client accepted dishes with order:",
			// 				orders_done[j].Order_id,
			// 				"\n")

			// 			orders_done = RemoveDish(orders_done, j)
			// 			tables[coordinates[k].Table_id].State = "free"
			// 			tables[coordinates[k].Table_id].My_order_id = 0

			// 			coordinates = RemoveCoordinate(coordinates, k)
			// 		} else {
			// 			fmt.Println(
			// 				"Client refused dishes with order:",
			// 				orders_done[j].Order_id,
			// 				"\n")
			// 		}

			// 		if k < len(coordinates)-1 {
			// 			break
			// 		}
			// 	}
			// }
			if found_kitchen_response == true && j < len(orders_done)-1 {
				break
			}
		}

		// take orders from clients
		for j := 0; j < len(tables); j++ {
			if tables[j].State == "WO" {
				approached_table_id = j

				tables[j].State = "WS"

				fmt.Println("Tables:", tables)
				fmt.Println("Waiter:", waiter_id, "| Got table:", tables[j].Id, "\n")

				new_order_id = order_id
				tables[j].My_order_id = new_order_id

				order_id += 1
				break
			}
		}

		m.Unlock()

		// if waiter took an order
		if new_order_id > 0 {
			new_order := build_order(new_order_id, approached_table_id, waiter_id)

			fmt.Println("Waiter", waiter_id, "| Got order:", new_order, "\n")
			send_order(new_order)
		}

		// waiter is resting (to not spend cpu cycles)
		time.Sleep(1 * time.Millisecond)
	}
	w.Done()
}

// Orders generator
func build_order(order_identifier int, table_id int, waiter_id int) models.Order {

	// client is making order, 2 min
	time.Sleep(2000 * time.Millisecond)

	n_items := rand.Intn(4) + 1
	items := []int{}
	for i := 0; i < n_items; i++ {
		items = append(items, app_data.GetDish(rand.Intn(9)).Dish_id)
	}

	max_time := 0
	for _, dish_id := range items {

		//fmt.Println(i, the_dish)
		prepation_time := app_data.GetDish(dish_id - 1).Preparation_time
		if max_time < prepation_time {
			max_time = prepation_time
		}
	}

	fmt.Println(items, max_time)

	order_priority := rand.Intn(4) + 1
	the_order := models.Order{
		order_identifier,
		table_id,
		waiter_id,
		items,
		order_priority,
		int(float32(max_time) * 1.3),
		int(time.Now().Unix()),
	}

	return the_order
}

// Order sending logic
func send_order(the_order models.Order) {
	json_data, err_marshall := json.Marshal(the_order)
	if err_marshall != nil {
		log.Fatal(err_marshall)
	}

	resp, err := http.Post("http://kitchen:8001/order", "application/json",
		bytes.NewBuffer(json_data))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Order sent to kitchen. Order id: %d. Status: %d\n\n", the_order.Order_id, resp.StatusCode)
}

// Hall endpoint: "/"
func call_hall(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint Hit: Hall")
	fmt.Fprintf(w, "Welcome to the Hall!")
}

// Hall endpoint: "/distribution"
func post_dishes(w http.ResponseWriter, r *http.Request) {
	var prepared models.Kitchen_response
	err := json.NewDecoder(r.Body).Decode(&prepared)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stars := 0
	if prepared.Cooking_time < prepared.Max_wait {
		stars = 5
	} else if float64(prepared.Cooking_time) < float64(prepared.Max_wait)*1.1 {
		stars = 4
	} else if float64(prepared.Cooking_time) < float64(prepared.Max_wait)*1.2 {
		stars = 3
	} else if float64(prepared.Cooking_time) < float64(prepared.Max_wait)*1.3 {
		stars = 2
	} else if float64(prepared.Cooking_time) < float64(prepared.Max_wait)*1.4 {
		stars = 1
	}

	m.Lock()

	orders_done = append(orders_done, prepared)
	reputation = append(reputation, stars)
	m.Unlock()

	fmt.Printf("Dishes received. Order id: %d\n\n", prepared.Order_id)
	fmt.Println("Dishes LIST:", orders_done)
}

func display_reputation() {
	for {
		time.Sleep(1 * time.Second)
		n := len(reputation)
		sum := 0
		for i := 0; i < n; i++ {
			sum += reputation[i]
		}
		rating := float64(sum) / float64(n)
		fmt.Println("Reputation:",
			rating,
			"|",
			reputation,
			"\n",
		)
	}
}

// Requests hadler
func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/", call_hall).Methods("GET")
	myRouter.HandleFunc("/distribution", post_dishes).Methods("POST")
	log.Fatal(http.ListenAndServe(":8002", myRouter))
}

func main() {

	// Start the hall threads

	n_tables := 7
	// Make tables
	build_tables(n_tables)

	// Initialize the mechanism of table occupation.
	table_occupation(n_tables)

	// Initialize 2 waiters.
	for i := 0; i < 2; i++ {
		w.Add(1)
		go waiter(i)
	}

	w.Add(1)
	go display_reputation()

	handleRequests()

	w.Wait()
}
