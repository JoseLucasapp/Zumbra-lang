#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

printf 'Running Z16 language foundation tests...\n'
go test ./mir ./nativec ./conformance ./object/builtins

go build -o "$TMP/zumbra" .
"$TMP/zumbra" check code_examples/core/ui_event_target.zum
"$TMP/zumbra" check code_examples/core/ui_scroll.zum
"$TMP/zumbra" check code_examples/core/ui_modal.zum
"$TMP/zumbra" check code_examples/core/array_append.zum
"$TMP/zumbra" check code_examples/core/data_exchange.zum
"$TMP/zumbra" check code_examples/core/ui_select_dropdown.zum
"$TMP/zumbra" check code_examples/core/ui_navigation_charts.zum
"$TMP/zumbra" check code_examples/core/ui_interaction_polish.zum
"$TMP/zumbra" check code_examples/core/ui_sidebar_line_chart.zum
"$TMP/zumbra" check code_examples/core/ui_text_editing.zum

VM_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TMP/zumbra" run code_examples/core/ui_event_target.zum)"
EXPECTED=$'edit-42\nbutton'
if [[ "$VM_OUTPUT" != "$EXPECTED" ]]; then
  printf 'Unexpected VM output:\n%s\n' "$VM_OUTPUT" >&2
  exit 1
fi

"$TMP/zumbra" build --release -o "$TMP/ui-event-target" code_examples/core/ui_event_target.zum
NATIVE_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TMP/ui-event-target")"
if [[ "$NATIVE_OUTPUT" != "$EXPECTED" ]]; then
  printf 'Unexpected native output:\n%s\n' "$NATIVE_OUTPUT" >&2
  exit 1
fi

SCROLL_VM_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TMP/zumbra" run code_examples/core/ui_scroll.zum)"
SCROLL_EXPECTED=$'256\n40\n-40\ntrue'
if [[ "$SCROLL_VM_OUTPUT" != "$SCROLL_EXPECTED" ]]; then
  printf 'Unexpected scroll VM output:\n%s\n' "$SCROLL_VM_OUTPUT" >&2
  exit 1
fi

"$TMP/zumbra" build --release -o "$TMP/ui-scroll" code_examples/core/ui_scroll.zum
SCROLL_NATIVE_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TMP/ui-scroll")"
if [[ "$SCROLL_NATIVE_OUTPUT" != "$SCROLL_EXPECTED" ]]; then
  printf 'Unexpected scroll native output:\n%s\n' "$SCROLL_NATIVE_OUTPUT" >&2
  exit 1
fi

MODAL_EXPECTED=$'2\n2\nmodal-confirm\n0'
MODAL_VM_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TMP/zumbra" run code_examples/core/ui_modal.zum)"
if [[ "$MODAL_VM_OUTPUT" != "$MODAL_EXPECTED" ]]; then
  printf 'Unexpected modal VM output:\n%s\n' "$MODAL_VM_OUTPUT" >&2
  exit 1
fi
"$TMP/zumbra" build --release -o "$TMP/ui-modal" code_examples/core/ui_modal.zum
MODAL_NATIVE_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TMP/ui-modal")"
if [[ "$MODAL_NATIVE_OUTPUT" != "$MODAL_EXPECTED" ]]; then
  printf 'Unexpected modal native output:\n%s\n' "$MODAL_NATIVE_OUTPUT" >&2
  exit 1
fi

ARRAY_EXPECTED=$'3\nfirst\nlast'
ARRAY_VM_OUTPUT="$("$TMP/zumbra" run code_examples/core/array_append.zum)"
if [[ "$ARRAY_VM_OUTPUT" != "$ARRAY_EXPECTED" ]]; then
  printf 'Unexpected array VM output:\n%s\n' "$ARRAY_VM_OUTPUT" >&2
  exit 1
fi
"$TMP/zumbra" build --release -o "$TMP/array-append" code_examples/core/array_append.zum
ARRAY_NATIVE_OUTPUT="$("$TMP/array-append")"
if [[ "$ARRAY_NATIVE_OUTPUT" != "$ARRAY_EXPECTED" ]]; then
  printf 'Unexpected array native output:\n%s\n' "$ARRAY_NATIVE_OUTPUT" >&2
  exit 1
fi

DATA_EXPECTED=$'true\ntrue\nFinal Fantasy IX\nfalse\ntrue\ntrue\n2\ncomma, preserved'
DATA_VM_OUTPUT="$("$TMP/zumbra" run code_examples/core/data_exchange.zum)"
if [[ "$DATA_VM_OUTPUT" != "$DATA_EXPECTED" ]]; then
  printf 'Unexpected recoverable data VM output:\n%s\n' "$DATA_VM_OUTPUT" >&2
  exit 1
fi
"$TMP/zumbra" build --release -o "$TMP/data-exchange" code_examples/core/data_exchange.zum
DATA_NATIVE_OUTPUT="$("$TMP/data-exchange")"
if [[ "$DATA_NATIVE_OUTPUT" != "$DATA_EXPECTED" ]]; then
  printf 'Unexpected recoverable data native output:\n%s\n' "$DATA_NATIVE_OUTPUT" >&2
  exit 1
