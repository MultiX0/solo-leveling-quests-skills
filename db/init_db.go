package db

import (
	"log"
	"os"

	"github.com/supabase-community/supabase-go"
)

var SupabaseClient *supabase.Client

func InitDB() {

	_url := os.Getenv("SUPA_URL")
	_key := os.Getenv("SUPA_KEY")

	client, err := supabase.NewClient(_url, _key, &supabase.ClientOptions{})
	if err != nil {
		log.Fatal(err)
	}

	SupabaseClient = client

}
