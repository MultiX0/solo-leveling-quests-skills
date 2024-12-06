# Solo Leveling Quests & Skills System

## Project Overview
A dynamic quest and skill progression system inspired by the Solo Leveling concept, built with Go and Supabase.

## Features
- Player creation and management
- Dynamic quest generation
- Skill acquisition system
- Time-based quest progression
- Randomized skill rewards

## Technical Stack
- Backend: Go (Golang)
- Database: Supabase PostgreSQL
- API Framework: Gorilla Mux

## Database Schema
- `players`: Store player basic information
- `quests`: Define available quests
- `skills`: Catalog of skills
- `player_quests`: Track player quest progress
- `player_skills`: Record player acquired skills

## Key Endpoints
- `POST /player`: Create new player
- `GET /player/{id}`: Retrieve player details
- `GET /player/{id}/quests`: Fetch active quests
- `GET /player/{id}/finish/{questId}`: Complete a quest

## Installation
1. Clone the repository
2. Set up Supabase project
3. Configure database connection
4. Run database migrations
5. Start the Go server

## Environment Setup
- Go 1.20+
- Supabase account
- PostgreSQL

## Contributing
1. Fork the repository
2. Create feature branch
3. Commit changes
4. Push and create pull request

## License
MIT License
