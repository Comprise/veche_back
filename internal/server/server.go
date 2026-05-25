package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	httpSrv *http.Server
}

func New(addr string, handler http.Handler) *Server {
	return &Server{
		httpSrv: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
	}
}

func (s *Server) Run() error {
	go func() {
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.httpSrv.Shutdown(ctx)
}
