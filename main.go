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
	"sync"
)



// This WaitGroup is used to wait for 
// all the goroutines launched to finish. 
var w sync.WaitGroup

// This mutex will synchronize access to the tables.
var m sync.Mutex



// Data types

type Table struct {
	Id             int
	State          string // free, WO (waiting to order), WS (waiting to be served)
	My_order_id    int
	Receive_dishes func(*Table, Kitchen_response) bool
}

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

type Table_Order struct {
	Table_id  int
	Order_id  int
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



// Global vars

var tables = []Table{}
var order_id = 1
var orders_done = []Kitchen_response{}

//NEW_CLIENT_TIME = 80
//WAITER_RESTING_TIME = 1 , waiter is resting (to not spend cpu cycles)
//MAKE_ORDER_TIME = 30



// Builds table instances stored globally
func build_tables(n int){

	for i := 0; i < n; i++ {
		table := Table{
			i, 
			"free",
			0,
			check_dish,
		}
		tables = append(tables, table)
    }
}

func check_dish(table *Table, dish Kitchen_response) bool {
	if table.My_order_id == dish.Order_id {
		return true
	}else{
		return false
	}
}


// Make 1 thread to invite clients to tables
func table_occupation(){
	w.Add(1)
	go occupy_table()
}

// Table occupation logic
func occupy_table(){
	for{
		m.Lock()
		
		i := rand.Intn(5)
		if tables[i].State == "free"{
			tables[i].State = "WO"
			fmt.Println("Tables:", tables)
			fmt.Println("New client to table", i, "\n")	
			
			// new client each 8 time units
			time.Sleep(80*time.Millisecond)	
		}
			
		m.Unlock()	
	  
	}
	w.Done()
}

// helping function to take out the dish from kitchen distribution
func RemoveDish(s []Kitchen_response, index int) []Kitchen_response {
	return append(s[:index], s[index+1:]...)
}

// helping function to remove dishes from waiter's memory
func RemoveCoordinate(s []Table_Order, index int) []Table_Order {
	return append(s[:index], s[index+1:]...)
}

// Make waiter threads
func waiter(i int) { 
	coordinates := []Table_Order{}
    for{
    	new_order_id := 0
    	approached_table_id := -1
    	found_kitchen_response := false
    	
		m.Lock() 
		
		// offer dishes to clients
		for j:= 0; j < len(orders_done); j++ {
			for k:= 0; k < len(coordinates); k++ {
				if coordinates[k].Order_id == orders_done[j].Order_id {
					found_kitchen_response = true
				
					this_table := tables[coordinates[k].Table_id]
					accepted := this_table.Receive_dishes(
						&this_table, 
						orders_done[j],
					)
					
					if accepted {
						fmt.Println(
							"Client accepted dishes with order:", 
							orders_done[j].Order_id,
							"\n")
						
						orders_done = RemoveDish(orders_done, j)
						tables[coordinates[k].Table_id].State = "free"
						tables[coordinates[k].Table_id].My_order_id = 0
						
						coordinates = RemoveCoordinate(coordinates, k)
					}else{
						fmt.Println(
							"Client refused dishes with order:", 
							orders_done[j].Order_id,
							"\n")
					}
										
					if k < len(coordinates)-1 {
						break
					}
				}
			}
			if found_kitchen_response == true && j < len(orders_done)-1 {
				break
			}
		}
		
		// take orders from clients
		for j:= 0; j < len(tables); j++ {
			if tables[j].State == "WO"{
				approached_table_id = j
				
				tables[j].State = "WS"
				
				fmt.Println("Tables:", tables)
				fmt.Println("Waiter:", i, "| Got table:", tables[j].Id, "\n")
				
				new_order_id = order_id
				tables[j].My_order_id = new_order_id
				
				order_id += 1
				break
			}
		}
				
		m.Unlock()
		
		// if waiter took nan order
		if new_order_id > 0 {
			new_order := build_order(new_order_id)
			
			new_coordinate := Table_Order{approached_table_id, new_order.Order_id}
			coordinates = append(coordinates, new_coordinate)
			fmt.Println("Waiter",i,"| Got order:", new_order,"| Remembered orders (table, order_id):", coordinates, "\n")
			send_order(new_order)
			
			new_order_id = 0
		}
		
		time.Sleep(1*time.Millisecond)
	}
	w.Done()
}

// Orders generator
func build_order(order_identifier int) Order{
	items := []int{3, 4, 4, 2}
	the_order := Order{
		order_identifier, 1, 1, items, 3, 45, int(time.Now().Unix()),
	}
	
	// client is making order, 3 time units
	time.Sleep(30*time.Millisecond)
	
	return the_order
}

// Order sending logic
func send_order(the_order Order){
	json_data, err_marshall := json.Marshal(the_order)
	if err_marshall != nil {
		log.Fatal(err_marshall)
	}

	resp, err := http.Post("http://localhost:8001/order", "application/json",
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
	var prepared Kitchen_response
	err := json.NewDecoder(r.Body).Decode(&prepared)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	m.Lock() 

	orders_done = append(orders_done, prepared)

	m.Unlock() 

	fmt.Printf("Dishes received. Order id: %d\n\n", prepared.Order_id)
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
    
    // Make 5 tables
    build_tables(5)
    
    // Initialize the mechanism of table occupation.
    w.Add(1)
    go table_occupation()
  
    // Initialize 3 waiters.
    for i := 0; i < 3; i++ {
        w.Add(1)        
        go waiter(i)
    }
    
    //
    handleRequests()
    
    // Block until the WaitGroup counter
    // goes back to 0; all the workers 
    // notified they’re done.
    w.Wait()
	
}

