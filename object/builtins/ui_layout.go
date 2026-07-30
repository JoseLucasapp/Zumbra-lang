package builtins

import (
	"fmt"
	"math"
	"strings"

	"zumbra/object"
)

func renderUIContext(ctx *object.UIContext) error {
	if ctx == nil || ctx.Window == nil || ctx.Window.Runtime == nil {
		return fmt.Errorf("invalid UI context")
	}
	ctx.Mu.RLock()
	if ctx.Closed {
		ctx.Mu.RUnlock()
		return nil
	}
	root, theme, focusID, hoverID := ctx.Root, ctx.Theme, ctx.FocusID, ctx.HoverID
	ctx.Mu.RUnlock()
	width, height := ctx.Window.Runtime.Size()
	if width < 1 || height < 1 {
		return fmt.Errorf("window has invalid size")
	}
	if theme == nil {
		theme = &object.UITheme{Name: "light", Values: defaultUITheme("light")}
	}
	var measurer object.DesktopUITextMeasurer
	if candidate, ok := ctx.Window.Runtime.(object.DesktopUITextMeasurer); ok {
		measurer = candidate
	}
	layoutUINode(root, object.UIRect{X: 0, Y: 0, Width: float64(width), Height: float64(height)}, theme, measurer)
	frame := &object.UIRenderFrame{Width: float64(width), Height: float64(height), Background: uiThemeString(theme, "background", "#f5f7fb")}
	flattenUIRender(root, theme, focusID, hoverID, nil, &frame.Items)
	if renderer, ok := ctx.Window.Runtime.(object.DesktopUIRenderer); ok {
		if err := renderer.RenderUI(frame); err != nil {
			return err
		}
	}
	ctx.Mu.Lock()
	ctx.LastFrame = frame
	ctx.Dirty = false
	ctx.Mu.Unlock()
	return nil
}

