package models

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/gorilla/mux"
	"gopkg.in/cheggaaa/pb.v1"
)

var (
	Objects map[string]*Book
)

const (
	SearchHref    = "<a href='book/index.php.+</a>"
	SearchId      = "book/index\\.php\\?md5=\\w{32}"
	SearchMD5     = "[A-Z0-9]{32}"
	SearchTitle   = ">[^<]+"
	SearchUrl     = "http://booksdl.org/get\\.php\\?md5=\\w{32}\\&key=\\w{16}"
	NumberOfBooks = "10"
)

// swagger:model
type Book struct {
	Title       string
	Id          string
	Author      string
	Filesize    string
	Extension   string
	Md5         string
	Year        string
	Url         string
	Coverurl    string
	Language    string
	Description string
	Isbn        string
	Publisher   string
}

type WriteCounter struct {
	Total uint64
	Pb    *pb.ProgressBar
}

type BookFile struct {
	size int64
	name string
	path string
	data []byte
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.Pb.Add64(int64(n))
	return n, nil
}

func GetHref(HttpResponse string) (href string) {
	re := regexp.MustCompile(SearchUrl)
	matchs := re.FindAllString(HttpResponse, -1)

	if len(matchs) > 0 {
		href = matchs[0]
	}

	return
}

func GetDownloadUrl(book *Book) error {
	BaseUrl := &url.URL{
		Scheme: "http",
		Host:   "booksdescr.org",
		Path:   "ads.php",
	}

	q := BaseUrl.Query()
	q.Set("md5", book.Md5)
	BaseUrl.RawQuery = q.Encode()

	res, err := http.Get(BaseUrl.String())
	if err != nil {
		log.Printf("http.Get(%q) error: %v", BaseUrl, err)
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("res.StatusCode = %d; want %d",
			res.StatusCode,
			http.StatusOK,
		)
	} else {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Printf("error reading resp body: %v", err)
			return err
		}
		book.Url = GetHref(string(b))
	}
	return nil
}

func WriteFile(book BookFile, dest string) (bytes int) {
	bytes = 0

	log.Println("Start writing")
	p := filepath.Join(dest, book.name)
	if f, err := os.Create(p); err == nil {
		defer f.Close()
		if bytes, err = f.Write(book.data); err == nil {
			f.Sync()
			return
		}
	}

	log.Println("Fail to write new file\n")
	return
}

func GetBookFilename(book Book) (filename string) {
	var tmp []string

	tmp = append(tmp, book.Title)
	tmp = append(tmp, fmt.Sprintf(" (%s - %s)", book.Year, book.Author))
	tmp = append(tmp, fmt.Sprintf(".%s", book.Extension))
	filename = strings.Join(tmp, "")
	return
}

func ParseHashes(response string) (hashes []string) {
	re := regexp.MustCompile(SearchHref)
	matchs := re.FindAllString(response, -1)

	for _, m := range matchs {
		re := regexp.MustCompile(SearchMD5)
		hash := re.FindString(m)
		if len(hash) == 32 {
			log.Printf("New hash found %s\n", hash)
			hashes = append(hashes, hash)
		}
	}

	return
}

func ParseResponse(data []byte) (book Book) {
	var cache []map[string]string

	if err := json.Unmarshal(data, &cache); err == nil {
		for _, item := range cache {
			for k, v := range item {
				switch k {
				case "id":
					book.Id = v
				case "title":
					book.Title = v
				case "author":
					book.Author = v
				case "filesize":
					book.Filesize = v
				case "extension":
					book.Extension = v
				case "md5":
					book.Md5 = v
					book.Url = "http://booksdl.org/get.php?md5=" + v
				case "coverurl":
					book.Coverurl = v
				case "language":
					book.Language = v
				case "descr":
					book.Description = v
				case "identifierwodash":
					book.Isbn = v
				case "publisher":
					book.Publisher = v
				}
			}
		}
	}

	return
}

func formatTitle(title string) (formatTitle string) {
	var cache []string
	var counter int

	if len(title) < 60 {
		return title
	}

	title = strings.TrimSpace(title)
	for _, t := range strings.Split(title, " ") {
		counter += len(t)

		if counter > 60 {
			counter = 0
			t = t + "\n"
		}
		cache = append(cache, t)
	}
	formatTitle = strings.Join(cache, " ")

	return
}

func pFormat(key string, value string, attr color.Attribute, align string) {
	c := color.New(attr).SprintFunc()
	a := fmt.Sprintf("%%%ss ", align)
	s := fmt.Sprintf("@%s "+a, c(key), value)
	fmt.Printf(a, s)
}

func GetDetails(hashes []string) (books []Book) {

	for _, md5 := range hashes {
		apiurl := fmt.Sprintf("http://libgen.io/json.php?md5=%s", md5)
		if r, err := http.Get(apiurl); err == nil {
			defer r.Body.Close()

			if b, err := ioutil.ReadAll(r.Body); err == nil {
				book := ParseResponse(b)
				books = append(books, book)
			}
		}
	}

	return
}

func SearchBook(query string, itemPerPage string, page string) (book []Book) {

	BaseUrl := &url.URL{
		Scheme: "http",
		Host:   "libgen.io",
		Path:   "search.php",
	}

	q := BaseUrl.Query()
	q.Set("req", query)
	q.Set("lg_topic", "libgen")
	q.Set("open", "0")
	q.Set("res", itemPerPage)
	q.Set("phrase", "1")
	q.Set("column", "def")
	q.Set("page", page)
	BaseUrl.RawQuery = q.Encode()

	res, err := http.Get(BaseUrl.String())
	if err != nil {
		log.Printf("http.Get(%q) error: %v", BaseUrl, err)
		os.Exit(-1)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("res.StatusCode = %d; want %d",
			res.StatusCode,
			http.StatusOK,
		)
		os.Exit(-1)
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal("error reading resp body: %v", err)
		os.Exit(-1)
	}
	hashes := ParseHashes(string(b))
	return GetDetails(hashes)
}

func Download(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	hashes := []string{params["md5"]}

	books := GetDetails(hashes)

	var (
		filename string
		filesize int64
		counter  *WriteCounter
	)

	filename = GetBookFilename(books[0])
	counter = &WriteCounter{}

	if res, err := http.Get(books[0].Url); err == nil {
		if res.StatusCode == http.StatusOK {
			defer res.Body.Close()

			filesize = res.ContentLength
			counter.Pb = pb.StartNew(int(filesize))
			w.Header().Set("Content-Disposition", "attachment; filename="+filename)
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", strconv.FormatInt(filesize, 10))
			_, err = io.Copy(w, io.TeeReader(res.Body, counter))
		}
	}
}