fi

SELECT_EXPECTED=$'true\nPrimeira\nSegunda\nfalse'
SELECT_VM_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TMP/zumbra" run code_examples/core/ui_select_dropdown.zum)"
if [[ "$SELECT_VM_OUTPUT" != "$SELECT_EXPECTED" ]]; then
  printf 'Unexpected select dropdown VM output:\n%s\n' "$SELECT_VM_OUTPUT" >&2
  exit 1
fi
"$TMP/zumbra" build --release -o "$TMP/ui-select-dropdown" code_examples/core/ui_select_dropdown.zum
SELECT_NATIVE_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TMP/ui-select-dropdown")"
if [[ "$SELECT_NATIVE_OUTPUT" != "$SELECT_EXPECTED" ]]; then
  printf 'Unexpected select dropdown native output:\n%s\n' "$SELECT_NATIVE_OUTPUT" >&2
  exit 1
fi


NAV_EXPECTED=$'280
2
56
Plataforma'
NAV_VM_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TMP/zumbra" run code_examples/core/ui_navigation_charts.zum)"
if [[ "$NAV_VM_OUTPUT" != "$NAV_EXPECTED" ]]; then
  printf 'Unexpected navigation/chart VM output:
%s
' "$NAV_VM_OUTPUT" >&2
  exit 1
fi
"$TMP/zumbra" build --release -o "$TMP/ui-navigation-charts" code_examples/core/ui_navigation_charts.zum
NAV_NATIVE_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TMP/ui-navigation-charts")"
if [[ "$NAV_NATIVE_OUTPUT" != "$NAV_EXPECTED" ]]; then
  printf 'Unexpected navigation/chart native output:
%s
' "$NAV_NATIVE_OUTPUT" >&2
  exit 1
fi


POLISH_EXPECTED=$'296
604
true
0
900
false'
POLISH_VM_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TMP/zumbra" run code_examples/core/ui_interaction_polish.zum)"
if [[ "$POLISH_VM_OUTPUT" != "$POLISH_EXPECTED" ]]; then
  printf 'Unexpected interaction polish VM output:
%s
' "$POLISH_VM_OUTPUT" >&2
  exit 1
fi
"$TMP/zumbra" build --release -o "$TMP/ui-interaction-polish" code_examples/core/ui_interaction_polish.zum
POLISH_NATIVE_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TMP/ui-interaction-polish")"
if [[ "$POLISH_NATIVE_OUTPUT" != "$POLISH_EXPECTED" ]]; then
  printf 'Unexpected interaction polish native output:
%s
' "$POLISH_NATIVE_OUTPUT" >&2
  exit 1
fi


SIDEBAR_LINE_EXPECTED=$'600
true
40
PlayStation 1'
SIDEBAR_LINE_VM_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TMP/zumbra" run code_examples/core/ui_sidebar_line_chart.zum)"
if [[ "$SIDEBAR_LINE_VM_OUTPUT" != "$SIDEBAR_LINE_EXPECTED" ]]; then
  printf 'Unexpected sidebar/line chart VM output:
%s
' "$SIDEBAR_LINE_VM_OUTPUT" >&2
  exit 1
fi
"$TMP/zumbra" build --release -o "$TMP/ui-sidebar-line-chart" code_examples/core/ui_sidebar_line_chart.zum
SIDEBAR_LINE_NATIVE_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TMP/ui-sidebar-line-chart")"
if [[ "$SIDEBAR_LINE_NATIVE_OUTPUT" != "$SIDEBAR_LINE_EXPECTED" ]]; then
  printf 'Unexpected sidebar/line chart native output:
%s
' "$SIDEBAR_LINE_NATIVE_OUTPUT" >&2
  exit 1
fi


TEXT_EDIT_EXPECTED=$'GXmmaAlpha beta
15
15'
TEXT_EDIT_VM_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TMP/zumbra" run code_examples/core/ui_text_editing.zum)"
if [[ "$TEXT_EDIT_VM_OUTPUT" != "$TEXT_EDIT_EXPECTED" ]]; then
  printf 'Unexpected text editing VM output:\n%s\n' "$TEXT_EDIT_VM_OUTPUT" >&2
  exit 1
fi
"$TMP/zumbra" build --release -o "$TMP/ui-text-editing" code_examples/core/ui_text_editing.zum
TEXT_EDIT_NATIVE_OUTPUT="$(ZUMBRA_DESKTOP_HEADLESS=1 "$TMP/ui-text-editing")"
if [[ "$TEXT_EDIT_NATIVE_OUTPUT" != "$TEXT_EDIT_EXPECTED" ]]; then
  printf 'Unexpected text editing native output:\n%s\n' "$TEXT_EDIT_NATIVE_OUTPUT" >&2
  exit 1
fi

printf 'Z16 language foundation tests passed.\n'
