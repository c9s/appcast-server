package main

import (
	"github.com/c9s/appcast"
	"log"
	"net/http"
	"regexp"
)

var routeAppcastRegExp = regexp.MustCompile("/appcast/([^/]+)/([^/]+)")

func AppcastXmlHandler(w http.ResponseWriter, r *http.Request) {
	var submatches = routeAppcastRegExp.FindStringSubmatch(r.URL.Path)
	if len(submatches) != 3 {
		ForbiddenHandler(w, r)
		return
	}
	var channelIdentity = submatches[1]
	var channelToken = submatches[2]
	if channel := FindChannelByIdentity(channelIdentity, channelToken); channel != nil {
		w.Header().Set("Content-Type", "text/xml; charset=UTF-8")

		rows, err := QueryReleasesByChannel(channelIdentity)
		if err != nil {
			log.Fatal("Query failed:", err)
		}
		defer rows.Close()

		appcastRss := appcast.New()
		appcastRss.Channel.Title = channel.Title
		appcastRss.Channel.Description = channel.Description
		appcastRss.Channel.Link = "http://" + r.Host + "/appcast/" + channelIdentity + "/" + channelToken
		// appcastRss.Channel.Language = channel.Language

		for rows.Next() {
			if item, err := ScanRowToAppcastItem(rows, channelIdentity, channelToken); err == nil {
				appcastRss.Channel.AddItem(item)
			}
		}
		appcastRss.WriteTo(w)
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Channel not found"))
	}
}
