package builtinspec

func init() {
	Names = append(Names,
		"sqliteQueryOne", "sqliteQueryStream", "sqliteMigrate", "sqliteSchemaVersion",
		"sqliteStatementQueryStream", "sqliteStatementParameterCount", "sqliteStatementColumns",
		"sqliteTransactionQueryStream", "sqliteSavepoint", "sqliteRollbackTo", "sqliteRelease",
		"sqlRowsNext", "sqlRowsColumns", "sqlRowsClose", "sqlRowsOpen",
		"postgresOpen", "postgresConfigurePool", "postgresPoolStats", "postgresPing", "postgresClose", "postgresIsOpen",
		"postgresExecDb", "postgresQueryDb", "postgresQueryOne", "postgresQueryStream", "postgresPrepare", "postgresBegin",
		"postgresStatementExec", "postgresStatementQuery", "postgresStatementStream", "postgresStatementClose", "postgresStatementOpen", "postgresStatementSQL",
		"postgresTransactionExec", "postgresTransactionQuery", "postgresTransactionStream", "postgresTransactionPrepare",
		"postgresSavepoint", "postgresRollbackTo", "postgresRelease", "postgresCommit", "postgresRollback", "postgresTransactionActive",
		"redisOpen", "redisPing", "redisClose", "redisIsOpen", "redisSetClient", "redisGetClient", "redisDelete", "redisExists", "redisExpire", "redisTTL", "redisIncrement", "redisPipeline", "redisPoolStats",
		"configLoad", "configFrom", "configEnv", "configMerge", "configRequired", "configString", "configInt", "configFloat", "configBool", "configSecret", "configRedacted",
		"logger", "loggerWith", "loggerSetLevel", "loggerLog", "loggerClose",
		"metrics", "metricsCounter", "metricsGauge", "metricsHistogram", "metricsSnapshot", "metricsReset",
		"traceStart", "traceChild", "traceSet", "traceEvent", "traceFinish", "traceActive",
		"sessionSQLite", "sessionRedis", "sessionCreate", "sessionGet", "sessionSet", "sessionDelete", "sessionRotate", "sessionTouch", "sessionCleanup", "sessionClose",
		"rateLimiter", "rateAllow", "rateReset",
		"jsonReadFile", "jsonWriteFile", "binaryEncode", "binaryDecode", "binaryWriteFile", "binaryReadFile",
	)
}
