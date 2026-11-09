package cmd

import (
	"context"
	"io/fs"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/handlers"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/authorization"
	"github.com/energiegedrag-nl/needforheat-server-api/repositories"
	"github.com/energiegedrag-nl/needforheat-server-api/services"
	"github.com/energiegedrag-nl/needforheat-server-api/swaggerdocs"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func init() {
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the needforheat API server",
		RunE:  handleServe,
	}

	rootCmd.AddCommand(serveCmd)
}

const (
	shutdownTimeout    = 30 * time.Second
	preRenewalDuration = 12 * time.Hour
)

func handleServe(cmd *cobra.Command, args []string) error {
	config := getConfiguration()

	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	dbCtx, dbCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dbCancel()

	db, err := repositories.NewDatabaseConnectionAndMigrate(dbCtx, config.DatabaseDSN)

	if err != nil {
		logrus.Fatal(err)
	}

	//Important services for admin and auth
	authService, err := services.NewAuthorizationServiceFromFile("./data/key.pem")
	if err != nil {
		logrus.Fatal(err)
	}
	authHandler := handlers.NewAuthorizationHandler(authService)

	adminRepository, err := repositories.NewAdminRepository("./data/admins.db")
	if err != nil {
		logrus.Fatal(err)
	}
	adminService := services.NewAdminService(adminRepository, authService)
	adminHandler := handlers.NewAdminHandler(adminService)

	adminAuth := authHandler.Middleware(authorization.AdminToken)
	accountActivationAuth := authHandler.Middleware(authorization.AccountActivationToken)
	accountAuth := authHandler.Middleware(authorization.AccountToken)
	deviceORaccountAuth := authHandler.DoubleMiddleware(authorization.DeviceToken, authorization.AccountToken)

	//Repositories
	appRepository := repositories.NewAppRepository(db)
	cloudFeedTypeRepository := repositories.NewCloudFeedTypeRepository(db)
	cloudFeedRepository := repositories.NewCloudFeedRepository(db)
	campaignRepository := repositories.NewCampaignRepository(db)
	propertyRepository := repositories.NewPropertyRepository(db)
	uploadRepository := repositories.NewUploadRepository(db)
	accountRepository := repositories.NewAccountRepository(db)
	deviceTypeRepository := repositories.NewDeviceTypeRepository(db)
	deviceRepository := repositories.NewDeviceRepository(db)
	dataSourceListRepository := repositories.NewDataSourceListRepository(db)
	dataSourceTypeRepository := repositories.NewDataSourceTypeRepository(db)
	energyQueryRepository := repositories.NewEnergyQueryRepository(db)
	energyQueryTypeRepository := repositories.NewEnergyQueryTypeRepository(db)
	apiKeyRepository := repositories.NewAPIKeyRepository(db)

	//Services
	appService := services.NewAppService(appRepository)
	cloudFeedTypeService := services.NewCloudFeedTypeService(cloudFeedTypeRepository)
	propertyService := services.NewPropertyService(propertyRepository)
	deviceTypeService := services.NewDeviceTypeService(deviceTypeRepository, propertyService)
	energyQueryTypeService := services.NewEnergyQueryTypeService(energyQueryTypeRepository, propertyService)
	dataSourceTypeService := services.NewDataSourceTypeService(
		dataSourceTypeRepository,
		deviceTypeService,
		cloudFeedTypeService,
		energyQueryTypeService,
	)
	dataSourceListService := services.NewDataSourceListService(dataSourceListRepository, dataSourceTypeService)
	campaignService := services.NewCampaignService(campaignRepository, appService, dataSourceListService)
	uploadService := services.NewUploadService(uploadRepository, deviceRepository, propertyService)
	cloudFeedService := services.NewCloudFeedService(cloudFeedRepository, cloudFeedTypeRepository, uploadService)
	accountService := services.NewAccountService(accountRepository, authService, appService, campaignService, cloudFeedService, dataSourceTypeService)
	energyQueryService := services.NewEnergyQueryService(energyQueryRepository, authService, energyQueryTypeService, accountService, uploadService)
	deviceService := services.NewDeviceService(deviceRepository, authService, deviceTypeService, accountService, uploadService)
	apiKeyService := services.NewAPIKeyService(apiKeyRepository)

	//Handlers
	appHandler := handlers.NewAppHandler(appService)
	cloudFeedTypeHandler := handlers.NewCloudFeedTypeHandler(cloudFeedTypeService)
	cloudFeedHandler := handlers.NewCloudFeedHandler(cloudFeedService)
	campaignHandler := handlers.NewCampaignHandler(campaignService)
	uploadHandler := handlers.NewUploadHandler(uploadService)
	accountHandler := handlers.NewAccountHandler(accountService)
	deviceTypeHandler := handlers.NewDeviceTypeHandler(deviceTypeService)
	deviceHandler := handlers.NewDeviceHandler(deviceService)
	dataSourceListHandler := handlers.NewDataSourceListHandler(dataSourceListService)
	dataSourceTypeHandler := handlers.NewDataSourceTypeHandler(dataSourceTypeService)
	energyQueryHandler := handlers.NewEnergyQueryHandler(energyQueryService)
	energyQueryTypeHandler := handlers.NewEnergyQueryTypeHandler(energyQueryTypeService)
	apiKeyHandler := handlers.NewAPIKeyHandler(apiKeyService)
	cockpitHandler := handlers.NewCockpitHandler(repositories.NewCockpitRepository(db), accountService)

	go cloudFeedService.RefreshTokensInBackground(ctx, preRenewalDuration)
	go cloudFeedService.DownloadInBackground(ctx, config.downloadStartTime)

	//Router
	r := chi.NewRouter()

	r.Use(middleware.Timeout(time.Second * 30))
	r.Use(middleware.Heartbeat("/healthcheck")) // Endpoint for health check.
	r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: logrus.StandardLogger()}))

	r.Method("POST", "/app", adminAuth(adminHandler.Middleware(appHandler.Create))) // POST on /app.

	r.Method("POST", "/cloud_feed_type", adminAuth(adminHandler.Middleware(cloudFeedTypeHandler.Create))) // POST on /cloud_feed.

	r.Method("POST", "/campaign", adminAuth(adminHandler.Middleware(campaignHandler.Create))) // POST on /campaign.

	r.Route("/account", func(r chi.Router) {
		r.Method("POST", "/", adminAuth(adminHandler.Middleware(accountHandler.Create))) // POST on /account.
		r.Method("POST", "/activate", accountActivationAuth(accountHandler.Activate))    // POST on /account/activate.

		r.Route("/{account_id}", func(r chi.Router) {
			r.Method("GET", "/", accountAuth(accountHandler.GetAccountByID))                     // GET on /account/{account_id}.
			r.Method("POST", "/cloud_feed", accountAuth(cloudFeedHandler.Create))                // POST on /account/{account_id}/cloud_feed_auth.
			r.Method("GET", "/cloud_feed", accountAuth(accountHandler.GetCloudFeedAuthStatuses)) // GET on /account/{account_id}/cloud_feed_auth.
		})
	})

	r.Method("POST", "/device_type", adminAuth(adminHandler.Middleware(deviceTypeHandler.Create))) // POST on /device_type.

	r.Route("/device", func(r chi.Router) {
		r.Method("POST", "/", accountAuth(deviceHandler.Create))                                         // POST on /device.
		r.Method("POST", "/activate", handlers.Handler(deviceHandler.Activate))                          // POST on /device/activate.
		r.Method("GET", "/{device_name}", accountAuth(deviceHandler.GetDeviceByName))                    // GET on /device/{device_name}.
		r.Method("GET", "/all", accountAuth(deviceHandler.GetDevicesByAccount))                          // GET on /device/all.
		r.Method("GET", "/{device_name}/measurements", accountAuth(deviceHandler.GetDeviceMeasurements)) // GET on /device/{device_name}/measurements.
		r.Method("GET", "/{device_name}/properties", accountAuth(deviceHandler.GetDeviceProperties))     // GET on /device/{device_name}/properties.
	})

	r.Method("POST", "/upload", deviceORaccountAuth(uploadHandler.Create)) // POST on /upload.

	r.Method("POST", "/data_source_list", adminAuth(dataSourceListHandler.Create)) // POST on /data_source_list
	r.Method("POST", "/data_source_type", adminAuth(dataSourceTypeHandler.Create)) // POST on /data_source_type

	r.Method("POST", "/energy_query_type", adminAuth(adminHandler.Middleware(energyQueryTypeHandler.Create))) // POST on /energy_query_type

	r.Route("/energy_query", func(r chi.Router) {
		r.Method("POST", "/", accountAuth(energyQueryHandler.Create))                                                    // POST on /energy_query.
		r.Method("GET", "/{energy_query_type}", accountAuth(energyQueryHandler.GetEnergyQueryByName))                    // GET on /energy_query/{energy_query_type}.
		r.Method("GET", "/all", accountAuth(energyQueryHandler.GetEnergyQueriesByAccount))                               // GET on /energy_query/all.
		r.Method("GET", "/{energy_query_type}/measurements", accountAuth(energyQueryHandler.GetEnergyQueryMeasurements)) // GET on /energy_query/{energy_query_type}/measurements.
		r.Method("GET", "/{energy_query_type}/properties", accountAuth(energyQueryHandler.GetEnergyQueryProperties))     // GET on /energy_query/{energy_query_type}/properties.
	})

	r.Method("GET", "/api_key/{api_name}", accountAuth(apiKeyHandler.GetAPIKey)) // GET on /api_key/{api_name}

	// Routes for the NeedForHeat Cockpit (docs/22). The reverse proxy only lets /admin through from the LAN/WireGuard.
	admin := func(h handlers.Handler) handlers.Handler { return adminAuth(adminHandler.Middleware(h)) }
	r.Route("/admin", func(r chi.Router) {
		r.Method("GET", "/overview", admin(cockpitHandler.Overview))
		r.Method("GET", "/campaigns/{campaign_id}/status", admin(cockpitHandler.CampaignStatus))
		r.Method("PATCH", "/campaigns/{campaign_id}", admin(cockpitHandler.UpdateCampaign))
		r.Method("POST", "/campaigns/{campaign_id}/accounts", admin(cockpitHandler.CreateAccounts))
		r.Method("POST", "/accounts/{account_id}/invitation", admin(cockpitHandler.NewInvitation))
	})

	setupSwaggerDocs(r, config.BaseURL)

	go setupRPCHandler(adminHandler, cloudFeedHandler)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second, // Protects against slowloris (gosec G112).
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	err = listenAndServe(ctx, server)
	if err != nil {
		return err
	}
	return nil
}

