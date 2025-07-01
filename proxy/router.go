package proxy

import (
	"context"
	"fmt"
	"time"

	routing "github.com/qiangxue/fasthttp-routing"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

const AnyHTTPMethod = "<ANY>"

// Route defines a single HTTP route for the proxy server.
type Route struct {
	Path    string
	Method  string
	Handler func(c *routing.Context) error
}

type Routable interface {
	Routes() []Route
	Name() string
}

func New(config *Config, routable Routable) *Server {
	router := routing.New()

	for _, route := range routable.Routes() {
		path := fmt.Sprintf("/%s/%s", routable.Name(), route.Path)
		logrus.Infof("applying route '%s' of method '%s'", path, route.Method)

		if route.Method == AnyHTTPMethod {
			router.Any(path, route.Handler)
			continue
		}

		router.To(route.Method, path, route.Handler)
	}

	return &Server{
		server: &fasthttp.Server{
			Handler:     router.HandleRequest,
			Concurrency: config.ConcurrencyLimit,
		},
		config: config,
	}
}

type Server struct {
	config *Config
	server *fasthttp.Server
}

func (s *Server) Address() string {
	return fmt.Sprintf("%s:%s", s.config.Bindaddr, s.config.Port)
}

func (s *Server) Serve() error {
	addr := s.Address()
	logrus.Infof("starting proxy server on %s...", addr)
	if err := s.server.ListenAndServe(addr); err != nil {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}

// GracefulShutdown shuts down the server gracefully and logs the reason.
func (s *Server) GracefulShutdown(ctx context.Context, reason string) error {
	addr := s.Address()
	start := time.Now()
	if deadline, ok := ctx.Deadline(); ok {
		logrus.Infof("shutting down proxy server on %s: %s (deadline: %s)", addr, reason, deadline.Format(time.RFC3339))
	} else {
		logrus.Infof("shutting down proxy server on %s: %s", addr, reason)
	}
	var err error
	if shutdownWithCtx := shutdownWithContextFunc(s.server); shutdownWithCtx != nil {
		err = shutdownWithCtx(ctx)
	} else {
		err = s.server.Shutdown()
	}
	dur := time.Since(start)
	if err != nil {
		logrus.Errorf("error during shutdown after %s: %v", dur, err)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	logrus.Infof("proxy server shutdown complete in %s", dur)
	return nil
}

// shutdownWithContextFunc checks if fasthttp.Server has ShutdownWithContext and returns it if available.
func shutdownWithContextFunc(s *fasthttp.Server) func(ctx context.Context) error {
	type shutdownWithContexter interface {
		ShutdownWithContext(ctx context.Context) error
	}
	if swc, ok := interface{}(s).(shutdownWithContexter); ok {
		return swc.ShutdownWithContext
	}
	return nil
}
