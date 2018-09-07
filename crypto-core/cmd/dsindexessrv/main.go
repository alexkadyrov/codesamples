package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/bfg-dev/crypto-core/pkg/api"
	"github.com/bfg-dev/crypto-core/pkg/api/dsindexeshandler"
	"github.com/bfg-dev/crypto-core/pkg/api/middlewares"
	"github.com/bfg-dev/crypto-core/pkg/helpers/cmd"
	"github.com/bfg-dev/crypto-core/pkg/services"
	servicesAmqp "github.com/bfg-dev/crypto-core/pkg/services/amqp"
	"github.com/bfg-dev/crypto-core/pkg/services/asset"
	assetPostgres "github.com/bfg-dev/crypto-core/pkg/services/asset/postgres"
	"github.com/bfg-dev/crypto-core/pkg/services/blockchain"
	"github.com/bfg-dev/crypto-core/pkg/services/blockchain/amqp/buyback"
	blockchainPostgres "github.com/bfg-dev/crypto-core/pkg/services/blockchain/postgres"
	"github.com/bfg-dev/crypto-core/pkg/services/cryptofund"
	"github.com/bfg-dev/crypto-core/pkg/services/tokenemission"
	tokenEmissionPostgres "github.com/bfg-dev/crypto-core/pkg/services/tokenemission/postgres"
	"github.com/bfg-dev/crypto-core/pkg/services/tokenredemption"
	tokenRedemptionPostgres "github.com/bfg-dev/crypto-core/pkg/services/tokenredemption/postgres"
	"github.com/bfg-dev/crypto-core/pkg/types/exchange"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
)

const defaultConfigName = "config.toml"

var config string

func main() {

	app := services.Application()
	err := app.Init(config)
	cmd.DieIfError(err, "app init error")

	amqpConnectionProvider, err := servicesAmqp.NewConnectionProvider(app.Config().GetString("CRYPTO_AMQP_HOST"),
		app.Config().GetInt("CRYPTO_AMQP_PORT"),
		app.Config().GetString("CRYPTO_AMQP_USER"),
		app.Config().GetString("CRYPTO_AMQP_PASS"),
		app.Config().GetString("CRYPTO_AMQP_VHOST"),
		app.Logger())
	cmd.DieIfError(err, "amqpConnectionProvider init error")

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

	assetRepo, err := assetPostgres.NewAssetRepository(dbConnection)
	cmd.DieIfError(err, "assetRepo init error")

	assetService, err := asset.NewService(assetRepo)
	cmd.DieIfError(err, "assetService init error")

	//TokenEmissionService
	tokenLedgerRepository, err := tokenEmissionPostgres.NewLedgerRepository(dbConnection)
	cmd.DieIfError(err, "tokenLedgerRepository init error")

	tokenRecordRepository, err := tokenEmissionPostgres.NewRecordRepository(dbConnection)
	cmd.DieIfError(err, "tokenRecordRepository init error")

	tokenEmissionService, err := tokenemission.NewService(tokenLedgerRepository, tokenRecordRepository)
	cmd.DieIfError(err, "tokenEmissionService init error")

	//TokenRedemptoionService
	tokenRedemptionBookRepository, err := tokenRedemptionPostgres.NewBookRepository(dbConnection)
	cmd.DieIfError(err, "tokenRedemptionBookRepository init error")

	tokenRedemptionBookEntryRepository, err := tokenRedemptionPostgres.NewBookEntryRepository(dbConnection)
	cmd.DieIfError(err, "tokenRedemptionBookEntryRepository init error")

	tokenRedemptionService, err := tokenredemption.NewService(tokenRedemptionBookRepository, tokenRedemptionBookEntryRepository)
	cmd.DieIfError(err, "tokenRedemptionService init error")

	buybackRepository, err := blockchainPostgres.NewBuybackRepository(dbConnection)
	cmd.DieIfError(err, "tokenRedemptionBookRepository init error")

	buybackEntryRepository, err := blockchainPostgres.NewBuybackEntryRepository(dbConnection)
	cmd.DieIfError(err, "tokenRedemptionBookEntryRepository init error")

	buybackPriceRepo, err := blockchainPostgres.NewBuybackPriceRepository(dbConnection)
	cmd.DieIfError(err, "buybackPriceRepository init error")

	buybackOracleClient, err := buyback.NewOracleClient(exchange.Middleware, amqpConnectionProvider)
	cmd.DieIfError(err, "NewOracleClient init error")

	buybackService, err := blockchain.NewBuybackService(buybackRepository, buybackEntryRepository, buybackPriceRepo, buybackOracleClient, app.Logger())
	cmd.DieIfError(err, "NewBuybackService init error")

	cryptofundService, err := cryptofund.NewService(
		app.Config().GetString("CRYPTO_INDEXES_URL"),
		app.Config().GetString("CRYPTO_INDEXES_USER"),
		app.Config().GetString("CRYPTO_INDEXES_PASS"),
	)
	cmd.DieIfError(err, "cryptofundService init error")

	handler, err := dsindexeshandler.New(
		app,
		assetService,
		tokenEmissionService,
		tokenRedemptionService,
		buybackService,
		cryptofundService)
	cmd.DieIfError(err, "dsindexeshandler init error")

	r := mux.NewRouter()

	r.Handle("/1.0/tokens/summary", common.With(
		negroni.WrapFunc(api.ResponseHandler(handler.GetSummary)))).Methods("GET")

	r.Handle("/1.1/tokens/summary", common.With(
		negroni.WrapFunc(api.ResponseHandler(handler.GetSummaryV2)))).Methods("GET")

	http.ListenAndServe(":8087", r)
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
