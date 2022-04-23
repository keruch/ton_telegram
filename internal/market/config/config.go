package config

type MarketConfig struct {
	Token         string        `json:"token"`
	Name          string        `json:"name"`
	RatingLimit   int           `json:"rating_limit"`
	Messages      Messages      `json:"messages"`
	FormatStrings FormatStrings `json:"format_strings"`
	Buttons       Buttons       `json:"buttons"`
}

type Messages struct {
	Start          string `json:"start"`
	Info           string `json:"info"`
	MissingCommand string `json:"missing_command"`
	Error          string `json:"error"`
}

type FormatStrings struct {
	Nft string `json:"nft"`
}

type Buttons struct {
	Verse  string `json:"verse"`
	Rating string `json:"rating"`
	Info   string `json:"info"`
}
