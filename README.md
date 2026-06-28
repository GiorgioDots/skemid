# skemid

Generate PostgreSQL schema documentation locally. Your schema never leaves your machine.

> Postgres schema docs generated locally — your schema never leaves your machine.

## Why

- Runs entirely on your machine. It connects to your database, reads the schema, and writes a file. No cloud, no account, no external service. Suitable for regulated, air-gapped, or GDPR-constrained environments.
- Zero runtime dependencies in the output. The HTML is a single file with inline CSS and no JavaScript. It makes no network calls when opened.
- One shareable file. Hand the HTML to someone or commit it; it works offline.

![screenshot](./docs/screenshots/html-out.png)

## Install

**Prebuilt binary.** Download the binary for your OS/arch from the [latest release](https://github.com/giorgiodots/skemid/releases/latest), then put it on your `PATH`.

**With Go** (requires Go, see [go.mod](go.mod) for the version):

```
go install github.com/giorgiodots/skemid@latest
```

**From source:**

```
go build -o skemid
```

This produces a `skemid` binary. You can also run it without building via `go run .`.

## Usage

The connection string is passed as a single positional argument.

```
skemid [-format html|json] [-o file] <postgres-connection-string>
```

Example:

```
skemid "postgres://user:password@localhost:5432/mydb?sslmode=disable"
```

Flags:

- `-format` — `html` (default) or `json`.
- `-o` — output file. Defaults to `{db_name}.out.{format}` (the database's own name, queried via `current_database()`), written to the current directory.

So the example above writes `mydb.out.html`. To get JSON instead:

```
skemid -format json "postgres://user:password@localhost:5432/mydb?sslmode=disable"
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
