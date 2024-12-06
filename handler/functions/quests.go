package functions

import (
	"encoding/json"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/MultiX0/solo_leveling_system/db"
	"github.com/MultiX0/solo_leveling_system/types"
	"github.com/MultiX0/solo_leveling_system/utils"
	"github.com/supabase-community/postgrest-go"
)

var wg = &sync.WaitGroup{}

func GetMainQuest(id string) (*types.Quest, error, time.Time) {
	var quest *types.Quest
	var err error
	var startTime time.Time
	var mu sync.Mutex
	wg.Add(1)
	go func() {
		defer wg.Done()
		completedQuestData, count, queryErr := db.SupabaseClient.From("player_quests").
			Select("*", "exact", false).
			Eq("player", id).
			Eq("status", "1").
			Eq("priority", "1").
			Order("start_at", &postgrest.OrderOpts{Ascending: false}).
			Limit(1, "").
			Execute()
		if queryErr != nil {
			mu.Lock()
			err = queryErr
			mu.Unlock()
			return
		}
		if count > 0 {
			var completedQuest []*types.PlayerQuest
			if unmarshalErr := json.Unmarshal(completedQuestData, &completedQuest); unmarshalErr != nil {
				mu.Lock()
				err = unmarshalErr
				mu.Unlock()
				return
			}
			if time.Since(completedQuest[0].StartAt) < 24*time.Hour {
				mu.Lock()
				quest = nil
				startTime = time.Time{}
				mu.Unlock()
				return
			}
		}
		currentQuestsData, count, queryErr := db.SupabaseClient.From("player_quests").
			Select("*", "exact", false).
			Eq("player", id).
			Eq("status", "0").
			Eq("priority", "1").
			Execute()
		if queryErr != nil {
			mu.Lock()
			err = queryErr
			mu.Unlock()
			return
		}
		if count == 0 {
			newQuest, fetchErr := fetchQuest(true)
			if fetchErr != nil {
				mu.Lock()
				err = fetchErr
				mu.Unlock()
				return
			}
			insertErr := insertQuestToPlayerQuests(newQuest, id)
			if insertErr != nil {
				mu.Lock()
				err = insertErr
				mu.Unlock()
				return
			}
			mu.Lock()
			quest = newQuest
			startTime = time.Now()
			mu.Unlock()
			return
		}
		var data []*types.PlayerQuest
		unmarshalErr := json.Unmarshal(currentQuestsData, &data)
		if unmarshalErr != nil {
			mu.Lock()
			err = unmarshalErr
			mu.Unlock()
			return
		}
		retrievedQuest, questErr := getQuestByID(strconv.Itoa(data[0].QuestID))
		if questErr != nil {
			mu.Lock()
			err = questErr
			mu.Unlock()
			return
		}
		mu.Lock()
		quest = retrievedQuest
		startTime = data[0].StartAt
		mu.Unlock()
	}()
	wg.Wait()
	return quest, err, startTime
}

