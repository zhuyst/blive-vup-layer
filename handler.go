package main

import (
	"blive-vup-layer/config"
	"blive-vup-layer/llm"
	"blive-vup-layer/tts"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/vtb-link/bianka/basic"
	"github.com/vtb-link/bianka/live"
	"github.com/vtb-link/bianka/proto"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func HandleImg(c *gin.Context) {
	imgUrl := c.Query("img_url")
	u, err := url.Parse(imgUrl)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"err": err.Error(),
		})
		return
	}
	if u.Host != "i0.hdslb.com" {
		c.JSON(http.StatusBadRequest, gin.H{
			"err": "invalid host",
		})
		return
	}

	referer := c.GetHeader("Referer")
	if referer != "" && strings.Contains(referer, "localhost") ||
		strings.Contains(referer, "blivechat.zhuyst.cc") ||
		strings.Contains(referer, "play-live.bilibili.com") {
		c.JSON(http.StatusForbidden, gin.H{
			"err": "blocked",
		})
		return
	}

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
}

type Handler struct {
	cfg        *config.Config
	liveClient *live.Client

	LLM *llm.LLM
	TTS *tts.TTS
}

func NewHandler(cfg *config.Config) (*Handler, error) {
	t, err := tts.NewTTS(cfg.AliyunTTS)
	if err != nil {
		return nil, fmt.Errorf("tts.NewTTS err: %w", err)
	}
	return &Handler{
		cfg:        cfg,
		liveClient: live.NewClient(live.NewConfig(cfg.BiliBili.AccessKey, cfg.BiliBili.SecretKey, cfg.BiliBili.AppId)),
		LLM:        llm.NewLLM(cfg.QianFan),
		TTS:        t,
	}, nil
}

type GiftWithTimer struct {
	Uname    string
	GiftName string
	GiftNum  int32
	Timer    *time.Timer
}

type LiveConfig struct {
	DisableLlm bool `json:"disable_llm"`
}

