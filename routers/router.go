// @APIVersion 1.0.0
// @Title Ketabshow Api
// @Description Api for search & download books
// @Contact javad10170@hotmail.com
// @TermsOfServiceUrl https://ketabshow.com/
// @License
// @LicenseUrl
package routers

import (
	"ketabshowapi/controllers"

	"github.com/astaxie/beego"
)

func init() {
	ns := beego.NewNamespace("/v1",
		beego.NSNamespace("/search",
			beego.NSInclude(
				&controllers.SearchController{},
			),
		),
		beego.NSNamespace("/download",
			beego.NSInclude(
				&controllers.DownloadController{},
			),
		),
	)
	beego.AddNamespace(ns)
}
