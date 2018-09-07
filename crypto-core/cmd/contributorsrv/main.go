package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/bfg-dev/crypto-core/pkg/api"
	"github.com/bfg-dev/crypto-core/pkg/api/middlewares"
	"github.com/bfg-dev/crypto-core/pkg/helpers/cmd"
	"github.com/bfg-dev/crypto-core/pkg/services"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/bfg-dev/crypto-core/pkg/api/contributorhandler"
	"github.com/bfg-dev/crypto-core/pkg/services/contributor"
	"github.com/bfg-dev/crypto-core/pkg/services/contributor/postgres"
)

const defaultConfigName = "config.toml"

var config string

func main() {

	app := services.Application()
	err := app.Init(config)
	cmd.DieIfError(err, "app init error")

	/**
	 *  Init middleware
	 */

	//Error interceptor middleware
	errorInterceptorMiddleware, err := middlewares.NewErrorInterceptor(app.Logger())
	cmd.DieIfError(err, "interceptor middleware init error")

	//Middleware for all routes
	//ORDER SENSITIVE!!!
	common := negroni.New(errorInterceptorMiddleware)

	dbConnection := app.DBConnection()
	if dbConnection == nil {
		cmd.Die("unable to get db connection from application")
	}


	missionRepository, err := postgres.NewMissionRepository(dbConnection)
	cmd.DieIfError(err, "NewMissionRepository init error")

	userMissionRepository, err := postgres.NewUserMissionRepository(dbConnection)

	contributorService, err := contributor.NewService(
		missionRepository,
		userMissionRepository,
	)
	cmd.DieIfError(err, "cryptofundService init error")

	handler, err := contributorhandler.New(
		app,
		contributorService,
	)
	cmd.DieIfError(err, "contributorhandler init error")

	r := mux.NewRouter()

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("pkg/templates/contributors/static/"))))

	r.Handle("/getNewUserMissionRequests", common.With(
		negroni.WrapFunc(api.ResponseHandler(handler.GetNewUserMissionRequests)))).Methods("GET")

	r.Handle("/admin/NewUserMissionRequests", common.With(
		negroni.WrapFunc(handler.RenderNewUserMissionRequestList))).Methods("GET")

	r.Handle("/admin/SetUserRequestStatus", common.With(
		negroni.WrapFunc(handler.SetUserRequestStatus))).Methods("POST")

	http.ListenAndServe(":8090", r)
}

func init() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))

	if err != nil {
		fmt.Println("Unable to get current path")
		os.Exit(1)
	}

	defaultConfigPath := path.Join(dir, defaultConfigName)

	flag.StringVar(&config, "config", defaultConfigPath, "You can set config file path")
	flag.Parse()
}
