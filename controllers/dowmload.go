package controllers

import (
	"io"
	"ketabshowapi/models"
	"net/http"
	"strconv"

	"github.com/astaxie/beego"
)

// Download Book
type DownloadController struct {
	beego.Controller
}

// @Title DownloadBook
// @Description download book by md5
// @Param	md5		path 	string	true		"md5 hash book"
// @Success 200 file return
// @Failure 403 :query is empty
// @router /:md5 [get]
func (o *DownloadController) DownloadBook() {
	md5 := o.Ctx.Input.Param(":md5")

	if md5 != "" {
		hashes := []string{md5}
		books := models.GetDetails(hashes)

		var (
			filename string
			filesize int64
		)

		filename = models.GetBookFilename(books[0])

		if res, err := http.Get(books[0].Url); err == nil {
			if res.StatusCode == http.StatusOK {
				defer res.Body.Close()

				filesize = res.ContentLength
				o.Ctx.Output.Context.ResponseWriter.Header().Set("Content-Disposition", "attachment; filename="+filename)
				o.Ctx.Output.Context.ResponseWriter.Header().Set("Content-Type", "application/octet-stream")
				o.Ctx.Output.Context.ResponseWriter.Header().Set("Content-Length", strconv.FormatInt(filesize, 10))
				io.Copy(o.Ctx.Output.Context.ResponseWriter, res.Body)
			}
		}

	}
}
