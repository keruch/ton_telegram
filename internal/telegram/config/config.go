package config

type BotConfig struct {
	Token                 string        `json:"token"`
	Name                  string        `json:"name"`
	RatingLimit           int           `json:"rating_limit"`
	RequiredSubscriptions []string      `json:"required_subscriptions"`
	Messages              Messages      `json:"messages"`
	FormatStrings         FormatStrings `json:"format_strings"`
	Buttons               Buttons       `json:"buttons"`
}

type Messages struct {
	Start             string `json:"start"`
	SubscribeToJoin   string `json:"subscribe_to_join"`
	SubscribedToAll   string `json:"subscribed_to_all"`
	AlreadyRegistered string `json:"already_registered"`
	MissingCommand    string `json:"missing_command"`
}

type FormatStrings struct {
	Unsubscribed          string `json:"unsubscribed"`
	FriendUnsubscribed    string `json:"friend_unsubscribed"`
	FriendSubscribedToAll string `json:"friend_subscribed_to_all"`
	Points                string `json:"points"`
	PersonalLink          string `json:"personal_link"`
	YouWereInvited        string `json:"you_were_invited"`
}

type Buttons struct {
	Invite string `json:"invite"`
	Points string `json:"points"`
	Rating string `json:"rating"`
	Info   string `json:"info"`
}