func layoutUINode(node *object.UINode, available object.UIRect, theme *object.UITheme, measurer object.DesktopUITextMeasurer) {
	if node == nil {
		return
	}
	node.Mu.RLock()
	props := cloneUIProps(node.Props)
	kind := node.Kind
	visible := node.Visible
	children := append([]*object.UINode{}, node.Children...)
	scrollOffsetY := node.ScrollOffsetY
	node.Mu.RUnlock()
	if !visible {
		node.Mu.Lock()
		node.Bounds = object.UIRect{}
		node.ContentBounds = object.UIRect{}
		node.ContentHeight = 0
		node.Mu.Unlock()
		return
	}
	flowChildren := make([]*object.UINode, 0, len(children))
	overlayChildren := make([]*object.UINode, 0, 1)
	for _, child := range children {
		child.Mu.RLock()
		childKind := child.Kind
		childVisible := child.Visible
		child.Mu.RUnlock()
		if !childVisible {
			child.Mu.Lock()
			child.Bounds = object.UIRect{}
			child.ContentBounds = object.UIRect{}
			child.ContentHeight = 0
			child.ScrollOffsetY = 0
			child.Mu.Unlock()
			continue
		}
		if childKind == "modal" {
			overlayChildren = append(overlayChildren, child)
		} else {
			flowChildren = append(flowChildren, child)
		}
	}

	margin := uiBox(props, "margin", 0)
	x := available.X + margin.left
	y := available.Y + margin.top
	width := available.Width - margin.left - margin.right
	height := available.Height - margin.top - margin.bottom
	// Modal width/height describe the dialog card, not an intermediate box.
	// Keeping the viewport dimensions here is what allows the dialog to be
	// centered correctly regardless of its requested size.
	if kind != "modal" {
		if v, ok := uiNumber(props["width"]); ok {
			width = v
		}
		if v, ok := uiNumber(props["height"]); ok {
			height = v
		}
	}
	if v, ok := uiNumber(props["minWidth"]); ok {
		width = math.Max(width, v)
	}
	if v, ok := uiNumber(props["maxWidth"]); ok {
		width = math.Min(width, v)
	}
	if v, ok := uiNumber(props["minHeight"]); ok {
		height = math.Max(height, v)
	}
	if v, ok := uiNumber(props["maxHeight"]); ok {
		height = math.Min(height, v)
	}
	if kind == "menu" {
		placement := optionString(props, "placement", "left")
		collapsed := optionBool(props, "collapsed", false)
		if placement == "top" || placement == "bottom" {
			size := uiPropNumber(props, "expandedSize", height)
			if collapsed {
				size = uiPropNumber(props, "collapsedSize", math.Min(size, 48))
			}
			height = size
		} else {
			size := uiPropNumber(props, "expandedSize", width)
			if collapsed {
				size = uiPropNumber(props, "collapsedSize", math.Min(size, 56))
			}
			width = size
		}
	}
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	natural := uiNaturalHeight(kind, props, theme, measurer)
	if _, explicit := uiNumber(props["height"]); !explicit && kind != "row" && kind != "column" && kind != "container" && kind != "menu" && kind != "modal" {
		height = math.Min(height, natural)
	}
	bounds := object.UIRect{X: x, Y: y, Width: width, Height: height}
	if kind == "modal" {
		modalMargin := math.Max(0, uiPropNumber(props, "viewportMargin", 24))
		requestedWidth := uiPropNumber(props, "contentWidth", uiPropNumber(props, "width", 480))
		requestedHeight := uiPropNumber(props, "contentHeight", uiPropNumber(props, "height", 320))
		modalWidth := math.Min(math.Max(0, width-modalMargin*2), requestedWidth)
		modalHeight := math.Min(math.Max(0, height-modalMargin*2), requestedHeight)
		bounds = object.UIRect{X: x + (width-modalWidth)/2, Y: y + (height-modalHeight)/2, Width: modalWidth, Height: modalHeight}
		x, y, width, height = bounds.X, bounds.Y, bounds.Width, bounds.Height
	}
	padding := uiBox(props, "padding", 0)
	inner := object.UIRect{X: x + padding.left, Y: y + padding.top, Width: math.Max(0, width-padding.left-padding.right), Height: math.Max(0, height-padding.top-padding.bottom)}

	node.Mu.Lock()
	node.Bounds = bounds
	node.ContentBounds = inner
	node.ContentHeight = inner.Height
	node.Mu.Unlock()
	if len(flowChildren) == 0 {
		for _, overlay := range overlayChildren {
			layoutUINode(overlay, bounds, theme, measurer)
		}
		return
	}

	direction := kind
	if direction != "row" && direction != "column" {
		direction = optionString(props, "direction", "column")
	}
	if kind == "menu" {
		placement := optionString(props, "placement", "left")
		if placement == "top" || placement == "bottom" {
			direction = "row"
		} else {
			direction = "column"
		}
	}
	if kind == "modal" {
		direction = "column"
	}
	gap := uiPropNumber(props, "gap", uiThemeNumber(theme, "spacing", 8))
	if gap < 0 {
		gap = 0
	}
	mainAvailable := inner.Height
	if direction == "row" {
		mainAvailable = inner.Width
	}
	mainAvailable -= gap * float64(maxInt(0, len(flowChildren)-1))
	if mainAvailable < 0 {
		mainAvailable = 0
	}
	fixed := 0.0
	totalGrow := 0.0
	for _, child := range flowChildren {
		child.Mu.RLock()
		cp := cloneUIProps(child.Props)
		ck := child.Kind
		child.Mu.RUnlock()
		grow := uiPropNumber(cp, "grow", 0)
		if grow > 0 {
			totalGrow += grow
			continue
		}
		size := uiNaturalHeight(ck, cp, theme, measurer)
		if direction == "row" {
			size = uiPropNumber(cp, "width", uiNaturalWidth(ck, cp, theme, measurer))
		} else {
			size = uiPropNumber(cp, "height", size)
		}
		fixed += size
	}
	remaining := math.Max(0, mainAvailable-fixed)
	contentHeight := inner.Height
	if direction == "column" {
		used := fixed + gap*float64(maxInt(0, len(flowChildren)-1))
		if totalGrow > 0 {
			used += remaining
		}
		contentHeight = math.Max(inner.Height, used)
	}

	scrollableY := uiScrollableY(props) && direction == "column"
	if !scrollableY {
		scrollOffsetY = 0
	} else {
		maxOffset := math.Max(0, contentHeight-inner.Height)
		if maxOffset <= uiScrollOverflowEpsilon {
			maxOffset = 0
		}
		scrollOffsetY = uiClamp(scrollOffsetY, 0, maxOffset)
		if uiHasVerticalOverflow(contentHeight, inner.Height) && !optionBool(props, "scrollbarOverlay", false) {
			scrollbarWidth := math.Max(4, uiPropNumber(props, "scrollbarWidth", 8))
			gutter := math.Max(2, uiPropNumber(props, "scrollbarGutter", 4))
			inner.Width = math.Max(0, inner.Width-scrollbarWidth-gutter)
		}
	}

	node.Mu.Lock()
	node.ContentBounds = inner
	node.ContentHeight = contentHeight
	node.ScrollOffsetY = scrollOffsetY
	node.Mu.Unlock()

	cursorX, cursorY := inner.X, inner.Y
	if scrollableY {
		cursorY -= scrollOffsetY
	}
	justify := optionString(props, "justify", "start")
	if totalGrow == 0 && remaining > 0 {
		switch justify {
		case "center":
			if direction == "row" {
				cursorX += remaining / 2
			} else {
				cursorY += remaining / 2
			}
		case "end":
			if direction == "row" {
				cursorX += remaining
			} else {
				cursorY += remaining
			}
		case "space-between":
			if len(flowChildren) > 1 {
				gap += remaining / float64(len(flowChildren)-1)
			}
		}
	}
	for _, child := range flowChildren {
		child.Mu.RLock()
		cp := cloneUIProps(child.Props)
		ck := child.Kind
		child.Mu.RUnlock()
		grow := uiPropNumber(cp, "grow", 0)
		var cw, ch float64
		if direction == "row" {
			cw = uiPropNumber(cp, "width", uiNaturalWidth(ck, cp, theme, measurer))
			if grow > 0 && totalGrow > 0 {
				cw = remaining * grow / totalGrow
			}
			ch = inner.Height
			align := optionString(props, "align", "stretch")
			naturalH := uiNaturalHeight(ck, cp, theme, measurer)
			if align != "stretch" {
				ch = math.Min(ch, uiPropNumber(cp, "height", naturalH))
			}
			cy := cursorY
			if align == "center" {
				cy = inner.Y + (inner.Height-ch)/2
			} else if align == "end" {
				cy = inner.Y + inner.Height - ch
			}
			layoutUINode(child, object.UIRect{X: cursorX, Y: cy, Width: cw, Height: ch}, theme, measurer)
			cursorX += cw + gap
		} else {
			ch = uiPropNumber(cp, "height", uiNaturalHeight(ck, cp, theme, measurer))
			if grow > 0 && totalGrow > 0 {
				ch = remaining * grow / totalGrow
			}
			cw = inner.Width
			align := optionString(props, "align", "stretch")
			naturalW := uiNaturalWidth(ck, cp, theme, measurer)
			if align != "stretch" {
				cw = math.Min(cw, uiPropNumber(cp, "width", naturalW))
			}
			cx := cursorX
			if align == "center" {
				cx = inner.X + (inner.Width-cw)/2
			} else if align == "end" {
				cx = inner.X + inner.Width - cw
			}
			layoutUINode(child, object.UIRect{X: cx, Y: cursorY, Width: cw, Height: ch}, theme, measurer)
			cursorY += ch + gap
		}
	}
	for _, overlay := range overlayChildren {
		layoutUINode(overlay, bounds, theme, measurer)
	}

}

