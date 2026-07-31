#if defined(ZUMBRA_ENABLE_SQLITE)

#include <limits.h>
#include <sqlite3.h>

/* Z12.1 SQLite runtime. Values are bound through sqlite3_bind_* only. */
typedef struct ZSQLiteDatabaseNative ZSQLiteDatabaseNative;
typedef struct ZSQLiteTransactionNative ZSQLiteTransactionNative;

typedef struct {
    ZValue receiver;
    char *name;
} ZSQLiteBoundMethodNative;

struct ZSQLiteDatabaseNative {
    sqlite3 *database;
    pthread_mutex_t mutex;
    bool open;
    bool transaction_active;
    char *path;
    struct ZSQLiteDatabaseNative *next;
};

typedef struct {
    ZSQLiteDatabaseNative *database;
    ZSQLiteTransactionNative *transaction;
    sqlite3_stmt *statement;
    bool open;
    char *sql;
} ZSQLiteStatementNative;

typedef struct {
    ZSQLiteDatabaseNative *database;
    sqlite3_stmt *statement;
    bool owns_statement;
    bool open;
    ZValue columns;
} ZSQLiteRowsNative;

struct ZSQLiteTransactionNative {
    ZSQLiteDatabaseNative *database;
    bool active;
};

static ZSQLiteDatabaseNative *z_sqlite_databases = NULL;
static pthread_mutex_t z_sqlite_registry_mutex = PTHREAD_MUTEX_INITIALIZER;

static ZValue z_sqlite_value(ZTag tag, ZKind kind, void *pointer) {
    ZValue result = z_value(tag, kind);
    result.as.p = pointer;
    return result;
}

static ZSQLiteDatabaseNative *z_sqlite_expect_database(ZValue value) {
    if (value.tag != ZV_SQLITE_DATABASE || value.as.p == NULL) z_fatal("expected SQLiteDatabase");
    return (ZSQLiteDatabaseNative *)value.as.p;
}

static ZSQLiteStatementNative *z_sqlite_expect_statement(ZValue value) {
    if (value.tag != ZV_SQLITE_STATEMENT || value.as.p == NULL) z_fatal("expected SQLiteStatement");
    return (ZSQLiteStatementNative *)value.as.p;
}

static ZSQLiteTransactionNative *z_sqlite_expect_transaction(ZValue value) {
    if (value.tag != ZV_SQLITE_TRANSACTION || value.as.p == NULL) z_fatal("expected SQLiteTransaction");
    return (ZSQLiteTransactionNative *)value.as.p;
}

static void z_sqlite_check_database(ZSQLiteDatabaseNative *database) {
    if (database == NULL || !database->open || database->database == NULL) z_fatal("SQLite database is closed");
}

static void z_sqlite_fail(ZSQLiteDatabaseNative *database, const char *operation, int code) {
    const char *message = database != NULL && database->database != NULL ? sqlite3_errmsg(database->database) : sqlite3_errstr(code);
    z_fatal("%s: %s", operation, message == NULL ? "unknown SQLite error" : message);
}

static ZValue z_sqlite_exec_result(ZSQLiteDatabaseNative *database) {
    ZValue pairs[2];
    pairs[0] = z_pair(z_string("lastInsertId"), z_int((int64_t)sqlite3_last_insert_rowid(database->database)));
    pairs[1] = z_pair(z_string("rowsAffected"), z_int((int64_t)sqlite3_changes64(database->database)));
    return z_dict_from(pairs, 2);
}

static void z_sqlite_bind_value(sqlite3_stmt *statement, int index, ZValue value) {
    int code = SQLITE_OK;
    switch (value.tag) {
    case ZV_NULL:
        code = sqlite3_bind_null(statement, index);
        break;
    case ZV_BOOL:
        code = sqlite3_bind_int64(statement, index, value.as.b ? 1 : 0);
        break;
    case ZV_INT:
        code = sqlite3_bind_int64(statement, index, value.as.i);
        break;
    case ZV_UINT:
        if (value.as.u > INT64_MAX) z_fatal("SQLite parameter %d exceeds signed INTEGER range", index);
        code = sqlite3_bind_int64(statement, index, (sqlite3_int64)value.as.u);
        break;
    case ZV_FLOAT:
        code = sqlite3_bind_double(statement, index, value.as.f);
        break;
    case ZV_STRING:
        code = sqlite3_bind_text(statement, index, value.as.s, -1, SQLITE_TRANSIENT);
        break;
    case ZV_BUFFER: {
        size_t byte_length = value.as.buffer->len * value.as.buffer->elem_size;
        if (byte_length > INT_MAX) z_fatal("SQLite parameter %d exceeds the maximum blob size supported by this runtime", index);
        code = byte_length == 0
            ? sqlite3_bind_zeroblob(statement, index, 0)
            : sqlite3_bind_blob(statement, index, value.as.buffer->data,
                                (int)byte_length, SQLITE_TRANSIENT);
        break;
    }
    default:
        z_fatal("unsupported SQLite parameter type at position %d", index);
    }
    if (code != SQLITE_OK) z_fatal("could not bind SQLite parameter %d: %s", index, sqlite3_errstr(code));
}

static void z_sqlite_bind_params(sqlite3_stmt *statement, ZValue params) {
    int expected = sqlite3_bind_parameter_count(statement);
    sqlite3_reset(statement);
    sqlite3_clear_bindings(statement);
    if (params.tag == ZV_ARRAY) {
        if ((int)params.as.array->len != expected) {
            z_fatal("SQLite statement expects %d parameters, got %zu", expected, params.as.array->len);
        }
        for (size_t index = 0; index < params.as.array->len; index++) {
            z_sqlite_bind_value(statement, (int)index + 1, params.as.array->items[index]);
        }
        return;
    }
    if (params.tag == ZV_DICT) {
        if ((int)params.as.dict->len != expected) {
            z_fatal("SQLite statement expects %d named parameters, got %zu", expected, params.as.dict->len);
        }
        for (int index = 1; index <= expected; index++) {
            const char *parameter = sqlite3_bind_parameter_name(statement, index);
            if (parameter == NULL || (*parameter != ':' && *parameter != '$' && *parameter != '@')) {
                z_fatal("SQLite parameter %d is positional; use an array", index);
            }
            ZValue value;
            bool found = z_z12_dict_find(params, parameter, &value);
            if (!found) found = z_z12_dict_find(params, parameter + 1, &value);
            if (!found) z_fatal("missing SQLite named parameter %s", parameter);
            z_sqlite_bind_value(statement, index, value);
        }
        return;
    }
    z_fatal("SQLite parameters must be an array or dictionary");
}

