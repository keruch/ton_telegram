package config

import (
	"net/url"

	"github.com/spf13/viper"
)

func SetupConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	err := viper.ReadInConfig()
	if err != nil {
		return err
	}

	return nil
}

func GetDatabaseURL() string {
	u := url.URL{
		Host:   viper.GetString("database.address") + viper.GetString("database.port"),
		User:   url.UserPassword(viper.GetString("database.username"), viper.GetString("database.password")),
		Scheme: viper.GetString("database.scheme"),
		Path:   viper.GetString("database.name"),
	}
	return u.String()
}

func GetTelegramBotToken() string {
	return viper.GetString("API.tg_bot_token")
}

func GetServerAddress() string {
	return viper.GetString("server.address")
}