const uiScrollOverflowEpsilon = 0.5

func uiHasVerticalOverflow(contentHeight, viewportHeight float64) bool {
	return viewportHeight > 0 && contentHeight > viewportHeight+uiScrollOverflowEpsilon
}

func uiShouldRenderVerticalScrollbar(props map[string]object.Object, contentHeight, viewportHeight float64) bool {
	return uiScrollableY(props) && uiHasVerticalOverflow(contentHeight, viewportHeight)
}

func uiScrollableY(props map[string]object.Object) bool {
	if optionBool(props, "scrollY", false) {
		return true
	}
	overflow := strings.ToLower(strings.TrimSpace(optionString(props, "overflowY", optionString(props, "overflow", ""))))
	return overflow == "scroll" || overflow == "auto"
}

func uiClipsChildren(props map[string]object.Object) bool {
	if uiScrollableY(props) || optionBool(props, "clipChildren", false) {
		return true
	}
	overflow := strings.ToLower(strings.TrimSpace(optionString(props, "overflowY", optionString(props, "overflow", ""))))
	return overflow == "hidden" || overflow == "clip"
}

type uiEdges struct{ left, top, right, bottom float64 }

func uiBox(props map[string]object.Object, key string, fallback float64) uiEdges {
	if v, ok := uiNumber(props[key]); ok {
		return uiEdges{v, v, v, v}
	}
	result := uiEdges{fallback, fallback, fallback, fallback}
	if v, ok := uiNumber(props[key+"Left"]); ok {
		result.left = v
	}
	if v, ok := uiNumber(props[key+"Top"]); ok {
		result.top = v
	}
	if v, ok := uiNumber(props[key+"Right"]); ok {
		result.right = v
	}
	if v, ok := uiNumber(props[key+"Bottom"]); ok {
		result.bottom = v
	}
	return result
}
func uiNumber(value object.Object) (float64, bool) {
	switch v := value.(type) {
	case *object.Integer:
		return float64(v.Value), true
	case *object.FixedInteger:
		return float64(v.SignedValue()), true
	case *object.Float:
		return v.Value, true
	}
	return 0, false
}
func uiPropNumber(props map[string]object.Object, key string, fallback float64) float64 {
	if v, ok := uiNumber(props[key]); ok {
		return v
	}
	return fallback
}
func uiThemeNumber(theme *object.UITheme, key string, fallback float64) float64 {
	if theme != nil {
		if v, ok := uiNumber(theme.Values[key]); ok {
			return v
		}
	}
	return fallback
}
func uiThemeString(theme *object.UITheme, key, fallback string) string {
	if theme != nil {
		if v, ok := theme.Values[key].(*object.String); ok {
			return v.Value
		}
	}
	return fallback
}
func uiTextStyle(props map[string]object.Object, theme *object.UITheme, wrapWidth float64) object.UITextStyle {
	lineHeight := uiPropNumber(props, "lineHeight", uiThemeNumber(theme, "lineHeight", 1.25))
	if lineHeight <= 0 {
		lineHeight = 1.25
	}
	return object.UITextStyle{
		FontFamily: uiString(props, "fontFamily", uiThemeString(theme, "fontFamily", "sans")),
		FontPath:   uiString(props, "fontPath", uiThemeString(theme, "fontPath", "")),
		FontSize:   uiPropNumber(props, "fontSize", uiThemeNumber(theme, "fontSize", 14)),
		FontWeight: uiString(props, "fontWeight", uiThemeString(theme, "fontWeight", "normal")),
		FontStyle:  uiString(props, "fontStyle", uiThemeString(theme, "fontStyle", "normal")),
		LineHeight: lineHeight,
		WrapWidth:  wrapWidth,
	}
}

