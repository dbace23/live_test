package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"
)

// models
type Shipment struct {
	ID             int       `json:"id"`
	Nama           string    `json:"nama"`
	Pengirim       string    `json:"pengirim"`
	NamaPenerima   string    `json:"namaPenerima"`
	AlamatPenerima string    `json:"alamatPenerima"`
	NamaItem       string    `json:"namaItem"`
	BeratItem      int       `json:"beratItem"`
	TimeStamp      time.Time `json:"datetime"`
	CreatedAt      time.Time `json:"createdAt"`
}

// repo: fallback
var (
	shipments = []Shipment{
		{ID: 1,
			Nama:           "Halim",
			Pengirim:       "Judy",
			NamaPenerima:   "Jasonn",
			AlamatPenerima: "Jalan agust 11,jakarta", NamaItem: "baju", BeratItem: 90,
			TimeStamp: time.Now(),
			CreatedAt: time.Now()},
	}
	nextID   = 2
	storeMux sync.RWMutex
)

// db handler
var db *sql.DB

// -- DB QUERIES
func dbListShipment() ([]Shipment, error) {
	rows, err := db.Query(`SELECT ID,Nama,Pengirim,NamaPenerima,AlamatPenerima,NamaItem,BeratItem,Timestamp,CreatedAt from shipments order by id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Shipment
	for rows.Next() {
		var s Shipment
		if err := rows.Scan(&s.ID, &s.Nama, &s.Pengirim, &s.NamaPenerima, &s.AlamatPenerima, &s.NamaItem, &s.BeratItem, &s.TimeStamp, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func dbGetShipment(id int) (*Shipment, error) {
	var s Shipment
	err := db.QueryRow(`SELECT ID,Nama,Pengirim,NamaPenerima,AlamatPenerima,NamaItem,BeratItem,Timestamp,CreatedAt from shipments WHERE id=$1`, id).
		Scan(&s.ID, &s.Nama, &s.Pengirim, &s.NamaPenerima, &s.AlamatPenerima, &s.NamaItem, &s.BeratItem, &s.TimeStamp, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}
func dbCreateShipment(s *Shipment) error {
	// If client omits TimeStamp, use NOW() in SQL
	return db.QueryRow(
		`INSERT INTO shipments
		 (nama, pengirim, nama_penerima, alamat_penerima, nama_item, berat_item, "timestamp")
		 VALUES ($1,$2,$3,$4,$5,$6, COALESCE($7, NOW()))
		 RETURNING id, created_at, "timestamp"`,
		s.Nama, s.Pengirim, s.NamaPenerima, s.AlamatPenerima, s.NamaItem, s.BeratItem, s.TimeStamp,
	).Scan(&s.ID, &s.CreatedAt, &s.TimeStamp)
}

func dbUpdateShipment(id int, s *Shipment) (*Shipment, error) {
	_, err := db.Exec(`
		UPDATE shipments SET
			nama = $1,
			pengirim = $2,
			nama_penerima = $3,
			alamat_penerima = $4,
			nama_item = $5,
			berat_item = $6,
			"timestamp" = COALESCE($7, "timestamp")
		WHERE id = $8
	`, s.Nama, s.Pengirim, s.NamaPenerima, s.AlamatPenerima, s.NamaItem, s.BeratItem,
		func() *time.Time {
			if s.TimeStamp.IsZero() {
				return nil
			}
			return &s.TimeStamp
		}(),
		id,
	)
	if err != nil {
		return nil, err
	}

	return dbGetShipment(id)
}
func dbDeleteShipment(id int) (bool, error) {
	res, err := db.Exec(`DELETE FROM shipments WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

// helpers
func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("writeJson encode error: %v", err)
	}
}
func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})

}
func parseID(param string) (int, error) {
	id, err := strconv.Atoi(param)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func validateShipment(s *Shipment) error {
	if strings.TrimSpace(s.Nama) == "" {
		return errors.New("nama is required")
	}
	if strings.TrimSpace(s.Pengirim) == "" {
		return errors.New("pengirim is required")
	}
	if strings.TrimSpace(s.NamaPenerima) == "" {
		return errors.New("namaPenerima is required")
	}
	if strings.TrimSpace(s.AlamatPenerima) == "" {
		return errors.New("alamatPenerima is required")
	}
	if strings.TrimSpace(s.NamaItem) == "" {
		return errors.New("namaItem is required")
	}
	if s.BeratItem < 0 {
		return errors.New("beratItem cannot be negative")
	}
	return nil
}

func findIndexByID(id int) int {
	for i, v := range shipments {
		if v.ID == id {
			return i
		}
	}
	return -1
}

// http handlers
func listShipments(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if db != nil {
		items, err := dbListShipment()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, items)
		return
	}
	storeMux.RLock()
	defer storeMux.RUnlock()
	writeJSON(w, http.StatusOK, shipments)

}

