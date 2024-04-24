package main

const (
	RequestTypeInit      = "init"
	RequestTypeHeartbeat = "heartbeat"

	ResultTypeHeartbeat = "heartbeat"
	ResultTypeRoom      = "room"
	ResultTypeDanmu     = "danmu"
	ResultTypeSuperChat = "superchat"
	ResultTypeGift      = "gift"
	ResultTypeGuard     = "guard"
)

type InitRequestData struct {
	Code string `json:"code" binding:"required"`
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
	GuardNum  int    `json:"guard_num"`
	GuardUnit string `json:"guard_unit"`
	MsgID     string `json:"msg_id"`
	Timestamp int    `json:"timestamp"`
}
