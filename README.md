# skemid

Generate PostgreSQL schema documentation locally. Your schema never leaves your machine.

> Postgres schema docs generated locally — your schema never leaves your machine.

## Why

- Runs entirely on your machine. It connects to your database, reads the schema, and writes a file. No cloud, no account, no external service. Suitable for regulated, air-gapped, or GDPR-constrained environments.
- Zero runtime dependencies in the output. The HTML is a single file with inline CSS and no JavaScript. It makes no network calls when opened.
- One shareable file. Hand the HTML to someone or commit it; it works offline.

![screenshot](<TODO: add screenshot>)

## Modules

skemid is organized into modules. Today there is one:

- **[db-reader](db-reader/)** — the CLI that connects to PostgreSQL, reads the schema, and generates the HTML or JSON output. This is the whole application at the moment.

Future modules (a UI, etc.) are planned but not present yet.

## Install

Requires Go (see [db-reader/go.mod](db-reader/go.mod) for the version). The module lives in [db-reader/](db-reader/).

```
cd db-reader
go build
```

This produces a `db-reader` binary. You can also run it without building via `go run .` from inside `db-reader/`.

## Usage

The connection string is passed as a single positional argument.

```
db-reader [-format html|json] [-o file] <postgres-connection-string>
```

Example:

```
db-reader "postgres://user:password@localhost:5432/mydb?sslmode=disable"
```

Flags:

- `-format` — `html` (default) or `json`.
- `-o` — output file. Defaults to `{db_name}.{format}` (the database's own name, queried via `current_database()`), written to the current directory.

So the example above writes `mydb.html`. To get JSON instead:

```
db-reader -format json "postgres://user:password@localhost:5432/mydb?sslmode=disable"
```

## What it generates

- **HTML**: a sidebar listing every table, and a per-table view of columns with their type, nullability, and default. Primary keys, NOT NULL, and single-column UNIQUE show as inline badges. Foreign keys render as links that jump to the referenced table. Composite UNIQUE constraints and CHECK constraints are listed at table level.
- **JSON**: the same schema model (tables, columns, primary/foreign keys, unique constraints, checks) for processing elsewhere.

## Limitations / Roadmap

- PostgreSQL only. This is v1.

Not implemented yet (no code for these exists today):

- ER diagram.
- Indexes.
- Other databases (MySQL, SQLite, etc.).
- Additional modules such as a UI.

## License

<TODO: license>
