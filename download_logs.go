package main

import "github.com/c9s/gatsby"

type DownloadLog struct {
	Id         int64  `field:"id"`
	RemoteAddr string `field:"remote_addr"`
	RequestURI string `field:"request_uri"`
	Referer    string `field:"referer"`
	UserAgent  string `field:"user_agent"`
	ReleaseId  int64  `field:"release_id"`
	gatsby.BaseRecord
}

func (self *DownloadLog) Init() {
	self.BaseRecord.SetTarget(self)
}