func getShipment(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := parseID(ps.ByName("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if db != nil {
		p, err := dbGetShipment(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if p == nil {
			writeError(w, http.StatusNotFound, "shipment not found")
			return
		}
		writeJSON(w, http.StatusOK, p)
		return
	}
	storeMux.RLock()
	defer storeMux.RUnlock()

	idx := findIndexByID(id)
	if idx := findIndexByID(id); idx == -1 {
		writeError(w, http.StatusNotFound, "shipment not found")
		return
	}
	writeJSON(w, http.StatusOK, shipments[idx])
}

func createShipment(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var in Shipment
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if err := validateShipment(&in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if db != nil {
		if err := dbCreateShipment(&in); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, in)
		return
	}

	storeMux.Lock()
	defer storeMux.Unlock()
	in.ID = nextID
	nextID++
	if in.TimeStamp.IsZero() {
		in.TimeStamp = time.Now()
	}
	in.CreatedAt = time.Now()
	shipments = append(shipments, in)
	writeJSON(w, http.StatusCreated, in)
}

func updateShipment(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := parseID(ps.ByName("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var in Shipment
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if err := validateShipment(&in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if db != nil {
		p, err := dbUpdateShipment(id, &in)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if p == nil {
			writeError(w, http.StatusNotFound, "shipment not found")
			return
		}
		writeJSON(w, http.StatusOK, p)
		return
	}
	storeMux.Lock()
	defer storeMux.Unlock()
	idx := findIndexByID(id)
	if idx == -1 {
		writeError(w, http.StatusNotFound, "shipment not found")
		return
	}
	in.ID = id
	shipments[idx] = in
	writeJSON(w, http.StatusOK, in)
}

func deleteShipment(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := parseID(ps.ByName("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if db != nil {
		ok, err := dbDeleteShipment(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(w, http.StatusNotFound, "shipment not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted_id": id})
		return
	}
	storeMux.Lock()
	defer storeMux.Unlock()
	idx := findIndexByID(id)
	if idx == -1 {
		writeError(w, http.StatusNotFound, "shipment not found")
		return
	}
	shipments = append(shipments[:idx], shipments[idx+1:]...)
	writeJSON(w, http.StatusOK, map[string]any{"deleted_id": id})
}

func main() {
	var err error
	dsn := os.Getenv("DATABASE_URL")
	if dsn != "" {
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			log.Fatalf("open db: %v", err)
		}
		if err := db.Ping(); err != nil {
			log.Fatalf("ping db: %v", err)
		}
		log.Println("DB connected")
	} else {
		log.Println("DATABASE_URL not set; using in-memory fallback")
	}

	router := httprouter.New()
	router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	router.GET("/shipment", listShipments)
	router.GET("/shipment/:id", getShipment)
	router.POST("/shipments", createShipment)
	router.PUT("/shipment/:id", updateShipment)
	router.DELETE("/shipment/:id", deleteShipment)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("shipment server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}
