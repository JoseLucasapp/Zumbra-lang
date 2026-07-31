package builtins

import (
	"bytes"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"zumbra/object"
)

// EmbeddedAsset is the portable runtime representation used by the VM and evaluator.
type EmbeddedAsset struct {
	Name string
	Data []byte
}

var embeddedAssets = struct {
	sync.RWMutex
	values map[string][]byte
}{values: map[string][]byte{}}

// ConfigureEmbeddedAssets replaces the process asset registry. Data is copied so
// callers can discard or mutate their input buffers safely after configuration.
func ConfigureEmbeddedAssets(assets []EmbeddedAsset) error {
	values := make(map[string][]byte, len(assets))
	for _, asset := range assets {
		name := normalizeAssetName(asset.Name)
		if name == "" {
			return &assetConfigurationError{message: "asset name cannot be empty"}
		}
		if strings.HasPrefix(name, "../") || name == ".." || strings.HasPrefix(name, "/") {
			return &assetConfigurationError{message: "asset name must be project-relative: " + asset.Name}
		}
		if _, exists := values[name]; exists {
			return &assetConfigurationError{message: "duplicate embedded asset: " + name}
		}
		values[name] = append([]byte(nil), asset.Data...)
	}
	embeddedAssets.Lock()
	embeddedAssets.values = values
	embeddedAssets.Unlock()
	return nil
}

func ResetEmbeddedAssets() {
	embeddedAssets.Lock()
	embeddedAssets.values = map[string][]byte{}
	embeddedAssets.Unlock()
}

type assetConfigurationError struct{ message string }

func (e *assetConfigurationError) Error() string { return e.message }

func normalizeAssetName(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	for strings.HasPrefix(value, "./") {
		value = strings.TrimPrefix(value, "./")
	}
	for strings.Contains(value, "//") {
		value = strings.ReplaceAll(value, "//", "/")
	}
	return value
}

func assetNameArgument(name string, args []object.Object) (string, *object.Error) {
	if len(args) != 1 {
		return "", NewError("%s expects 1 argument, got %d", name, len(args))
	}
	value, ok := args[0].(*object.String)
	if !ok {
		return "", NewError("%s expects an asset name string", name)
	}
	normalized := normalizeAssetName(value.Value)
	if normalized == "" {
		return "", NewError("%s asset name cannot be empty", name)
	}
	return normalized, nil
}

func lookupEmbeddedAsset(name string) ([]byte, bool) {
	embeddedAssets.RLock()
	data, exists := embeddedAssets.values[name]
	if exists {
		data = append([]byte(nil), data...)
	}
	embeddedAssets.RUnlock()
	return data, exists
}

func AssetExistsBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		name, err := assetNameArgument("assetExists", args)
		if err != nil {
			return err
		}
		_, exists := lookupEmbeddedAsset(name)
		return NewBoolean(exists)
	}}
}

func AssetTextBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		name, err := assetNameArgument("assetText", args)
		if err != nil {
			return err
		}
		data, exists := lookupEmbeddedAsset(name)
		if !exists {
			return NewError("assetText: embedded asset %q was not found", name)
		}
		if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
			return NewError("assetText: embedded asset %q is not valid UTF-8", name)
		}
		return NewString(string(data))
	}}
}

func AssetBytesBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		name, err := assetNameArgument("assetBytes", args)
		if err != nil {
			return err
		}
		data, exists := lookupEmbeddedAsset(name)
		if !exists {
			return NewError("assetBytes: embedded asset %q was not found", name)
		}
		return &object.ByteArray{Data: data}
	}}
}

func AssetListBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return NewError("assetList expects 0 arguments, got %d", len(args))
		}
		embeddedAssets.RLock()
		names := make([]string, 0, len(embeddedAssets.values))
		for name := range embeddedAssets.values {
			names = append(names, name)
		}
		embeddedAssets.RUnlock()
		sort.Strings(names)
		items := make([]object.Object, len(names))
		for index, name := range names {
			items[index] = NewString(name)
		}
		return &object.Array{Elements: items}
	}}
}
