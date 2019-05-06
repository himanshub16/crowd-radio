module github.com/himanshub16/upnext-backend

go 1.12

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/google/uuid v1.1.1
	github.com/jmoiron/sqlx v1.2.0
	github.com/labstack/echo v3.3.10+incompatible
	github.com/labstack/gommon v0.2.8 // indirect
	github.com/lib/pq v1.1.0
	github.com/mattn/go-colorable v0.1.1 // indirect
	github.com/mattn/go-isatty v0.0.6 // indirect
	github.com/mattn/go-sqlite3 v1.10.0
	github.com/stretchr/testify v1.3.0 // indirect
	github.com/valyala/fasttemplate v1.0.0 // indirect
	google.golang.org/appengine v1.5.0 // indirect
)

require (
	github.com/google/pprof v0.0.0-20190502144155-8358a9778bd1 // indirect
	github.com/himanshub16/upnext-backend/cluster v0.0.0
	github.com/ianlancetaylor/demangle v0.0.0-20181102032728-5e5cf60278f6 // indirect
	golang.org/x/arch v0.0.0-20190312162104-788fe5ffcd8c // indirect
	golang.org/x/tools v0.0.0-20190503185657-3b6f9c0030f7 // indirect
)

replace github.com/himanshub16/upnext-backend/cluster v0.0.0 => ./cluster
