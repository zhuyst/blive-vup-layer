package main

const (
	RequestTypeInit      = "init"
	RequestTypeHeartbeat = "heartbeat"
	RequestTypeConfig    = "config"

	ResultTypeHeartbeat = "heartbeat"
	ResultTypeRoom      = "room"
	ResultTypeConfig    = "config"
	ResultTypeDanmu     = "danmu"
	ResultTypeSuperChat = "superchat"
	ResultTypeGift      = "gift"
	ResultTypeGuard     = "guard"
	ResultTypeEnterRoom = "enter_room"

	ResultTypeTTS = "tts"
	ResultTypeLLM = "llm"

	CmdLiveRoomEnter = "LIVE_OPEN_PLATFORM_LIVE_ROOM_ENTER"
	CmdLiveStart     = "LIVE_OPEN_PLATFORM_LIVE_START"
	CmdLiveEnd       = "LIVE_OPEN_PLATFORM_LIVE_END"
)

var danmuGiftList = []string{
	// 红包
	"老板大气！点点红包抽礼物",
	"老板大气！点点红包抽礼物！",
	"点点红包，关注主播抽礼物～",
	"喜欢主播加关注，点点红包抽礼物",
	"红包抽礼物，开启今日好运！",
	"中奖喷雾！中奖喷雾！",
	// 节奏风暴
	"前方高能预警，注意这不是演习",
	"我从未见过如此厚颜无耻之人",
	"那万一赢了呢",
	"你们城里人真会玩",
	"左舷弹幕太薄了",
	"要优雅，不要污",
	"我选择狗带",
	"可爱即正义~~",
	"糟了，是心动的感觉！",
	"这个直播间已经被我们承包了！",
	"妈妈问我为什么跪着看直播 w(ﾟДﾟ)w",
	"你们对力量一无所知~(￣▽￣)~",
}

var danmuGiftMap map[string]struct{}

func init() {
	danmuGiftMap = make(map[string]struct{}, len(danmuGiftList))
	for _, d := range danmuGiftList {
		danmuGiftMap[d] = struct{}{}
	}
}

type InitRequestData struct {
	Code      string     `json:"code" binding:"required"`
	Timestamp int64      `json:"timestamp"`
	RoomId    int64      `json:"room_id"`
	Mid       int64      `json:"mid"`
	Caller    string     `json:"caller"`
	CodeSign  string     `json:"code_sign"`
	Config    LiveConfig `json:"config"`
}

type RoomData struct {
	RoomID int    `json:"room_id"`
	Uname  string `json:"uname"`
	UFace  string `json:"uface"`
}

type UserData struct {
	OpenID                 string `json:"open_id"`
	Uname                  string `json:"uname"`
	UFace                  string `json:"uface"`
	FansMedalLevel         int    `json:"fans_medal_level"`
	FansMedalName          string `json:"fans_medal_name"`
	FansMedalWearingStatus bool   `json:"fans_medal_wearing_status"`
	GuardLevel             int    `json:"guard_level"`
}

type DanmuData struct {
	UserData
	Msg         string `json:"msg"`
	MsgID       string `json:"msg_id"`
	Timestamp   int    `json:"timestamp"`
	EmojiImgUrl string `json:"emoji_img_url"`
	DmType      int    `json:"dm_type"`
}

type SuperChatData struct {
	UserData
	Msg       string  `json:"msg"`
	MsgID     string  `json:"msg_id"`
	MessageID int     `json:"message_id"`
	Rmb       float64 `json:"rmb"`
	Timestamp int     `json:"timestamp"`
	StartTime int     `json:"start_time"`
	EndTime   int     `json:"end_time"`
}

type GiftData struct {
	UserData
	GiftID    int                `json:"gift_id"`
	GiftName  string             `json:"gift_name"`
	GiftNum   int                `json:"gift_num"`
	Rmb       float64            `json:"rmb"`
	Paid      bool               `json:"paid"`
	Timestamp int                `json:"timestamp"`
	MsgID     string             `json:"msg_id"`
	GiftIcon  string             `json:"gift_icon"`
	ComboGift bool               `json:"combo_gift"`
	ComboInfo *GiftDataComboInfo `json:"combo_info"`
}

type GiftDataComboInfo struct {
	ComboBaseNum int    `json:"combo_base_num"`
	ComboCount   int    `json:"combo_count"`
	ComboID      string `json:"combo_id"`
	ComboTimeout int    `json:"combo_timeout"`
}

type GuardData struct {
	UserData
	GuardLevel int    `json:"guard_level"`
	GuardNum   int    `json:"guard_num"`
	GuardUnit  string `json:"guard_unit"`
	MsgID      string `json:"msg_id"`
	Timestamp  int    `json:"timestamp"`
}

type CmdRoomEnterData struct {
	RoomId    int64  `json:"room_id"`
	Uface     string `json:"uface"`
	Uname     string `json:"uname"`
	OpenId    string `json:"open_id"`
	Timestamp int    `json:"timestamp"`
}

type RoomEnterData struct {
	UserData
	Timestamp int64 `json:"timestamp"`
}

var GuardLevelMap = map[int]string{
	1: "总督",
	2: "提督",
	3: "舰长",
}
