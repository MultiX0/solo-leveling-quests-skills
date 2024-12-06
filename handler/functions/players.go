package functions

import (
	"encoding/json"
	"log"

	"github.com/MultiX0/solo_leveling_system/db"
	"github.com/MultiX0/solo_leveling_system/types"
	"github.com/MultiX0/solo_leveling_system/utils"
)

func CreateNewPlayer(player *types.Player) (*types.Player, error) {
	data, err := utils.InsertToDB("players", map[string]any{
		"name":   player.Name,
		"gender": player.Gender,
	})
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var newPlayer types.Player

	err = json.Unmarshal(data, &newPlayer)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	log.Println(newPlayer)

	return &newPlayer, nil
}

func GetPlayerByID(id string) (*types.Player, error) {
	data, _, err := db.SupabaseClient.From("players").Select("*", "exact", false).Eq("id", id).Single().Execute()

	if err != nil {
		log.Println(err)
		return nil, err
	}

	var player types.Player
	err = json.Unmarshal(data, &player)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &player, nil

}
