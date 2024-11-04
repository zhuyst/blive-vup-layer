package main

import (
	"blive-vup-layer/config"
	"errors"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	cachecontrol "go.eigsys.de/gin-cachecontrol/v2"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{})

	if err := os.MkdirAll("logs", os.ModePerm); err != nil {
		log.Fatalf("failed to create logs dir: %v", err)
		return
	}
	logFile, err := os.OpenFile(fmt.Sprintf("logs/%s.txt", time.Now().Format("2006-01-02-15-04-05")), os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
	if err != nil {
		log.Fatalf("failed to create log file: %v", err)
		return
	}
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))

	configFilePath := flag.String("config", "./etc/config-dev.toml", "config file path")
	flag.Parse()
	cfg, err := config.ParseConfig(*configFilePath)
	if err != nil {
		log.Fatalf("failed to parse config file: %v", err)
		return
	}

	h := NewHandler(cfg)

	gin.SetMode(gin.ReleaseMode)
	g := gin.New()
	g.Use(gin.Recovery())

	staticRouter := g.Group("/")
	staticRouter.Use(func(c *gin.Context) {
		c.Header("X-Frame-Options", "ALLOW-FROM https://play-live.bilibili.com/")
	})
	assetsRouter := staticRouter.Group("/")
	assetsRouter.Use(cachecontrol.New(cachecontrol.CacheAssetsForeverPreset))

	g.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	g.GET("/server/ws", h.WebSocket)
	//assetsRouter.GET("/server/img", HandleImg)
	staticRouter.StaticFile("/", "./frontend/dist/index.html")

	assetsRouter.StaticFile("/favicon.ico", "./frontend/dist/favicon.ico")
	assetsRouter.Static("/assets/", "./frontend/dist/assets")

	const addr = ":8080"
	server := http.Server{
		Addr:    addr,
		Handler: g,
	}
	log.Infof("server started at %s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server.ListenAndServe err: %v", err)
		return
	}

	// 退出
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	<-stopCh
	server.Close()
	log.Infof("server shutdown")
}
