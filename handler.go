package main

import (
	"blive-vup-layer/config"
	"blive-vup-layer/dao"
	"blive-vup-layer/llm"
	"blive-vup-layer/tts"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/golang-lru/v2/expirable"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/vtb-link/bianka/basic"
	"github.com/vtb-link/bianka/live"
	"github.com/vtb-link/bianka/proto"
	"golang.org/x/exp/slog"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	FansMedalName = "巫女酱" // 粉丝牌名称

	LlmReplyFansMedalLevel     = 10 // 可以触发大模型响应的最小粉丝牌等级
	RoomEnterTTSFansMedalLevel = 15 // 可以触发进入直播间TTS提示的最小粉丝牌等级

	MessageExpiration     = 15 * time.Minute // 历史消息过期时间
	GiftComboDuration     = 4 * time.Second  // 礼物连击时间，连击结束后会合并播放TTS
	LlmHistoryDuration    = 10 * time.Minute // 大模型使用历史弹幕去理解上下文的时间范围
	LastEnterUserDuration = 10 * time.Minute // 最后一个进入直播间用户将会播放TTS的等待时间

	DisableLlmByUserCountDuration = 1 * time.Minute // 统计间隔时间内用户数量，用于触发暂停大模型
	DisableLlmByUserCount         = 5               // 触发暂停大模型的用户数量

	LlmReplyLimitDuration = 5 * time.Minute // 大模型最大回复数量的统计时间
	LlmReplyLimitCount    = 10              // 大模型统计窗口内最大的回复数量

	ProbabilityLlmTriggerDuration    = 5 * time.Minute // 概率触发大模型回复的统计时间
	ProbabilityLlmTriggerLevel1      = 0.0             // 100%触发
	ProbabilityLlmTriggerLevel1Count = 0               // 统计人数为0
	ProbabilityLlmTriggerLevel2      = 0.3             // 70%触发
	ProbabilityLlmTriggerLevel2Count = 10              // 统计人数为[1, 10]
	ProbabilityLlmTriggerLevel3      = 0.7             // 30%触发
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
	Dao *dao.Dao

	slog *slog.Logger
}

