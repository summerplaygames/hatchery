package hatchery

import (
	"encoding/json"
	"net/http"
)

func writeJSONResponse(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-type", "application/json")
	json.NewEncoder(w).Encode(v)
}
