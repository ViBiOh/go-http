package main

import (
	"flag"
	"net/http"
	"os"

	httputils "github.com/ViBiOh/httputils/pkg"
	"github.com/ViBiOh/httputils/pkg/alcotest"
	"github.com/ViBiOh/httputils/pkg/cors"
	"github.com/ViBiOh/httputils/pkg/logger"
	"github.com/ViBiOh/httputils/pkg/owasp"
	"github.com/ViBiOh/viws/pkg/env"
	"github.com/ViBiOh/viws/pkg/viws"
)

func main() {
	fs := flag.NewFlagSet("viws", flag.ExitOnError)

	serverConfig := httputils.Flags(fs, "")
	alcotestConfig := alcotest.Flags(fs, "")
	owaspConfig := owasp.Flags(fs, "")
	corsConfig := cors.Flags(fs, "cors")

	viwsConfig := viws.Flags(fs, "")
	envConfig := env.Flags(fs, "")

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)

	serverApp, err := httputils.New(serverConfig)
	logger.Fatal(err)

	owaspApp := owasp.New(owaspConfig)
	corsApp := cors.New(corsConfig)

	viwsApp, err := viws.New(viwsConfig)
	if err != nil {
		logger.Error("%#v", err)
	}
	envApp := env.New(envConfig)

	viwsHandler := httputils.ChainMiddlewares(viwsApp.Handler(), owaspApp)
	envHandler := httputils.ChainMiddlewares(envApp.Handler(), owaspApp, corsApp)
	requestHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/env" {
			envHandler.ServeHTTP(w, r)
		} else {
			viwsHandler.ServeHTTP(w, r)
		}
	})

	serverApp.ListenAndServe(requestHandler, httputils.HealthHandler(nil), nil)
}
