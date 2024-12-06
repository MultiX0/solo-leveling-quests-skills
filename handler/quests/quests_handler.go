package quests

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/MultiX0/solo_leveling_system/handler/functions"
	"github.com/MultiX0/solo_leveling_system/utils"
	"github.com/gorilla/mux"
)

var (
	handlerInstance *QuestsHandler
	handlerOnce     sync.Once
)

type QuestsHandler struct {
	mu sync.RWMutex
}

func GetNewQuestsHandler() *QuestsHandler {
	handlerOnce.Do(func() {
		handlerInstance = &QuestsHandler{}
	})

	return handlerInstance
}

func (h *QuestsHandler) RoutesHandler(router *mux.Router) {
	router.HandleFunc("/player/{id}/quests", h.FetchQuests).Methods("GET")
	router.HandleFunc("/player/{id}/finish/{questId}", h.FinishQuest).Methods("GET")

}

func (h *QuestsHandler) FinishQuest(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	playerId := params["id"]
	questId := params["questId"]

	if len(playerId) == 0 || len(questId) == 0 {
		log.Println("Empty player ID and questID received")
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid player ID and quest ID"))
		return
	}

	skill, err := functions.FinishQuest(playerId, questId)
	if err != nil {
		log.Println(err)
		utils.WriteError(w, http.StatusBadGateway, err)
		return
	}

	utils.WriteJsonResponse(w, http.StatusAccepted, map[string]any{
		"message": "congrats you got a new skill!",
		"skill":   skill,
	})

}

func (h *QuestsHandler) FetchQuests(w http.ResponseWriter, r *http.Request) {

	h.mu.Lock()
	defer h.mu.Unlock()

	params := mux.Vars(r)
	playerId := params["id"]

	log.Printf("Received request for player ID: %s", playerId)

	if playerId == "" {
		log.Println("Empty player ID received")
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid player ID"))
		return
	}

	mainQuest, err, startTime := functions.GetMainQuest(playerId)
	if err != nil {
		log.Println(err)
		utils.WriteError(w, http.StatusBadGateway, err)
		return
	}

	sideQuests, err := functions.GetSideQuests(playerId)
	if err != nil {
		log.Println(err)
		utils.WriteError(w, http.StatusBadGateway, err)
		return
	}

	timeLeft := time.Until(startTime.Add(24 * time.Hour))
	if timeLeft < 0 {
		timeLeft = 0
	}
	timeLeftStr := timeLeft.Round(time.Minute).String()

	type Response struct {
		MainQuest  any    `json:"main_quest"`
		SideQuests any    `json:"side_quests"`
		TimeLeft   string `json:"time_left"`
		Punishment string `json:"punishment"`
	}

	response := Response{
		MainQuest:  mainQuest,
		SideQuests: sideQuests,
		TimeLeft:   timeLeftStr,
		Punishment: "You will lose 250 xp points.",
	}

	utils.WriteJsonResponse(w, http.StatusOK, response)
}
