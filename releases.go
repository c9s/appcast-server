package main

import (
	"database/sql"
	// "github.com/c9s/appcast"
	"github.com/c9s/gatsby"
	"time"
)

type Release struct {
	Id                 int64      `field:"id"`
	Title              string     `field:"title"`
	Description        string     `field:"desc"`
	ReleaseNotes       string     `field:"release_notes"`
	PubDate            *time.Time `field:"pubdate"`
	Filename           string     `field:"filename"`
	Channel            string     `field:"channel"`
	Length             int64      `field:"length"`
	Mimetype           string     `field:"mimetype"`
	DSASignature       string     `field:"dsa_signature"`
	Version            string     `field:"version"`
	ShortVersionString string     `field:"short_version_string"`
	Token              string     `field:"token"`
	Downloaded         int64      `field:"downloaded"`
	gatsby.BaseRecord
}

func (self *Release) Init() {
	self.BaseRecord.SetTarget(self)
}

func FindReleaseByToken(token string) *Release {
	r := Release{}
	r.Init()
	res := r.LoadByCols(map[string]interface{}{
		"token": token,
	})
	if res.IsEmpty {
		return nil
	}
	if res.Error != nil {
		panic(res)
	}
	return &r
}

func FindReleaseByTokenAndChannel(token string, channel string) *Release {
	r := Release{}
	r.Init()
	res := r.LoadByCols(map[string]interface{}{
		"token":   token,
		"channel": channel,
	})
	if res.IsEmpty {
		return nil
	}
	if res.Error != nil {
		panic(res)
	}
	return &r
}

func LoadReleaseByChannelAndToken(identity string, token string) *Release {
	r := Release{}
	r.Init()
	var res = r.LoadWith("WHERE channel = ? AND token = ?", identity, token)
	if res.IsEmpty {
		return nil
	}
	if res.Error != nil {
		panic(res)
	}
	return &r
}

func QueryReleasesByChannel(identity string) (*sql.Rows, error) {
	return db.Query(`SELECT 
		title, desc, pubdate, version, 
		short_version_string, filename, mimetype, length, 
		dsa_signature, token
		FROM releases WHERE channel = ? ORDER BY pubdate DESC`, identity)
}
