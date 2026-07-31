package builtins

import "zumbra/object"

func registerZ12Builtin(name string, builtin *object.Builtin) {
	Builtins = append(Builtins, struct {
		Name    string
		Builtin *object.Builtin
	}{name, builtin})
}

func init() {
	entries := []struct {
		name    string
		builtin *object.Builtin
	}{
		// SQLite completion.
		{"sqliteQueryOne", SQLiteQueryOneBuiltin()},
		{"sqliteQueryStream", SQLiteQueryStreamBuiltin()},
		{"sqliteMigrate", SQLiteMigrateBuiltin()},
		{"sqliteSchemaVersion", SQLiteSchemaVersionBuiltin()},
		{"sqliteBackup", SQLiteBackupBuiltin()},
		{"sqliteRestore", SQLiteRestoreBuiltin()},
		{"sqliteIntegrityCheck", SQLiteIntegrityCheckBuiltin()},
		{"sqliteStatementQueryStream", SQLiteStatementQueryStreamBuiltin()},
		{"sqliteStatementParameterCount", SQLiteStatementParameterCountBuiltin()},
		{"sqliteStatementColumns", SQLiteStatementColumnsBuiltin()},
		{"sqliteTransactionQueryStream", SQLiteTransactionQueryStreamBuiltin()},
		{"sqliteSavepoint", SQLiteSavepointBuiltin()},
		{"sqliteRollbackTo", SQLiteRollbackToBuiltin()},
		{"sqliteRelease", SQLiteReleaseBuiltin()},
		{"sqlRowsNext", SQLRowsNextBuiltin()},
		{"sqlRowsColumns", SQLRowsColumnsBuiltin()},
		{"sqlRowsClose", SQLRowsCloseBuiltin()},
		{"sqlRowsOpen", SQLRowsOpenBuiltin()},

		// PostgreSQL object API and pools.
		{"postgresOpen", PostgresOpenBuiltin()},
		{"postgresConfigurePool", PostgresConfigurePoolBuiltin()},
		{"postgresPoolStats", PostgresPoolStatsBuiltin()},
		{"postgresPing", PostgresPingBuiltin()},
		{"postgresClose", PostgresCloseBuiltin()},
		{"postgresIsOpen", PostgresIsOpenBuiltin()},
		{"postgresExecDb", PostgresExecObjectBuiltin()},
		{"postgresQueryDb", PostgresQueryObjectBuiltin()},
		{"postgresQueryOne", PostgresQueryOneBuiltin()},
		{"postgresQueryStream", PostgresQueryStreamBuiltin()},
		{"postgresPrepare", PostgresPrepareBuiltin()},
		{"postgresBegin", PostgresBeginBuiltin()},
		{"postgresStatementExec", PostgresStatementExecBuiltin()},
		{"postgresStatementQuery", PostgresStatementQueryBuiltin()},
		{"postgresStatementStream", PostgresStatementStreamBuiltin()},
		{"postgresStatementClose", PostgresStatementCloseBuiltin()},
		{"postgresStatementOpen", PostgresStatementOpenBuiltin()},
		{"postgresStatementSQL", PostgresStatementSQLBuiltin()},
		{"postgresTransactionExec", PostgresTransactionExecBuiltin()},
		{"postgresTransactionQuery", PostgresTransactionQueryBuiltin()},
		{"postgresTransactionStream", PostgresTransactionStreamBuiltin()},
		{"postgresTransactionPrepare", PostgresTransactionPrepareBuiltin()},
		{"postgresSavepoint", PostgresSavepointBuiltin()},
		{"postgresRollbackTo", PostgresRollbackToBuiltin()},
		{"postgresRelease", PostgresReleaseBuiltin()},
		{"postgresCommit", PostgresCommitBuiltin()},
		{"postgresRollback", PostgresRollbackBuiltin()},
		{"postgresTransactionActive", PostgresTransactionActiveBuiltin()},

		// Redis object API and pipelines.
		{"redisOpen", RedisOpenBuiltin()},
		{"redisPing", RedisPingBuiltin()},
		{"redisClose", RedisCloseBuiltin()},
		{"redisIsOpen", RedisIsOpenBuiltin()},
		{"redisSetClient", RedisSetObjectBuiltin()},
		{"redisGetClient", RedisGetObjectBuiltin()},
		{"redisDelete", RedisDelObjectBuiltin()},
		{"redisExists", RedisExistsBuiltin()},
		{"redisExpire", RedisExpireBuiltin()},
		{"redisTTL", RedisTTLBuiltin()},
		{"redisIncrement", RedisIncrementBuiltin()},
		{"redisPipeline", RedisPipelineBuiltin()},
		{"redisPoolStats", RedisPoolStatsBuiltin()},

		// Typed configuration.
		{"configLoad", ConfigLoadBuiltin()},
		{"configFrom", ConfigFromBuiltin()},
		{"configEnv", ConfigEnvBuiltin()},
		{"configMerge", ConfigMergeBuiltin()},
		{"configRequired", ConfigRequiredBuiltin()},
		{"configString", ConfigStringBuiltin()},
		{"configInt", ConfigIntBuiltin()},
		{"configFloat", ConfigFloatBuiltin()},
		{"configBool", ConfigBoolBuiltin()},
		{"configSecret", ConfigSecretBuiltin()},
		{"configRedacted", ConfigRedactedBuiltin()},

		// Structured logs, metrics and tracing.
		{"logger", LoggerBuiltin()},
		{"loggerWith", LoggerWithBuiltin()},
		{"loggerSetLevel", LoggerSetLevelBuiltin()},
		{"loggerLog", LoggerLogBuiltin()},
		{"loggerClose", LoggerCloseBuiltin()},
		{"metrics", MetricsBuiltin()},
		{"metricsCounter", MetricsCounterBuiltin()},
		{"metricsGauge", MetricsGaugeBuiltin()},
		{"metricsHistogram", MetricsHistogramBuiltin()},
		{"metricsSnapshot", MetricsSnapshotBuiltin()},
		{"metricsReset", MetricsResetBuiltin()},
		{"traceStart", TraceStartBuiltin()},
		{"traceChild", TraceChildBuiltin()},
		{"traceSet", TraceSetBuiltin()},
		{"traceEvent", TraceEventBuiltin()},
		{"traceFinish", TraceFinishBuiltin()},
		{"traceActive", TraceActiveBuiltin()},

		// Persistent sessions.
		{"sessionSQLite", SessionSQLiteBuiltin()},
		{"sessionRedis", SessionRedisBuiltin()},
		{"sessionCreate", SessionCreateBuiltin()},
		{"sessionGet", SessionGetBuiltin()},
		{"sessionSet", SessionSetBuiltin()},
		{"sessionDelete", SessionDeleteBuiltin()},
		{"sessionRotate", SessionRotateBuiltin()},
		{"sessionTouch", SessionTouchBuiltin()},
		{"sessionCleanup", SessionCleanupBuiltin()},
		{"sessionClose", SessionCloseBuiltin()},

		// Z11 deferred rate limiting.
		{"rateLimiter", RateLimiterBuiltin()},
		{"rateAllow", RateAllowBuiltin()},
		{"rateReset", RateResetBuiltin()},

		// JSON files and versioned binary serialization.
		{"fileExists", FileExistsBuiltin()},
		{"jsonReadFile", JSONReadFileBuiltin()},
		{"jsonWriteFile", JSONWriteFileBuiltin()},
		{"jsonReadResult", JSONReadResultBuiltin()},
		{"jsonWriteResult", JSONWriteResultBuiltin()},
		{"csvReadFile", CSVReadFileBuiltin()},
		{"csvWriteFile", CSVWriteFileBuiltin()},
		{"csvReadResult", CSVReadResultBuiltin()},
		{"csvWriteResult", CSVWriteResultBuiltin()},
		{"binaryEncode", BinaryEncodeBuiltin()},
		{"binaryDecode", BinaryDecodeBuiltin()},
		{"binaryWriteFile", BinaryWriteFileBuiltin()},
		{"binaryReadFile", BinaryReadFileBuiltin()},
	}
	for _, entry := range entries {
		registerZ12Builtin(entry.name, entry.builtin)
	}
}
