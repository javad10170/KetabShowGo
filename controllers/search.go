package controllers

import (
	"ketabshowapi/models"

	"github.com/astaxie/beego"
)

// Search book
type SearchController struct {
	beego.Controller
}

// @Title SearchBook
// @Description find book by title
// @Param	query		path 	string	true		"title book"
// @Param	res			path 	string	true		"item per page"
// @Param	page		path 	string	true		"page"
// @Success 200 {object} Book
// @Failure 403 :query is empty
// @router /:query/:res/:page [get]
func (o *SearchController) SearchBook() {
	query := o.Ctx.Input.Param(":query")
	res := o.Ctx.Input.Param(":res")
	page := o.Ctx.Input.Param(":page")

	if query != "" {
		o.Data["json"] = models.SearchBook(query, res, page)
	}

	o.ServeJSON()
}
