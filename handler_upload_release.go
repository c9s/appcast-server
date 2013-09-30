package main

import (
	"github.com/c9s/jsondata"
	"net/http"
)

func UploadReleaseHandler(w http.ResponseWriter, r *http.Request) {
	submatches := routeReleaseCreateRegExp.FindStringSubmatch(r.URL.Path)
	if len(submatches) != 3 {
		ForbiddenHandler(w, r)
		return
	}
	if r.Method != "POST" {
		ForbiddenHandler(w, r)
		return
	}

	channelIdentity := submatches[1]
	channelToken := submatches[2]

	if channel := FindChannelByIdentity(channelIdentity, channelToken); channel != nil {
		if _, err := CreateNewReleaseFromRequest(r, channelIdentity); err != nil {
			var msg = jsondata.Map{"error": err}
			msg.WriteTo(w)
		} else {
			var msg = jsondata.Map{"success": true}
			msg.WriteTo(w)
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Channel not found"))
	}
}
