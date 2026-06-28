package main

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"sort"

	_ "github.com/lib/pq"
)

//go:embed schema.html.gohtml
var htmlTemplate string

var tmpl = template.Must(template.New("schema").Parse(htmlTemplate))

type Column struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Nullable bool    `json:"nullable"`
	Default  *string `json:"default,omitempty"`
	PK       bool    `json:"primary_key"`
	Unique   bool    `json:"unique"`
	FK       *FKRef  `json:"foreign_key,omitempty"`
}

type FKRef struct {
	Schema string `json:"schema"`
	Table  string `json:"table"`
	Column string `json:"column"`
}

type Table struct {
	Schema  string     `json:"schema"`
	Name    string     `json:"name"`
	Columns []Column   `json:"columns"`
	Uniques [][]string `json:"unique_constraints,omitempty"` // composite UNIQUE(col, col, ...)
	Checks  []string   `json:"checks,omitempty"`
}

func tableID(schema, name string) string { return schema + "." + name }

// ID and Target are referenced by schema.html.tmpl.
func (t Table) ID() string     { return tableID(t.Schema, t.Name) }
func (f FKRef) Target() string { return tableID(f.Schema, f.Table) }

func introspect(db *sql.DB) ([]Table, error) {
	tables := map[string]*Table{}
	colIdx := map[string]int{} // "schema.table\x00column" -> index

	rows, err := db.Query(`
		SELECT table_schema, table_name, column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema NOT IN ('pg_catalog','information_schema')
		ORDER BY table_schema, table_name, ordinal_position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var sch, tbl, col, typ, nullable string
		var def sql.NullString
		if err := rows.Scan(&sch, &tbl, &col, &typ, &nullable, &def); err != nil {
			return nil, err
		}
		id := tableID(sch, tbl)
		t := tables[id]
		if t == nil {
			t = &Table{Schema: sch, Name: tbl}
			tables[id] = t
		}
		c := Column{Name: col, Type: typ, Nullable: nullable == "YES"}
		if def.Valid {
			c.Default = &def.String
		}
		colIdx[id+"\x00"+col] = len(t.Columns)
		t.Columns = append(t.Columns, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// flag mutates the column referenced by (schema,table,column)
	flagCol := func(sch, tbl, col string, fn func(*Column)) {
		id := tableID(sch, tbl)
		t := tables[id]
		if t == nil {
			return
		}
		if i, ok := colIdx[id+"\x00"+col]; ok {
			fn(&t.Columns[i])
		}
	}

	// primary keys
	pkRows, err := db.Query(`
		SELECT tc.table_schema, tc.table_name, kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'`)
	if err != nil {
		return nil, err
	}
	defer pkRows.Close()
	for pkRows.Next() {
		var sch, tbl, col string
		if err := pkRows.Scan(&sch, &tbl, &col); err != nil {
			return nil, err
		}
		flagCol(sch, tbl, col, func(c *Column) { c.PK = true })
	}
	if err := pkRows.Err(); err != nil {
		return nil, err
	}

	// unique constraints, grouped by constraint so composite keys stay composite.
	// single-column -> column badge; multi-column -> table-level UNIQUE(...).
	uqRows, err := db.Query(`
		SELECT tc.table_schema, tc.table_name, tc.constraint_name, kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'UNIQUE'
		ORDER BY tc.table_schema, tc.table_name, tc.constraint_name, kcu.ordinal_position`)
	if err != nil {
		return nil, err
	}
	defer uqRows.Close()
	type uqKey struct{ sch, tbl, name string }
	uqCols := map[uqKey][]string{}
	var uqOrder []uqKey
	for uqRows.Next() {
		var sch, tbl, name, col string
		if err := uqRows.Scan(&sch, &tbl, &name, &col); err != nil {
			return nil, err
		}
		k := uqKey{sch, tbl, name}
		if _, ok := uqCols[k]; !ok {
			uqOrder = append(uqOrder, k)
		}
		uqCols[k] = append(uqCols[k], col)
	}
	if err := uqRows.Err(); err != nil {
		return nil, err
	}
	for _, k := range uqOrder {
		cols := uqCols[k]
		if len(cols) == 1 {
			flagCol(k.sch, k.tbl, cols[0], func(c *Column) { c.Unique = true })
		} else if t := tables[tableID(k.sch, k.tbl)]; t != nil {
			t.Uniques = append(t.Uniques, cols)
		}
	}

	// foreign keys
	fkRows, err := db.Query(`
		SELECT tc.table_schema, tc.table_name, kcu.column_name,
		       ccu.table_schema, ccu.table_name, ccu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
		  ON ccu.constraint_name = tc.constraint_name AND ccu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'`)
	if err != nil {
		return nil, err
	}
	defer fkRows.Close()
	for fkRows.Next() {
		var sch, tbl, col, fsch, ftbl, fcol string
		if err := fkRows.Scan(&sch, &tbl, &col, &fsch, &ftbl, &fcol); err != nil {
			return nil, err
		}
		flagCol(sch, tbl, col, func(c *Column) {
			c.FK = &FKRef{Schema: fsch, Table: ftbl, Column: fcol}
		})
	}
	if err := fkRows.Err(); err != nil {
		return nil, err
	}

	// named check constraints (skip the implicit NOT NULL ones)
	chkRows, err := db.Query(`
		SELECT tc.table_schema, tc.table_name, cc.check_clause
		FROM information_schema.table_constraints tc
		JOIN information_schema.check_constraints cc
		  ON tc.constraint_name = cc.constraint_name AND tc.table_schema = cc.constraint_schema
		WHERE tc.constraint_type = 'CHECK'
		  AND tc.table_schema NOT IN ('pg_catalog','information_schema')
		  AND cc.check_clause NOT LIKE '%IS NOT NULL'`)
	if err != nil {
		return nil, err
	}
	defer chkRows.Close()
	for chkRows.Next() {
		var sch, tbl, clause string
		if err := chkRows.Scan(&sch, &tbl, &clause); err != nil {
			return nil, err
		}
		if t := tables[tableID(sch, tbl)]; t != nil {
			t.Checks = append(t.Checks, clause)
		}
	}
	if err := chkRows.Err(); err != nil {
		return nil, err
	}

	out := make([]Table, 0, len(tables))
	for _, t := range tables {
		out = append(out, *t)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Schema != out[j].Schema {
			return out[i].Schema < out[j].Schema
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func renderHTML(w io.Writer, tables []Table) error {
	return tmpl.Execute(w, tables)
}

func main() {
	format := flag.String("format", "html", "output format: html or json")
	out := flag.String("o", "", "output file (default {db_name}.{format})")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [-format html|json] [-o file] <postgres-connection-string>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	db, err := sql.Open("postgres", flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		fmt.Fprintln(os.Stderr, "ping:", err)
		os.Exit(1)
	}

	tables, err := introspect(db)
	if err != nil {
		fmt.Fprintln(os.Stderr, "introspect:", err)
		os.Exit(1)
	}

	path := *out
	if path == "" {
		var dbName string
		if err := db.QueryRow("SELECT current_database()").Scan(&dbName); err != nil {
			fmt.Fprintln(os.Stderr, "db name:", err)
			os.Exit(1)
		}
		path = dbName + ".out." + *format
	}
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	defer f.Close()
	w := f

	switch *format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		err = enc.Encode(tables)
	case "html":
		err = renderHTML(w, tables)
	default:
		fmt.Fprintln(os.Stderr, "unknown format:", *format)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
}
