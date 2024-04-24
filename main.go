package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/vtb-link/bianka/basic"
	"github.com/vtb-link/bianka/live"
	"github.com/vtb-link/bianka/proto"
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

	liveClient := live.NewClient(rCfg)

	gin.SetMode(gin.ReleaseMode)
	g := gin.New()
	g.Use(gin.Recovery())

	g.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	g.GET("/server/img", func(c *gin.Context) {
		imgUrl := c.Query("img_url")
		resp, err := http.Get(imgUrl)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"err": err.Error(),
			})
			return
		}
		defer resp.Body.Close()

		contentType := resp.Header.Get("Content-Type")
		c.DataFromReader(resp.StatusCode, resp.ContentLength, contentType, resp.Body, nil)
	})
	g.GET("/server/ws", func(c *gin.Context) {
		conn, err := NewWebSocketConn(c)
		if err != nil {
			log.Errorf("NewWebSocketConn err: %v", err)
			return
		}
		defer conn.Close()

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		var (
			startResp *live.AppStartResponse
			tk        *time.Ticker
			wcs       *basic.WsClient
		)
		defer func() {
			if wcs != nil {
				wcs.Close()
			}
			if tk != nil {
				tk.Stop()
			}
			if startResp != nil {
				liveClient.AppEnd(startResp.GameInfo.GameID)
			}
		}()
		init := func(code string) {
			if startResp != nil {
				conn.WriteResultError(ResultTypeRoom, http.StatusBadRequest, "connection already init")
				return
			}

			log.Infof("init code: %s", code)
			startResp, err = liveClient.AppStart(code)
			if err != nil {
				conn.WriteResultError(ResultTypeRoom, http.StatusInternalServerError, err.Error())
				return
			}

			tk = time.NewTicker(time.Second * 20)
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-tk.C:
						// 心跳
						if err := liveClient.AppHeartbeat(startResp.GameInfo.GameID); err != nil {
							log.Errorf("Heartbeat fail, err: %v", err)
							cancel()
							conn.Close()
							return
						}
					}
				}
			}()

			// close 事件处理
			onCloseHandle := func(wcs *basic.WsClient, startResp basic.StartResp, closeType int) {
				// 注册关闭回调
				log.Infof("WebsocketClient onClose, startResp: %v", startResp)

				// 注意检查关闭类型, 避免无限重连
				if closeType == basic.CloseActively || closeType == basic.CloseReceivedShutdownMessage || closeType == basic.CloseAuthFailed {
					log.Infof("WebsocketClient exit")
					return
				}

				// 对于可能的情况下重新连接
				// 注意: 在某些场景下 startResp 会变化, 需要重新获取
				// 此外, 一但 AppHeartbeat 失败, 会导致 startResp.GameInfo.GameID 变化, 需要重新获取
				err := wcs.Reconnection(startResp)
				if err != nil {
					log.Errorf("Reconnection fail, err: %v", err)
					conn.WriteResultError(ResultTypeRoom, CodeInternalError, err.Error())
					cancel()
					conn.Close()
					return
				}
			}

			// 消息处理 Handle
			dispatcherHandleMap := basic.DispatcherHandleMap{
				proto.OperationMessage: func(_ *basic.WsClient, msg *proto.Message) error {
					// 单条消息raw
					log.Infof(string(msg.Payload()))

					// 自动解析
					_, data, err := proto.AutomaticParsingMessageCommand(msg.Payload())
					if err != nil {
						log.Errorf("proto.AutomaticParsingMessageCommand err: %v", err)
						return err
					}

					// Switch cmd
					switch d := data.(type) {
					case *proto.CmdDanmuData:
						{
							conn.WriteResultOK(ResultTypeDanmu, &DanmuData{
								UserData: UserData{
									OpenID:                 d.OpenID,
									Uname:                  d.Uname,
									UFace:                  convertImgUrl(d.UFace),
									FansMedalLevel:         d.FansMedalLevel,
									FansMedalName:          d.FansMedalName,
									FansMedalWearingStatus: d.FansMedalWearingStatus,
									GuardLevel:             d.GuardLevel,
								},
								Msg:         d.Msg,
								MsgID:       d.MsgID,
								Timestamp:   d.Timestamp,
								EmojiImgUrl: d.EmojiImgUrl,
								DmType:      d.DmType,
							})
							break
						}
					case *proto.CmdSuperChatData:
						{
							conn.WriteResultOK(ResultTypeSuperChat, &SuperChatData{
								UserData: UserData{
									OpenID:                 d.OpenID,
									Uname:                  d.Uname,
									UFace:                  convertImgUrl(d.Uface),
									FansMedalLevel:         d.FansMedalLevel,
									FansMedalName:          d.FansMedalName,
									FansMedalWearingStatus: d.FansMedalWearingStatus,
									GuardLevel:             d.GuardLevel,
								},
								Msg:       d.Message,
								MsgID:     d.MsgID,
								MessageID: d.MessageID,
								Rmb:       float64(d.Rmb),
								Timestamp: d.Timestamp,
								StartTime: d.StartTime,
								EndTime:   d.EndTime,
							})
							break
						}
					case *proto.CmdSendGiftData:
						{
							conn.WriteResultOK(ResultTypeGift, &GiftData{
								UserData: UserData{
									OpenID:                 d.OpenID,
									Uname:                  d.Uname,
									UFace:                  convertImgUrl(d.Uface),
									FansMedalLevel:         d.FansMedalLevel,
									FansMedalName:          d.FansMedalName,
									FansMedalWearingStatus: d.FansMedalWearingStatus,
									GuardLevel:             d.GuardLevel,
								},
								GiftID:    d.GiftID,
								GiftName:  d.GiftName,
								GiftNum:   d.GiftNum,
								Rmb:       float64(d.Price) / 1000,
								Paid:      d.Paid,
								Timestamp: d.Timestamp,
								MsgID:     d.MsgID,
								GiftIcon:  d.GiftIcon,
								ComboGift: d.ComboGift,
								ComboInfo: &GiftDataComboInfo{
									ComboBaseNum: d.ComboInfo.ComboBaseNum,
									ComboCount:   d.ComboInfo.ComboCount,
									ComboID:      d.ComboInfo.ComboID,
									ComboTimeout: d.ComboInfo.ComboTimeout,
								},
							})
							break
						}
					case *proto.CmdGuardData:
						{
							conn.WriteResultOK(ResultTypeGuard, &GuardData{
								UserData: UserData{
									OpenID:                 d.UserInfo.OpenID,
									Uname:                  d.UserInfo.Uname,
									UFace:                  convertImgUrl(d.UserInfo.Uface),
									FansMedalLevel:         d.FansMedalLevel,
									FansMedalName:          d.FansMedalName,
									FansMedalWearingStatus: d.FansMedalWearingStatus,
									GuardLevel:             d.GuardLevel,
								},
								GuardNum:  d.GuardNum,
								GuardUnit: d.GuardUnit,
								Timestamp: d.Timestamp,
								MsgID:     d.MsgID,
							})
							break
						}
					default:
						{

						}
					}

					return nil
				},
			}

			wcs, err = basic.StartWebsocket(startResp, dispatcherHandleMap, onCloseHandle, basic.DefaultLoggerGenerator())
			if err != nil {
				log.Errorf("basic.StartWebsocket err: %v", err)
				conn.WriteResultError(ResultTypeRoom, CodeInternalError, err.Error())
				return
			}

			log.Infof("room_info: %v", startResp.AnchorInfo)
			conn.WriteResultOK(ResultTypeRoom, &RoomData{
				RoomID: startResp.AnchorInfo.RoomID,
				Uname:  startResp.AnchorInfo.Uname,
				UFace:  convertImgUrl(startResp.AnchorInfo.UFace),
			})
		}

		for {
			var req WebSocketRequest
			if err := conn.ReadJSON(&req); err != nil {
				if !errors.Is(err, io.EOF) {
					conn.WriteResultError(ResultTypeRoom, CodeBadRequest, err.Error())
				}
				return
			}

			switch req.Type {
			case RequestTypeInit:
				{
					if req.Data == nil {
						conn.WriteResultError(ResultTypeRoom, CodeBadRequest, "data is null")
						return
					}
					var initData InitRequestData
					if err := json.Unmarshal(req.Data, &initData); err != nil {
						conn.WriteResultError(ResultTypeRoom, CodeBadRequest, err.Error())
						return
					}
					init(initData.Code)
					break
				}
			case RequestTypeHeartbeat:
				{
					conn.WriteResultOK(ResultTypeHeartbeat, nil)
					break
				}
			default:
				{
					conn.WriteResultError(ResultTypeRoom, CodeBadRequest, "unknown type")
				}
			}
		}
	})

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
