package main

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/vtb-link/bianka/live"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
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

	ak := os.Getenv("ACCESS_KEY")
	sk := os.Getenv("SECRET_KEY")
	appIdStr := os.Getenv("APP_ID")
	if ak == "" || sk == "" || appIdStr == "" {
		log.Fatalf("Environment variables ACCESS_KEY, SECRET_KEY, APP_ID must be set")
		return
	}
	appId, err := strconv.ParseInt(appIdStr, 10, 64)
	if err != nil {
		log.Fatalf("failed to convert APP_ID to int: %v", err)
		return
	}

	rCfg := live.NewConfig(
		ak,
		sk,
		appId, // 应用id
	)

	h := NewHandler(rCfg)

	gin.SetMode(gin.ReleaseMode)
	g := gin.New()
	g.Use(gin.Recovery())

	g.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	g.GET("/server/img", HandleImg)
	g.GET("/server/ws", h.WebSocket)

	staticRouter := g.Group("/")
	staticRouter.Use(func(c *gin.Context) {
		c.Header("X-Frame-Options", "ALLOW-FROM https://play-live.bilibili.com/")
	})
	staticRouter.StaticFile("/", "./frontend/dist/index.html")
	staticRouter.StaticFile("/favicon.ico", "./frontend/dist/favicon.ico")
	staticRouter.Static("/assets/", "./frontend/dist/assets")

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

func convertImgUrl(imgUrl string) string {
	query := url.Values{}
	query.Set("img_url", imgUrl)
	return "/server/img?" + query.Encode()
}
