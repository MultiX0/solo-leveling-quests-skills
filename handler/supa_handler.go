package supa

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/MultiX0/solo_leveling_system/db"
	"github.com/MultiX0/solo_leveling_system/handler/functions"
	"github.com/MultiX0/solo_leveling_system/types"
	"github.com/MultiX0/solo_leveling_system/utils"
	"github.com/gorilla/mux"
)

var (
	handlerInstance *SupabaseHandler
	handlerOnce     sync.Once
)

var wg = &sync.WaitGroup{}

type SupabaseHandler struct {
	mu sync.RWMutex
}

func GetSupabaseHandler() *SupabaseHandler {

	handlerOnce.Do(func() {
		handlerInstance = &SupabaseHandler{}
	})

	return handlerInstance
}

func (h *SupabaseHandler) HandleRequests(router *mux.Router) {
	router.HandleFunc("/init", h.initDB).Methods("POST")
	router.HandleFunc("/player/{id}", h.GetPlayerByID).Methods("GET")
	router.HandleFunc("/player", h.CreateNewPlayer).Methods("POST")
}

func (h *SupabaseHandler) CreateNewPlayer(w http.ResponseWriter, r *http.Request) {

	type RequestBody struct {
		Name   string `json:"name"`
		Gender *bool  `json:"gender"`
	}

	var body RequestBody

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Println(err)
		utils.WriteError(w, http.StatusBadGateway, fmt.Errorf("please provide all the player data such as (name, gender)\nbe sure to set the gender to true if the player is male"))
		return
	}

	if len(body.Name) == 0 || body.Gender == nil {
		utils.WriteError(w, http.StatusBadGateway, fmt.Errorf("please provide all the player data such as (name, gender)\nbe sure to set the gender to true if the player is male"))
		return
	}

	player := &types.Player{Name: body.Name, Gender: *body.Gender}
	player, err := functions.CreateNewPlayer(player)

	if err != nil {
		log.Println(err)
		utils.WriteError(w, http.StatusBadGateway, err)
		return
	}

	utils.WriteJsonResponse(w, http.StatusOK, player)

}

func (h *SupabaseHandler) GetPlayerByID(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	params := mux.Vars(r)
	playerId := params["id"]
	if len(playerId) == 0 {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("please provide valid player id"))
		return
	}

	player, err := functions.GetPlayerByID(playerId)
	if err != nil {
		log.Println(err)
		utils.WriteError(w, http.StatusBadGateway, err)
		return
	}

	skills, err := functions.GetPlayerSkills(playerId)
	if err != nil {
		log.Println(err)
		utils.WriteError(w, http.StatusBadGateway, err)
		return
	}

	type PlayerDataResponse struct {
		Player *types.Player  `json:"player"`
		Skills []*types.Skill `json:"skills"`
	}

	response := PlayerDataResponse{
		Player: player,
		Skills: skills,
	}

	utils.WriteJsonResponse(w, http.StatusOK, response)

}

func (h *SupabaseHandler) initDB(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	questsJson, err := os.Open("quests.json")
	if err != nil {
		log.Println(err)
		utils.WriteError(w, http.StatusBadGateway, err)
	}

	defer questsJson.Close()

	bytesValue, _ := io.ReadAll(questsJson)
	var quests []types.Quest

	err = json.Unmarshal(bytesValue, &quests)
	if err != nil {
		log.Println(err)
		utils.WriteError(w, http.StatusBadGateway, err)
	}

	h.insertQuests(quests)

	skillsJson, err := os.Open("skills.json")
	if err != nil {
		log.Println(err)
		utils.WriteError(w, http.StatusBadGateway, err)
	}

	defer skillsJson.Close()

	bytesValue, _ = io.ReadAll(skillsJson)
	var skills []types.Skill

	err = json.Unmarshal(bytesValue, &skills)
	if err != nil {
		log.Println(err)
		utils.WriteError(w, http.StatusBadGateway, err)
	}

	h.insertSkills(skills)

	utils.WriteJsonResponse(w, http.StatusOK, map[string]any{"quests": quests, "skills": skills})

}

func (h *SupabaseHandler) insertSkills(skills []types.Skill) {
	for _, skill := range skills {
		wg.Add(1)
		go func(skill types.Skill) {
			defer wg.Done()
			skills, err := h.getSkillByName(skill.Name)
			if err != nil || len(skills) != 0 {
				return
			}
			log.Println(skill)
			db.SupabaseClient.From("skills").Insert(map[string]any{"name": skill.Name, "description": skill.Description, "level": skill.Level}, false, "", "", "exact").Execute()
		}(skill)
	}
	wg.Wait()
}

func (h *SupabaseHandler) insertQuests(quests []types.Quest) {
	for _, quest := range quests {
		wg.Add(1)
		go func(q types.Quest) {
			defer wg.Done()
			quest, err := h.getQuestByTitle(q.Title)
			if err != nil || len(quest) != 0 {
				return
			}
			log.Println(q)
			db.SupabaseClient.From("quests").Insert(map[string]any{
				"title":       q.Title,
				"description": q.Description,
				"priority":    q.Priority,
			}, false, "", "", "exact").Execute()
		}(quest)
	}
	wg.Wait()
}

func (h *SupabaseHandler) getQuestByTitle(title string) ([]types.Quest, error) {

	data, _, err := db.SupabaseClient.From("quests").Select("title", "exact", false).Eq("title", title).Execute()
	if err != nil {
		return nil, err
	}

	var q []types.Quest
	err = json.Unmarshal(data, &q)

	if err != nil {
		return nil, err
	}

	return q, nil

}

func (h *SupabaseHandler) getSkillByName(name string) ([]types.Skill, error) {

	data, _, err := db.SupabaseClient.From("skills").Select("name", "exact", false).Eq("name", name).Execute()
	if err != nil {
		return nil, err
	}

	var s []types.Skill
	err = json.Unmarshal(data, &s)

	if err != nil {
		return nil, err
	}

	return s, nil

}
