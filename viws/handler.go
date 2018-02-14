package viws

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils"
	"github.com/ViBiOh/httputils/tools"
	"github.com/ViBiOh/viws/utils"
)

const notFoundFilename = `404.html`

// App stores informations and secret of API
type App struct {
	spa          bool
	directory    string
	pushPaths    []string
	notFoundPath *string
}

// NewApp creates new App from Flags' config
func NewApp(config map[string]interface{}, tls bool) (*App, error) {
	spa := *(config[`spa`].(*bool))
	notFound := *(config[`notFound`].(*bool))
	directory := *(config[`directory`].(*string))
	push := *(config[`push`].(*string))

	if utils.IsFileExist(directory) == nil {
		return nil, fmt.Errorf(`Directory %s is unreachable or does not contains index`, directory)
	}
	log.Printf(`[viws] Serving file from %s`, directory)

	var notFoundPath *string
	if notFound {
		if spa {
			return nil, errors.New(`Incompatible options provided: -notFound and -spa`)
		}

		if notFoundPath = utils.IsFileExist(directory, notFoundFilename); notFoundPath == nil {
			return nil, fmt.Errorf(`Not found page %s%s is unreachable`, directory, notFoundFilename)
		}

		log.Printf(`[viws] 404 will be %s`, *notFoundPath)
	}

	var pushPaths []string
	if push != `` {
		if !tls {
			return nil, errors.New(`HTTP/2 Server Push works only when TLS in enabled`)
		}

		pushPaths = strings.Split(push, `,`)
	}

	if spa {
		log.Print(`[viws] Working in SPA mode`)
	}

	return &App{
		spa:          spa,
		directory:    directory,
		pushPaths:    pushPaths,
		notFoundPath: notFoundPath,
	}, nil
}

// Flags add flags for given prefix
func Flags(prefix string) map[string]interface{} {
	return map[string]interface{}{
		`directory`: flag.String(tools.ToCamel(prefix+`Directory`), `/www/`, `[viws] Directory to serve`),
		`notFound`:  flag.Bool(tools.ToCamel(prefix+`NotFound`), false, `[viws] Graceful 404 page at /404.html`),
		`spa`:       flag.Bool(tools.ToCamel(prefix+`Spa`), false, `[viws] Indicate Single Page Application mode`),
		`push`:      flag.String(tools.ToCamel(prefix+`Push`), ``, `[viws] Paths for HTTP/2 Server Push on index, comma separated`),
	}
}

// ServerPushHandler add server push when serving index
func (a *App) ServerPushHandler(next http.Handler) http.Handler {
	if len(a.pushPaths) == 0 {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.URL.Path == `/` {
			if pusher, ok := w.(http.Pusher); ok {
				for _, path := range a.pushPaths {
					if err := pusher.Push(path, nil); err != nil {
						log.Printf(`Failed to push %s: %v`, path, err)
					}
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// FileHandler serve file given configuration
func (a *App) FileHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if filename := utils.IsFileExist(a.directory, r.URL.Path); filename != nil {
			http.ServeFile(w, r, *filename)
		} else if a.notFoundPath != nil {
			w.WriteHeader(http.StatusNotFound)
			http.ServeFile(w, r, *a.notFoundPath)
		} else if a.spa {
			w.Header().Add(`Cache-Control`, `no-cache`)
			http.ServeFile(w, r, a.directory)
		} else {
			httputils.NotFound(w)
		}
	})
}