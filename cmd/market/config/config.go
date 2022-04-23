package config

import (
	"net/url"

	"github.com/keruch/ton_telegram/internal/market/config"
	"github.com/spf13/viper"
)

func SetupConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./cmd/market/config")

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

func GetBotConfig() config.MarketConfig {
	return config.MarketConfig{
		Token:       viper.GetString("bot.token"),
		Name:        viper.GetString("bot.name"),
		RatingLimit: viper.GetInt("bot.rating_limit"),
		Messages: config.Messages{
			Start:          viper.GetString("bot.messages.start"),
			Info:           viper.GetString("bot.messages.info"),
			MissingCommand: viper.GetString("bot.messages.missing_command"),
			Error:          viper.GetString("bot.messages.error"),
		},
		FormatStrings: config.FormatStrings{
			Nft: viper.GetString("bot.format_strings.nft"),
		},
		Buttons: config.Buttons{
			Verse:  viper.GetString("bot.buttons.verse"),
			Rating: viper.GetString("bot.buttons.rating"),
			Info:   viper.GetString("bot.buttons.info"),
		},
	}
}

func GetTelegramBotToken() string {
	return viper.GetString("bot.token")
}

func GetTelegramBotTag() string {
	return viper.GetString("bot.name")
}

func GetRequiredChannels() []string {
	return viper.GetStringSlice("BotImplementation.required_subscriptions")
}

func GetStartMessage() string {
	return viper.GetString("BotImplementation.start_message")
}

func GetSubscribeToJoinMessage() string {
	return viper.GetString("BotImplementation.subscribe_to_join_message")
}

func GetSubscribedToAllMessage() string {
	return viper.GetString("BotImplementation.subscribed_to_all_message")
}

func GetServerAddress() string {
	return viper.GetString("server.address")
}