static sqlite3_stmt *z_sqlite_prepare_locked(ZSQLiteDatabaseNative *database, const char *sql) {
    z_sqlite_check_database(database);
    sqlite3_stmt *statement = NULL;
    int code = sqlite3_prepare_v2(database->database, sql, -1, &statement, NULL);
    if (code != SQLITE_OK) z_sqlite_fail(database, "prepare SQLite statement", code);
    if (statement == NULL) z_fatal("SQLite query is empty");
    return statement;
}

static ZValue z_sqlite_column_value(sqlite3_stmt *statement, int column) {
    switch (sqlite3_column_type(statement, column)) {
    case SQLITE_INTEGER:
        return z_int((int64_t)sqlite3_column_int64(statement, column));
    case SQLITE_FLOAT:
        return z_float(sqlite3_column_double(statement, column));
    case SQLITE_TEXT:
        return z_string((const char *)sqlite3_column_text(statement, column));
    case SQLITE_BLOB: {
        int length = sqlite3_column_bytes(statement, column);
        ZValue bytes = z_bytes((size_t)length);
        if (length > 0) memcpy(bytes.as.buffer->data, sqlite3_column_blob(statement, column), (size_t)length);
        return bytes;
    }
    default:
        return z_null();
    }
}

static ZValue z_sqlite_query_statement_locked(ZSQLiteDatabaseNative *database, sqlite3_stmt *statement, ZValue params) {
    z_sqlite_bind_params(statement, params);
    ZValue rows = z_array_from(NULL, 0);
    int columns = sqlite3_column_count(statement);
    for (;;) {
        int code = sqlite3_step(statement);
        if (code == SQLITE_DONE) break;
        if (code != SQLITE_ROW) z_sqlite_fail(database, "query SQLite statement", code);
        ZValue *pairs = (ZValue *)z_alloc(sizeof(ZValue) * (size_t)columns);
        for (int column = 0; column < columns; column++) {
            pairs[column] = z_pair(z_string(sqlite3_column_name(statement, column)), z_sqlite_column_value(statement, column));
        }
        ZValue row = z_dict_from(pairs, (size_t)columns);
        ZArray *array = rows.as.array;
        if (array->len == array->cap) {
            size_t capacity = array->cap == 0 ? 4 : array->cap * 2;
            ZValue *items = (ZValue *)z_alloc(sizeof(ZValue) * capacity);
            if (array->len > 0) memcpy(items, array->items, sizeof(ZValue) * array->len);
            array->items = items;
            array->cap = capacity;
        }
        array->items[array->len++] = row;
    }
    return rows;
}

static ZValue z_sqlite_exec_statement_locked(ZSQLiteDatabaseNative *database, sqlite3_stmt *statement, ZValue params) {
    z_sqlite_bind_params(statement, params);
    int code = sqlite3_step(statement);
    if (code == SQLITE_ROW) z_fatal("SQLite statement returned rows; use query instead of exec");
    if (code != SQLITE_DONE) z_sqlite_fail(database, "execute SQLite statement", code);
    return z_sqlite_exec_result(database);
}

static ZValue z_sqlite_open(const char *path) {
    sqlite3 *handle = NULL;
    int flags = SQLITE_OPEN_READWRITE | SQLITE_OPEN_CREATE | SQLITE_OPEN_FULLMUTEX | SQLITE_OPEN_URI;
    int code = sqlite3_open_v2(path, &handle, flags, NULL);
    if (code != SQLITE_OK) {
        const char *source = handle != NULL ? sqlite3_errmsg(handle) : sqlite3_errstr(code);
        char *message = z_strdup(source == NULL ? "unknown SQLite error" : source);
        if (handle != NULL) sqlite3_close_v2(handle);
        z_fatal("open SQLite database %s: %s", path, message);
    }
    sqlite3_extended_result_codes(handle, 1);
    sqlite3_busy_timeout(handle, 5000);
    ZSQLiteDatabaseNative *database = (ZSQLiteDatabaseNative *)z_alloc(sizeof(ZSQLiteDatabaseNative));
    memset(database, 0, sizeof(*database));
    database->database = handle;
    database->open = true;
    database->path = z_strdup(path);
    pthread_mutex_init(&database->mutex, NULL);
    pthread_mutex_lock(&z_sqlite_registry_mutex);
    database->next = z_sqlite_databases;
    z_sqlite_databases = database;
    pthread_mutex_unlock(&z_sqlite_registry_mutex);
    return z_sqlite_value(ZV_SQLITE_DATABASE, ZK_SQLITE_DATABASE, database);
}

static bool z_sqlite_close_database(ZSQLiteDatabaseNative *database) {
    pthread_mutex_lock(&database->mutex);
    if (!database->open) { pthread_mutex_unlock(&database->mutex); return true; }
    if (database->transaction_active) { pthread_mutex_unlock(&database->mutex); z_fatal("cannot close SQLite database with an active transaction"); }
    int code = sqlite3_close_v2(database->database);
    if (code != SQLITE_OK) { pthread_mutex_unlock(&database->mutex); z_sqlite_fail(database, "close SQLite database", code); }
    database->database = NULL;
    database->open = false;
    pthread_mutex_unlock(&database->mutex);
    return true;
}

static ZValue z_sqlite_database_exec(ZSQLiteDatabaseNative *database, const char *sql, ZValue params, bool transaction_call) {
    pthread_mutex_lock(&database->mutex);
    z_sqlite_check_database(database);
    if (database->transaction_active && !transaction_call) { pthread_mutex_unlock(&database->mutex); z_fatal("SQLite database has an active transaction; use the transaction handle"); }
    sqlite3_stmt *statement = z_sqlite_prepare_locked(database, sql);
    ZValue result = z_sqlite_exec_statement_locked(database, statement, params);
    sqlite3_finalize(statement);
    pthread_mutex_unlock(&database->mutex);
    return result;
}

static ZValue z_sqlite_database_query(ZSQLiteDatabaseNative *database, const char *sql, ZValue params, bool transaction_call) {
    pthread_mutex_lock(&database->mutex);
    z_sqlite_check_database(database);
    if (database->transaction_active && !transaction_call) { pthread_mutex_unlock(&database->mutex); z_fatal("SQLite database has an active transaction; use the transaction handle"); }
    sqlite3_stmt *statement = z_sqlite_prepare_locked(database, sql);
    ZValue result = z_sqlite_query_statement_locked(database, statement, params);
    sqlite3_finalize(statement);
    pthread_mutex_unlock(&database->mutex);
    return result;
}

