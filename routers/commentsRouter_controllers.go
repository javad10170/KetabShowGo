package routers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context/param"
)

func init() {

    beego.GlobalControllerRouter["ketabshowapi/controllers:DownloadController"] = append(beego.GlobalControllerRouter["ketabshowapi/controllers:DownloadController"],
        beego.ControllerComments{
            Method: "DownloadBook",
            Router: `/:md5`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["ketabshowapi/controllers:SearchController"] = append(beego.GlobalControllerRouter["ketabshowapi/controllers:SearchController"],
        beego.ControllerComments{
            Method: "SearchBook",
            Router: `/:query/:res/:page`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

}
