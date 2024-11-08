package tts

import (
	"blive-vup-layer/config"
	nls "blive-vup-layer/tts/alibabacloud-nls-go-sdk"
	"errors"
	"fmt"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"io"
	syslog "log"
	"os"
	"path"
	"time"
)

type TTS struct {
	cfg *config.AliyunTTSConfig
}

func NewTTS(cfg *config.AliyunTTSConfig) (*TTS, error) {
	if err := os.MkdirAll(config.ResultFilePath, os.ModePerm); err != nil {
		return nil, err
	}
	return &TTS{cfg: cfg}, nil
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

type NewTaskParams struct {
	Text      string
	PitchRate int
}

func (tts *TTS) NewTask(params *NewTaskParams) (*Task, error) {
	taskId := uuid.NewV4().String()
	l := log.WithField("task_id", uuid.NewV4().String())

	fname := path.Join(config.ResultFilePath, fmt.Sprintf("tts-%s.wav", taskId))
	fout, err := os.OpenFile(fname, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	param := nls.SpeechSynthesisStartParam{
		Voice:      "voice-3e06127",
		Format:     "wav",
		SampleRate: 24000,
		Volume:     50,
		SpeechRate: -100,
		PitchRate:  params.PitchRate,
	}

	t := &Task{
		TaskId: taskId,
		Logger: l,
		File:   fout,
		Fname:  fname,

		param: param,
		text:  params.Text,
	}

	l.Infof("new tts: %s", t.text)
	nlsLog := nls.NewNlsLogger(io.Discard, "NLS", syslog.LstdFlags|syslog.Lmicroseconds)
	//nlsLog.SetDebug(true)

	nlsCfg, err := nls.NewConnectionConfigWithAKInfoDefault(
		nls.DEFAULT_URL,
		tts.cfg.AppKey, tts.cfg.AccessKey, tts.cfg.SecretKey,
	)
	if err != nil {
		log.Errorf("NewConnectionConfigWithAKInfoDefault err: %v", err)
		return nil, err
	}

	ss, err := nls.NewSpeechSynthesis(nlsCfg, nlsLog, false,
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

	go func() {
		cleanTimer := time.NewTimer(time.Hour)
		defer cleanTimer.Stop()
		<-cleanTimer.C
		os.Remove(task.Fname)
	}()

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