static ZValue z_sqlite_prepare_value(ZSQLiteDatabaseNative *database, ZSQLiteTransactionNative *transaction, const char *sql) {
    pthread_mutex_lock(&database->mutex);
    z_sqlite_check_database(database);
    if (database->transaction_active && transaction == NULL) { pthread_mutex_unlock(&database->mutex); z_fatal("SQLite database has an active transaction; prepare through the transaction handle"); }
    sqlite3_stmt *statement = z_sqlite_prepare_locked(database, sql);
    ZSQLiteStatementNative *native = (ZSQLiteStatementNative *)z_alloc(sizeof(ZSQLiteStatementNative));
    native->database = database;
    native->transaction = transaction;
    native->statement = statement;
    native->open = true;
    native->sql = z_strdup(sql);
    pthread_mutex_unlock(&database->mutex);
    return z_sqlite_value(ZV_SQLITE_STATEMENT, ZK_SQLITE_STATEMENT, native);
}

static void z_sqlite_check_statement(ZSQLiteStatementNative *statement) {
    if (statement == NULL || !statement->open || statement->statement == NULL) z_fatal("SQLite statement is closed");
    z_sqlite_check_database(statement->database);
    if (statement->transaction != NULL && !statement->transaction->active) z_fatal("SQLite transaction is no longer active");
    if (statement->transaction == NULL && statement->database->transaction_active) z_fatal("SQLite database has an active transaction; use a transaction statement");
}

static ZValue z_sqlite_statement_exec(ZSQLiteStatementNative *statement, ZValue params) {
    pthread_mutex_lock(&statement->database->mutex);
    z_sqlite_check_statement(statement);
    ZValue result = z_sqlite_exec_statement_locked(statement->database, statement->statement, params);
    pthread_mutex_unlock(&statement->database->mutex);
    return result;
}

static ZValue z_sqlite_statement_query(ZSQLiteStatementNative *statement, ZValue params) {
    pthread_mutex_lock(&statement->database->mutex);
    z_sqlite_check_statement(statement);
    ZValue result = z_sqlite_query_statement_locked(statement->database, statement->statement, params);
    pthread_mutex_unlock(&statement->database->mutex);
    return result;
}

static bool z_sqlite_statement_close(ZSQLiteStatementNative *statement) {
    pthread_mutex_lock(&statement->database->mutex);
    if (!statement->open) { pthread_mutex_unlock(&statement->database->mutex); return true; }
    int code = sqlite3_finalize(statement->statement);
    statement->statement = NULL;
    statement->open = false;
    pthread_mutex_unlock(&statement->database->mutex);
    if (code != SQLITE_OK) z_sqlite_fail(statement->database, "close SQLite statement", code);
    return true;
}

static ZValue z_sqlite_begin(ZSQLiteDatabaseNative *database) {
    pthread_mutex_lock(&database->mutex);
    z_sqlite_check_database(database);
    if (database->transaction_active) { pthread_mutex_unlock(&database->mutex); z_fatal("SQLite database already has an active transaction"); }
    char *error = NULL;
    int code = sqlite3_exec(database->database, "BEGIN IMMEDIATE", NULL, NULL, &error);
    if (code != SQLITE_OK) {
        const char *message = error == NULL ? sqlite3_errmsg(database->database) : error;
        pthread_mutex_unlock(&database->mutex);
        z_fatal("begin SQLite transaction: %s", message);
    }
    if (error != NULL) sqlite3_free(error);
    database->transaction_active = true;
    ZSQLiteTransactionNative *transaction = (ZSQLiteTransactionNative *)z_alloc(sizeof(ZSQLiteTransactionNative));
    transaction->database = database;
    transaction->active = true;
    pthread_mutex_unlock(&database->mutex);
    return z_sqlite_value(ZV_SQLITE_TRANSACTION, ZK_SQLITE_TRANSACTION, transaction);
}

static bool z_sqlite_finish_transaction(ZSQLiteTransactionNative *transaction, const char *command) {
    ZSQLiteDatabaseNative *database = transaction->database;
    pthread_mutex_lock(&database->mutex);
    if (!transaction->active) { pthread_mutex_unlock(&database->mutex); z_fatal("SQLite transaction is no longer active"); }
    z_sqlite_check_database(database);
    char *error = NULL;
    int code = sqlite3_exec(database->database, command, NULL, NULL, &error);
    if (code != SQLITE_OK) {
        const char *message = error == NULL ? sqlite3_errmsg(database->database) : error;
        pthread_mutex_unlock(&database->mutex);
        z_fatal("finish SQLite transaction: %s", message);
    }
    if (error != NULL) sqlite3_free(error);
    transaction->active = false;
    database->transaction_active = false;
    pthread_mutex_unlock(&database->mutex);
    return true;
}


