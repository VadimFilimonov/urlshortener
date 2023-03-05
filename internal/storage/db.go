package storage

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type dataDB struct {
	databaseDNS string
}

func runMigrations(databaseDNS string) {
	db, err := sql.Open("postgres", databaseDNS)

	if err != nil {
		db.Close()
		log.Fatal("unable to open db")
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})

	if err != nil {
		db.Close()
		log.Fatalf("unable to init db driver %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://schema",
		"postgres",
		driver,
	)

	if err != nil {
		db.Close()
		log.Fatalf("unable to init db migrator %v", err)
	}

	m.Up()
	db.Close()
}

func NewDB(databaseDNS string) dataDB {
	runMigrations(databaseDNS)

	return dataDB{databaseDNS: databaseDNS}
}

func (data dataDB) Get(shortenURL string) (string, error) {
	db, err := sql.Open("postgres", data.databaseDNS)

	if err != nil {
		db.Close()
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var originalURL string
	err = db.QueryRowContext(ctx, "SELECT original_url FROM urls WHERE shorten_url = $1 LIMIT 1", shortenURL).Scan(&originalURL)

	if err != nil {
		db.Close()
		return "", err
	}

	db.Close()
	return originalURL, nil
}

func (data dataDB) GetItemsOfUser(userID string) ([]item, error) {
	items := make([]item, 0)
	db, err := sql.Open("postgres", data.databaseDNS)

	if err != nil {
		db.Close()
		log.Println(err)
		return items, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, "SELECT * FROM urls WHERE user_id = $1", userID)

	if err != nil {
		db.Close()
		log.Println(err)
		return items, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var item item
		err = rows.Scan(&id, &item.userID, &item.ShortenURL, &item.OriginalURL)

		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	db.Close()
	return items, nil
}

func (data dataDB) Add(originalURL, shortenURL, userID string) bool {
	db, err := sql.Open("postgres", data.databaseDNS)

	if err != nil {
		db.Close()
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = db.ExecContext(ctx, "INSERT INTO urls(user_id, shorten_url, original_url) VALUES($1,$2,$3)", userID, shortenURL, originalURL)

	if err != nil {
		db.Close()
		return false
	}

	db.Close()
	return true
}
