package main

import "github.com/c9s/gatsby"

type DownloadLog struct {
	Id         int64  `field:"id"`
	RemoteAddr string `field:"remoteAddr"`
	UserAgent  string `field:"useragent"`
	ReleaseId  int64  `field:"releaseId"`
	gatsby.BaseRecord
}
