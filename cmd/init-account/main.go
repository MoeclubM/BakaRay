package main

import (
	"errors"
	"flag"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"bakaray/internal/config"
	"bakaray/internal/models"
	"bakaray/internal/repository"
)

func main() {
	username := flag.String("username", "", "Username for the account")
	password := flag.String("password", "", "Password for the account")
	role := flag.String("role", "admin", "Role assigned to the account (default admin)")
	groupID := flag.Uint("group", 0, "User group ID for the account")
	configFile := flag.String("config", "", "Path to YAML configuration file")
	flag.Parse()

	if *username == "" || *password == "" {
		log.Fatalf("--username and --password are required")
	}

	if *configFile != "" {
		os.Setenv("CONFIG_FILE", *configFile)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := repository.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	if err := repository.AutoMigrate(db); err != nil {
		log.Printf("Warning: AutoMigrate failed: %v", err)
	}

	var existing models.User
	if err := db.Where("username = ?", *username).First(&existing).Error; err == nil {
		log.Printf("User %s already exists, skipping creation", *username)
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Fatalf("Failed to verify existing user: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	user := models.User{
		Username:     *username,
		PasswordHash: string(hash),
		UserGroupID:  uint(*groupID),
		Role:         *role,
	}
	if err := db.Create(&user).Error; err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	log.Printf("User %s created with role %s", *username, *role)
}