func uiTextFromProps(props map[string]object.Object) string {
	for _, key := range []string{"text", "label", "value", "placeholder", "title"} {
		if value, ok := props[key].(*object.String); ok && value.Value != "" {
			return value.Value
		}
	}
	return ""
}

func uiMeasureText(text string, props map[string]object.Object, theme *object.UITheme, measurer object.DesktopUITextMeasurer, wrapWidth float64) object.UITextMetrics {
	style := uiTextStyle(props, theme, wrapWidth)
	if style.FontSize <= 0 {
		style.FontSize = 14
	}
	if measurer != nil {
		metrics := measurer.MeasureUIText(text, style)
		if metrics.Width >= 0 && metrics.Height > 0 {
			return metrics
		}
	}
	return approximateUITextMetrics(text, style)
}

func approximateUITextMetrics(text string, style object.UITextStyle) object.UITextMetrics {
	fontSize := style.FontSize
	if fontSize <= 0 {
		fontSize = 14
	}
	lineHeight := style.LineHeight
	if lineHeight <= 0 {
		lineHeight = 1.25
	}
	measureLine := func(line string) float64 {
		width := 0.0
		for _, r := range line {
			switch {
			case r == '\t':
				width += fontSize * 1.32
			case r == ' ':
				width += fontSize * 0.34
			case strings.ContainsRune("ilI.,'`!|:;", r):
				width += fontSize * 0.32
			case strings.ContainsRune("MW@#%&", r):
				width += fontSize * 0.88
			case r >= 0x2E80:
				width += fontSize
			default:
				width += fontSize * 0.58
			}
		}
		return width
	}
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	maxWidth := 0.0
	lineCount := 0
	for _, line := range lines {
		width := measureLine(line)
		if style.WrapWidth > 0 && width > style.WrapWidth {
			wrapped := int(math.Ceil(width / style.WrapWidth))
			lineCount += wrapped
			if style.WrapWidth > maxWidth {
				maxWidth = style.WrapWidth
			}
		} else {
			lineCount++
			if width > maxWidth {
				maxWidth = width
			}
		}
	}
	if lineCount < 1 {
		lineCount = 1
	}
	return object.UITextMetrics{Width: maxWidth, Height: float64(lineCount) * fontSize * lineHeight}
}

