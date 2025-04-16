module github.com/jbeshir/alignment-research-feed

go 1.23.1

require (
	github.com/go-sql-driver/mysql v1.8.1
	github.com/gorilla/feeds v1.1.2
	github.com/gorilla/mux v1.8.1
	github.com/huandu/go-sqlbuilder v1.28.1
	github.com/joho/godotenv v1.5.1
	github.com/pinecone-io/go-pinecone v1.1.0
	github.com/stretchr/testify v1.9.0
	golang.org/x/crypto v0.36.0
	golang.org/x/sync v0.12.0
	google.golang.org/protobuf v1.34.2
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/oapi-codegen/runtime v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/grpc v1.66.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/gorilla/feeds v1.1.2 => github.com/jbeshir/gorilla-feeds v0.0.0-20240110072658-f3d0c21c0bd5