func NewHandler(cfg *config.Config, logWriter io.Writer) (*Handler, error) {
	t, err := tts.NewTTS(cfg.AliyunTTS)
	if err != nil {
		return nil, fmt.Errorf("tts.NewTTS err: %w", err)
	}
	d, err := dao.NewDao(cfg.DbPath)
	if err != nil {
		return nil, fmt.Errorf("dao.NewDao err: %w", err)
	}
	return &Handler{
		cfg:        cfg,
		liveClient: live.NewClient(live.NewConfig(cfg.BiliBili.AccessKey, cfg.BiliBili.SecretKey, cfg.BiliBili.AppId)),
		LLM:        llm.NewLLM(cfg.QianFan),
		TTS:        t,
		Dao:        d,
		slog:       slog.New(slog.NewJSONHandler(logWriter, &slog.HandlerOptions{Level: slog.LevelInfo})),
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

type ChatMessage struct {
	OpenId    string
	User      string
	Message   string
	Timestamp time.Time
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

	var lastEnterUser *UserData = nil
	lastEnterUserTimer := time.NewTimer(LastEnterUserDuration)
	defer lastEnterUserTimer.Stop()

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
			lastEnterUserTimer.Reset(LastEnterUserDuration)
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

	go func() {
		for range lastEnterUserTimer.C {
			if lastEnterUser == nil {
				lastEnterUserTimer.Reset(LastEnterUserDuration)
				continue
			}
			pushTTS(&tts.NewTaskParams{
				Text: fmt.Sprintf("欢迎%s酱来到直播间", lastEnterUser.Uname),
			}, false)
			lastEnterUserTimer.Reset(LastEnterUserDuration)
		}
	}()

	historyMsgLru := expirable.NewLRU[string, *ChatMessage](512, nil, MessageExpiration)
	llmReplyLru := expirable.NewLRU[string, struct{}](LlmReplyLimitCount, nil, LlmReplyLimitDuration)
	probabilityLlmTriggerRandom := rand.New(rand.NewSource(time.Now().UnixNano()))

	isLlmProcessing := false
	startLlmReply := func(force bool) {
		if !isLiving || livingCfg.DisableLlm {
			return
		}

		var msgs []*ChatMessage
		userMap := map[string]struct{}{}
		probabilityLlmTriggerCounter := -1 // 当前尝试触发的用户不算，所以初始值为-1
		for _, msg := range historyMsgLru.Values() {
			if time.Since(msg.Timestamp) <= LlmHistoryDuration {
				msgs = append(msgs, msg)
			}
			if time.Since(msg.Timestamp) <= DisableLlmByUserCountDuration {
				userMap[msg.OpenId] = struct{}{}
			}
			if time.Since(msg.Timestamp) <= ProbabilityLlmTriggerDuration {
				probabilityLlmTriggerCounter++
			}
		}

		if !force {
			llmReplyLruLen := llmReplyLru.Len()
			if llmReplyLruLen >= LlmReplyLimitCount {
				log.Infof("disable llm by reply count: %d", llmReplyLruLen)
				return
			}

			if len(userMap) >= DisableLlmByUserCount {
				log.Infof("disable llm by user count: %d", len(userMap))
				return
			}

			currentMsg := msgs[len(msgs)-1]
			if IsRepeatedChar(currentMsg.Message) {
				log.Infof("disable llm by repeated msg: %s", currentMsg.Message)
				return
			}

			var probability float64
			if probabilityLlmTriggerCounter > ProbabilityLlmTriggerLevel2Count {
				probability = ProbabilityLlmTriggerLevel3
			} else if probabilityLlmTriggerCounter > ProbabilityLlmTriggerLevel1Count {
				probability = ProbabilityLlmTriggerLevel2
			} else {
				probability = ProbabilityLlmTriggerLevel1
			}

			r := probabilityLlmTriggerRandom.Float64()
			fmt.Printf("r: %.2f, probability: %.2f\n", r, probability)
			if r <= probability {
				log.Infof("disable llm by probability: %.2f, counter: %d, compare: %.2f", r, probabilityLlmTriggerCounter, probability)
				return
			}
		}

		isLlmProcessing = true
		go func(msgs []*ChatMessage) {
			defer func() {
				isLlmProcessing = false
			}()

			llmMsgs := make([]*llm.ChatMessage, len(msgs))
			for i, msg := range msgs {
				llmMsgs[i] = &llm.ChatMessage{
					User:    msg.User,
					Message: msg.Message,
				}
			}
			llmRes, err := h.LLM.ChatWithLLM(context.Background(), llmMsgs)
			if err != nil {
				conn.WriteResultError(ResultTypeLLM, CodeInternalError, err.Error())
				log.Errorf("ChatWithLLM err: %v", err)
				return
			}
			conn.WriteResultOK(ResultTypeLLM, gin.H{
				"llm_result": llmRes,
			})
			llmReplyLru.Add(uuid.NewV4().String(), struct{}{})
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
				_, data, err := proto.AutomaticParsingMessageCommand(msg.Payload())
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
						u := UserData{
							OpenID:                 d.OpenID,
							Uname:                  d.Uname,
							UFace:                  convertImgUrl(d.UFace),
							FansMedalLevel:         d.FansMedalLevel,
							FansMedalName:          d.FansMedalName,
							FansMedalWearingStatus: d.FansMedalWearingStatus,
							GuardLevel:             d.GuardLevel,
						}
						danmuData := &DanmuData{
							UserData:    u,
							Msg:         d.Msg,
							MsgID:       d.MsgID,
							Timestamp:   d.Timestamp,
							EmojiImgUrl: d.EmojiImgUrl,
							DmType:      d.DmType,
						}
						conn.WriteResultOK(ResultTypeDanmu, danmuData)

						go h.setUser(u)

						historyMsgLru.Add(d.MsgID, &ChatMessage{
							OpenId:    danmuData.OpenID,
							User:      danmuData.Uname,
							Message:   danmuData.Msg,
							Timestamp: time.Now(),
						})

						pitchRate := 0
						//if !livingCfg.DisableLlm {
						//	pitchRate = -100
						//}
						pushTTS(&tts.NewTaskParams{
							Text:      fmt.Sprintf("%s说：%s", d.Uname, d.Msg),
							PitchRate: pitchRate,
						}, false)

						if isLlmProcessing {
							break
						}

						if (danmuData.FansMedalWearingStatus &&
							danmuData.FansMedalName == FansMedalName &&
							danmuData.FansMedalLevel >= LlmReplyFansMedalLevel) || // 带10级粉丝牌
							danmuData.GuardLevel > 0 || // 舰长
							(danmuData.Uname == "巫女酱子" || danmuData.Uname == "青云-_-z") {
							startLlmReply(false)
						}

						break
					}
				case *proto.CmdSuperChatData:
					{
						u := UserData{
							OpenID:                 d.OpenID,
							Uname:                  d.Uname,
							UFace:                  convertImgUrl(d.Uface),
							FansMedalLevel:         d.FansMedalLevel,
							FansMedalName:          d.FansMedalName,
							FansMedalWearingStatus: d.FansMedalWearingStatus,
							GuardLevel:             d.GuardLevel,
						}
						scData := &SuperChatData{
							UserData:  u,
							Msg:       d.Message,
							MsgID:     d.MsgID,
							MessageID: d.MessageID,
							Rmb:       float64(d.Rmb),
							Timestamp: d.Timestamp,
							StartTime: d.StartTime,
							EndTime:   d.EndTime,
						}
						conn.WriteResultOK(ResultTypeSuperChat, scData)

						go h.setUser(u)

						historyMsgLru.Add(d.MsgID, &ChatMessage{
							OpenId:    scData.OpenID,
							User:      scData.Uname,
							Message:   scData.Msg,
							Timestamp: time.Now(),
						})
						pushTTS(&tts.NewTaskParams{
							Text: fmt.Sprintf("谢谢%s酱的醒目留言：%s", d.Uname, d.Message),
						}, false)
						startLlmReply(true)
						break
					}
				case *proto.CmdSendGiftData:
					{
						u := UserData{
							OpenID:                 d.OpenID,
							Uname:                  d.Uname,
							UFace:                  convertImgUrl(d.Uface),
							FansMedalLevel:         d.FansMedalLevel,
							FansMedalName:          d.FansMedalName,
							FansMedalWearingStatus: d.FansMedalWearingStatus,
							GuardLevel:             d.GuardLevel,
						}
						conn.WriteResultOK(ResultTypeGift, &GiftData{
							UserData:  u,
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

						go h.setUser(u)

						key := fmt.Sprintf("%s-%d", d.OpenID, d.GiftID)

						giftTimerMapMutex.RLock()
						gt, ok := giftTimerMap[key]
						giftTimerMapMutex.RUnlock()
						if ok {
							atomic.AddInt32(&gt.GiftNum, int32(d.GiftNum))
							gt.Timer.Reset(GiftComboDuration)
							break
						}

						gt = &GiftWithTimer{
							Uname:    d.Uname,
							GiftNum:  int32(d.GiftNum),
							GiftName: d.GiftName,
							Timer:    time.NewTimer(GiftComboDuration),
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
						u := UserData{
							OpenID:                 d.UserInfo.OpenID,
							Uname:                  d.UserInfo.Uname,
							UFace:                  convertImgUrl(d.UserInfo.Uface),
							FansMedalLevel:         d.FansMedalLevel,
							FansMedalName:          d.FansMedalName,
							FansMedalWearingStatus: d.FansMedalWearingStatus,
							GuardLevel:             d.GuardLevel,
						}
						conn.WriteResultOK(ResultTypeGuard, &GuardData{
							UserData:   u,
							GuardLevel: d.GuardLevel,
							GuardNum:   d.GuardNum,
							GuardUnit:  d.GuardUnit,
							Timestamp:  d.Timestamp,
							MsgID:      d.MsgID,
						})
						go h.setUser(u)
						guardName := getGuardLevelName(d.GuardLevel)
						pushTTS(&tts.NewTaskParams{
							Text: fmt.Sprintf("谢谢%s酱赠送的%d个%s%s，么么哒", d.UserInfo.Uname, d.GuardNum, d.GuardUnit, guardName),
						}, false)
						break
					}
				case *proto.CmdLiveStartData:
					{
						pushTTS(&tts.NewTaskParams{
							Text: "主人开始直播啦，弹幕姬启动！",
						}, true)
						isLiving = true
						break
					}
				case *proto.CmdLiveEndData:
					{
						pushTTS(&tts.NewTaskParams{
							Text: "主人直播结束啦，今天辛苦了！",
						}, true)
						isLiving = false
						break
					}
				case *proto.CmdLiveRoomEnterData:
					{
						u := UserData{
							OpenID: d.OpenID,
							Uname:  d.Uname,
							UFace:  d.Uface,
						}
						conn.WriteResultOK(ResultTypeEnterRoom, &RoomEnterData{
							UserData:  u,
							Timestamp: d.Timestamp,
						})

						lastEnterUser = &u

						go func(openId string) {
							u, err := h.Dao.GetUser(context.Background(), openId)
							if err != nil {
								log.Errorf("GetUser open_id: %s err: %v", openId, err)
								return
							}

							if u == nil {
								return
							}

							if (u.FansMedalWearingStatus && u.FansMedalLevel >= RoomEnterTTSFansMedalLevel) ||
								u.GuardLevel > 0 {

								name := d.Uname
								if u.GuardLevel > 0 {
									guardName := getGuardLevelName(u.GuardLevel)
									name = guardName + name
								}

								pushTTS(&tts.NewTaskParams{
									Text: fmt.Sprintf("欢迎%s酱来到直播间", name),
								}, false)
							}
						}(d.OpenID)

						break
					}
				default:
					{
						break
					}
				}

				return nil
			},
		}

		wcs, err = basic.StartWebsocket(
			startResp,
			dispatcherHandleMap,
			onCloseHandle,
			h.slog,
		)
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

func (h *Handler) setUser(userData UserData) {
	err := h.Dao.CreateOrUpdateUser(context.Background(), &dao.User{
		OpenID:                 userData.OpenID,
		FansMedalWearingStatus: userData.FansMedalWearingStatus,
		FansMedalLevel:         userData.FansMedalLevel,
		GuardLevel:             userData.GuardLevel,
	})
	if err != nil {
		log.Errorf("CreateOrUpdateUser open_id: %s, err: %v", userData.OpenID, err)
	}
}

func getGuardLevelName(guardLevel int) string {
	guardName, ok := GuardLevelMap[guardLevel]
	if !ok {
		guardName = "舰长"
	}
	return guardName
}

func convertImgUrl(imgUrl string) string {
	return imgUrl
	//query := url.Values{}
	//query.Set("img_url", imgUrl)
	//return "/server/img?" + query.Encode()
}
