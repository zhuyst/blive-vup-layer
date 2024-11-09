package dao

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Dao struct {
	db *gorm.DB

	userMap      map[string]*User
	userMapMutex sync.RWMutex
}

const MemoryFilePath = ":memory:"

func NewDao(filePath string) (*Dao, error) {
	if !strings.HasPrefix(filePath, MemoryFilePath) {
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return nil, err
		}

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			file, err := os.Create(filePath)
			if err != nil {
				return nil, err
			}
			file.Close()
		}
	}

	db, err := gorm.Open(sqlite.Open(filePath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(User{}); err != nil {
		return nil, err
	}

	return &Dao{
		db:      db,
		userMap: make(map[string]*User),
	}, nil
}
