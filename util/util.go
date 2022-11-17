package util

import (
	"time"

	"github.com/spf13/viper"
)

func InitializeViper() (err error) {
	viper.SetConfigName("listener")
	viper.AddConfigPath("../../config")
	viper.AddConfigPath("../config")
	viper.AddConfigPath("./config")

	viper.SetEnvPrefix("listener")
	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	return
}

func Crc16(data []byte, poly uint16) (crc uint16) {
	for i := range data {
		crc = crc ^ uint16(data[i])
		for ucBit := 0; ucBit < 8; ucBit++ {
			ucCarry := byte(crc & 1)
			crc >>= 1
			if ucCarry != 0 {
				crc = crc ^ poly
			}
		}
	}
	return
}

func MakeTimeout(timeout int) time.Time {
	return time.Now().Add(time.Duration(timeout) * time.Second)
}