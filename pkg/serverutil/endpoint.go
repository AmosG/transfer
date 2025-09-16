package serverutil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/transferia/transferia/internal/logger"
	"go.ytsaurus.tech/library/go/core/log"
)

func PingFunc(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "Application/json")
	res, _ := json.Marshal(map[string]interface{}{"ping": "pong", "ts": time.Now()})
	if _, err := w.Write(res); err != nil {
		logger.Log.Error("unable to write", log.Error(err))
		return
	}
}

func RunHealthCheck() {
	RunHealthCheckOnPort(80)
}

func RunHealthCheckOnPort(port int) {
	rootMux := http.NewServeMux()
	rootMux.HandleFunc("/ping", PingFunc)
	logger.Log.Infof("healthcheck is upraising on port 80")
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), rootMux); err != nil { // it must be on 80 port - bcs of dataplane instance-group
		logger.Log.Error("failed to serve health check", log.Error(err))
	}
}

func RunPprof(port int) {
	logger.Log.Infof("init pprof on port %d", port)
	server, err := NewServer("tcp", fmt.Sprintf(":%d", port), logger.Log)
	if err != nil {
		logger.Log.Info(fmt.Sprintf("failed to serve pprof on %d, try random port", port), log.Error(err))
		server, err = NewServer("tcp", ":0", logger.Log)
		if err != nil {
			logger.Log.Error("failed to add listener for pprof", log.Error(err))
			return
		}
		logger.Log.Infof("pprof listen on: %v", server.Addr().String())
	}
	if err := server.Serve(); err != nil {
		logger.Log.Error("failed to serve pprof", log.Error(err))
	}
}