const (
	day                 = time.Hour * 24
	defaultDownloadTime = "04h00s"
)

type Configuration struct {
	DatabaseDSN       string
	BaseURL           string
	downloadStartTime time.Time
}

func getConfiguration() Configuration {
	dsn, ok := os.LookupEnv("NFH_DSN")
	if !ok {
		logrus.Fatal("NFH_DSN was not set")
	}

	baseURL, ok := os.LookupEnv("NFH_BASE_URL")
	if !ok {
		logrus.Fatal("NFH_BASE_URL was not set")
	}

	downloadTime, ok := os.LookupEnv("NFH_DOWNLOAD_TIME")
	if !ok {
		logrus.Warning("NFH_DOWNLOAD_TIME was not set. defaulting to", defaultDownloadTime)
		downloadTime = defaultDownloadTime
	}

	duration, err := time.ParseDuration(downloadTime)
	if err != nil {
		logrus.Fatal(err)
	}

	downloadStartTime := time.Now().Truncate(day)
	downloadStartTime = downloadStartTime.Add(duration)
	// If time is in the past, add 1 day.
	if downloadStartTime.Before(time.Now()) {
		downloadStartTime = downloadStartTime.Add(day)
	}

	return Configuration{
		DatabaseDSN:       dsn,
		BaseURL:           baseURL,
		downloadStartTime: downloadStartTime,
	}
}

