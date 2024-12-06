package functions

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"

	"github.com/MultiX0/solo_leveling_system/db"
	"github.com/MultiX0/solo_leveling_system/types"
)

func GetPlayerSkills(id string) ([]*types.Skill, error) {

	data, _, err := db.SupabaseClient.From("player_skills").Select("*", "", false).Eq("player", id).Execute()
	if err != nil {
		return nil, err
	}

	var skillsData []*types.PlayerSkills
	err = json.Unmarshal(data, &skillsData)

	if err != nil {
		return nil, err
	}

	var skills []*types.Skill

	for _, s := range skillsData {

		recivedSkill, err := getSkillByID(strconv.Itoa(s.SkillID))
		if err != nil {
			return nil, err
		}

		skills = append(skills, recivedSkill)
	}

	return skills, nil

}

func getSkillByID(id string) (*types.Skill, error) {
	data, _, err := db.SupabaseClient.From("skills").Select("*", "", false).Eq("id", id).Single().Execute()
	if err != nil {
		return nil, err
	}

	var skill *types.Skill
	err = json.Unmarshal(data, &skill)

	if err != nil {
		return nil, err
	}

	return skill, nil
}

func RandomSkillLevelBased(playerId string, level int) (*types.Skill, error) {

	if level > 100 {
		return nil, fmt.Errorf("you already have all the skills")
	}

	levelStr := strconv.Itoa(level)
	data, _, err := db.SupabaseClient.From("skills").Select("*", "", false).Eq("level", levelStr).Execute()
	if err != nil {
		return nil, err
	}

	var skills []*types.Skill

	err = json.Unmarshal(data, &skills)
	if err != nil {
		return nil, err
	}

	rand.Shuffle(len(skills), func(i, j int) {
		skills[i], skills[j] = skills[j], skills[i]
	})

	for _, skill := range skills {
		hasSkill, err := checkHavedSkill(skill)
		if err != nil {
			return nil, err
		}
		if *hasSkill {
			return skill, nil
		}
	}

	return RandomSkillLevelBased(playerId, level+1)
}

func checkHavedSkill(skill *types.Skill) (*bool, error) {

	idStr := strconv.Itoa(skill.ID)

	data, _, err := db.SupabaseClient.From("player_skills").Select("*", "", false).Eq("skill", idStr).Execute()

	if err != nil {
		return nil, err
	}

	var playerSkills []*types.PlayerSkills
	err = json.Unmarshal(data, &playerSkills)
	if err != nil {
		return nil, err
	}

	res := (len(playerSkills) == 0)

	return &res, nil

}

func GivePlayerNewSkill(playerId string, skill *types.Skill) error {
	_, _, err := db.SupabaseClient.From("player_skills").Insert(map[string]any{
		"skill":  skill.ID,
		"player": playerId,
	}, false, "", "", "exact").Execute()

	return err
}
