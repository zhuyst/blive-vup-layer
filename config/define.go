package config

const (
	ResultFilePath = "./result/"
)

type Config struct {
	QianFan   *QianFanConfig   `toml:"qianfan"`
	AliyunTTS *AliyunTTSConfig `toml:"aliyun_tts"`
	BiliBili  *BiliBiliConfig  `toml:"biliBili"`
}

type QianFanConfig struct {
	AccessKey string `toml:"access_key"`
	SecretKey string `toml:"secret_key"`
}

type AliyunTTSConfig struct {
	AccessKey string `toml:"access_key"`
	SecretKey string `toml:"secret_key"`
	AppKey    string `toml:"app_key"`
}

type BiliBiliConfig struct {
	AccessKey           string `toml:"access_key"`
	SecretKey           string `toml:"secret_key"`
	AppId               int64  `toml:"app_id"`
	DisableValidateSign bool   `toml:"disable_validate_sign"`
}