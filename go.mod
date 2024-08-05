module github.com/jbeshir/alignment-research-feed

go 1.22.1

require (
	github.com/go-sql-driver/mysql v1.8.1
	github.com/gorilla/feeds v1.1.2
	github.com/gorilla/mux v1.8.1
	github.com/joho/godotenv v1.5.1
	github.com/stretchr/testify v1.9.0
	golang.org/x/crypto v0.22.0
	golang.org/x/sync v0.7.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/huandu/go-sqlbuilder v1.28.1 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.23.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/gorilla/feeds v1.1.2 => github.com/jbeshir/gorilla-feeds v0.0.0-20240110072658-f3d0c21c0bd5
