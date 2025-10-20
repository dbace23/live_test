package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	_  "github.com/lib/pq"
	"github.com/julienschmidt/httprouter"
)

//models
type Shipment struct{
	ID int `json:"id"`
	Nama string `json:"nama"`
	Pengirim string `json:"pengirim"`
	NamaPenerima string `json:"namaPenerima"`
	AlamatPenerima string `json:"alatPenerima"`
	NamaItem string `json:"namaItem"`
	BeratItem int `json:"beratItem"`
	TimeStamp time.Time `json:"datetime"`
	CreatedAt time.Time `json:"createdAt"`
}

// repo: fallback
var (
	shipment=[] Shipment{
		{ID:1,Nama:"Halim",Pengirim:"Judy",NamaPenerima:"Jasonn",AlamatPenerima:"Jalan agust 11,jakarta",NamaItem:"baju",BeratItem:90,TimeStamp: time.Now(), CreatedAt: time.Now()}
	}
	nextID=2
	storeMux Sync.RWMutex

)
// db handler

var db *sql.DB

//-- DB QUERIES
func dbListShipment() ([]Shipment,error){
	rows,err:= db.Query(`SELECT ID,Nama,Pengirim,NamaPenerima,AlamatPenerima,NamaItem,BeratItem,Timestamp,CreatedAt from shipments order by id`)
	if err != nil {
		return nil,err
	}
	defer rows.Close()
	var out []Shipment
	for rows.Next(){
		var s Shipment
		if err:= rows.Scan(&s.ID,&s.Nama,&s.Pengirim,&s.NamaPenerima,&s.AlamatPenerima,&s.NamaItem,&s.BeratItem,&s.Timestamp,&s.CreatedAt); err != nil{
			return nil,err
		}
		out.append(out,s)
	}
	return out,rows.Err()
}

func dbGetShipment(id int) (*Shipment, error) {
	var s Shipment
	err := db.QueryRow(`SELECT ID,Nama,Pengirim,NamaPenerima,AlamatPenerima,NamaItem,BeratItem,Timestamp,CreatedAt from shipments WHERE id=$1`, id).
		Scan(&s.ID,&s.Nama,&s.Pengirim,&s.NamaPenerima,&s.AlamatPenerima,&s.NamaItem,&s.BeratItem,&s.Timestamp,&s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}
//todo list queries
			//create shipment
			//update shipment
			// delete shipment

//helpers
func writeJSON(w http.ResponseWriter,code int, data any){
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(code)
	if err:= json.NewEncoder(w).Encode(data);err !=nil{
		log.Printf("writeJson encode error: %v",err)
	}
}
func writeError(w http.ResponseWriter,code int, msg string){
	writeJSON(w,code,map[string]string{"error":msg})

}
func parseID(param string) (int, error) {
	id, err := strconv.Atoi(param)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func validateItem(it *Product) error {
	if strings.TrimSpace(it.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(it.Description) == "" {
		return errors.New("description is required")
	}
	if it.Price < 0 {
		return errors.New("price cannot be negative")
	}
	return nil
}

func findIndexByID(id int) int {
	for i, v := range products {
		if v.ID == id {
			return i
		}
	}
	return -1
}
//http handlers 
func listShipments(w http.ResponseWriter, r *http.Request, _ httprouter.Params){
	if db!=nil{
		items,err:=dbListShipment()
		if err!=nil{
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w,http.StatusOK,items)
		return
	}
	storeMux.RLock()
	defer storeMux.RUnlock()
	writeJSON(w, http.StatusOK,shipment)

}

func getShipment(w http.ResponseWriter, r* http.Request, ps httprouter.Params){
	id,err:=parseID(ps.ByName("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest,er.Error())
		return
	}
	if db!=nil{
		p,err:=dbGetShipment(id)
		if err:=nil{
			writeError(w,http.StatusInternalServerError,err.Error())
			return
		}
		if p==nil{
			writeError(w,http.StatusNotFound,err.Error())
			return
		}
		writeJSON(w,http.StatusOK,p)
		return
	}
	storeMux.RLock()
	defer storeMux.RUnlock()
	if idx:=findIndexByID(id);idx==-1{
		writeError(w,http.StatusNotFound,"product not found")
		return
	}
	writeJSON(w,http.StatusOK,shipment[idx])
}

func main (){
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	router := httprouter.New()
	router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	router.GET("/shipment",listShipments)
	router.GET("/shipment/:id",getShipment)
	// router.POST("/shipments",createShipment)
	// router.PUT("/shipment/:id",updateShipment)
	// router.DELETE("/shipment/:id",deleteShipment)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("shipment server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}

