module github.com/jbeshir/alignment-research-feed

go 1.24.1

require (
	github.com/auth0/go-jwt-middleware/v2 v2.3.0
	github.com/go-sql-driver/mysql v1.9.2
	github.com/gorilla/feeds v1.2.0
	github.com/gorilla/mux v1.8.1
	github.com/huandu/go-sqlbuilder v1.35.0
	github.com/joho/godotenv v1.5.1
	github.com/pinecone-io/go-pinecone v1.1.1
	github.com/stretchr/testify v1.10.0
	golang.org/x/crypto v0.45.0
	golang.org/x/sync v0.18.0
	google.golang.org/protobuf v1.36.6
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/oapi-codegen/runtime v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250414145226-207652e42e2e // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250414145226-207652e42e2e // indirect
	google.golang.org/grpc v1.71.1 // indirect
	gopkg.in/go-jose/go-jose.v2 v2.6.3 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/gorilla/feeds v1.1.2 => github.com/jbeshir/gorilla-feeds v0.0.0-20240110072658-f3d0c21c0bd5
