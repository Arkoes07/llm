package domain

type StudyPlanParam struct {
	Topic        string `json:"topic"`
	CurrentLevel string `json:"current_level"` // "beginner" | "intermediate" | "advanced"
	Weeks        int    `json:"weeks"`
	HoursPerWeek int    `json:"hours_per_week"`
}

type StudyPlan struct {
	Topic       string      `json:"topic"`
	TotalWeeks  int         `json:"total_weeks"`
	Weeks       []StudyWeek `json:"weeks"`
	Assumptions []string    `json:"assumptions"`
}

type StudyWeek struct {
	Week       int      `json:"week"`
	Focus      string   `json:"focus"`
	Activities []string `json:"activities"`
	Outcome    string   `json:"outcome"`
}
