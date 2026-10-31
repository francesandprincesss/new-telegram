// Package database owns PostgreSQL/GORM connection lifecycle and pool settings.
package database

// TODO: Open and configure *gorm.DB from config. Do not call AutoMigrate: the
// schema is managed exclusively by golang-migrate SQL migrations.
