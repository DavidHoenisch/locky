module github.com/locky

go 1.22

replace github.com/locky/auth => ./auth

require github.com/locky/auth v0.0.0

require (
	github.com/casbin/casbin/v2 v2.87.1 // indirect
	github.com/casbin/govaluate v1.1.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20231201235250-de7065d80cb9 // indirect
	github.com/jackc/pgx/v5 v5.5.4 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/crypto v0.19.0 // indirect
	golang.org/x/sync v0.9.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.20.0 // indirect
	gorm.io/driver/postgres v1.5.7 // indirect
	gorm.io/gorm v1.30.0 // indirect
)
