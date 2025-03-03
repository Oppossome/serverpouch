# Serverpouch
A server management tool for the effortless deployment and adminstration of servers. Designed to be simple, easy to use, and efficient.

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. 

### Prerequisites

What things you need to install the software and how to install them

- [Go](https://go.dev/doc/install)
- [Docker](https://docs.docker.com/get-docker/)

### Installing

A step by step series of examples that tell you how to get a development env running

1. Clone the repository
2. Run the following command to start the development database
```bash
docker-compose up -d
```
3. Run the following command to migrate the database
```bash
cd ./tools
go run github.com/rubenv/sql-migrate/sql-migrate up
```
4. Run the following command to start the server
```bash
make dev
```

## Running the tests

Run the following command to run the tests
```bash
make test
```

