package main

import (
	"crypto/sha1"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/c9s/jsondata"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"text/template"
	"time"
)

import (
	"github.com/c9s/appcast"
	_ "github.com/c9s/appcast-server/uploader"
	"github.com/c9s/gatsby"
	"github.com/c9s/rss"
	"github.com/gorilla/pat"
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

func ChannelUpdateHandler(w http.ResponseWriter, r *http.Request) {
	channelId, _ := strconv.Atoi(r.URL.Query().Get(":channelId"))
	channel := Channel{}
	channel.Init()

	var payload map[string]interface{}
	body, err := ioutil.ReadAll(r.Body)
	err = json.Unmarshal(body, &payload)
	if err != nil {
		log.Println(err)
		writeJson(w, jsondata.Map{"error": err})
		return
	}

	var res = channel.Load(int64(channelId))
	if res.IsEmpty {
		writeJson(w, jsondata.Map{"error": "Channel not found"})
		return
	}
	if res.Error != nil {
		log.Println(res.Error)
		writeJson(w, jsondata.Map{"error": res.Error})
		return
	}

	if title, ok := payload["title"]; ok {
		channel.Title = title.(string)
	}
	if description, ok := payload["description"]; ok {
		channel.Description = description.(string)
	}
	if identity, ok := payload["identity"]; ok {
		channel.Identity = identity.(string)
	}
	if token, ok := payload["token"]; ok {
		channel.Token = token.(string)
	}
	res = channel.Update()
	if res.Error != nil {
		panic(res)
	}
	writeJson(w, channel)
}

func ChannelGetHandler(w http.ResponseWriter, r *http.Request) {
	channelId, _ := strconv.Atoi(r.URL.Query().Get(":channelId"))
	channel := Channel{}
	channel.Init()
	var res = channel.Load(int64(channelId))
	if res.IsEmpty {
		writeJson(w, jsondata.Map{"error": "Channel not found"})
		return
	}
	if res.Error != nil {
		writeJson(w, jsondata.Map{"error": res.Error})
		return
	}
	writeJson(w, channel)
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
	http.HandleFunc("/channel/create", ChannelCreateHandler)
	http.HandleFunc("/channel", ChannelListHandler)

	http.HandleFunc("/=/channels", ChannelCollectionHandler)
	http.HandleFunc("/=/channel/:channelId", ChannelCollectionHandler)

	r := pat.New()
	r.Get("/=/channel/{channelId}", ChannelGetHandler)
	r.Post("/=/channel/{channelId}", ChannelUpdateHandler)
	http.Handle("/", r)

	http.Handle("/partials/", http.StripPrefix("/partials/", http.FileServer(http.Dir("views/partials"))))

	http.HandleFunc("/appcast/", AppcastXmlHandler)
	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))

	log.Println("Listening " + *bind + " ...")
	if err := http.ListenAndServe(*bind, nil); err != nil {
		panic(err)
	}
}
