package builtins

import (
	"database/sql"
	"fmt"
	"zumbra/object"

	_ "github.com/go-sql-driver/mysql"
)

var db_connection *sql.DB

func MySqlConnectionBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 5 {
				return NewError("wrong number of arguments, mysqlConnection(host, port, user, password, database). got=%d, want=0", len(args))
			}

			if args[0].Type() != object.STRING_OBJ || args[1].Type() != object.STRING_OBJ || args[2].Type() != object.STRING_OBJ || args[3].Type() != object.STRING_OBJ || args[4].Type() != object.STRING_OBJ {
				return NewError("All arguments to `mysqlConnection` must be STRING, got %s", args[0].Type())
			}

			host := args[0].(*object.String).Value
			port := args[1].(*object.String).Value
			user := args[2].(*object.String).Value
			password := args[3].(*object.String).Value
			database := args[4].(*object.String).Value

			var err error
			db_connection, err = sql.Open("mysql", user+":"+password+"@tcp("+host+":"+port+")/"+database)
			if err != nil {
				return NewError("Failed to open database, mysqlConnection('%s', '%s', '%s', '%s', '%s'). got %s", host, port, user, password, database, err)
			}

			err = db_connection.Ping()
			if err != nil {
				return NewError("Failed to ping database, mysqlConnection('%s', '%s', '%s', '%s', '%s'). got %s", host, port, user, password, database, err)
			}

			fmt.Printf("Database '%s' connected successfully\n", database)

			return nil
		},
	}
}

func mysqlCreateTableBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, mysqlCreateTable(tableName, fields). got=%d, want=2", len(args))
			}

			if args[0].Type() != object.STRING_OBJ || args[1].Type() != object.STRING_OBJ {
				return NewError("All arguments to `mysqlCreateTable` must be STRING, got %s", args[0].Type())
			}

			if db_connection == nil {
				return NewError("Database is not connected. Use mysqlConnection(...) before creating tables.")
			}

			tableName := args[0].(*object.String).Value
			fields := args[1].(*object.String).Value

			_, err := db_connection.Exec("CREATE TABLE " + tableName + " (" + fields + ");")
			if err != nil {
				return NewError("Failed to create table, mysqlCreateTable('%s', '%s'). got %s", tableName, fields, err)
			}

			fmt.Printf("Table '%s' created successfully\n", tableName)

			return nil
		},
	}
}

func mysqlShowTablesBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 0 {
				return NewError("wrong number of arguments, mysqlShowTables(). got=%d, want=0", len(args))
			}

			if db_connection == nil {
				return NewError("Database is not connected. Use mysqlConnection(...) before creating tables.")
			}

			rows, err := db_connection.Query("SHOW TABLES")
			if err != nil {
				return NewError("Failed to show tables, mysqlShowTables(). got %s", err)
			}

			var tables []string
			for rows.Next() {
				var table string
				err := rows.Scan(&table)
				if err != nil {
					return NewError("Failed to scan table, mysqlShowTables(). got %s", err)
				}
				tables = append(tables, table)
			}

			elements := []object.Object{}
			for _, table := range tables {
				elements = append(elements, &object.String{Value: table})
			}

			return &object.Array{Elements: elements}
		},
	}
}

func mysqlGetFromTableBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return NewError("wrong number of arguments, mysqlGetFromTable(tableName, fields, condition). got=%d, want=1", len(args))
			}

			if args[0].Type() != object.STRING_OBJ || args[1].Type() != object.STRING_OBJ || args[2].Type() != object.STRING_OBJ {
				return NewError("All arguments to `mysqlGetFromTable` must be STRING, got %s", args[0].Type())
			}

			if db_connection == nil {
				return NewError("Database is not connected. Use mysqlConnection(...) before creating tables.")
			}

			tableName := args[0].(*object.String).Value
			fields := args[1].(*object.String).Value
			condition := " WHERE " + args[2].(*object.String).Value + ";"

			if args[2].(*object.String).Value == "" {
				condition = ";"
			}

			rows, err := db_connection.Query("SELECT " + fields + " FROM " + tableName + condition)
			if err != nil {
				return NewError("Failed to get from table, mysqlGetFromTable('%s', '%s', '%s'). got %s", tableName, fields, condition, err)
			}

			var records []map[string]interface{}
			for rows.Next() {
				record := make(map[string]interface{})
				err := rows.Scan(&record)
				if err != nil {
					return NewError("Failed to scan record, mysqlGetFromTable('%s', '%s', '%s'). got %s", tableName, fields, condition, err)
				}
				records = append(records, record)
			}

			elements := []object.Object{}
			for _, record := range records {
				elements = append(elements, &object.Record{Fields: record})
			}

			return &object.Array{Elements: elements}
		}}
}
