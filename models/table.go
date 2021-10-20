package models

type Table struct {
	Id             int
	State          string // free, WO (waiting to order), WS (waiting to be served)
	My_order_id    int
	Receive_dishes func(*Table, Kitchen_response) bool
}
