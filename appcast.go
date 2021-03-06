package main

import (
	"crypto/sha1"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"regexp"
	"text/template"
	"time"
)

import (
	"github.com/c9s/appcast"
	_ "github.com/c9s/appcast-server/uploader"
	"github.com/c9s/gatsby"
	"github.com/c9s/jsondata"
	"github.com/c9s/rss"
	_ "github.com/mattn/go-sqlite3"
)

const BIND = ":5000"
const UPLOAD_DIR = "uploads"
const DEFAULT_SQLITEDB = "appcast.db"

var BASEURL string
var domain = flag.String("domain", "localhost", "base url")
var bind = flag.String("bind", ":5000", "bind")
var dbname = flag.String("db", DEFAULT_SQLITEDB, "database name")

var ErrFileIsRequired = errors.New("file is required.")
var ErrReleaseInsertFailed = errors.New("release insert failed.")

var routeUploadPageRegExp = regexp.MustCompile("/release/upload/([^/]+)/([^/]+)")
var routeReleaseCreateRegExp = regexp.MustCompile("/release/create/([^/]+)/([^/]+)")
var routeDownloadRegExp = regexp.MustCompile("/release/download/([^/]+)/([^/]+)/([^/]+)")

var db *sql.DB
var templates = template.Must(template.ParseGlob("templates/*.html"))

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

func CreateNewReleaseFromRequest(r *http.Request, channelIdentity string) (*Release, error) {
	file, fileReader, err := r.FormFile("file")
	if err == http.ErrMissingFile {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var dstFilePath = path.Join(UPLOAD_DIR, fileReader.Filename)
	if err = ioutil.WriteFile(dstFilePath, data, 0777); err != nil {
		return nil, err
	}

	length, _ := GetFileLength(dstFilePath)
	mimetype := GetMimeTypeByFilename(fileReader.Filename)

	title := r.FormValue("title")
	desc := r.FormValue("desc")
	pubDate := r.FormValue("pubDate")
	version := r.FormValue("version")
	shortVersionString := r.FormValue("shortVersionString")
	releaseNotes := r.FormValue("releaseNotes")
	dsaSignature := r.FormValue("dsaSignature")

	h := sha1.New()
	h.Write([]byte(title))
	h.Write([]byte(version))
	h.Write([]byte(shortVersionString))
	h.Write(data)
	token := fmt.Sprintf("%x", h.Sum(nil))

	pubDateTime, err := time.Parse("2006-01-02", pubDate)
	if err != nil {
		panic(err)
	}

	newRelease := Release{}
	newRelease.Title = title
	newRelease.Description = desc
	newRelease.Version = version
	newRelease.ShortVersionString = shortVersionString
	newRelease.ReleaseNotes = releaseNotes
	newRelease.DSASignature = dsaSignature
	newRelease.Token = token
	newRelease.Filename = fileReader.Filename
	newRelease.PubDate = &pubDateTime
	newRelease.Length = length
	newRelease.Mimetype = mimetype
	newRelease.Channel = channelIdentity
	newRelease.Init()
	var result = newRelease.Create()
	if result.Error != nil {
		panic(result)
	}
	log.Println("New Release Uploaded", newRelease)
	return &newRelease, nil
}

func CreateChannelHandler(w http.ResponseWriter, r *http.Request) {
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

func UploadPageHandler(w http.ResponseWriter, r *http.Request) {
	submatches := routeUploadPageRegExp.FindStringSubmatch(r.URL.Path)
	if len(submatches) != 3 {
		ForbiddenHandler(w, r)
		return
	}
	channelIdentity := submatches[1]
	channelToken := submatches[2]

	if channel := FindChannelByIdentity(channelIdentity, channelToken); channel != nil {
		if r.Method == "POST" {
			if _, err := CreateNewReleaseFromRequest(r, channelIdentity); err != nil {
				panic(err)
			}
		}

		t := templates.Lookup("upload.html")
		if t != nil {
			err := t.Execute(w, channel)
			if err != nil {
				panic(err)
			}
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Channel not found"))
		return
	}
}

func ScanRowToAppcastItem(rows *sql.Rows, channelIdentity, channelToken string) (*appcast.Item, error) {
	var title, desc, version, shortVersionString, filename, mimetype, dsaSignature, token string
	var pubDate time.Time
	var length int64
	var err = rows.Scan(&title, &desc, &pubDate, &version, &shortVersionString, &filename, &mimetype, &length, &dsaSignature, &token)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var item = appcast.Item{}
	item.Title = title
	item.Description = desc
	item.PubDate = rss.Date(pubDate.Format(time.RFC822Z))
	item.Enclosure.Length = length
	item.Enclosure.Type = mimetype
	item.Enclosure.SparkleVersion = version
	item.Enclosure.SparkleVersionShortString = shortVersionString
	item.Enclosure.SparkleDSASignature = dsaSignature
	item.Enclosure.URL = BASEURL + "/release/download/" + channelIdentity + "/" + channelToken + "/" + token
	return &item, nil
}

func ForbiddenHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("Forbidden"))
	return
}

/*
For route: /download/gotray/{token}

/download/gotray/be24d1c54d0ba415b8897b02f0c38d89
*/

func DownloadFileHandler(w http.ResponseWriter, r *http.Request) {
	submatches := routeDownloadRegExp.FindStringSubmatch(r.URL.Path)
	if len(submatches) != 4 {
		ForbiddenHandler(w, r)
		return
	}

	channelIdentity := submatches[1]
	channelToken := submatches[2]
	releaseToken := submatches[3]

	if channel := FindChannelByIdentity(channelIdentity, channelToken); channel != nil {
		if release := LoadReleaseByChannelAndToken(channelIdentity, releaseToken); release != nil {
			log.Println("Download release", release.Filename, release.Mimetype, release.Downloaded)

			// Update downloaded counter
			db.Exec(`UPDATE releases SET downloaded = downloaded + 1 WHERE id = ?`, release.Id)
			w.Header().Set("Content-Type", release.Mimetype)
			w.Header().Set("Content-Disposition", "inline; filename=\""+release.Filename+"\"")

			dlog := DownloadLog{
				RemoteAddr: r.RemoteAddr,
				RequestURI: r.RequestURI,
				UserAgent:  r.UserAgent(),
				Referer:    r.Referer(),
				ReleaseId:  release.Id,
			}
			dlog.Init()
			dlog.Create()

			data, err := ioutil.ReadFile(path.Join(UPLOAD_DIR, release.Filename))
			if err != nil {
				panic(err)
			}
			w.Write(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Release not found"))
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Channel not found"))
	}
}

func main() {
	flag.Parse()

	BASEURL = "http://" + *domain
	log.Println(BASEURL)

	db = ConnectDB(*dbname)
	gatsby.SetupConnection(db, gatsby.DriverSqlite)
	defer db.Close()

	/*
		/release/download/{channel identity}/{channel token}/{release token}
		/release/upload/{channel identity}/{channel token}
		/release/new/{channel identity}
		/appcast/{channel identity}.xml
	*/
	http.HandleFunc("/release/download/", DownloadFileHandler)
	http.HandleFunc("/release/upload/", UploadPageHandler)
	http.HandleFunc("/release/create/", UploadReleaseHandler)
	http.HandleFunc("/channel/create", CreateChannelHandler)
	http.HandleFunc("/appcast/", AppcastXmlHandler)
	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))

	log.Println("Listening " + *bind + " ...")
	if err := http.ListenAndServe(*bind, nil); err != nil {
		panic(err)
	}
}
