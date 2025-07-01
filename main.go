package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	logrusStack "github.com/Gurpartap/logrus-stack"
	"github.com/jaffee/commandeer"
	"github.com/sirupsen/logrus"
	"github.com/stash86/kucoin-proxy/proxy"
	"github.com/stash86/kucoin-proxy/proxy/kucoin"
	"github.com/stash86/kucoin-proxy/store"
	"github.com/valyala/fasthttp"
)

var (
	//go:embed disclaimer.txt
	disclaimer string

	version = "dev"
)

type app struct {
	Verbose         int           `help:"verbose level: 0 - info, 1 - debug, 2 - trace"`
	CacheSize       int           `help:"amount of candles to cache"`
	TTLCacheTimeout time.Duration `help:"ttl of blobs of cached data"`
	ClientTimeout   time.Duration `help:"client timeout"`

	ProxyConfig  proxy.Config  `flag:"!embed"`
	KucoinConfig kucoin.Config `flag:"!embed"`
}

func newApp() *app {
	return &app{
		Verbose:         0,
		CacheSize:       1000,
		TTLCacheTimeout: time.Minute * 10,
		ClientTimeout:   time.Second * 15,
		KucoinConfig: kucoin.Config{
			KucoinTopicsPerWs: 200,
			KucoinApiURL:      "https://openapi-v2.kucoin.com",
		},
		ProxyConfig: proxy.Config{
			Port:             "8080",
			Bindaddr:         "0.0.0.0",
			ConcurrencyLimit: fasthttp.DefaultConcurrency,
		},
	}
}

func (app *app) configure() {
	switch app.Verbose {
	case 0:
		logrus.SetLevel(logrus.InfoLevel)
	case 1:
		logrus.SetLevel(logrus.DebugLevel)
	case 2:
		logrus.SetLevel(logrus.TraceLevel)
	}
}

func (app *app) Run() error {
	logrus.SetOutput(os.Stdout)
	logrus.AddHook(logrusStack.StandardHook())

	fmt.Println(disclaimer)

	logrus.Infof("starting kucoin-proxy: version - '%s'... ", version)

	if app.Verbose > 2 {
		return fmt.Errorf("wrong verbose level '%d'", app.Verbose)
	}

	app.configure()

	logrus.Infof("Validating proxy config: %+v", app.ProxyConfig)
	if err := app.ProxyConfig.Validate(); err != nil {
		logrus.Errorf("Proxy config validation failed: %v", err)
		return err
	}

	logrus.Infof("Validating kucoin config: %+v", app.KucoinConfig)
	if err := app.KucoinConfig.Validate(); err != nil {
		logrus.Errorf("Kucoin config validation failed: %v", err)
		return err
	}

	logrus.Infof("Initializing HTTP client with timeout: %s", app.ClientTimeout)
	client := &proxy.Client{
		Client: fasthttp.Client{
			ReadTimeout:  app.ClientTimeout,
			WriteTimeout: app.ClientTimeout,
		},
	}

	logrus.Infof("Initializing proxy server with cache size: %d, TTL cache timeout: %s", app.CacheSize, app.TTLCacheTimeout)
	proxySrv := proxy.New(&app.ProxyConfig,
		kucoin.New(
			store.NewStore(app.CacheSize),
			store.NewTTLCache(app.TTLCacheTimeout),
			client,
			&app.KucoinConfig,
		),
	)

	// Set up signal handling for graceful shutdown
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sig := <-shutdownCh
		logrus.Warnf("Received shutdown signal: %s", sig)
		err := proxySrv.GracefulShutdown(ctx, fmt.Sprintf("received signal: %s", sig))
		if err != nil {
			logrus.Errorf("Graceful shutdown error: %v", err)
		} else {
			logrus.Info("Graceful shutdown completed successfully")
		}
		os.Exit(0)
	}()

	logrus.Info("Proxy server starting...")
	err := proxySrv.Serve()
	if err != nil {
		logrus.Errorf("Proxy server error: %v", err)
		return fmt.Errorf("proxy server error: %w", err)
	}
	logrus.Info("Proxy server stopped.")

	return nil
}

func main() {
	app := newApp()

	if err := commandeer.Run(app); err != nil {
		logrus.Fatal(err)
	}
}
