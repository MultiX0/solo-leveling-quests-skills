package jobs

import (
	"log"

	"github.com/MultiX0/solo_leveling_system/handler/functions"
	"github.com/robfig/cron/v3"
)

func InitCronJobs() {
	c := cron.New()
	c.AddFunc("@every 24h00m00s", QuestsJob)
}

func QuestsJob() {
	err := functions.UpdateOutdatedQuests()
	if err != nil {
		log.Println(err)
	}
}
