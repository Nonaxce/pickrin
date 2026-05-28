package main

import "time"

type Project struct {
	ClientSide       string    `json:"client_side"`
	ServerSide       string    `json:"server_side"`
	GameVersions     []string  `json:"game_versions"`
	ID               string    `json:"id"`
	Slug             string    `json:"slug"`
	ProjectType      string    `json:"project_type"`
	Team             string    `json:"team"`
	Organization     string    `json:"organization"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Body             string    `json:"body"`
	BodyURL          string    `json:"body_url"`
	Published        time.Time `json:"published"`
	Updated          time.Time `json:"updated"`
	Approved         time.Time `json:"approved"`
	Queued           string    `json:"queued"`
	Status           string    `json:"status"`
	RequestedStatus  string    `json:"requested_status"`
	ModeratorMessage struct {
		Message string
		Body    string
	} `json:"moderator_message"`
	License struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"license"`
	Downloads            int      `json:"downloads"`
	Followers            int      `json:"followers"`
	Categories           []string `json:"categories"`
	AdditionalCategories []string `json:"additional_categories"`
	Loaders              []string `json:"loaders"`
	Versions             []string `json:"versions"`
	IconURL              string   `json:"icon_url"`
	IssuesURL            string   `json:"issues_url"`
	SourceURL            string   `json:"source_url"`
	WikiURL              string   `json:"wiki_url"`
	DiscordURL           string   `json:"discord_url"`
	//DonationUrls         []string `json:"donation_urls"`
	// Gallery []struct {
	// 	URL         string    `json:"url"`
	// 	RawURL      string    `json:"raw_url"`
	// 	Featured    bool      `json:"featured"`
	// 	Title       string    `json:"title"`
	// 	Description string    `json:"description"`
	// 	Created     time.Time `json:"created"`
	// 	Ordering    int       `json:"ordering"`
	// } `json:"gallery"`
	//Color              int    `json:"color"`
	ThreadID string `json:"thread_id"`
	//MonetizationStatus string `json:"monetization_status"`
}
