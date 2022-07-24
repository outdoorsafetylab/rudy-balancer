package controller

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

func writeJSON(w http.ResponseWriter, r *http.Request, body interface{}) {
	if boolVar(r, "pretty", false) {
		data, err := json.Marshal(body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		var dst bytes.Buffer
		err = json.Indent(&dst, data, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = io.Copy(w, &dst)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	} else {
		enc := json.NewEncoder(w)
		err := enc.Encode(body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
}
