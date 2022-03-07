package config

import (
	"net/url"

	"github.com/keruch/ton_masks_bot/internal/telegram/config"
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

func GetBotConfig() config.BotConfig {
	return config.BotConfig{
		Token:                 viper.GetString("bot.token"),
		Name:                  viper.GetString("bot.name"),
		RatingLimit:           viper.GetInt("bot.rating_limit"),
		RequiredSubscriptions: viper.GetStringSlice("bot.required_subscriptions"),
		Messages: config.Messages{
			Start:             viper.GetString("bot.messages.start"),
			SubscribeToJoin:   viper.GetString("bot.messages.subscribe_to_join"),
			SubscribedToAll:   viper.GetString("bot.messages.subscribed_to_all"),
			AlreadyRegistered: viper.GetString("bot.messages.already_registered"),
			MissingCommand:    viper.GetString("bot.messages.missing_command"),
		},
		FormatStrings: config.FormatStrings{
			Unsubscribed:          viper.GetString("bot.format_strings.unsubscribed"),
			FriendUnsubscribed:    viper.GetString("bot.format_strings.friend_unsubscribed"),
			FriendSubscribedToAll: viper.GetString("bot.format_strings.friend_subscribed_to_all"),
			Points:                viper.GetString("bot.format_strings.points"),
			PersonalLink:          viper.GetString("bot.format_strings.personal_link"),
			YouWereInvited:        viper.GetString("bot.format_strings.you_were_invited"),
		},
		Buttons: config.Buttons{
			Invite: viper.GetString("bot.buttons.invite"),
			Points: viper.GetString("bot.buttons.points"),
			Rating: viper.GetString("bot.buttons.rating"),
			Info:   viper.GetString("bot.buttons.info"),
		},
	}
}

func GetTelegramBotToken() string {
	return viper.GetString("TelegramAPI.bot_token")
}

func GetTelegramBotTag() string {
	return viper.GetString("TelegramAPI.bot_name")
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