static ZValue z_sqlite_rows_next_native(void *raw, bool *ok) {
    ZSQLiteRowsNative *rows = (ZSQLiteRowsNative *)raw;
    pthread_mutex_lock(&rows->database->mutex);
    if (!rows->open || rows->statement == NULL) {
        pthread_mutex_unlock(&rows->database->mutex);
        *ok = false;
        return z_null();
    }
    int code = sqlite3_step(rows->statement);
    if (code == SQLITE_DONE) {
        rows->open = false;
        if (rows->owns_statement) { sqlite3_finalize(rows->statement); rows->statement = NULL; }
        else { sqlite3_reset(rows->statement); sqlite3_clear_bindings(rows->statement); }
        pthread_mutex_unlock(&rows->database->mutex);
        *ok = false;
        return z_null();
    }
    if (code != SQLITE_ROW) {
        pthread_mutex_unlock(&rows->database->mutex);
        z_sqlite_fail(rows->database, "stream SQLite row", code);
    }
    int columns = sqlite3_column_count(rows->statement);
    ZValue *pairs = columns == 0 ? NULL : (ZValue *)z_alloc(sizeof(ZValue) * (size_t)columns);
    for (int column = 0; column < columns; column++) pairs[column] = z_pair(z_string(sqlite3_column_name(rows->statement, column)), z_sqlite_column_value(rows->statement, column));
    ZValue result = z_dict_from(pairs, (size_t)columns);
    pthread_mutex_unlock(&rows->database->mutex);
    *ok = true;
    return result;
}
static ZValue z_sqlite_rows_columns_native(void *raw) { return ((ZSQLiteRowsNative *)raw)->columns; }
static void z_sqlite_rows_close_native(void *raw) {
    ZSQLiteRowsNative *rows = (ZSQLiteRowsNative *)raw;
    pthread_mutex_lock(&rows->database->mutex);
    if (rows->open && rows->statement != NULL) {
        if (rows->owns_statement) sqlite3_finalize(rows->statement);
        else { sqlite3_reset(rows->statement); sqlite3_clear_bindings(rows->statement); }
    }
    rows->statement = NULL;
    rows->open = false;
    pthread_mutex_unlock(&rows->database->mutex);
}
static ZValue z_sqlite_stream_from_statement(ZSQLiteDatabaseNative *database, sqlite3_stmt *statement, ZValue params, bool owns_statement) {
    z_sqlite_bind_params(statement, params);
    int count = sqlite3_column_count(statement);
    ZValue *names = count == 0 ? NULL : (ZValue *)z_alloc(sizeof(ZValue) * (size_t)count);
    for (int column = 0; column < count; column++) names[column] = z_string(sqlite3_column_name(statement, column));
    ZSQLiteRowsNative *rows = (ZSQLiteRowsNative *)z_alloc(sizeof(ZSQLiteRowsNative));
    rows->database = database; rows->statement = statement; rows->owns_statement = owns_statement; rows->open = true; rows->columns = z_array_from(names, (size_t)count);
    return z_z12_rows_new(rows, z_sqlite_rows_next_native, z_sqlite_rows_columns_native, z_sqlite_rows_close_native);
}
static ZValue z_sqlite_database_stream(ZSQLiteDatabaseNative *database, const char *sql, ZValue params, bool transaction_call) {
    pthread_mutex_lock(&database->mutex);
    z_sqlite_check_database(database);
    if (database->transaction_active && !transaction_call) { pthread_mutex_unlock(&database->mutex); z_fatal("SQLite database has an active transaction; use the transaction handle"); }
    sqlite3_stmt *statement = z_sqlite_prepare_locked(database, sql);
    ZValue result = z_sqlite_stream_from_statement(database, statement, params, true);
    pthread_mutex_unlock(&database->mutex);
    return result;
}
static ZValue z_sqlite_statement_stream(ZSQLiteStatementNative *statement, ZValue params) {
    pthread_mutex_lock(&statement->database->mutex);
    z_sqlite_check_statement(statement);
    ZValue result = z_sqlite_stream_from_statement(statement->database, statement->statement, params, false);
    pthread_mutex_unlock(&statement->database->mutex);
    return result;
}
static bool z_sqlite_safe_identifier(const char *name) {
    if (name == NULL || *name == '\0') return false;
    for (const char *cursor = name; *cursor != '\0'; cursor++) if (!isalnum((unsigned char)*cursor) && *cursor != '_') return false;
    return true;
}
static bool z_sqlite_transaction_control(ZSQLiteTransactionNative *transaction, const char *prefix, const char *name) {
    if (!z_sqlite_safe_identifier(name)) z_fatal("invalid SQLite savepoint name");
    ZZ12Buffer sql; z_z12_buffer_init(&sql); z_z12_buffer_append(&sql, prefix); z_z12_buffer_append(&sql, name);
    ZSQLiteDatabaseNative *database = transaction->database;
    pthread_mutex_lock(&database->mutex);
    if (!transaction->active || !database->transaction_active) { pthread_mutex_unlock(&database->mutex); z_z12_buffer_free(&sql); z_fatal("SQLite transaction is no longer active"); }
    char *error = NULL; int code = sqlite3_exec(database->database, sql.data, NULL, NULL, &error);
    if (code != SQLITE_OK) { const char *message = error == NULL ? sqlite3_errmsg(database->database) : error; pthread_mutex_unlock(&database->mutex); z_z12_buffer_free(&sql); z_fatal("SQLite savepoint: %s", message); }
    if (error != NULL) sqlite3_free(error);
    pthread_mutex_unlock(&database->mutex); z_z12_buffer_free(&sql); return true;
}
static int64_t z_sqlite_schema_version(ZSQLiteDatabaseNative *database) {
    pthread_mutex_lock(&database->mutex);
    z_sqlite_check_database(database);
    char *error = NULL;
    int code = sqlite3_exec(database->database, "CREATE TABLE IF NOT EXISTS _zumbra_migrations(version INTEGER PRIMARY KEY,name TEXT NOT NULL,applied_at INTEGER NOT NULL)", NULL, NULL, &error);
    if (code != SQLITE_OK) { const char *message=error==NULL?sqlite3_errmsg(database->database):error; pthread_mutex_unlock(&database->mutex); z_fatal("initialize migrations: %s", message); }
    if (error != NULL) sqlite3_free(error);
    sqlite3_stmt *statement = NULL; sqlite3_prepare_v2(database->database, "SELECT COALESCE(MAX(version),0) FROM _zumbra_migrations", -1, &statement, NULL);
    int64_t result = 0; if (sqlite3_step(statement) == SQLITE_ROW) result = sqlite3_column_int64(statement, 0); sqlite3_finalize(statement);
    pthread_mutex_unlock(&database->mutex); return result;
}
static int64_t z_sqlite_migrate(ZSQLiteDatabaseNative *database, ZValue migrations) {
    if (migrations.tag != ZV_ARRAY) z_fatal("SQLite migrations must be array");
    int64_t current = z_sqlite_schema_version(database), applied = 0, previous = current;
    pthread_mutex_lock(&database->mutex);
    char *error = NULL; if (sqlite3_exec(database->database, "BEGIN IMMEDIATE", NULL, NULL, &error) != SQLITE_OK) { const char *message=error==NULL?sqlite3_errmsg(database->database):error; pthread_mutex_unlock(&database->mutex); z_fatal("begin migrations: %s", message); }
    for (size_t index = 0; index < migrations.as.array->len; index++) {
        ZValue migration = migrations.as.array->items[index]; if (migration.tag != ZV_DICT) { sqlite3_exec(database->database,"ROLLBACK",NULL,NULL,NULL); pthread_mutex_unlock(&database->mutex); z_fatal("migration must be dictionary"); }
        ZValue version_value, name_value, sql_value;
        if (!z_z12_dict_find(migration,"version",&version_value) || !z_z12_dict_find(migration,"name",&name_value) || !z_z12_dict_find(migration,"sql",&sql_value)) { sqlite3_exec(database->database,"ROLLBACK",NULL,NULL,NULL); pthread_mutex_unlock(&database->mutex); z_fatal("migration requires version, name and sql"); }
        int64_t version = z_as_i64(version_value); if (version <= previous && version > current) { sqlite3_exec(database->database,"ROLLBACK",NULL,NULL,NULL); pthread_mutex_unlock(&database->mutex); z_fatal("migration versions must be strictly increasing"); }
        if (version <= current) continue;
        if (name_value.tag != ZV_STRING || sql_value.tag != ZV_STRING) { sqlite3_exec(database->database,"ROLLBACK",NULL,NULL,NULL); pthread_mutex_unlock(&database->mutex); z_fatal("migration name and sql must be strings"); }
        if (sqlite3_exec(database->database, sql_value.as.s, NULL, NULL, &error) != SQLITE_OK) { const char *message=error==NULL?sqlite3_errmsg(database->database):error; sqlite3_exec(database->database,"ROLLBACK",NULL,NULL,NULL); pthread_mutex_unlock(&database->mutex); z_fatal("apply migration %s: %s", name_value.as.s, message); }
        sqlite3_stmt *record=NULL; sqlite3_prepare_v2(database->database,"INSERT INTO _zumbra_migrations(version,name,applied_at) VALUES(?,?,?)",-1,&record,NULL); sqlite3_bind_int64(record,1,version);sqlite3_bind_text(record,2,name_value.as.s,-1,SQLITE_TRANSIENT);sqlite3_bind_int64(record,3,z_z12_now_ms());if(sqlite3_step(record)!=SQLITE_DONE){sqlite3_finalize(record);sqlite3_exec(database->database,"ROLLBACK",NULL,NULL,NULL);pthread_mutex_unlock(&database->mutex);z_fatal("record migration: %s",sqlite3_errmsg(database->database));}sqlite3_finalize(record);previous=version;applied++;
    }
    if (sqlite3_exec(database->database,"COMMIT",NULL,NULL,&error)!=SQLITE_OK){const char*message=error==NULL?sqlite3_errmsg(database->database):error;sqlite3_exec(database->database,"ROLLBACK",NULL,NULL,NULL);pthread_mutex_unlock(&database->mutex);z_fatal("commit migrations: %s",message);}pthread_mutex_unlock(&database->mutex);return applied;
}

