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
    if (params.tag != ZV_ARRAY) z_fatal("SQLite parameters must be an array");
    int expected = sqlite3_bind_parameter_count(statement);
    if ((int)params.as.array->len != expected) {
        z_fatal("SQLite statement expects %d parameters, got %zu", expected, params.as.array->len);
    }
    sqlite3_reset(statement);
    sqlite3_clear_bindings(statement);
    for (size_t index = 0; index < params.as.array->len; index++) {
        z_sqlite_bind_value(statement, (int)index + 1, params.as.array->items[index]);
    }
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
        else if (strcmp(method->name, "prepare") == 0) builtin = "sqlitePrepare";
        else if (strcmp(method->name, "begin") == 0) builtin = "sqliteBegin";
        else if (strcmp(method->name, "close") == 0) builtin = "sqliteClose";
        else if (strcmp(method->name, "isOpen") == 0) builtin = "sqliteIsOpen";
        else if (strcmp(method->name, "path") == 0) builtin = "sqlitePath";
    } else if (method->receiver.tag == ZV_SQLITE_STATEMENT) {
        if (strcmp(method->name, "exec") == 0) builtin = "sqliteStatementExec";
        else if (strcmp(method->name, "query") == 0) builtin = "sqliteStatementQuery";
        else if (strcmp(method->name, "close") == 0) builtin = "sqliteStatementClose";
        else if (strcmp(method->name, "isOpen") == 0) builtin = "sqliteStatementOpen";
        else if (strcmp(method->name, "sql") == 0) builtin = "sqliteStatementSQL";
    } else if (method->receiver.tag == ZV_SQLITE_TRANSACTION) {
        if (strcmp(method->name, "exec") == 0) builtin = "sqliteTransactionExec";
        else if (strcmp(method->name, "query") == 0) builtin = "sqliteTransactionQuery";
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
        if (strcmp(name,"exec")==0||strcmp(name,"query")==0||strcmp(name,"prepare")==0||strcmp(name,"begin")==0||strcmp(name,"close")==0||strcmp(name,"isOpen")==0||strcmp(name,"path")==0) return z_sqlite_bound_method(value,name);
        z_fatal("SQLiteDatabase has no method %s", name);
    }
    if (value.tag == ZV_SQLITE_STATEMENT) {
        if (strcmp(name,"exec")==0||strcmp(name,"query")==0||strcmp(name,"close")==0||strcmp(name,"isOpen")==0||strcmp(name,"sql")==0) return z_sqlite_bound_method(value,name);
        z_fatal("SQLiteStatement has no method %s", name);
    }
    if (value.tag == ZV_SQLITE_TRANSACTION) {
        if (strcmp(name,"exec")==0||strcmp(name,"query")==0||strcmp(name,"prepare")==0||strcmp(name,"commit")==0||strcmp(name,"rollback")==0||strcmp(name,"active")==0) return z_sqlite_bound_method(value,name);
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

static ZValue z_sqlite_call_builtin(const char *name, const ZValue *args, size_t argc, bool *handled) {
    *handled = true;
    if (strcmp(name,"sqliteOpen")==0) { z_expect_args(name,argc,1); return z_sqlite_open(z_as_cstring(args[0])); }
    if (strcmp(name,"sqliteMemory")==0) { z_expect_args(name,argc,0); return z_sqlite_open(":memory:"); }
    if (strcmp(name,"sqliteExec")==0) { z_expect_args(name,argc,3); return z_sqlite_database_exec(z_sqlite_expect_database(args[0]),z_as_cstring(args[1]),args[2],false); }
    if (strcmp(name,"sqliteQuery")==0) { z_expect_args(name,argc,3); return z_sqlite_database_query(z_sqlite_expect_database(args[0]),z_as_cstring(args[1]),args[2],false); }
    if (strcmp(name,"sqlitePrepare")==0) { z_expect_args(name,argc,2); return z_sqlite_prepare_value(z_sqlite_expect_database(args[0]),NULL,z_as_cstring(args[1])); }
    if (strcmp(name,"sqliteBegin")==0) { z_expect_args(name,argc,1); return z_sqlite_begin(z_sqlite_expect_database(args[0])); }
    if (strcmp(name,"sqliteClose")==0) { z_expect_args(name,argc,1); return z_bool(z_sqlite_close_database(z_sqlite_expect_database(args[0]))); }
    if (strcmp(name,"sqliteIsOpen")==0) { z_expect_args(name,argc,1); return z_bool(z_sqlite_database_is_open(z_sqlite_expect_database(args[0]))); }
    if (strcmp(name,"sqlitePath")==0) { z_expect_args(name,argc,1); return z_string(z_sqlite_expect_database(args[0])->path); }
    if (strcmp(name,"sqliteStatementExec")==0) { z_expect_args(name,argc,2); return z_sqlite_statement_exec(z_sqlite_expect_statement(args[0]),args[1]); }
    if (strcmp(name,"sqliteStatementQuery")==0) { z_expect_args(name,argc,2); return z_sqlite_statement_query(z_sqlite_expect_statement(args[0]),args[1]); }
    if (strcmp(name,"sqliteStatementClose")==0) { z_expect_args(name,argc,1); return z_bool(z_sqlite_statement_close(z_sqlite_expect_statement(args[0]))); }
    if (strcmp(name,"sqliteStatementOpen")==0) { z_expect_args(name,argc,1); return z_bool(z_sqlite_statement_is_open(z_sqlite_expect_statement(args[0]))); }
    if (strcmp(name,"sqliteStatementSQL")==0) { z_expect_args(name,argc,1); return z_string(z_sqlite_expect_statement(args[0])->sql); }
    if (strcmp(name,"sqliteTransactionExec")==0) { z_expect_args(name,argc,3); return z_sqlite_transaction_exec(z_sqlite_expect_transaction(args[0]),z_as_cstring(args[1]),args[2],false); }
    if (strcmp(name,"sqliteTransactionQuery")==0) { z_expect_args(name,argc,3); return z_sqlite_transaction_exec(z_sqlite_expect_transaction(args[0]),z_as_cstring(args[1]),args[2],true); }
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