func GetSideQuests(id string) ([]*types.Quest, error, time.Time) {
	var quests []*types.Quest
	var err error
	var startTime time.Time
	var mu sync.Mutex
	wg.Add(1)
	go func() {
		defer wg.Done()
		completedQuestsData, count, queryErr := db.SupabaseClient.From("player_quests").
			Select("*", "exact", false).
			Eq("player", id).
			Eq("status", "1").
			Gt("priority", "1").
			Order("start_at", &postgrest.OrderOpts{Ascending: false}).
			Limit(2, "").
			Execute()
		if queryErr != nil {
			mu.Lock()
			err = queryErr
			mu.Unlock()
			return
		}
		if count > 0 {
			var completedQuests []*types.PlayerQuest
			if unmarshalErr := json.Unmarshal(completedQuestsData, &completedQuests); unmarshalErr != nil {
				mu.Lock()
				err = unmarshalErr
				mu.Unlock()
				return
			}
			allRecent := true
			earliestStart := time.Now()
			for _, quest := range completedQuests {
				if quest.StartAt.Before(earliestStart) {
					earliestStart = quest.StartAt
				}
				if time.Since(quest.StartAt) >= 24*time.Hour {
					allRecent = false
					break
				}
			}
			if allRecent && len(completedQuests) == 2 {
				mu.Lock()
				quests = []*types.Quest{}
				startTime = earliestStart
				mu.Unlock()
				return
			}
		}
		currentQuestsData, count, queryErr := db.SupabaseClient.From("player_quests").
			Select("*", "exact", false).
			Eq("player", id).
			Eq("status", "0").
			Gt("priority", "1").
			Execute()
		if queryErr != nil {
			mu.Lock()
			err = queryErr
			mu.Unlock()
			return
		}
		if count == 0 {
			var tempQuests []*types.Quest
			currentTime := time.Now()
			for len(tempQuests) < 2 {
				quest, fetchErr := fetchQuest(false)
				if fetchErr != nil {
					mu.Lock()
					err = fetchErr
					mu.Unlock()
					return
				}
				duplicate := false
				for _, q := range tempQuests {
					if quest.Title == q.Title {
						duplicate = true
						break
					}
				}
				if !duplicate {
					tempQuests = append(tempQuests, quest)
					insertErr := insertQuestToPlayerQuests(quest, id)
					if insertErr != nil {
						mu.Lock()
						err = insertErr
						mu.Unlock()
						return
					}
				}
			}
			mu.Lock()
			quests = tempQuests
			startTime = currentTime
			mu.Unlock()
			return
		}
		var data []*types.PlayerQuest
		unmarshalErr := json.Unmarshal(currentQuestsData, &data)
		if unmarshalErr != nil {
			mu.Lock()
			err = unmarshalErr
			mu.Unlock()
			return
		}
		var tempQuests []*types.Quest
		earliestStart := time.Now()
		for _, d := range data {
			quest, questErr := getQuestByID(strconv.Itoa(d.QuestID))
			if questErr != nil {
				mu.Lock()
				err = questErr
				mu.Unlock()
				return
			}
			if d.StartAt.Before(earliestStart) {
				earliestStart = d.StartAt
			}
			tempQuests = append(tempQuests, quest)
		}
		mu.Lock()
		quests = tempQuests
		startTime = earliestStart
		mu.Unlock()
	}()
	wg.Wait()
	return quests, err, startTime
}

func getQuestByID(id string) (*types.Quest, error) {
	data, _, err := db.SupabaseClient.From("quests").Select("*", "", false).Eq("id", id).Single().Execute()
	if err != nil {
		return nil, err
	}

	var quest *types.Quest
	err = json.Unmarshal(data, &quest)
	if err != nil {
		return nil, err
	}

	return quest, err

}

func fetchQuest(main bool) (*types.Quest, error) {

	var data []byte
	var err error

	if main {
		data, _, err = db.SupabaseClient.From("quests").Select("*", "exact", false).Eq("priority", "1").Execute()
	} else {
		data, _, err = db.SupabaseClient.From("quests").Select("*", "exact", false).Gt("priority", "1").Execute()
	}

	if err != nil {
		return nil, err
	}
	var quests []types.Quest
	if err = json.Unmarshal(data, &quests); err != nil {
		return nil, err
	}

	if len(quests) == 1 {
		return &quests[0], err
	}

	random := rand.Intn(len(quests) - 1)
	quest := quests[random]

	return &quest, nil

}

func insertQuestToPlayerQuests(quest *types.Quest, playerId string) error {

	id, err := strconv.Atoi(playerId)
	if err != nil {
		return err
	}

	_, _, err = db.SupabaseClient.From("player_quests").Insert(map[string]any{
		"start_at": utils.NowDate(),
		"player":   id, "quest": quest.ID,
		"status":   0,
		"priority": quest.Priority,
	}, false, "", "", "exact").Execute()

	return err

}

func FinishQuest(playerId string, questId string) (*types.Skill, error) {

	_, _, err := db.SupabaseClient.From("player_quests").Update(map[string]any{"status": 1}, "", "exact").Eq("status", "0").Eq("player", playerId).Eq("quest", questId).Execute()
	if err != nil {
		return nil, err
	}

	quest, err := getQuestByID(questId)
	if err != nil {
		return nil, err

	}

	skill, err := RandomSkillLevelBased(playerId, quest.Priority)

	if err != nil {
		return nil, err

	}

	err = GivePlayerNewSkill(playerId, skill)
	if err != nil {
		return nil, err

	}

	return skill, nil
}

func UpdateOutdatedQuests() error {
	outDatedTime := time.Now().Add(-time.Hour * 24)
	_, _, err := db.SupabaseClient.From("player_quests").Select("*", "exact", false).Eq("status", "0").Lt("start_at", outDatedTime.UTC().Format("2006-01-02T15:04:05.999999Z")).Execute()

	return err
}