static ZValue z_sqlite_bound_method(ZValue receiver, const char *name) {
    ZSQLiteBoundMethodNative *method = (ZSQLiteBoundMethodNative *)z_alloc(sizeof(ZSQLiteBoundMethodNative));
    method->receiver = receiver;
    method->name = z_strdup(name);
    return z_sqlite_value(ZV_SQLITE_METHOD, ZK_FUNCTION, method);
}

static ZValue z_sqlite_call_method(ZValue callable, const ZValue *args, size_t argc) {
    ZSQLiteBoundMethodNative *method = (ZSQLiteBoundMethodNative *)callable.as.p;
    if (argc > 3) z_fatal("SQLite method %s received too many arguments", method->name);
    ZValue forwarded[4];
    forwarded[0] = method->receiver;
    for (size_t index = 0; index < argc; index++) forwarded[index + 1] = args[index];
    const char *builtin = NULL;
    if (method->receiver.tag == ZV_SQLITE_DATABASE) {
        if (strcmp(method->name, "exec") == 0) builtin = "sqliteExec";
        else if (strcmp(method->name, "query") == 0) builtin = "sqliteQuery";
        else if (strcmp(method->name, "queryOne") == 0) builtin = "sqliteQueryOne";
        else if (strcmp(method->name, "stream") == 0) builtin = "sqliteQueryStream";
        else if (strcmp(method->name, "migrate") == 0) builtin = "sqliteMigrate";
        else if (strcmp(method->name, "schemaVersion") == 0) builtin = "sqliteSchemaVersion";
        else if (strcmp(method->name, "prepare") == 0) builtin = "sqlitePrepare";
        else if (strcmp(method->name, "begin") == 0) builtin = "sqliteBegin";
        else if (strcmp(method->name, "close") == 0) builtin = "sqliteClose";
        else if (strcmp(method->name, "isOpen") == 0) builtin = "sqliteIsOpen";
        else if (strcmp(method->name, "path") == 0) builtin = "sqlitePath";
    } else if (method->receiver.tag == ZV_SQLITE_STATEMENT) {
        if (strcmp(method->name, "exec") == 0) builtin = "sqliteStatementExec";
        else if (strcmp(method->name, "query") == 0) builtin = "sqliteStatementQuery";
        else if (strcmp(method->name, "stream") == 0) builtin = "sqliteStatementQueryStream";
        else if (strcmp(method->name, "parameterCount") == 0) builtin = "sqliteStatementParameterCount";
        else if (strcmp(method->name, "columns") == 0) builtin = "sqliteStatementColumns";
        else if (strcmp(method->name, "close") == 0) builtin = "sqliteStatementClose";
        else if (strcmp(method->name, "isOpen") == 0) builtin = "sqliteStatementOpen";
        else if (strcmp(method->name, "sql") == 0) builtin = "sqliteStatementSQL";
    } else if (method->receiver.tag == ZV_SQLITE_TRANSACTION) {
        if (strcmp(method->name, "exec") == 0) builtin = "sqliteTransactionExec";
        else if (strcmp(method->name, "query") == 0) builtin = "sqliteTransactionQuery";
        else if (strcmp(method->name, "stream") == 0) builtin = "sqliteTransactionQueryStream";
        else if (strcmp(method->name, "savepoint") == 0) builtin = "sqliteSavepoint";
        else if (strcmp(method->name, "rollbackTo") == 0) builtin = "sqliteRollbackTo";
        else if (strcmp(method->name, "release") == 0) builtin = "sqliteRelease";
        else if (strcmp(method->name, "prepare") == 0) builtin = "sqliteTransactionPrepare";
        else if (strcmp(method->name, "commit") == 0) builtin = "sqliteCommit";
        else if (strcmp(method->name, "rollback") == 0) builtin = "sqliteRollback";
        else if (strcmp(method->name, "active") == 0) builtin = "sqliteTransactionActive";
    }
    if (builtin == NULL) z_fatal("unknown SQLite method %s", method->name);
    bool handled = false;
    ZValue result = z_sqlite_call_builtin(builtin, forwarded, argc + 1, &handled);
    if (!handled) z_fatal("SQLite method %s is not implemented", method->name);
    return result;
}

static ZValue z_sqlite_get_field(ZValue value, const char *name, bool *handled) {
    *handled = true;
    if (value.tag == ZV_SQLITE_DATABASE) {
        if (strcmp(name,"exec")==0||strcmp(name,"query")==0||strcmp(name,"queryOne")==0||strcmp(name,"stream")==0||strcmp(name,"prepare")==0||strcmp(name,"begin")==0||strcmp(name,"migrate")==0||strcmp(name,"schemaVersion")==0||strcmp(name,"close")==0||strcmp(name,"isOpen")==0||strcmp(name,"path")==0) return z_sqlite_bound_method(value,name);
        z_fatal("SQLiteDatabase has no method %s", name);
    }
    if (value.tag == ZV_SQLITE_STATEMENT) {
        if (strcmp(name,"exec")==0||strcmp(name,"query")==0||strcmp(name,"stream")==0||strcmp(name,"parameterCount")==0||strcmp(name,"columns")==0||strcmp(name,"close")==0||strcmp(name,"isOpen")==0||strcmp(name,"sql")==0) return z_sqlite_bound_method(value,name);
        z_fatal("SQLiteStatement has no method %s", name);
    }
    if (value.tag == ZV_SQLITE_TRANSACTION) {
        if (strcmp(name,"exec")==0||strcmp(name,"query")==0||strcmp(name,"stream")==0||strcmp(name,"prepare")==0||strcmp(name,"savepoint")==0||strcmp(name,"rollbackTo")==0||strcmp(name,"release")==0||strcmp(name,"commit")==0||strcmp(name,"rollback")==0||strcmp(name,"active")==0) return z_sqlite_bound_method(value,name);
        z_fatal("SQLiteTransaction has no method %s", name);
    }
    *handled = false;
    return z_null();
}

