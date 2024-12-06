package utils

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/MultiX0/solo_leveling_system/db"
)

func WriteJsonResponse(w http.ResponseWriter, statusCode int, v any) error {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	return json.NewEncoder(w).Encode(v)

}

func WriteError(w http.ResponseWriter, statusCode int, err error) error {
	return WriteJsonResponse(w, statusCode, map[string]any{"error": err.Error()})
}

func InsertToDB(table string, data any) ([]byte, error) {
	newData, _, err := db.SupabaseClient.From(table).Insert(data, false, "", "", "exact").Single().Execute()

	if err != nil {
		return nil, err
	}

	return newData, nil
}

func NowDate() string {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.999999Z")
	return now
}
