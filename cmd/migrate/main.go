package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/yourname/api-gateway/internal/config"
	"github.com/yourname/api-gateway/internal/database"
)

func main() {
	cfg := config.Load()

	files, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		log.Fatalf("failed to read migrations: %v", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		log.Println("no migrations to apply")
		return
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	for _, file := range files {
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("failed to read %s: %v", file, err)
		}
		if _, err := db.Exec(string(sqlBytes)); err != nil {
			log.Fatalf("failed to apply %s: %v", file, err)
		}
		log.Printf("applied %s", filepath.Base(file))
	}

	fmt.Printf("applied %d migration(s)\n", len(files))
}