static bool z_sqlite_database_is_open(ZSQLiteDatabaseNative *database) {
    pthread_mutex_lock(&database->mutex);
    bool open = database->open && database->database != NULL;
    pthread_mutex_unlock(&database->mutex);
    return open;
}

static bool z_sqlite_statement_is_open(ZSQLiteStatementNative *statement) {
    pthread_mutex_lock(&statement->database->mutex);
    bool open = statement->open && statement->statement != NULL &&
                statement->database->open && statement->database->database != NULL;
    pthread_mutex_unlock(&statement->database->mutex);
    return open;
}

static bool z_sqlite_transaction_is_active(ZSQLiteTransactionNative *transaction) {
    pthread_mutex_lock(&transaction->database->mutex);
    bool active = transaction->active && transaction->database->transaction_active &&
                  transaction->database->open && transaction->database->database != NULL;
    pthread_mutex_unlock(&transaction->database->mutex);
    return active;
}

static ZValue z_sqlite_transaction_exec(ZSQLiteTransactionNative *transaction, const char *sql, ZValue params, bool query) {
    ZSQLiteDatabaseNative *database = transaction->database;
    pthread_mutex_lock(&database->mutex);
    z_sqlite_check_database(database);
    if (!transaction->active || !database->transaction_active) {
        pthread_mutex_unlock(&database->mutex);
        z_fatal("SQLite transaction is no longer active");
    }
    sqlite3_stmt *statement = z_sqlite_prepare_locked(database, sql);
    ZValue result = query
        ? z_sqlite_query_statement_locked(database, statement, params)
        : z_sqlite_exec_statement_locked(database, statement, params);
    sqlite3_finalize(statement);
    pthread_mutex_unlock(&database->mutex);
    return result;
}

static ZValue z_sqlite_transaction_prepare(ZSQLiteTransactionNative *transaction, const char *sql) {
    ZSQLiteDatabaseNative *database = transaction->database;
    pthread_mutex_lock(&database->mutex);
    z_sqlite_check_database(database);
    if (!transaction->active || !database->transaction_active) {
        pthread_mutex_unlock(&database->mutex);
        z_fatal("SQLite transaction is no longer active");
    }
    sqlite3_stmt *statement = z_sqlite_prepare_locked(database, sql);
    ZSQLiteStatementNative *native = (ZSQLiteStatementNative *)z_alloc(sizeof(ZSQLiteStatementNative));
    native->database = database;
    native->transaction = transaction;
    native->statement = statement;
    native->open = true;
    native->sql = z_strdup(sql);
    pthread_mutex_unlock(&database->mutex);
    return z_sqlite_value(ZV_SQLITE_STATEMENT, ZK_SQLITE_STATEMENT, native);
}


static ZValue z_sqlite_operation_result(bool ok, ZValue value, const char *error) {
    ZValue pairs[3] = {
        z_pair(z_string("ok"), z_bool(ok)),
        z_pair(z_string("value"), value),
        z_pair(z_string("error"), z_string(error == NULL ? "" : error))
    };
    return z_dict_from(pairs, 3);
}

static bool z_sqlite_integrity_raw(sqlite3 *handle, char *message, size_t message_size) {
    sqlite3_stmt *statement = NULL;
    int code = sqlite3_prepare_v2(handle, "PRAGMA quick_check", -1, &statement, NULL);
    if (code != SQLITE_OK || statement == NULL) {
        snprintf(message, message_size, "prepare integrity check: %s", sqlite3_errmsg(handle));
        if (statement != NULL) sqlite3_finalize(statement);
        return false;
    }
    code = sqlite3_step(statement);
    if (code != SQLITE_ROW) {
        snprintf(message, message_size, "run integrity check: %s", sqlite3_errmsg(handle));
        sqlite3_finalize(statement);
        return false;
    }
    const unsigned char *text = sqlite3_column_text(statement, 0);
    snprintf(message, message_size, "%s", text == NULL ? "no integrity result" : (const char *)text);
    bool ok = text != NULL && strcasecmp((const char *)text, "ok") == 0;
    sqlite3_finalize(statement);
    return ok;
}

static bool z_sqlite_copy_raw(sqlite3 *destination, sqlite3 *source, char *message, size_t message_size) {
    sqlite3_backup *backup = sqlite3_backup_init(destination, "main", source, "main");
    if (backup == NULL) {
        snprintf(message, message_size, "initialize SQLite backup: %s", sqlite3_errmsg(destination));
        return false;
    }
    int step = sqlite3_backup_step(backup, -1);
    int finish = sqlite3_backup_finish(backup);
    if (step != SQLITE_DONE) {
        snprintf(message, message_size, "copy SQLite backup: %s", sqlite3_errmsg(destination));
        return false;
    }
    if (finish != SQLITE_OK) {
        snprintf(message, message_size, "finish SQLite backup: %s", sqlite3_errmsg(destination));
        return false;
    }
    return true;
}

static ZValue z_sqlite_backup_result(ZSQLiteDatabaseNative *database, const char *path) {
    if (path == NULL || *path == '\0') return z_sqlite_operation_result(false, z_null(), "SQLite backup path is empty");
    pthread_mutex_lock(&database->mutex);
    if (!database->open || database->database == NULL) { pthread_mutex_unlock(&database->mutex); return z_sqlite_operation_result(false, z_null(), "SQLite database is closed"); }
    if (database->transaction_active) { pthread_mutex_unlock(&database->mutex); return z_sqlite_operation_result(false, z_null(), "cannot back up SQLite database with an active transaction"); }
    if (database->path != NULL && strcmp(database->path, path) == 0) { pthread_mutex_unlock(&database->mutex); return z_sqlite_operation_result(false, z_null(), "SQLite backup destination must differ from the open database"); }
    size_t temporary_size = strlen(path) + 48;
    char *temporary = (char *)malloc(temporary_size);
    if (temporary == NULL) { pthread_mutex_unlock(&database->mutex); return z_sqlite_operation_result(false, z_null(), "out of memory creating backup path"); }
    snprintf(temporary, temporary_size, "%s.zumbra-%ld.tmp", path, (long)getpid());
    unlink(temporary);
    sqlite3 *target = NULL;
    int code = sqlite3_open_v2(temporary, &target, SQLITE_OPEN_READWRITE | SQLITE_OPEN_CREATE | SQLITE_OPEN_FULLMUTEX, NULL);
    char message[1024] = {0};
    bool ok = code == SQLITE_OK && target != NULL;
    if (!ok) snprintf(message, sizeof(message), "open SQLite backup destination: %s", target == NULL ? sqlite3_errstr(code) : sqlite3_errmsg(target));
    if (ok) ok = z_sqlite_copy_raw(target, database->database, message, sizeof(message));
    if (target != NULL) sqlite3_close_v2(target);
    pthread_mutex_unlock(&database->mutex);
    if (ok && rename(temporary, path) != 0) { snprintf(message, sizeof(message), "replace SQLite backup: %s", strerror(errno)); ok = false; }
    if (!ok) unlink(temporary);
    free(temporary);
    return z_sqlite_operation_result(ok, ok ? z_string(path) : z_null(), ok ? "" : message);
}

