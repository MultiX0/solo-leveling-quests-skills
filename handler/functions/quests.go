package functions

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/MultiX0/solo_leveling_system/db"
	"github.com/MultiX0/solo_leveling_system/types"
	"github.com/MultiX0/solo_leveling_system/utils"
	"github.com/supabase-community/postgrest-go"
)

var (
	questCache     = make(map[string]*types.Quest)
	questCacheMux  sync.RWMutex
	questPoolCache = make(map[bool][]types.Quest)
	poolCacheMux   sync.RWMutex
)

// Worker pool to manage concurrent operations
var workerPool = make(chan struct{}, runtime.NumCPU()*4)

func acquireWorker() {
	workerPool <- struct{}{}
}

func releaseWorker() {
	<-workerPool
}

// Cached quest retrieval
func getQuestByID(id string) (*types.Quest, error) {
	questCacheMux.RLock()
	if quest, exists := questCache[id]; exists {
		questCacheMux.RUnlock()
		return quest, nil
	}
	questCacheMux.RUnlock()

	data, _, err := db.SupabaseClient.From("quests").Select("*", "", false).Eq("id", id).Single().Execute()
	if err != nil {
		return nil, err
	}

	var quest *types.Quest
	err = json.Unmarshal(data, &quest)
	if err != nil {
		return nil, err
	}

	questCacheMux.Lock()
	questCache[id] = quest
	questCacheMux.Unlock()

	return quest, nil
}

func lazyInitQuestPool(main bool) {
	poolCacheMux.Lock()
	defer poolCacheMux.Unlock()

	// Check if pool is already populated
	if _, exists := questPoolCache[main]; exists {
		return
	}

	quests, err := fetchQuestPool(main)
	if err != nil {
		log.Println(err)
		return
	}

	questPoolCache[main] = quests
}

func fetchQuestPool(main bool) ([]types.Quest, error) {
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

	return quests, nil
}

// Cached quest pool retrieval
func fetchQuest(main bool) (*types.Quest, error) {
	poolCacheMux.RLock()
	if pool, exists := questPoolCache[main]; exists && len(pool) > 0 {
		poolCacheMux.RUnlock()
		random := rand.Intn(len(pool))
		return &pool[random], nil
	}
	poolCacheMux.RUnlock()

	// Lazy load the quest pool if not exists
	lazyInitQuestPool(main)

	poolCacheMux.RLock()
	defer poolCacheMux.RUnlock()

	pool, exists := questPoolCache[main]
	if !exists || len(pool) == 0 {
		return nil, fmt.Errorf("no quests found")
	}

	random := rand.Intn(len(pool))
	return &pool[random], nil
}

func GetMainQuest(id string) (*types.Quest, error, time.Time) {
	var quest *types.Quest
	var err error
	var startTime time.Time
	var mu sync.Mutex
	var wg sync.WaitGroup

	acquireWorker()
	defer releaseWorker()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				mu.Lock()
				err = fmt.Errorf("panic in GetMainQuest: %v", r)
				mu.Unlock()
			}
		}()

		completedQuestData, count, queryErr := db.SupabaseClient.From("player_quests").
			Select("*", "exact", false).Order("start_at", &postgrest.OrderOpts{Ascending: false}).
			Eq("player", id).
			Eq("status", "1").
			Eq("priority", "1").
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
			Select("*", "exact", false).Order("start_at", &postgrest.OrderOpts{Ascending: false}).
			Eq("player", id).
			Eq("status", "0").
			Eq("priority", "1").
			Limit(1, "").
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
	var wg sync.WaitGroup

	acquireWorker()
	defer releaseWorker()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				mu.Lock()
				err = fmt.Errorf("panic in GetSideQuests: %v", r)
				mu.Unlock()
			}
		}()

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

func insertQuestToPlayerQuests(quest *types.Quest, playerId string) error {
	id, err := strconv.Atoi(playerId)
	if err != nil {
		return err
	}

	_, _, err = db.SupabaseClient.From("player_quests").Insert(map[string]any{
		"start_at": utils.NowDate(),
		"player":   id,
		"quest":    quest.ID,
		"status":   0,
		"priority": quest.Priority,
	}, false, "", "", "exact").Execute()

	return err
}

func FinishQuest(playerId string, questId string) (*types.Skill, error) {
	_, _, err := db.SupabaseClient.From("player_quests").Update(
		map[string]any{"status": 1},
		"",
		"exact",
	).Eq("status", "0").Eq("player", playerId).Eq("quest", questId).Execute()

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

func TimeForQuest(main bool, playerId string) (*time.Time, error) {
	var data []byte
	var err error

	if main {
		data, _, err = db.SupabaseClient.From("player_quests").
			Select("*", "exact", false).
			Eq("player", playerId).
			Eq("priority", "1").
			Order("start_at", &postgrest.OrderOpts{Ascending: false}).
			Limit(1, "").
			Execute()
	} else {
		data, _, err = db.SupabaseClient.From("player_quests").
			Select("*", "exact", false).
			Eq("player", playerId).
			Neq("priority", "1").
			Order("start_at", &postgrest.OrderOpts{Ascending: false}).
			Limit(1, "").
			Execute()
	}

	if err != nil {
		return nil, err
	}

	var quests []types.PlayerQuest
	err = json.Unmarshal(data, &quests)
	if err != nil {
		return nil, err
	}

	if len(quests) == 0 {
		return nil, fmt.Errorf("no quests found")
	}

	return &quests[0].StartAt, nil
}

func UpdateOutdatedQuests() error {
	outDatedTime := time.Now().Add(-time.Hour * 24)
	_, _, err := db.SupabaseClient.From("player_quests").
		Update(map[string]any{"status": 2}, "", "exact").
		Eq("status", "0").
		Lt("start_at", outDatedTime.UTC().Format("2006-01-02T15:04:05.999999Z")).
		Execute()

	return err
}
