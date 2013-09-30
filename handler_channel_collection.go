package main

import "net/http"
import "github.com/c9s/gatsby"
import "encoding/json"
import "fmt"

func ChannelCollectionHandler(w http.ResponseWriter, r *http.Request) {
	data, res := gatsby.SelectWith(db, &Channel{}, "")
	_ = res
	channels := data.([]Channel)
	writeJson(w, channels)
}

func writeJson(w http.ResponseWriter, val interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	fmt.Fprint(w, string(b))
	return nil
}
