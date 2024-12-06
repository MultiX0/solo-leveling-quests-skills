package types

import "time"

type Quest struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
}

type Skill struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Level       int    `json:"level"`
}

type Player struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	Gender   bool      `json:"gender"`
	JoinedAt time.Time `json:"joined_at"`
}

type PlayerQuest struct {
	ID       int       `json:"id"`
	StartAt  time.Time `json:"start_at"`
	PlayerID int       `json:"player"`
	QuestID  int       `json:"quest"`
	Status   int       `json:"status"`
}

type PlayerSkills struct {
	ID        int       `json:"id"`
	SkillID   int       `json:"skill"`
	PlayerID  int       `json:"player"`
	RecivedAt time.Time `json:"recived_at"`
}
