package conformance

import "testing"

func TestFixedWidthComparisonBooleanConformance(t *testing.T) {
	requireZ17EvaluatorVMMatch(t, "fixed-width-boolean", `
		struct Header { battery: bool; }
		var flags << 0u8;
		var parsed << Header((flags band 0x02u8) != 0u8);
		!parsed.battery;
	`, "true")
}
