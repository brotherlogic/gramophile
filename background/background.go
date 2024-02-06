package background

import "github.com/brotherlogic/gramophile/db"

type BackgroundRunner struct {
	db                    db.Database
	key, secret, callback string
	ReleaseRefresh        int64
}

func GetBackgroundRunner(db db.Database, key, secret, callback string) *BackgroundRunner {
	return &BackgroundRunner{db: db, key: key, secret: secret, callback: callback}
}
