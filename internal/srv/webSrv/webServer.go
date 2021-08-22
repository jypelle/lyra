package webSrv

import (
	"github.com/gorilla/mux"
	"github.com/jypelle/mifasol/internal/srv/config"
	"github.com/jypelle/mifasol/internal/srv/store"
	"github.com/jypelle/mifasol/internal/srv/webSrv/clients"
	"github.com/jypelle/mifasol/internal/srv/webSrv/static"
	"github.com/jypelle/mifasol/internal/srv/webSrv/templates"
	"github.com/sirupsen/logrus"
	"github.com/vearutop/statigz"
	"html/template"
	"io/fs"
	"net/http"
)

type WebServer struct {
	store        *store.Store
	router       *mux.Router
	serverConfig *config.ServerConfig

	templateHelpers template.FuncMap

	StaticFs    fs.FS
	TemplatesFs fs.FS
	ClientsFs   fs.FS

	log *logrus.Entry
}

func NewWebServer(store *store.Store, router *mux.Router, serverConfig *config.ServerConfig) *WebServer {

	webServer := &WebServer{
		store:        store,
		router:       router,
		serverConfig: serverConfig,
		log:          logrus.WithField("origin", "web"),
	}

	// Ressources
	//	if serverConfig.EmbeddedFs {
	webServer.StaticFs = static.Fs
	webServer.TemplatesFs = templates.Fs
	//webServer.ClientsFs = clients.Fs
	//	} else {
	//		webServer.StaticFs = os.DirFS("internal/srv/webSrv/static")
	//		webServer.TemplatesFs = os.DirFS("internal/srv/webSrv/templates")
	//	}

	// Set routes

	// Static files
	//
	staticFileHandler := http.StripPrefix("/static", http.FileServer(http.FS(webServer.StaticFs)))
	webServer.router.PathPrefix("/static").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Expires", "")
		w.Header().Set("Cache-Control", "public, max-age=2592000") // 30 days
		w.Header().Set("Pragma", "")
		staticFileHandler.ServeHTTP(w, r)
	})

	// Clients binary executables files
	//
	clientsFileHandler := http.StripPrefix("/clients", statigz.FileServer(clients.Fs))
	webServer.router.PathPrefix("/clients").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Expires", "")
		w.Header().Set("Cache-Control", "public, max-age=2592000") // 30 days
		w.Header().Set("Pragma", "")
		clientsFileHandler.ServeHTTP(w, r)
	})

	// Start page
	webServer.router.HandleFunc("/", webServer.IndexAction).Methods("GET").Name("start")

	return webServer
}

func (s *WebServer) Log() *logrus.Entry {
	return s.log
}

func (d *WebServer) Config() *config.ServerConfig {
	return d.serverConfig
}

type IndexView struct {
	Title string
}

func (d *WebServer) IndexAction(w http.ResponseWriter, r *http.Request) {

	view := &IndexView{
		Title: "Mifasol",
	}

	d.HtmlWriterRender(w, view, "main.html")
}