static ZValue z_sqlite_restore_result(ZSQLiteDatabaseNative *database, const char *path) {
    if (path == NULL || *path == '\0') return z_sqlite_operation_result(false, z_null(), "SQLite restore path is empty");
    if (database->path != NULL && strcmp(database->path, path) == 0) return z_sqlite_operation_result(false, z_null(), "SQLite restore source must differ from the open database");
    sqlite3 *source = NULL;
    int code = sqlite3_open_v2(path, &source, SQLITE_OPEN_READONLY | SQLITE_OPEN_FULLMUTEX | SQLITE_OPEN_URI, NULL);
    char message[1024] = {0};
    if (code != SQLITE_OK || source == NULL) {
        snprintf(message, sizeof(message), "open SQLite restore source: %s", source == NULL ? sqlite3_errstr(code) : sqlite3_errmsg(source));
        if (source != NULL) sqlite3_close_v2(source);
        return z_sqlite_operation_result(false, z_null(), message);
    }
    if (!z_sqlite_integrity_raw(source, message, sizeof(message))) {
        sqlite3_close_v2(source);
        char detailed[1200]; snprintf(detailed, sizeof(detailed), "validate SQLite restore source: %s", message);
        return z_sqlite_operation_result(false, z_null(), detailed);
    }
    pthread_mutex_lock(&database->mutex);
    if (!database->open || database->database == NULL) { pthread_mutex_unlock(&database->mutex); sqlite3_close_v2(source); return z_sqlite_operation_result(false, z_null(), "SQLite database is closed"); }
    if (database->transaction_active) { pthread_mutex_unlock(&database->mutex); sqlite3_close_v2(source); return z_sqlite_operation_result(false, z_null(), "cannot restore SQLite database with an active transaction"); }
    bool ok = z_sqlite_copy_raw(database->database, source, message, sizeof(message));
    if (ok) ok = z_sqlite_integrity_raw(database->database, message, sizeof(message));
    pthread_mutex_unlock(&database->mutex);
    sqlite3_close_v2(source);
    return z_sqlite_operation_result(ok, ok ? z_string(path) : z_null(), ok ? "" : message);
}

static ZValue z_sqlite_integrity_result(ZSQLiteDatabaseNative *database) {
    char message[1024] = {0};
    pthread_mutex_lock(&database->mutex);
    if (!database->open || database->database == NULL) { pthread_mutex_unlock(&database->mutex); return z_sqlite_operation_result(false, z_null(), "SQLite database is closed"); }
    if (database->transaction_active) { pthread_mutex_unlock(&database->mutex); return z_sqlite_operation_result(false, z_null(), "cannot check SQLite integrity with an active transaction"); }
    bool ok = z_sqlite_integrity_raw(database->database, message, sizeof(message));
    pthread_mutex_unlock(&database->mutex);
    return z_sqlite_operation_result(ok, z_string(message), ok ? "" : message);
}

