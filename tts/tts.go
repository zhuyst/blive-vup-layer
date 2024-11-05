package tts

import (
	"blive-vup-layer/config"
	"errors"
	"fmt"
	nls "github.com/aliyun/alibabacloud-nls-go-sdk"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path"
	"time"
)

type TTS struct {
	cfg *nls.ConnectionConfig
}

const template = `
<speak voice="zhimiao_emo" encodeType="wav" sampleRate="48000">%s</speak>
`

func NewTTS(cfg *config.AliyunTTSConfig) (*TTS, error) {
	nlsCfg, err := nls.NewConnectionConfigWithAKInfoDefault(nls.DEFAULT_URL, cfg.AppKey, cfg.AccessKey, cfg.SecretKey)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(config.ResultFilePath, os.ModePerm); err != nil {
		return nil, err
	}
	return &TTS{cfg: nlsCfg}, nil
}

type Task struct {
	TaskId string
	Logger *log.Entry

	File  io.Writer
	Fname string
	Err   error

	param           nls.SpeechSynthesisStartParam
	text            string
	speechSynthesis *nls.SpeechSynthesis
}

func (tts *TTS) NewTask(input string) (*Task, error) {
	taskId := uuid.NewV4().String()
	l := log.WithField("task_id", uuid.NewV4().String())

	fname := path.Join(config.ResultFilePath, fmt.Sprintf("tts-%s.wav", taskId))
	fout, err := os.OpenFile(fname, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	param := nls.DefaultSpeechSynthesisParam()
	t := &Task{
		TaskId: taskId,
		Logger: l,
		File:   fout,
		Fname:  fname,

		param: param,
		text:  fmt.Sprintf(template, input),
	}

	l.Infof("new tts: %s", t.text)
	ss, err := nls.NewSpeechSynthesis(tts.cfg, nls.DefaultNlsLog(), false,
		t.onTaskFailed, t.onSynthesisResult, nil,
		t.onCompleted, t.onClose, param)
	if err != nil {
		log.Errorf("NewTask err: %v", err)
		return nil, err
	}

	t.speechSynthesis = ss
	return t, nil
}

func (task *Task) Run() (string, error) {
	ch, err := task.speechSynthesis.Start(task.text, task.param, nil)
	if err != nil {
		task.Logger.Errorf("Start err: %v", err)
		task.speechSynthesis.Shutdown()
		task.Err = err
		return "", err
	}

	err = task.waitReady(ch)
	if err != nil {
		task.Err = err
		return "", err
	}
	task.Logger.Infof("Synthesis done")
	task.speechSynthesis.Shutdown()

	return task.Fname, nil
}

func (task *Task) onTaskFailed(text string, param interface{}) {
	task.Logger.Errorf("TaskFailed: %s", text)
}

func (task *Task) onSynthesisResult(data []byte, param interface{}) {
	task.File.Write(data)
}

func (task *Task) onCompleted(text string, param interface{}) {
	task.Logger.Infof("onCompleted: %s", text)
}

func (task *Task) onClose(param interface{}) {
	task.Logger.Infof("onClosed")
}

func (task *Task) waitReady(ch chan bool) error {
	select {
	case done := <-ch:
		{
			if !done {
				task.Logger.Error("wait failed")
				return errors.New("wait failed")
			}
			task.Logger.Debugf("Wait done")
		}
	case <-time.After(60 * time.Second):
		{
			task.Logger.Error("Wait timeout")
			return errors.New("wait timeout")
		}
	}
	return nil
}
