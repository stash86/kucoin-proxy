package proxy_test

import (
	"context"
	"testing"
	"time"

	routing "github.com/qiangxue/fasthttp-routing"
	"github.com/sirupsen/logrus"
	"github.com/stash86/kucoin-proxy/proxy"
)

type dummyRoutable struct{}

func (d dummyRoutable) Routes() []proxy.Route {
	return []proxy.Route{
		{
			Path:    "test",
			Method:  "GET",
			Handler: func(c *routing.Context) error { return nil }, // Handler signature matches expected type
		},
	}
}

func (d dummyRoutable) Name() string { return "dummy" }

func TestServerAddress(t *testing.T) {
	cfg := &proxy.Config{Port: "1234", Bindaddr: "127.0.0.1", ConcurrencyLimit: 1}
	srv := proxy.New(cfg, dummyRoutable{})
	want := "127.0.0.1:1234"
	if got := srv.Address(); got != want {
		t.Errorf("Address() = %q, want %q", got, want)
	}
}

func TestGracefulShutdown(t *testing.T) {
	cfg := &proxy.Config{Port: "1234", Bindaddr: "127.0.0.1", ConcurrencyLimit: 1}
	srv := proxy.New(cfg, dummyRoutable{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := srv.GracefulShutdown(ctx, "test shutdown")
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("GracefulShutdown() error = %v", err)
	}
}

func TestServeReturnsError(t *testing.T) {
	cfg := &proxy.Config{Port: "0", Bindaddr: "invalid", ConcurrencyLimit: 1}
	srv := proxy.New(cfg, dummyRoutable{})
	logrus.SetLevel(logrus.PanicLevel) // Suppress log output
	err := srv.Serve()
	if err == nil {
		t.Error("Serve() should return error for invalid address")
	}
}