static ZValue z_sqlite_call_builtin(const char *name, const ZValue *args, size_t argc, bool *handled) {
    *handled = true;
    if (strcmp(name,"sqliteOpen")==0) { z_expect_args(name,argc,1); return z_sqlite_open(z_as_cstring(args[0])); }
    if (strcmp(name,"sqliteMemory")==0) { z_expect_args(name,argc,0); return z_sqlite_open(":memory:"); }
    if (strcmp(name,"sqliteExec")==0) { if(argc<2||argc>3)z_fatal("sqliteExec expects database, SQL and optional parameters"); return z_sqlite_database_exec(z_sqlite_expect_database(args[0]),z_as_cstring(args[1]),argc==3?args[2]:z_array_from(NULL,0),false); }
    if (strcmp(name,"sqliteQuery")==0) { if(argc<2||argc>3)z_fatal("sqliteQuery expects database, SQL and optional parameters"); return z_sqlite_database_query(z_sqlite_expect_database(args[0]),z_as_cstring(args[1]),argc==3?args[2]:z_array_from(NULL,0),false); }
    if (strcmp(name,"sqliteQueryOne")==0) { if(argc<2||argc>3)z_fatal("sqliteQueryOne expects database, SQL and optional parameters"); ZValue rows=z_sqlite_database_query(z_sqlite_expect_database(args[0]),z_as_cstring(args[1]),argc==3?args[2]:z_array_from(NULL,0),false); return rows.as.array->len==0?z_null():rows.as.array->items[0]; }
    if (strcmp(name,"sqliteQueryStream")==0) { if(argc<2||argc>3)z_fatal("sqliteQueryStream expects database, SQL and optional parameters"); return z_sqlite_database_stream(z_sqlite_expect_database(args[0]),z_as_cstring(args[1]),argc==3?args[2]:z_array_from(NULL,0),false); }
    if (strcmp(name,"sqliteMigrate")==0) { z_expect_args(name,argc,2); return z_int(z_sqlite_migrate(z_sqlite_expect_database(args[0]),args[1])); }
    if (strcmp(name,"sqliteSchemaVersion")==0) { z_expect_args(name,argc,1); return z_int(z_sqlite_schema_version(z_sqlite_expect_database(args[0]))); }
    if (strcmp(name,"sqliteBackup")==0) { z_expect_args(name,argc,2); return z_sqlite_backup_result(z_sqlite_expect_database(args[0]),z_as_cstring(args[1])); }
    if (strcmp(name,"sqliteRestore")==0) { z_expect_args(name,argc,2); return z_sqlite_restore_result(z_sqlite_expect_database(args[0]),z_as_cstring(args[1])); }
    if (strcmp(name,"sqliteIntegrityCheck")==0) { z_expect_args(name,argc,1); return z_sqlite_integrity_result(z_sqlite_expect_database(args[0])); }
    if (strcmp(name,"sqlitePrepare")==0) { z_expect_args(name,argc,2); return z_sqlite_prepare_value(z_sqlite_expect_database(args[0]),NULL,z_as_cstring(args[1])); }
    if (strcmp(name,"sqliteBegin")==0) { z_expect_args(name,argc,1); return z_sqlite_begin(z_sqlite_expect_database(args[0])); }
    if (strcmp(name,"sqliteClose")==0) { z_expect_args(name,argc,1); return z_bool(z_sqlite_close_database(z_sqlite_expect_database(args[0]))); }
    if (strcmp(name,"sqliteIsOpen")==0) { z_expect_args(name,argc,1); return z_bool(z_sqlite_database_is_open(z_sqlite_expect_database(args[0]))); }
    if (strcmp(name,"sqlitePath")==0) { z_expect_args(name,argc,1); return z_string(z_sqlite_expect_database(args[0])->path); }
    if (strcmp(name,"sqliteStatementExec")==0) { if(argc<1||argc>2)z_fatal("sqliteStatementExec expects statement and optional parameters"); return z_sqlite_statement_exec(z_sqlite_expect_statement(args[0]),argc==2?args[1]:z_array_from(NULL,0)); }
    if (strcmp(name,"sqliteStatementQuery")==0) { if(argc<1||argc>2)z_fatal("sqliteStatementQuery expects statement and optional parameters"); return z_sqlite_statement_query(z_sqlite_expect_statement(args[0]),argc==2?args[1]:z_array_from(NULL,0)); }
    if (strcmp(name,"sqliteStatementQueryStream")==0) { if(argc<1||argc>2)z_fatal("sqliteStatementQueryStream expects statement and optional parameters"); return z_sqlite_statement_stream(z_sqlite_expect_statement(args[0]),argc==2?args[1]:z_array_from(NULL,0)); }
    if (strcmp(name,"sqliteStatementParameterCount")==0) { z_expect_args(name,argc,1); return z_int(sqlite3_bind_parameter_count(z_sqlite_expect_statement(args[0])->statement)); }
    if (strcmp(name,"sqliteStatementColumns")==0) { z_expect_args(name,argc,1); ZSQLiteStatementNative *statement=z_sqlite_expect_statement(args[0]); int count=sqlite3_column_count(statement->statement); ZValue *items=count==0?NULL:(ZValue*)z_alloc(sizeof(ZValue)*(size_t)count); for(int i=0;i<count;i++)items[i]=z_string(sqlite3_column_name(statement->statement,i)); return z_array_from(items,(size_t)count); }
    if (strcmp(name,"sqliteStatementClose")==0) { z_expect_args(name,argc,1); return z_bool(z_sqlite_statement_close(z_sqlite_expect_statement(args[0]))); }
    if (strcmp(name,"sqliteStatementOpen")==0) { z_expect_args(name,argc,1); return z_bool(z_sqlite_statement_is_open(z_sqlite_expect_statement(args[0]))); }
    if (strcmp(name,"sqliteStatementSQL")==0) { z_expect_args(name,argc,1); return z_string(z_sqlite_expect_statement(args[0])->sql); }
    if (strcmp(name,"sqliteTransactionExec")==0) { if(argc<2||argc>3)z_fatal("sqliteTransactionExec expects transaction, SQL and optional parameters"); return z_sqlite_transaction_exec(z_sqlite_expect_transaction(args[0]),z_as_cstring(args[1]),argc==3?args[2]:z_array_from(NULL,0),false); }
    if (strcmp(name,"sqliteTransactionQuery")==0) { if(argc<2||argc>3)z_fatal("sqliteTransactionQuery expects transaction, SQL and optional parameters"); return z_sqlite_transaction_exec(z_sqlite_expect_transaction(args[0]),z_as_cstring(args[1]),argc==3?args[2]:z_array_from(NULL,0),true); }
    if (strcmp(name,"sqliteTransactionQueryStream")==0) { if(argc<2||argc>3)z_fatal("sqliteTransactionQueryStream expects transaction, SQL and optional parameters"); ZSQLiteTransactionNative *tx=z_sqlite_expect_transaction(args[0]); return z_sqlite_database_stream(tx->database,z_as_cstring(args[1]),argc==3?args[2]:z_array_from(NULL,0),true); }
    if (strcmp(name,"sqliteSavepoint")==0) { z_expect_args(name,argc,2); return z_bool(z_sqlite_transaction_control(z_sqlite_expect_transaction(args[0]),"SAVEPOINT ",z_as_cstring(args[1]))); }
    if (strcmp(name,"sqliteRollbackTo")==0) { z_expect_args(name,argc,2); return z_bool(z_sqlite_transaction_control(z_sqlite_expect_transaction(args[0]),"ROLLBACK TO SAVEPOINT ",z_as_cstring(args[1]))); }
    if (strcmp(name,"sqliteRelease")==0) { z_expect_args(name,argc,2); return z_bool(z_sqlite_transaction_control(z_sqlite_expect_transaction(args[0]),"RELEASE SAVEPOINT ",z_as_cstring(args[1]))); }
    if (strcmp(name,"sqliteTransactionPrepare")==0) { z_expect_args(name,argc,2); return z_sqlite_transaction_prepare(z_sqlite_expect_transaction(args[0]),z_as_cstring(args[1])); }
    if (strcmp(name,"sqliteCommit")==0) { z_expect_args(name,argc,1); return z_bool(z_sqlite_finish_transaction(z_sqlite_expect_transaction(args[0]),"COMMIT")); }
    if (strcmp(name,"sqliteRollback")==0) { z_expect_args(name,argc,1); return z_bool(z_sqlite_finish_transaction(z_sqlite_expect_transaction(args[0]),"ROLLBACK")); }
    if (strcmp(name,"sqliteTransactionActive")==0) { z_expect_args(name,argc,1); return z_bool(z_sqlite_transaction_is_active(z_sqlite_expect_transaction(args[0]))); }
    *handled = false;
    return z_null();
}

static void z_sqlite_runtime_init(void) {
    z_sqlite_databases = NULL;
}

static void z_sqlite_runtime_shutdown(void) {
    pthread_mutex_lock(&z_sqlite_registry_mutex);
    for (ZSQLiteDatabaseNative *database = z_sqlite_databases; database != NULL; database = database->next) {
        pthread_mutex_lock(&database->mutex);
        if (database->open && database->database != NULL) {
            if (database->transaction_active) sqlite3_exec(database->database, "ROLLBACK", NULL, NULL, NULL);
            sqlite3_close_v2(database->database);
            database->database = NULL;
            database->open = false;
            database->transaction_active = false;
        }
        pthread_mutex_unlock(&database->mutex);
    }
    z_sqlite_databases = NULL;
    pthread_mutex_unlock(&z_sqlite_registry_mutex);
}

#endif
