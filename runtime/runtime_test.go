package runtime

import (
	"strings"
	"testing"
)

func TestRuntimeContainsMysqlDropTableWrapper(t *testing.T) {
	rt := Runtime()

	if !strings.Contains(rt, "func mysqlDropTable(tableName string) error") {
		t.Fatalf("runtime is missing mysqlDropTable wrapper")
	}

	if !strings.Contains(rt, "return mysqlDeleteTable(tableName)") {
		t.Fatalf("runtime wrapper does not delegate to mysqlDeleteTable")
	}
}
