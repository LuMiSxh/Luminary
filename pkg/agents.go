package pkg

type AgentStatus string

const (
	AgentStatusStable       AgentStatus = "stable"
	AgentStatusExperimental AgentStatus = "experimental"
	AgentStatusOutdated     AgentStatus = "outdated"
)

type Agent struct {
	ID          string
	Name        string
	Description string
	Status      AgentStatus
}

func GetAgents() []Agent {
	return []Agent{
		{"mangadex", "MangaDex", "World's largest manga community and scanlation site", AgentStatusStable},
		{"mangaplus", "MANGA Plus", "Official source for Shueisha titles", AgentStatusStable},
		{"mangasee", "MangaSee", "Large collection of scanlated manga", AgentStatusStable},
		{"webtoons", "WEBTOONS", "Free comics platform for webtoons", AgentStatusStable},
		{"crunchyroll", "Crunchyroll Manga", "Subscription-based manga service", AgentStatusExperimental},
	}
}

func GetAgentByID(id string) *Agent {
	for _, agent := range GetAgents() {
		if agent.ID == id {
			return &agent
		}
	}
	return nil
}