func (h *Handler) WebSocket(c *gin.Context) {
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
			h.liveClient.AppEnd(startResp.GameInfo.GameID)
		}
	}()

	isLiving := true
	livingCfg := LiveConfig{
		DisableLlm: false,
	}

	ttsQueue := tts.NewTTSQueue(h.TTS)
	defer ttsQueue.Close()
	ttsCh := ttsQueue.ListenResult()
	go func() {
		for r := range ttsCh {
			if err := r.Err; err != nil {
				conn.WriteResultError(ResultTypeTTS, CodeInternalError, err.Error())
				continue
			}
			conn.WriteResultOK(ResultTypeTTS, gin.H{
				"audio_file_path": r.Fname,
			})
		}
	}()
	pushTTS := func(params *tts.NewTaskParams, force bool) {
		if !isLiving && !force {
			return
		}
		if err := ttsQueue.Push(params); err != nil {
			conn.WriteResultError(ResultTypeTTS, CodeInternalError, err.Error())
		}
	}

	var historyMsgList []*llm.ChatMessage
	isLlmProcessing := false
	startLlmReply := func() {
		if !isLiving || livingCfg.DisableLlm {
			return
		}

		const expireDuration = 10 * time.Minute

		isLlmProcessing = true

		var msgs []*llm.ChatMessage
		for _, msg := range historyMsgList {
			if time.Since(msg.Timestamp) > expireDuration {
				continue
			}
			msgs = append(msgs, msg)
		}
		historyMsgList = msgs

		go func(msgs []*llm.ChatMessage) {
			defer func() {
				isLlmProcessing = false
			}()
			llmRes, err := h.LLM.ChatWithLLM(context.Background(), msgs)
			if err != nil {
				conn.WriteResultError(ResultTypeLLM, CodeInternalError, err.Error())
				log.Errorf("ChatWithLLM err: %v", err)
				return
			}
			conn.WriteResultOK(ResultTypeLLM, gin.H{
				"llm_result": llmRes,
			})
			pushTTS(&tts.NewTaskParams{
				Text: llmRes,
			}, false)
		}(msgs)
	}

	giftTimerMap := make(map[string]*GiftWithTimer)
	var giftTimerMapMutex sync.RWMutex

	init := func(code string) {
		if startResp != nil {
			conn.WriteResultError(ResultTypeRoom, http.StatusBadRequest, "connection already init")
			return
		}

		log.Infof("init code: %s", code)
		startResp, err = h.liveClient.AppStart(code)
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
					if err := h.liveClient.AppHeartbeat(startResp.GameInfo.GameID); err != nil {
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
				cmd, data, err := proto.AutomaticParsingMessageCommand(msg.Payload())
				if err != nil {
					log.Errorf("proto.AutomaticParsingMessageCommand err: %v", err)
					return err
				}

				// Switch cmd
				switch d := data.(type) {
				case *proto.CmdDanmuData:
					{
						if _, ok := danmuGiftMap[d.Msg]; ok {
							break
						}
						danmuData := &DanmuData{
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
						}
						conn.WriteResultOK(ResultTypeDanmu, danmuData)

						historyMsgList = append(historyMsgList, &llm.ChatMessage{
							User:      danmuData.Uname,
							Message:   danmuData.Msg,
							Timestamp: time.Now(),
						})

						pitchRate := 0
						if !livingCfg.DisableLlm {
							pitchRate = -100
						}
						pushTTS(&tts.NewTaskParams{
							Text:      fmt.Sprintf("%s说：%s", d.Uname, d.Msg),
							PitchRate: pitchRate,
						}, false)

						if isLlmProcessing {
							break
						}

						if (danmuData.FansMedalWearingStatus &&
							danmuData.FansMedalName == "巫女酱" &&
							danmuData.FansMedalLevel >= 10) || // 带10级粉丝牌
							danmuData.GuardLevel > 0 || // 舰长
							(danmuData.Uname == "巫女酱子" || danmuData.Uname == "青云-_-z") {
							startLlmReply()
						}

						break
					}
				case *proto.CmdSuperChatData:
					{
						scData := &SuperChatData{
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
						}
						conn.WriteResultOK(ResultTypeSuperChat, scData)
						historyMsgList = append(historyMsgList, &llm.ChatMessage{
							User:      scData.Uname,
							Message:   scData.Msg,
							Timestamp: time.Now(),
						})
						pushTTS(&tts.NewTaskParams{
							Text: fmt.Sprintf("谢谢%s酱的醒目留言：%s", d.Uname, d.Message),
						}, false)
						startLlmReply()
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

						const comboDuration = 4 * time.Second
						key := fmt.Sprintf("%s-%d", d.OpenID, d.GiftID)

						giftTimerMapMutex.RLock()
						gt, ok := giftTimerMap[key]
						giftTimerMapMutex.RUnlock()
						if ok {
							atomic.AddInt32(&gt.GiftNum, int32(d.GiftNum))
							gt.Timer.Reset(comboDuration)
							break
						}

						gt = &GiftWithTimer{
							Uname:    d.Uname,
							GiftNum:  int32(d.GiftNum),
							GiftName: d.GiftName,
							Timer:    time.NewTimer(comboDuration),
						}

						giftTimerMapMutex.Lock()
						giftTimerMap[key] = gt
						giftTimerMapMutex.Unlock()
						go func(gt *GiftWithTimer) {
							defer gt.Timer.Stop()
							<-gt.Timer.C

							giftTimerMapMutex.Lock()
							delete(giftTimerMap, key)
							giftTimerMapMutex.Unlock()

							giftNum := atomic.LoadInt32(&gt.GiftNum)
							pushTTS(&tts.NewTaskParams{
								Text: fmt.Sprintf("谢谢%s酱赠送的%d个%s 么么哒", gt.Uname, giftNum, gt.GiftName),
							}, false)
						}(gt)
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
							GuardLevel: d.GuardLevel,
							GuardNum:   d.GuardNum,
							GuardUnit:  d.GuardUnit,
							Timestamp:  d.Timestamp,
							MsgID:      d.MsgID,
						})
						guardName, ok := GuardLevelMap[d.GuardLevel]
						if !ok {
							guardName = "舰长"
						}
						pushTTS(&tts.NewTaskParams{
							Text: fmt.Sprintf("谢谢%s酱赠送的%d个%s%s，么么哒", d.UserInfo.Uname, d.GuardNum, d.GuardUnit, guardName),
						}, false)
						break
					}
				case map[string]interface{}:
					{
						switch cmd {
						case CmdLiveStart:
							{
								pushTTS(&tts.NewTaskParams{
									Text: "主人开始直播啦，弹幕姬启动！",
								}, true)
								isLiving = true
								break
							}
						case CmdLiveEnd:
							{
								pushTTS(&tts.NewTaskParams{
									Text: "主人直播结束啦，今天辛苦了！",
								}, true)
								isLiving = false
								break
							}
						case CmdLiveRoomEnter:
							{
								var dd CmdRoomEnterData
								if err := MapToStruct(d, &dd); err != nil {
									log.Errorf("MapToStruct err: %v", err)
									break
								}

								conn.WriteResultOK(ResultTypeEnterRoom, &RoomEnterData{
									UserData: UserData{
										OpenID: dd.OpenId,
										Uname:  dd.Uname,
										UFace:  dd.Uface,
									},
									Timestamp: dd.Timestamp,
								})

								//pushTTS(&tts.NewTaskParams{
								//	Text: fmt.Sprintf("欢迎%s酱进入直播间", dd.Uname),
								//}, false)

								break
							}
						default:
							{
								break
							}
						}
					}
				default:
					{
						break
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
				if !h.cfg.BiliBili.DisableValidateSign {
					signParams := live.H5SignatureParams{
						Timestamp: strconv.FormatInt(initData.Timestamp, 10),
						Code:      initData.Code,
						Mid:       strconv.FormatInt(initData.Mid, 10),
						Caller:    initData.Caller,

						CodeSign: initData.CodeSign,
					}
					if ok := signParams.ValidateSignature(h.cfg.BiliBili.SecretKey); !ok {
						conn.WriteResultError(ResultTypeRoom, CodeBadRequest, "invalid signature")
						return
					}
				}

				livingCfg = initData.Config
				conn.WriteResultOK(ResultTypeConfig, livingCfg)

				init(initData.Code)
				break
			}
		case RequestTypeConfig:
			{
				if req.Data == nil {
					conn.WriteResultError(ResultTypeConfig, CodeBadRequest, "data is null")
					return
				}
				var configData LiveConfig
				if err := json.Unmarshal(req.Data, &configData); err != nil {
					conn.WriteResultError(ResultTypeConfig, CodeBadRequest, err.Error())
					return
				}
				livingCfg = configData
				conn.WriteResultOK(ResultTypeConfig, livingCfg)
			}
		case RequestTypeHeartbeat:
			{
				conn.WriteResultOK(ResultTypeHeartbeat, nil)
				break
			}
		default:
			{
				conn.WriteResultError(ResultTypeRoom, CodeBadRequest, "unknown type")
				break
			}
		}
	}
}

func convertImgUrl(imgUrl string) string {
	return imgUrl
	//query := url.Values{}
	//query.Set("img_url", imgUrl)
	//return "/server/img?" + query.Encode()
}
