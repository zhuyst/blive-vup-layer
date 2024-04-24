package main

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
)

type WebSocketConn struct {
	conn *websocket.Conn

	connMutex sync.Mutex
}

func NewWebSocketConn(c *gin.Context) (*WebSocketConn, error) {
	wsUpgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	header := http.Header{}
	header.Set("X-Frame-Options", "ALLOW-FROM https://play-live.bilibili.com/")
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return nil, err
	}

	return &WebSocketConn{
		conn: conn,
	}, nil
}

type WebSocketRequest struct {
	Type string          `json:"type" binding:"required"`
	Data json.RawMessage `json:"data"`
}

func (c *WebSocketConn) ReadJSON(v *WebSocketRequest) error {
	return c.conn.ReadJSON(v)
}

type WebSocketResult struct {
	Type string      `json:"type"`
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

const (
	CodeOK            = 0
	CodeBadRequest    = 400
	CodeInternalError = 500
)

func (c *WebSocketConn) WriteResultOK(resultType string, data interface{}) error {
	return c.WriteResult(&WebSocketResult{
		Type: resultType,
		Code: CodeOK,
		Msg:  "success",
		Data: data,
	})
}

func (c *WebSocketConn) WriteResultError(resultType string, code int, msg string) error {
	return c.WriteResult(&WebSocketResult{
		Type: resultType,
		Code: code,
		Msg:  msg,
	})
}

func (c *WebSocketConn) WriteResult(res *WebSocketResult) error {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	msg, _ := json.Marshal(res)
	if res.Code == CodeOK {
		log.Infof("write result type: %s, code: %d, data: %s", res.Type, res.Code, msg)
	} else {
		log.Errorf("write result type: %s, code: %d, data: %s", res.Type, res.Code, msg)
	}
	return c.conn.WriteMessage(websocket.TextMessage, msg)
}

func (c *WebSocketConn) Close() error { return c.conn.Close() }
