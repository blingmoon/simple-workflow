module github.com/blingmoon/simple-workflow/examples

go 1.24.0

// 使用本地主模块（开发时）
replace github.com/blingmoon/simple-workflow => ../../

require (
	github.com/blingmoon/simple-workflow v0.0.0-00010101000000-000000000000
	github.com/pkg/errors v0.9.1
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.31.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gabriel-vasile/mimetype v1.4.12 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	github.com/redis/go-redis/v9 v9.17.2 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
)