func listenAndServe(ctx context.Context, server *http.Server) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		err := server.ListenAndServe()
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	})
	logrus.Infoln("listening on", server.Addr)

	g.Go(func() error {
		<-gCtx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		return server.Shutdown(shutdownCtx)
	})

	err := g.Wait()
	if err != http.ErrServerClosed {
		return err
	}
	return nil
}

func setupSwaggerDocs(r *chi.Mux, baseURL string) {
	swaggerUI, err := fs.Sub(swaggerdocs.StaticFiles, "swagger-ui")
	if err != nil {
		logrus.Fatal(err)
	}

	docsHandler, err := handlers.NewDocsHandler(swaggerdocs.StaticFiles, baseURL)
	if err != nil {
		logrus.Fatal(err)
	}

	r.Method("GET", "/openapi.yml", handlers.Handler(docsHandler.OpenAPISpec))                        // Serve openapi.yml
	r.Method("GET", "/docs/*", http.StripPrefix("/docs/", http.FileServer(http.FS(swaggerUI))))       // Serve static files.
	r.Method("GET", "/docs", handlers.Handler(docsHandler.RedirectDocs(http.StatusMovedPermanently))) // Redirect /docs to /docs/
	r.Method("GET", "/", handlers.Handler(docsHandler.RedirectDocs(http.StatusSeeOther)))             // Redirect / to /docs/
}

func setupRPCHandler(handlers ...any) {

	for _, handler := range handlers {
		rpc.Register(handler)
	}

	rpc.HandleHTTP()

	listener, err := net.Listen("tcp4", "127.0.0.1:8081")
	if err != nil {
		logrus.Fatal(err)
	}

	err = http.Serve(listener, nil)
	if err != nil {
		logrus.Fatal(err)
	}
}