func uiNaturalHeight(kind string, props map[string]object.Object, theme *object.UITheme, measurer object.DesktopUITextMeasurer) float64 {
	control := uiThemeNumber(theme, "controlHeight", 36)
	font := uiPropNumber(props, "fontSize", uiThemeNumber(theme, "fontSize", 14))
	textMetrics := uiMeasureText(uiTextFromProps(props), props, theme, measurer, uiPropNumber(props, "wrapWidth", 0))
	switch kind {
	case "text":
		return math.Max(font+8, textMetrics.Height+8)
	case "textarea":
		rows := uiPropNumber(props, "rows", 4)
		return math.Max(rows*(font*uiPropNumber(props, "lineHeight", uiThemeNumber(theme, "lineHeight", 1.25)))+16, control)
	case "table":
		rows := len(uiObjectArray(props["rows"]))
		if rows == 0 {
			rows = 1
		}
		return float64(rows+1) * control
	case "list", "tree":
		items := len(uiObjectArray(props["items"]))
		if items == 0 {
			items = 1
		}
		return float64(items) * control
	case "tabs":
		return control
	case "menu":
		placement := optionString(props, "placement", "left")
		if placement == "top" || placement == "bottom" {
			size := uiPropNumber(props, "expandedSize", control)
			if optionBool(props, "collapsed", false) {
				size = uiPropNumber(props, "collapsedSize", math.Min(size, 48))
			}
			return size
		}
		return uiPropNumber(props, "height", control)
	case "modal":
		return uiPropNumber(props, "height", 320)
	case "tooltip":
		return math.Max(font+16, textMetrics.Height+12)
	case "progress":
		return 20
	case "image", "canvas":
		return uiPropNumber(props, "height", 160)
	case "spacer":
		return uiPropNumber(props, "size", 8)
	case "row", "column", "container":
		return uiPropNumber(props, "height", control)
	}
	return control
}
func uiNaturalWidth(kind string, props map[string]object.Object, theme *object.UITheme, measurer object.DesktopUITextMeasurer) float64 {
	text := uiTextFromProps(props)
	metrics := uiMeasureText(text, props, theme, measurer, 0)
	base := math.Max(48, metrics.Width+24)
	switch kind {
	case "input", "textarea", "select":
		return math.Max(base, 180)
	case "checkbox", "radio":
		return base + 20
	case "image", "canvas":
		return uiPropNumber(props, "width", 240)
	case "menu":
		placement := optionString(props, "placement", "left")
		if placement == "left" || placement == "right" {
			size := uiPropNumber(props, "expandedSize", uiPropNumber(props, "width", 240))
			if optionBool(props, "collapsed", false) {
				size = uiPropNumber(props, "collapsedSize", math.Min(size, 56))
			}
			return size
		}
		return uiPropNumber(props, "width", base)
	case "spacer":
		return uiPropNumber(props, "size", 8)
	}
	return base
}
func uiObjectArray(value object.Object) []object.Object {
	if a, ok := value.(*object.Array); ok {
		return a.Elements
	}
	return nil
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func flattenUIRender(node *object.UINode, theme *object.UITheme, focusID, hoverID string, inheritedClip *object.UIRect, items *[]object.UIRenderItem) {
	if node == nil {
		return
	}
	node.Mu.RLock()
	visible := node.Visible
	id, kind, bounds := node.ID, node.Kind, node.Bounds
	contentBounds := node.ContentBounds
	scrollOffsetY := node.ScrollOffsetY
	contentHeight := node.ContentHeight
	props := cloneUIProps(node.Props)
	children := append([]*object.UINode{}, node.Children...)
	node.Mu.RUnlock()
	if !visible {
		return
	}
	applyUIStyleDefaults(kind, props, theme)
	commands := []object.UICanvasCommand{}
	if kind == "canvas" {
		commands = parseCanvasCommands(props["commands"])
	}
	var itemClip *object.UIRect
	if inheritedClip != nil {
		copy := *inheritedClip
		itemClip = &copy
	}
	*items = append(*items, object.UIRenderItem{
		ID: id, Kind: kind, Bounds: bounds, ContentBounds: contentBounds, Clip: itemClip,
		Props: props, Focused: id == focusID, Hovered: id == hoverID, Commands: commands,
		ScrollOffsetY: scrollOffsetY, ScrollContentHeight: contentHeight,
	})
	childClip := inheritedClip
	if uiClipsChildren(props) {
		viewport := contentBounds
		if inheritedClip != nil {
			intersection, ok := uiRectIntersection(*inheritedClip, viewport)
			if !ok {
				return
			}
			viewport = intersection
		}
		childClip = &viewport
	}
	for _, child := range children {
		flattenUIRender(child, theme, focusID, hoverID, childClip, items)
	}
}

func uiRectIntersection(a, b object.UIRect) (object.UIRect, bool) {
	x1 := math.Max(a.X, b.X)
	y1 := math.Max(a.Y, b.Y)
	x2 := math.Min(a.X+a.Width, b.X+b.Width)
	y2 := math.Min(a.Y+a.Height, b.Y+b.Height)
	if x2 <= x1 || y2 <= y1 {
		return object.UIRect{}, false
	}
	return object.UIRect{X: x1, Y: y1, Width: x2 - x1, Height: y2 - y1}, true
}

func cloneUIProps(source map[string]object.Object) map[string]object.Object {
	result := make(map[string]object.Object, len(source))
	for k, v := range source {
		result[k] = v
	}
	return result
}
func applyUIStyleDefaults(kind string, props map[string]object.Object, theme *object.UITheme) {
	set := func(key string, value object.Object) {
		if _, ok := props[key]; !ok {
			props[key] = value
		}
	}
	set("fontSize", NewFloat(uiThemeNumber(theme, "fontSize", 14)))
	set("fontFamily", NewString(uiThemeString(theme, "fontFamily", "sans")))
	set("fontPath", NewString(uiThemeString(theme, "fontPath", "")))
	set("fontWeight", NewString(uiThemeString(theme, "fontWeight", "normal")))
	set("fontStyle", NewString(uiThemeString(theme, "fontStyle", "normal")))
	set("lineHeight", NewFloat(uiThemeNumber(theme, "lineHeight", 1.25)))
	set("textColor", NewString(uiThemeString(theme, "text", "#172033")))
	set("borderColor", NewString(uiThemeString(theme, "border", "#cfd6e2")))
	set("radius", NewFloat(uiThemeNumber(theme, "radius", 6)))
	switch kind {
	case "container", "row", "column", "table", "list", "tree", "tabs", "menu":
		set("background", NewString("transparent"))
	case "text":
		set("background", NewString("transparent"))
	case "button":
		set("background", NewString(uiThemeString(theme, "primary", "#3867e8")))
		set("textColor", NewString(uiThemeString(theme, "primaryText", "#ffffff")))
		set("textAlign", NewString("center"))
		set("textOverflow", NewString("ellipsis"))
		set("cursor", NewString("pointer"))
	case "input", "textarea":
		set("background", NewString(uiThemeString(theme, "surface", "#ffffff")))
		set("focusBackground", NewString(uiThemeString(theme, "surfaceAlt", "#eef2f7")))
		set("cursor", NewString("text"))
		set("caretColor", NewString(uiThemeString(theme, "primary", "#3867e8")))
	case "checkbox", "radio":
		set("background", NewString(uiThemeString(theme, "surface", "#ffffff")))
		set("cursor", NewString("pointer"))
	case "select":
		set("background", NewString(uiThemeString(theme, "surface", "#ffffff")))
		set("textAlign", NewString("left"))
		set("textOverflow", NewString("ellipsis"))
		set("dropdownBackground", NewString(uiThemeString(theme, "surface", "#ffffff")))
		set("selectedOptionBackground", NewString(uiThemeString(theme, "surfaceAlt", "#eef2f7")))
		set("dropdownBorderColor", NewString(uiThemeString(theme, "border", "#cfd6e2")))
		set("cursor", NewString("pointer"))
	case "modal":
		set("background", NewString(uiThemeString(theme, "surface", "#ffffff")))
		set("overlay", NewString("#00000055"))
		set("backdropBlur", NewFloat(6))
		set("modalShadow", NewString("#00000040"))
	case "tooltip":
		set("background", NewString("#202633"))
		set("textColor", NewString("#ffffff"))
	case "progress":
		set("background", NewString(uiThemeString(theme, "surfaceAlt", "#eef2f7")))
		set("fill", NewString(uiThemeString(theme, "primary", "#3867e8")))
	case "canvas":
		set("background", NewString(optionString(props, "background", "transparent")))
	}
}
func parseCanvasCommands(value object.Object) []object.UICanvasCommand {
	array, ok := value.(*object.Array)
	if !ok {
		return nil
	}
	commands := []object.UICanvasCommand{}
	for _, entry := range array.Elements {
		m, err := objectDictMap(entry, "canvas command")
		if err != nil {
			continue
		}
		kind := optionString(m, "kind", "")
		values := map[string]object.Object{}
		if dict, ok := m["values"].(*object.Dict); ok {
			values, _ = objectDictMap(dict, "canvas values")
		}
		commands = append(commands, object.UICanvasCommand{Kind: kind, Values: values})
	}
	return commands
}

func uiNodeText(item object.UIRenderItem) string {
	for _, key := range []string{"text", "label", "value", "placeholder", "title"} {
		if v, ok := item.Props[key].(*object.String); ok && v.Value != "" {
			return v.Value
		}
	}
	return ""
}
func uiColor(props map[string]object.Object, key, fallback string) string {
	if v, ok := props[key].(*object.String); ok {
		return v.Value
	}
	return fallback
}
func uiBool(props map[string]object.Object, key string, fallback bool) bool {
	return optionBool(props, key, fallback)
}
func uiString(props map[string]object.Object, key, fallback string) string {
	return optionString(props, key, fallback)
}
func uiFloat(props map[string]object.Object, key string, fallback float64) float64 {
	return uiPropNumber(props, key, fallback)
}
func uiClamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
func uiTextDisplay(value object.Object) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case *object.String:
		return v.Value
	case *object.Integer:
		return fmt.Sprintf("%d", v.Value)
	case *object.Float:
		return fmt.Sprintf("%g", v.Value)
	case *object.Boolean:
		return fmt.Sprintf("%t", v.Value)
	}
	return value.Inspect()
}
func uiRowDisplay(row object.Object) string {
	if d, ok := row.(*object.Dict); ok {
		parts := []string{}
		for _, pair := range d.Pairs {
			parts = append(parts, uiTextDisplay(pair.Value))
		}
		return strings.Join(parts, " | ")
	}
	return uiTextDisplay(row)
}
