# Serverpouch Development Tools
This directory contains tools that are used in the development of the serverpouch project. The tools are used to format code, generate code, manage database migrations, and generate mock implementations of interfaces.

## gofumpt
[gofumpt](https://github.com/mvdan/gofumpt) is a tool that formats Go code. It is used to format the Go code in the serverpouch project.

To install gofumpt, use the following command:
```bash
go install mvdan.cc/gofumpt@latest
```

To run gofumpt, use the following command while in the main directory:
```bash
gofumpt -l -w . 
```

If you'd like to configure gofumpt to run when you save, see this [gofumpt README](https://github.com/mvdan/gofumpt?tab=readme-ov-file#installation) for more information.

## sqlc
sqlc is a tool that generates Go code from SQL queries. It is used to create type safe Go code from SQL queries. It is used to generate the code for the database queries in the serverpouch project.

To run sqlc, use the following command while in the tools directory:
```bash
go run github.com/sqlc-dev/sqlc/cmd/sqlc 
```

Alternatively, you may run the following command to execute all go:generate commands in the tools directory:
```bash
go generate .
```

## sql-migrate
[sql-migrate](https://github.com/rubenv/sql-migrate) is a tool that manages SQL database migrations. It is used to create and run migrations for the serverpouch project.

To run sql-migrate, use the following command while in the tools directory:
```bash
go run github.com/rubenv/sql-migrate/sql-migrate 
```

### Creating a new migration

To create a new migration, use the following command:
```bash
go run github.com/rubenv/sql-migrate/sql-migrate new <migration_name>
```

This will create a new migration file in the migrations directory. The file will contain an up and down function that will be used to apply and rollback the migration.

### Running migrations

To run all pending migrations, use the following command:
```bash
go run github.com/rubenv/sql-migrate/sql-migrate up
```

## mockery
[mockery](https://github.com/vektra/mockery) is a tool that generates mock implementations of Go interfaces. It is used to create mock implementations of interfaces for testing in the serverpouch project.

To run mockery, use the following command while in the tools directory:
```bash
go generate .
```

This will generate mock implementations of the interfaces specified in the `.mockery.yml` file.
