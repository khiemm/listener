# project structure

- cmd:
- internal:
- pkg:

# module

- has a `go.mod`
- has multiple packages

# package

- one package is one directory
- one package has one `main`, if it not is dependency package
- a package need run by `go run`, must named `package main`, if not, it just is dependency package
- use `go run .` instead of `go main.go` to compile all files in package

# call function from another file

- same package (directory): functions in files call each other
- outside package:
  - in case package is public to any repository (same module or another public module): `import github.com/khiemm/listener/pkg/storage`
  - in case package is not public to any repository (a local module): `go mod edit -replace example.com/storage=../storage`
  - https://go.dev/doc/tutorial/call-module-code

# database

- can connect: use some packages, need learn more
  - database/sql
  - gorp: to map to database
  - github.com/go-sql-driver/mysql: Just registering the driver
- can query from other package

# feature

- listen TCP incoming - GPS data
- handle parse and save to database
- beacon data