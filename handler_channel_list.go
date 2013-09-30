package main

import (
	"net/http"
)

func ChannelListHandler(w http.ResponseWriter, r *http.Request) {
	t := templates.Lookup("channel_list.html")
	if t != nil {
		err := t.Execute(w, nil)
		if err != nil {
			panic(err)
		}
	}
}
