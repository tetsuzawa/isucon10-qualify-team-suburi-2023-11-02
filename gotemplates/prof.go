package gotemplates

import (
	"net/http"
	"net/http/pprof"
	_ "net/http/pprof"

	"github.com/labstack/echo/v4"
)

// go tool pprof -list main. -cum -seconds 60 http://43.206.3.190/debug/pprof/profile
// https://www.tetsuzawa.com/docs/ISUCON/go/pprof
func Pprof(e *echo.Echo) *echo.Echo {
	prefixRouter := e.Group("/debug/pprof")
	{
		prefixRouter.GET("/", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
		prefixRouter.GET("/allocs", echo.WrapHandler(pprof.Handler("allocs")))
		prefixRouter.GET("/block", echo.WrapHandler(pprof.Handler("block")))
		prefixRouter.GET("/cmdline", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
		prefixRouter.GET("/goroutine", echo.WrapHandler(pprof.Handler("goroutine")))
		prefixRouter.GET("/heap", echo.WrapHandler(pprof.Handler("heap")))
		prefixRouter.GET("/mutex", echo.WrapHandler(pprof.Handler("mutex")))
		prefixRouter.GET("/profile", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
		prefixRouter.POST("/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
		prefixRouter.GET("/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
		prefixRouter.GET("/threadcreate", echo.WrapHandler(pprof.Handler("threadcreate")))
		//prefixRouter.GET("/trace", echo.WrapHandler(pprof.Trace))
	}
	return e
}
