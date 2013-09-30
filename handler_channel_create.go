package main

import "net/http"

func ChannelCreateHandler(w http.ResponseWriter, r *http.Request) {
	var data = map[string]interface{}{}

	if r.Method == "POST" {
		newChannel := Channel{}
		newChannel.Title = r.FormValue("title")
		newChannel.Token = r.FormValue("token")
		newChannel.Identity = r.FormValue("identity")
		newChannel.Description = r.FormValue("desc")
		newChannel.Init()
		newChannel.Create()

		data["Created"] = true
	}

	t := templates.Lookup("channel_create.html")
	if t != nil {
		err := t.Execute(w, data)
		if err != nil {
			panic(err)
		}
	}
}
