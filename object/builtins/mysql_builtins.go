package builtins

import (
	"database/sql"
	"fmt"
	"strings"
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

func mysqlShowTableColumnsBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments, mysqlShowTableColumns(tableName). got=%d, want=1", len(args))
			}

			if args[0].Type() != object.STRING_OBJ {
				return NewError("All arguments to `mysqlShowTableColumns` must be STRING, got %s", args[0].Type())
			}

			if db_connection == nil {
				return NewError("Database is not connected. Use mysqlConnection(...) before creating tables.")
			}

			tableName := args[0].(*object.String).Value

			rows, err := db_connection.Query("SHOW COLUMNS FROM " + tableName)
			if err != nil {
				return NewError("Failed to show table columns, mysqlShowTableColumns('%s'). got %s", tableName, err)
			}

			var columns []string
			var (
				field, columnType, null, key, extra string
				defaultValue                        sql.NullString
			)

			for rows.Next() {
				err := rows.Scan(&field, &columnType, &null, &key, &defaultValue, &extra)
				if err != nil {
					return NewError("Failed to scan column, mysqlShowTableColumns('%s'). got %s", tableName, err)
				}
				columns = append(columns, field)
			}

			elements := []object.Object{}
			for _, column := range columns {
				elements = append(elements, &object.String{Value: column})
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

			columns, err := rows.Columns()
			if err != nil {
				return NewError("Failed to get columns from result set: %s", err)
			}

			var records []map[string]interface{}

			for rows.Next() {
				values := make([]interface{}, len(columns))
				valuePtrs := make([]interface{}, len(columns))
				for i := range values {
					valuePtrs[i] = &values[i]
				}

				if err := rows.Scan(valuePtrs...); err != nil {
					return NewError("Failed to scan row: %s", err)
				}

				record := make(map[string]interface{})
				for i, col := range columns {
					var v interface{}
					val := values[i]

					b, ok := val.([]byte)
					if ok {
						v = string(b)
					} else {
						v = val
					}

					record[col] = v
				}

				records = append(records, record)
			}

			elements := []object.Object{}
			for _, record := range records {
				pairs := map[object.DictKey]object.DictPair{}
				for key, val := range record {
					keyObj := &object.String{Value: key}
					pairs[keyObj.DictKey()] = object.DictPair{
						Key:   keyObj,
						Value: objectFromGoValue(val),
					}
				}
				elements = append(elements, &object.Dict{Pairs: pairs})
			}

			return &object.Array{Elements: elements}
		},
	}
}

func mysqlInsertIntoTableBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, mysqlInsertIntoTable(tableName, dict). got=%d, want=2", len(args))
			}

			if args[0].Type() != object.STRING_OBJ {
				return NewError("First argument to `mysqlInsertIntoTable` must be STRING, got %s", args[0].Type())
			}

			if args[1].Type() != object.DICT_OBJ {
				return NewError("Second argument to `mysqlInsertIntoTable` must be a DICT, got %s", args[1].Type())
			}

			if db_connection == nil {
				return NewError("Database is not connected. Use mysqlConnection(...) before creating tables.")
			}

			tableName := args[0].(*object.String).Value
			dict := args[1].(*object.Dict)

			keys := []string{}
			placeholders := []string{}
			argsValues := []interface{}{}

			for _, pair := range dict.Pairs {
				keys = append(keys, pair.Key.Inspect())
				placeholders = append(placeholders, "?")
				argsValues = append(argsValues, goValueFromObject(pair.Value))
			}

			query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);", tableName, strings.Join(keys, ","), strings.Join(placeholders, ","))

			_, err := db_connection.Exec(query, argsValues...)
			if err != nil {
				return NewError("Failed to insert into table, mysqlInsertIntoTable('%s', '%v'). got %s", tableName, dict.Inspect(), err)
			}

			fmt.Println("Record inserted successfully")
			return nil
		},
	}
}

func mysqlUpdateIntoTableBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return nil
		},
	}
}

func goValueFromObject(obj object.Object) interface{} {
	switch v := obj.(type) {
	case *object.String:
		return v.Value
	case *object.Integer:
		return v.Value
	case *object.Boolean:
		return v.Value
	default:
		return v.Inspect()
	}
}

func objectFromGoValue(v interface{}) object.Object {
	switch val := v.(type) {
	case string:
		return &object.String{Value: val}
	case int64:
		return &object.Integer{Value: val}
	case int:
		return &object.Integer{Value: int64(val)}
	case float64:
		return &object.Float{Value: val}
	case bool:
		return &object.Boolean{Value: val}
	case nil:
		return nil
	default:
		return &object.String{Value: fmt.Sprintf("%v", val)}
	}
}
