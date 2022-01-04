# project structure

# package

- same package: functions in files call each other
- outside package:
  - need go mod init and replace: go mod edit -replace example.com/storage=pkg/storage, in case package is not public to any repository
  - to use that package

# database

- can connect: use some packages, need learn more
  - database/sql
  - gorp: to map to database
  - github.com/go-sql-driver/mysql: Just registering the driver
- can query from other package

# next

- listen TCP incoming
- handle parse and save to database
