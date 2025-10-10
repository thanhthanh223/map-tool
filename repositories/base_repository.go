package repositories

import (
	"gorm.io/gorm"
)

// BaseRepository provides common database operations
type BaseRepository struct {
	db *gorm.DB
}

// NewBaseRepository creates a new base repository
func NewBaseRepository(db *gorm.DB) *BaseRepository {
	return &BaseRepository{
		db: db,
	}
}

// GetDB returns the database connection
func (r *BaseRepository) GetDB() *gorm.DB {
	return r.db
}

// Begin starts a transaction
func (r *BaseRepository) Begin() *gorm.DB {
	return r.db.Begin()
}

// Commit commits a transaction
func (r *BaseRepository) Commit(tx *gorm.DB) error {
	return tx.Commit().Error
}

// Rollback rolls back a transaction
func (r *BaseRepository) Rollback(tx *gorm.DB) error {
	return tx.Rollback().Error
}
