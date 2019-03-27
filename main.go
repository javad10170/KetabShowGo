package main

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

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/gorilla/mux"
	"gopkg.in/cheggaaa/pb.v1"
)

const (
	SearchHref    = "<a href='book/index.php.+</a>"
	SearchId      = "book/index\\.php\\?md5=\\w{32}"
	SearchMD5     = "[A-Z0-9]{32}"
	SearchTitle   = ">[^<]+"
	SearchUrl     = "http://booksdl.org/get\\.php\\?md5=\\w{32}\\&key=\\w{16}"
	NumberOfBooks = "10"
)

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
			log.Printf("%d written bytes\n", bytes)
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

func DownloadBook(book Book) error {
	var (
		filename string
		filesize int64
		counter  *WriteCounter
	)

	filename = GetBookFilename(book)
	counter = &WriteCounter{}

	log.Println("Download Started")
	if res, err := http.Get(book.Url); err == nil {
		if res.StatusCode == http.StatusOK {
			defer res.Body.Close()

			filesize = res.ContentLength
			counter.Pb = pb.StartNew(int(filesize))
			out, err := os.Create(filename + ".tmp")

			if err != nil {
				return err
			}
			defer out.Close()
			_, err = io.Copy(out, io.TeeReader(res.Body, counter))
			if err != nil {
				return err
			}
			err = os.Rename(filename+".tmp", filename)
			if err != nil {
				return err
			}

			log.Printf("[OK] %s\n", filename)
		}
	}
	return nil
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
	var formatAuthor string

	for _, md5 := range hashes {
		apiurl := fmt.Sprintf("http://libgen.io/json.php?md5=%s", md5)
		if r, err := http.Get(apiurl); err == nil {
			defer r.Body.Close()

			if b, err := ioutil.ReadAll(r.Body); err == nil {
				book := ParseResponse(b)
				size, _ := strconv.Atoi(book.Filesize)
				fsize := humanize.Bytes(uint64(size))

				fmt.Println(strings.Repeat("-", 80))
				fTitle := fmt.Sprintf("%5s %s", book.Id, book.Title)
				fTitle = formatTitle(fTitle)
				fmt.Printf("%s\n    ++ ", fTitle)
				if len(book.Author) > 25 {
					formatAuthor = book.Author[:25]
				} else {
					formatAuthor = book.Author
				}
				pFormat("author", formatAuthor, color.FgYellow, "-25")
				pFormat("year", book.Year, color.FgCyan, "4")
				pFormat("size", fsize, color.FgGreen, "6")
				pFormat("type", book.Extension, color.FgRed, "4")
				fmt.Println()
				books = append(books, book)
			}
		}
	}

	return
}

func SearchBook(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	BaseUrl := &url.URL{
		Scheme: "http",
		Host:   "libgen.io",
		Path:   "search.php",
	}

	q := BaseUrl.Query()
	q.Set("req", params["query"])
	q.Set("lg_topic", "libgen")
	q.Set("open", "0")
	q.Set("res", string(params["res"]))
	q.Set("phrase", "1")
	q.Set("column", "def")
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
	json.NewEncoder(w).Encode(GetDetails(hashes))
}

func Download(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	hashes := []string{params["md5"]}

	books := GetDetails(hashes)
	DownloadBook(books[0])
	json.NewEncoder(w).Encode(books[0])
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/{query}", SearchBook).Methods("GET")
	router.HandleFunc("/download/{md5}", Download).Methods("GET")
	log.Fatal(http.ListenAndServe(":8000", router))
}
