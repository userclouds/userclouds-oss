package main

func main() {
	const q = `SELECT * FROM users WHERE deleted=0::TIMESTAMP;` // want `avoid using SELECT \* because it will fail during a deploy if a column is added`

	const r = `SELECT tbl, version, dsc, up, down, unknown_column FROM migrations WHERE deleted=0::TIMESTAMP;` // want `apparent SQL statement for table migrations contains unsafe column unknown_column`

	const s = `SELECT tbl, dsc, up, down FROM migrations; /* lint-sql-deleted-ok */` // want `missing column version`

	const t = `SELECT tbl, version, dsc, up, down FROM migrations WHERE deleted=0::TIMESTAMP;`
}
