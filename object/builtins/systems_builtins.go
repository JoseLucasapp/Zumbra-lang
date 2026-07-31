package builtins

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"
	"unsafe"

	"zumbra/collections"
	"zumbra/object"
)

func systemTypeArg(value object.Object, label string) (object.SystemType, *object.Error) {
	name, ok := value.(*object.String)
	if !ok {
		return object.SystemType{}, NewError("%s must be a type name string", label)
	}
	t, ok := object.ParseSystemType(strings.ToLower(strings.TrimSpace(name.Value)))
	if !ok {
		return object.SystemType{}, NewError("%s uses unsupported native type %q", label, name.Value)
	}
	return t, nil
}

func pointerArg(value object.Object, label string) (*object.Pointer, *object.Error) {
	pointer, ok := value.(*object.Pointer)
	if !ok {
		return nil, NewError("%s expects Pointer, got %s", label, value.Type())
	}
	return pointer, nil
}

func integerArg(value object.Object, label string) (int64, *object.Error) {
	result, err := collections.IntegerValue(value)
	if err != nil {
		return 0, NewError("%s: %s", label, err)
	}
	return result, nil
}

func stringArg(value object.Object, label string) (string, *object.Error) {
	result, ok := value.(*object.String)
	if !ok {
		return "", NewError("%s expects string, got %s", label, value.Type())
	}
	return result.Value, nil
}

func dictFromObjects(values map[string]object.Object) *object.Dict {
	pairs := make(map[object.DictKey]object.DictPair, len(values))
	for key, value := range values {
		k := &object.String{Value: key}
		pairs[k.DictKey()] = object.DictPair{Key: k, Value: value}
	}
	return &object.Dict{Pairs: pairs}
}

func dictInt64(values map[string]int64) *object.Dict {
	objects := make(map[string]object.Object, len(values))
	for key, value := range values {
		objects[key] = &object.Integer{Value: value}
	}
	return dictFromObjects(objects)
}

func dictGet(dict *object.Dict, key string) object.Object {
	if dict == nil {
		return nil
	}
	value := &object.String{Value: key}
	pair, ok := dict.Pairs[value.DictKey()]
	if !ok {
		return nil
	}
	return pair.Value
}

func validAllocationCount(t object.SystemType, count int64) bool {
	if count < 0 || t.Size <= 0 {
		return false
	}
	maxInt := int64(^uint(0) >> 1)
	return count <= maxInt/int64(t.Size)
}

func AllocBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("alloc expects 2 arguments, got %d", len(args))
		}
		t, typeErr := systemTypeArg(args[0], "alloc type")
		if typeErr != nil {
			return typeErr
		}
		count, countErr := integerArg(args[1], "alloc count")
		if countErr != nil {
			return countErr
		}
		if count < 0 {
			return NewError("alloc count must be non-negative")
		}
		if !validAllocationCount(t, count) {
			return NewError("alloc size overflows the native address space")
		}
		return object.NewManagedPointer(t, int(count), "heap")
	}}
}

func CallocBuiltin() *object.Builtin { return AllocBuiltin() }

func NullPointerBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("nullPointer expects 1 argument, got %d", len(args))
		}
		t, err := systemTypeArg(args[0], "nullPointer type")
		if err != nil {
			return err
		}
		return object.NewNullPointer(t)
	}}
}

func ReallocBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("realloc expects 2 arguments, got %d", len(args))
		}
		pointer, pointerErr := pointerArg(args[0], "realloc")
		if pointerErr != nil {
			return pointerErr
		}
		count, countErr := integerArg(args[1], "realloc count")
		if countErr != nil {
			return countErr
		}
		if count < 0 || !validAllocationCount(pointer.ElementType(), count) {
			return NewError("realloc count is invalid or overflows the native address space")
		}
		if err := object.ReallocateMemoryBlock(pointer, int(count)); err != nil {
			return NewError("realloc: %s", err)
		}
		return pointer
	}}
}

func FreeBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("free expects 1 argument, got %d", len(args))
		}
		pointer, pointerErr := pointerArg(args[0], "free")
		if pointerErr != nil {
			return pointerErr
		}
		if err := object.FreeMemoryBlock(pointer); err != nil {
			return NewError("free: %s", err)
		}
		return NewBoolean(true)
	}}
}

func PointerReadBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 || len(args) > 2 {
			return NewError("pointerRead expects 1 or 2 arguments, got %d", len(args))
		}
		pointer, pointerErr := pointerArg(args[0], "pointerRead")
		if pointerErr != nil {
			return pointerErr
		}
		index := int64(0)
		if len(args) == 2 {
			var indexErr *object.Error
			index, indexErr = integerArg(args[1], "pointerRead index")
			if indexErr != nil {
				return indexErr
			}
		}
		value, err := pointer.Read(int(index))
		if err != nil {
			return NewError("pointerRead: %s", err)
		}
		return value
	}}
}

func DereferenceBuiltin() *object.Builtin { return PointerReadBuiltin() }

func PointerWriteBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 && len(args) != 3 {
			return NewError("pointerWrite expects 2 or 3 arguments, got %d", len(args))
		}
		pointer, pointerErr := pointerArg(args[0], "pointerWrite")
		if pointerErr != nil {
			return pointerErr
		}
		index := int64(0)
		valueIndex := 1
		if len(args) == 3 {
			var indexErr *object.Error
			index, indexErr = integerArg(args[1], "pointerWrite index")
			if indexErr != nil {
				return indexErr
			}
			valueIndex = 2
		}
		if err := pointer.Write(int(index), args[valueIndex]); err != nil {
			return NewError("pointerWrite: %s", err)
		}
		return pointer
	}}
}

func PointerFromAddressBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 3 || len(args) > 4 {
			return NewError("pointerFromAddress expects type, address, length and optional mutable flag")
		}
		t, typeErr := systemTypeArg(args[0], "pointerFromAddress type")
		if typeErr != nil {
			return typeErr
		}
		address, addressErr := collections.IntegerValue(args[1])
		if addressErr != nil {
			if fixed, ok := args[1].(*object.FixedInteger); ok {
				address = int64(fixed.UnsignedValue())
			} else {
				return NewError("pointerFromAddress address: %s", addressErr)
			}
		}
		length, lengthErr := integerArg(args[2], "pointerFromAddress length")
		if lengthErr != nil {
			return lengthErr
		}
		if length < 0 {
			return NewError("pointerFromAddress length must be non-negative")
		}
		mutable := false
		if len(args) == 4 {
			flag, ok := args[3].(*object.Boolean)
			if !ok {
				return NewError("pointerFromAddress mutable must be bool")
			}
			mutable = flag.Value
		}
		return &object.Pointer{Raw: unsafe.Pointer(uintptr(uint64(address))), RawType: t, Length: int(length), Mutable: mutable}
	}}
}

func PointerEqualBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("pointerEqual expects 2 arguments, got %d", len(args))
		}
		left, err := pointerArg(args[0], "pointerEqual")
		if err != nil {
			return err
		}
		right, err := pointerArg(args[1], "pointerEqual")
		if err != nil {
			return err
		}
		return NewBoolean(left.Address() == right.Address())
	}}
}

func PointerCompareBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("pointerCompare expects 2 arguments, got %d", len(args))
		}
		left, err := pointerArg(args[0], "pointerCompare")
		if err != nil {
			return err
		}
		right, err := pointerArg(args[1], "pointerCompare")
		if err != nil {
			return err
		}
		la, ra := left.Address(), right.Address()
		result := int64(0)
		if la < ra {
			result = -1
		} else if la > ra {
			result = 1
		}
		return NewInteger(result)
	}}
}

func PointerIsAlignedBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("pointerIsAligned expects 2 arguments, got %d", len(args))
		}
		pointer, err := pointerArg(args[0], "pointerIsAligned")
		if err != nil {
			return err
		}
		alignment, alignErr := integerArg(args[1], "pointerIsAligned alignment")
		if alignErr != nil {
			return alignErr
		}
		if alignment <= 0 || alignment&(alignment-1) != 0 {
			return NewError("pointerIsAligned alignment must be a positive power of two")
		}
		return NewBoolean(pointer.Address()%uintptr(alignment) == 0)
	}}
}

func PointerOffsetBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("pointerOffset expects 2 arguments, got %d", len(args))
		}
		pointer, pointerErr := pointerArg(args[0], "pointerOffset")
		if pointerErr != nil {
			return pointerErr
		}
		offset, offsetErr := integerArg(args[1], "pointerOffset offset")
		if offsetErr != nil {
			return offsetErr
		}
		if offset < 0 || offset > int64(pointer.Length) {
			return NewError("pointerOffset out of bounds: %d (length %d)", offset, pointer.Length)
		}
		result := &object.Pointer{
			Block: pointer.Block, Offset: pointer.Offset + int(offset), Length: pointer.Length - int(offset),
			Mutable: pointer.Mutable, Borrowed: true, Raw: pointer.Raw, RawType: pointer.RawType,
		}
		if pointer.Block == nil && pointer.Raw != nil {
			result.Raw = unsafe.Pointer(uintptr(pointer.Raw) + uintptr(offset)*uintptr(pointer.ElementType().Size))
			result.Offset = 0
		}
		return result
	}}
}

func PointerLengthBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("pointerLength expects 1 argument, got %d", len(args))
		}
		pointer, err := pointerArg(args[0], "pointerLength")
		if err != nil {
			return err
		}
		return NewInteger(int64(pointer.Length))
	}}
}

func PointerByteLengthBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("pointerByteLength expects 1 argument, got %d", len(args))
		}
		pointer, err := pointerArg(args[0], "pointerByteLength")
		if err != nil {
			return err
		}
		return NewInteger(int64(pointer.Length * pointer.ElementType().Size))
	}}
}

func PointerTypeBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("pointerType expects 1 argument, got %d", len(args))
		}
		pointer, err := pointerArg(args[0], "pointerType")
		if err != nil {
			return err
		}
		return NewString(pointer.ElementType().Name)
	}}
}

func PointerAddressBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("pointerAddress expects 1 argument, got %d", len(args))
		}
		pointer, err := pointerArg(args[0], "pointerAddress")
		if err != nil {
			return err
		}
		return &object.FixedInteger{Kind: object.FixedU64, Raw: uint64(pointer.Address())}
	}}
}

func PointerIsNullBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("pointerIsNull expects 1 argument, got %d", len(args))
		}
		pointer, err := pointerArg(args[0], "pointerIsNull")
		if err != nil {
			return err
		}
		return NewBoolean(pointer.IsNull())
	}}
}

func PointerIsValidBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("pointerIsValid expects 1 argument, got %d", len(args))
		}
		pointer, err := pointerArg(args[0], "pointerIsValid")
		if err != nil {
			return err
		}
		if pointer.IsNull() || pointer.Moved || pointer.Released {
			return NewBoolean(false)
		}
		if pointer.Block != nil {
			pointer.Block.Mu.RLock()
			valid := !pointer.Block.Freed
			pointer.Block.Mu.RUnlock()
			return NewBoolean(valid)
		}
		return NewBoolean(pointer.Raw != nil)
	}}
}

func PointerOwnedBuiltin() *object.Builtin {
	return pointerFlagBuiltin("pointerOwned", func(p *object.Pointer) bool { return p.Owned })
}
func PointerBorrowedBuiltin() *object.Builtin {
	return pointerFlagBuiltin("pointerBorrowed", func(p *object.Pointer) bool { return p.Borrowed })
}
func PointerMutableBuiltin() *object.Builtin {
	return pointerFlagBuiltin("pointerMutable", func(p *object.Pointer) bool { return p.Mutable })
}

func pointerFlagBuiltin(name string, flag func(*object.Pointer) bool) *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("%s expects 1 argument, got %d", name, len(args))
		}
		pointer, err := pointerArg(args[0], name)
		if err != nil {
			return err
		}
		return NewBoolean(flag(pointer))
	}}
}

func BorrowPointerBuiltin(mutable bool) *object.Builtin {
	name := "borrowPointer"
	if mutable {
		name = "borrowPointerMut"
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("%s expects 1 argument, got %d", name, len(args))
		}
		pointer, pointerErr := pointerArg(args[0], name)
		if pointerErr != nil {
			return pointerErr
		}
		if pointer.Block == nil || pointer.IsNull() || pointer.Moved || pointer.Released {
			return NewError("%s expects a valid managed pointer", name)
		}
		pointer.Block.Mu.Lock()
		defer pointer.Block.Mu.Unlock()
		if pointer.Block.Freed {
			return NewError("%s: use after free", name)
		}
		if mutable {
			if !pointer.Mutable || pointer.Block.MutableBorrow || pointer.Block.BorrowCount != 0 {
				return NewError("%s requires exclusive mutable access", name)
			}
			pointer.Block.MutableBorrow = true
		} else {
			if pointer.Block.MutableBorrow {
				return NewError("%s cannot overlap a mutable borrow", name)
			}
			pointer.Block.BorrowCount++
		}
		return &object.Pointer{
			Block: pointer.Block, Offset: pointer.Offset, Length: pointer.Length,
			Mutable: mutable, Borrowed: true,
		}
	}}
}

func ReleaseBorrowBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("releaseBorrow expects 1 argument, got %d", len(args))
		}
		pointer, err := pointerArg(args[0], "releaseBorrow")
		if err != nil {
			return err
		}
		if !pointer.Borrowed || pointer.Released || pointer.Block == nil {
			return NewError("releaseBorrow expects an active borrowed pointer")
		}
		pointer.Block.Mu.Lock()
		if pointer.Mutable {
			pointer.Block.MutableBorrow = false
		} else if pointer.Block.BorrowCount > 0 {
			pointer.Block.BorrowCount--
		}
		pointer.Block.Mu.Unlock()
		pointer.Released = true
		return NewBoolean(true)
	}}
}

func MovePointerBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("movePointer expects 1 argument, got %d", len(args))
		}
		pointer, err := pointerArg(args[0], "movePointer")
		if err != nil {
			return err
		}
		if !pointer.Owned || pointer.Moved || pointer.Released || pointer.Block == nil {
			return NewError("movePointer expects a valid owning pointer")
		}
		pointer.Block.Mu.RLock()
		hasBorrows := pointer.Block.BorrowCount != 0 || pointer.Block.MutableBorrow
		pointer.Block.Mu.RUnlock()
		if hasBorrows {
			return NewError("movePointer cannot move memory while borrows are active")
		}
		result := &object.Pointer{Block: pointer.Block, Offset: pointer.Offset, Length: pointer.Length, Owned: true, Mutable: pointer.Mutable}
		pointer.Moved = true
		pointer.Owned = false
		pointer.Length = 0
		return result
	}}
}

func AddressOfBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 || len(args) > 2 {
			return NewError("addressOf expects 1 or 2 arguments, got %d", len(args))
		}
		index := int64(0)
		if len(args) == 2 {
			var indexErr *object.Error
			index, indexErr = integerArg(args[1], "addressOf index")
			if indexErr != nil {
				return indexErr
			}
		}
		switch value := args[0].(type) {
		case *object.Pointer:
			if index < 0 || index > int64(value.Length) {
				return NewError("addressOf index out of bounds")
			}
			result := &object.Pointer{
				Block: value.Block, Offset: value.Offset + int(index), Length: value.Length - int(index),
				Mutable: value.Mutable, Borrowed: true, Raw: value.Raw, RawType: value.RawType,
			}
			if value.Block == nil && value.Raw != nil {
				result.Raw = unsafe.Pointer(uintptr(value.Raw) + uintptr(index)*uintptr(value.ElementType().Size))
				result.Offset = 0
			}
			return result
		case *object.ByteArray:
			if index < 0 || index > int64(len(value.Data)) {
				return NewError("addressOf index out of bounds")
			}
			t, _ := object.ParseSystemType("u8")
			block := &object.MemoryBlock{Data: value.Data, Type: t, Length: len(value.Data), Allocator: "external", Backing: value}
			return &object.Pointer{Block: block, Offset: int(index), Length: len(value.Data) - int(index), Mutable: true, Borrowed: true}
		case *object.TypedArray:
			if index < 0 || index > int64(value.Length) {
				return NewError("addressOf index out of bounds")
			}
			t, _ := object.ParseSystemType(string(value.Kind))
			block := &object.MemoryBlock{Data: value.Data, Type: t, Length: value.Length, Allocator: "external", Backing: value}
			return &object.Pointer{Block: block, Offset: int(index), Length: value.Length - int(index), Mutable: true, Borrowed: true}
		case *object.Slice:
			if index < 0 || index > int64(value.Length) {
				return NewError("addressOf index out of bounds")
			}
			base := AddressOfBuiltin().Fn(value.Source, &object.Integer{Value: int64(value.Start) + index})
			if pointer, ok := base.(*object.Pointer); ok {
				pointer.Length = value.Length - int(index)
			}
			return base
		default:
			return NewError("addressOf expects byte array, typed array, slice or pointer, got %s", args[0].Type())
		}
	}}
}

func PointerCopyBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 5 {
			return NewError("pointerCopy expects 5 arguments, got %d", len(args))
		}
		destination, destinationErr := pointerArg(args[0], "pointerCopy destination")
		if destinationErr != nil {
			return destinationErr
		}
		destinationStart, destinationStartErr := integerArg(args[1], "pointerCopy destination start")
		if destinationStartErr != nil {
			return destinationStartErr
		}
		source, sourceErr := pointerArg(args[2], "pointerCopy source")
		if sourceErr != nil {
			return sourceErr
		}
		sourceStart, sourceStartErr := integerArg(args[3], "pointerCopy source start")
		if sourceStartErr != nil {
			return sourceStartErr
		}
		count, countErr := integerArg(args[4], "pointerCopy count")
		if countErr != nil {
			return countErr
		}
		if count < 0 || destinationStart < 0 || sourceStart < 0 || destinationStart+count > int64(destination.Length) || sourceStart+count > int64(source.Length) {
			return NewError("pointerCopy range is out of bounds")
		}
		values := make([]object.Object, int(count))
		for index := int64(0); index < count; index++ {
			value, err := source.Read(int(sourceStart + index))
			if err != nil {
				return NewError("pointerCopy: %s", err)
			}
			values[index] = value
		}
		for index, value := range values {
			if err := destination.Write(int(destinationStart)+index, value); err != nil {
				return NewError("pointerCopy: %s", err)
			}
		}
		return destination
	}}
}

func PointerFillBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("pointerFill expects 2 arguments, got %d", len(args))
		}
		pointer, err := pointerArg(args[0], "pointerFill")
		if err != nil {
			return err
		}
		for index := 0; index < pointer.Length; index++ {
			if writeErr := pointer.Write(index, args[1]); writeErr != nil {
				return NewError("pointerFill: %s", writeErr)
			}
		}
		return pointer
	}}
}

func SizeOfTypeBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sizeOfType expects 1 argument, got %d", len(args))
		}
		t, err := systemTypeArg(args[0], "sizeOfType")
		if err != nil {
			return err
		}
		return NewInteger(int64(t.Size))
	}}
}

func AlignOfTypeBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("alignOfType expects 1 argument, got %d", len(args))
		}
		t, err := systemTypeArg(args[0], "alignOfType")
		if err != nil {
			return err
		}
		return NewInteger(int64(t.Align))
	}}
}

func ByteSizeOfBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("byteSizeOf expects 1 argument, got %d", len(args))
		}
		switch value := args[0].(type) {
		case *object.Pointer:
			return NewInteger(int64(value.Length * value.ElementType().Size))
		case *object.ByteArray:
			return NewInteger(int64(len(value.Data)))
		case *object.TypedArray:
			return NewInteger(int64(len(value.Data)))
		case *object.Slice:
			base := ByteSizeOfBuiltin().Fn(value.Source)
			if integer, ok := base.(*object.Integer); ok {
				length, _ := collections.Length(value.Source)
				if length == 0 {
					return NewInteger(0)
				}
				return NewInteger(integer.Value / int64(length) * int64(value.Length))
			}
			return base
		default:
			return NewError("byteSizeOf expects pointer or compact memory value, got %s", args[0].Type())
		}
	}}
}

func NativeStructLayoutBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("nativeStructLayout expects 1 argument, got %d", len(args))
		}
		fields, ok := args[0].(*object.Array)
		if !ok {
			return NewError("nativeStructLayout expects an array of field descriptors")
		}
		offset := 0
		maxAlign := 1
		fieldResults := make([]object.Object, 0, len(fields.Elements))
		for _, item := range fields.Elements {
			descriptor, ok := item.(*object.Dict)
			if !ok {
				return NewError("nativeStructLayout fields must be dictionaries")
			}
			nameValue, nameOK := dictGet(descriptor, "name").(*object.String)
			typeValue, typeOK := dictGet(descriptor, "type").(*object.String)
			if !nameOK || !typeOK {
				return NewError("nativeStructLayout fields require string name and type")
			}
			t, typeErr := systemTypeArg(typeValue, "nativeStructLayout field type")
			if typeErr != nil {
				return typeErr
			}
			offset = alignUp(offset, t.Align)
			fieldResults = append(fieldResults, dictFromObjects(map[string]object.Object{
				"name": NewString(nameValue.Value), "type": NewString(t.Name), "offset": NewInteger(int64(offset)),
				"size": NewInteger(int64(t.Size)), "align": NewInteger(int64(t.Align)),
			}))
			offset += t.Size
			if t.Align > maxAlign {
				maxAlign = t.Align
			}
		}
		size := alignUp(offset, maxAlign)
		return dictFromObjects(map[string]object.Object{
			"size": NewInteger(int64(size)), "align": NewInteger(int64(maxAlign)), "fields": &object.Array{Elements: fieldResults},
		})
	}}
}

func alignUp(value, alignment int) int {
	if alignment <= 1 {
		return value
	}
	return (value + alignment - 1) &^ (alignment - 1)
}

func ArenaCreateBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return NewError("arenaCreate expects 0 arguments, got %d", len(args))
		}
		return &object.MemoryArena{}
	}}
}

func arenaArg(value object.Object, label string) (*object.MemoryArena, *object.Error) {
	arena, ok := value.(*object.MemoryArena)
	if !ok {
		return nil, NewError("%s expects MemoryArena, got %s", label, value.Type())
	}
	return arena, nil
}

func ArenaAllocBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("arenaAlloc expects 3 arguments, got %d", len(args))
		}
		arena, arenaErr := arenaArg(args[0], "arenaAlloc")
		if arenaErr != nil {
			return arenaErr
		}
		t, typeErr := systemTypeArg(args[1], "arenaAlloc type")
		if typeErr != nil {
			return typeErr
		}
		count, countErr := integerArg(args[2], "arenaAlloc count")
		if countErr != nil {
			return countErr
		}
		if count < 0 {
			return NewError("arenaAlloc count must be non-negative")
		}
		if !validAllocationCount(t, count) {
			return NewError("arenaAlloc size overflows the native address space")
		}
		arena.Mu.Lock()
		defer arena.Mu.Unlock()
		if arena.Closed {
			return NewError("arenaAlloc cannot use a closed arena")
		}
		block := object.NewMemoryBlock(t, int(count), "arena", nil)
		arena.Blocks = append(arena.Blocks, block)
		arena.Bytes += int64(len(block.Data))
		return &object.Pointer{Block: block, Length: int(count), Mutable: true}
	}}
}

func ArenaResetBuiltin() *object.Builtin { return arenaCloseBuiltin("arenaReset", false) }
func ArenaFreeBuiltin() *object.Builtin  { return arenaCloseBuiltin("arenaFree", true) }

func arenaCloseBuiltin(name string, closeArena bool) *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("%s expects 1 argument, got %d", name, len(args))
		}
		arena, err := arenaArg(args[0], name)
		if err != nil {
			return err
		}
		arena.Mu.Lock()
		defer arena.Mu.Unlock()
		if arena.Closed {
			return NewError("%s cannot use a closed arena", name)
		}
		for _, block := range arena.Blocks {
			object.ReleaseMemoryBlock(block)
		}
		arena.Blocks = nil
		arena.Bytes = 0
		arena.Closed = closeArena
		return NewBoolean(true)
	}}
}

func ArenaStatsBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("arenaStats expects 1 argument, got %d", len(args))
		}
		arena, err := arenaArg(args[0], "arenaStats")
		if err != nil {
			return err
		}
		arena.Mu.Lock()
		defer arena.Mu.Unlock()
		return dictFromObjects(map[string]object.Object{
			"bytes": NewInteger(arena.Bytes), "allocations": NewInteger(int64(len(arena.Blocks))), "open": NewBoolean(!arena.Closed),
		})
	}}
}

func MemoryStatsBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return NewError("memoryStats expects 0 arguments, got %d", len(args))
		}
		return dictInt64(object.MemoryStatsSnapshot())
	}}
}

func MemoryLeaksBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return NewError("memoryLeaks expects 0 arguments, got %d", len(args))
		}
		blocks := object.MemoryLeakBlocks()
		sort.Slice(blocks, func(i, j int) bool { return blocks[i].ID < blocks[j].ID })
		items := make([]object.Object, 0, len(blocks))
		for _, block := range blocks {
			items = append(items, dictFromObjects(map[string]object.Object{
				"id": NewInteger(int64(block.ID)), "type": NewString(block.Type.Name), "elements": NewInteger(int64(block.Length)),
				"bytes": NewInteger(int64(len(block.Data))), "allocator": NewString(block.Allocator),
			}))
		}
		return &object.Array{Elements: items}
	}}
}

func MemoryValidateBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return NewError("memoryValidate expects 0 arguments, got %d", len(args))
		}
		stats := object.MemoryStatsSnapshot()
		return dictFromObjects(map[string]object.Object{
			"ok":            NewBoolean(stats["invalid_accesses"] == 0 && stats["double_frees"] == 0),
			"active_blocks": NewInteger(stats["active_blocks"]), "invalid_accesses": NewInteger(stats["invalid_accesses"]),
			"double_frees": NewInteger(stats["double_frees"]),
		})
	}}
}

func MemoryResetStatsBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return NewError("memoryResetStats expects 0 arguments, got %d", len(args))
		}
		object.ResetMemoryStats()
		return NewBoolean(true)
	}}
}

func MmapOpenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 2 || len(args) > 3 {
			return NewError("mmapOpen expects 2 or 3 arguments, got %d", len(args))
		}
		path, pathErr := stringArg(args[0], "mmapOpen path")
		if pathErr != nil {
			return pathErr
		}
		mode, modeErr := stringArg(args[1], "mmapOpen mode")
		if modeErr != nil {
			return modeErr
		}
		size := int64(0)
		if len(args) == 3 {
			var sizeErr *object.Error
			size, sizeErr = integerArg(args[2], "mmapOpen size")
			if sizeErr != nil {
				return sizeErr
			}
		}
		mapped, err := openMappedMemory(path, mode, size)
		if err != nil {
			return NewError("mmapOpen: %s", err)
		}
		return mapped
	}}
}

func mappedArg(value object.Object, label string) (*object.MappedMemory, *object.Error) {
	mapped, ok := value.(*object.MappedMemory)
	if !ok {
		return nil, NewError("%s expects MappedMemory, got %s", label, value.Type())
	}
	return mapped, nil
}

func MmapPointerBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("mmapPointer expects 1 argument, got %d", len(args))
		}
		mapped, err := mappedArg(args[0], "mmapPointer")
		if err != nil {
			return err
		}
		mapped.Mu.Lock()
		defer mapped.Mu.Unlock()
		if mapped.Closed {
			return NewError("mmapPointer cannot use a closed mapping")
		}
		return &object.Pointer{Block: mapped.Block, Length: len(mapped.Data), Mutable: mapped.Mode != "read", Borrowed: true}
	}}
}

func MmapFlushBuiltin() *object.Builtin { return mappedActionBuiltin("mmapFlush", flushMappedMemory) }
func MmapCloseBuiltin() *object.Builtin { return mappedActionBuiltin("mmapClose", closeMappedMemory) }

func mappedActionBuiltin(name string, action func(*object.MappedMemory) error) *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("%s expects 1 argument, got %d", name, len(args))
		}
		mapped, err := mappedArg(args[0], name)
		if err != nil {
			return err
		}
		if actionErr := action(mapped); actionErr != nil {
			return NewError("%s: %s", name, actionErr)
		}
		return NewBoolean(true)
	}}
}

func MmapSizeBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("mmapSize expects 1 argument, got %d", len(args))
		}
		mapped, err := mappedArg(args[0], "mmapSize")
		if err != nil {
			return err
		}
		return NewInteger(int64(len(mapped.Data)))
	}}
}

func SharedMemoryOpenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 2 || len(args) > 3 {
			return NewError("sharedMemoryOpen expects 2 or 3 arguments, got %d", len(args))
		}
		name, nameErr := stringArg(args[0], "sharedMemoryOpen name")
		if nameErr != nil {
			return nameErr
		}
		size, sizeErr := integerArg(args[1], "sharedMemoryOpen size")
		if sizeErr != nil {
			return sizeErr
		}
		create := true
		if len(args) == 3 {
			value, ok := args[2].(*object.Boolean)
			if !ok {
				return NewError("sharedMemoryOpen create must be bool")
			}
			create = value.Value
		}
		shared, err := openSharedMemory(name, size, create)
		if err != nil {
			return NewError("sharedMemoryOpen: %s", err)
		}
		return shared
	}}
}

func sharedArg(value object.Object, label string) (*object.SharedMemory, *object.Error) {
	shared, ok := value.(*object.SharedMemory)
	if !ok {
		return nil, NewError("%s expects SharedMemory, got %s", label, value.Type())
	}
	return shared, nil
}

func SharedMemoryPointerBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sharedMemoryPointer expects 1 argument, got %d", len(args))
		}
		shared, err := sharedArg(args[0], "sharedMemoryPointer")
		if err != nil {
			return err
		}
		shared.Mu.Lock()
		defer shared.Mu.Unlock()
		if shared.Closed {
			return NewError("sharedMemoryPointer cannot use closed shared memory")
		}
		return &object.Pointer{Block: shared.Block, Length: len(shared.Data), Mutable: true, Borrowed: true}
	}}
}

func SharedMemoryCloseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sharedMemoryClose expects 1 argument, got %d", len(args))
		}
		shared, err := sharedArg(args[0], "sharedMemoryClose")
		if err != nil {
			return err
		}
		if closeErr := closeSharedMemory(shared); closeErr != nil {
			return NewError("sharedMemoryClose: %s", closeErr)
		}
		return NewBoolean(true)
	}}
}

func SharedMemoryUnlinkBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sharedMemoryUnlink expects 1 argument, got %d", len(args))
		}
		name, err := stringArg(args[0], "sharedMemoryUnlink name")
		if err != nil {
			return err
		}
		if unlinkErr := unlinkSharedMemory(name); unlinkErr != nil {
			return NewError("sharedMemoryUnlink: %s", unlinkErr)
		}
		return NewBoolean(true)
	}}
}

func VolatileReadBuiltin() *object.Builtin  { return PointerReadBuiltin() }
func VolatileWriteBuiltin() *object.Builtin { return PointerWriteBuiltin() }

func MemoryFenceBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) > 1 {
			return NewError("memoryFence expects zero or one argument, got %d", len(args))
		}
		mode := "seq_cst"
		if len(args) == 1 {
			var err *object.Error
			mode, err = stringArg(args[0], "memoryFence mode")
			if err != nil {
				return err
			}
		}
		if err := platformMemoryFence(mode); err != nil {
			return NewError("memoryFence: %s", err)
		}
		return &object.Null{}
	}}
}

func AtomicPointerLoadBuiltin() *object.Builtin        { return atomicPointerBuiltin("load") }
func AtomicPointerStoreBuiltin() *object.Builtin       { return atomicPointerBuiltin("store") }
func AtomicPointerAddBuiltin() *object.Builtin         { return atomicPointerBuiltin("add") }
func AtomicPointerSwapBuiltin() *object.Builtin        { return atomicPointerBuiltin("swap") }
func AtomicPointerCompareSwapBuiltin() *object.Builtin { return atomicPointerBuiltin("compare_swap") }

func atomicPointerBuiltin(operation string) *object.Builtin {
	name := "atomicPointer" + strings.Title(strings.ReplaceAll(operation, "_", " "))
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		minimum := 1
		maximum := 2
		if operation == "compare_swap" {
			minimum, maximum = 3, 3
		}
		if len(args) < minimum || len(args) > maximum {
			return NewError("%s expects %d arguments, got %d", name, maximum, len(args))
		}
		pointer, err := pointerArg(args[0], name)
		if err != nil {
			return err
		}
		return platformAtomicPointer(operation, pointer, args[1:]...)
	}}
}

func MemoryProtectBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("memoryProtect expects 2 arguments, got %d", len(args))
		}
		pointer, err := pointerArg(args[0], "memoryProtect")
		if err != nil {
			return err
		}
		mode, modeErr := stringArg(args[1], "memoryProtect mode")
		if modeErr != nil {
			return modeErr
		}
		if protectErr := protectPointerMemory(pointer, mode); protectErr != nil {
			return NewError("memoryProtect: %s", protectErr)
		}
		return NewBoolean(true)
	}}
}

func MemoryLockBuiltin() *object.Builtin {
	return pointerPlatformActionBuiltin("memoryLock", lockPointerMemory)
}
func MemoryUnlockBuiltin() *object.Builtin {
	return pointerPlatformActionBuiltin("memoryUnlock", unlockPointerMemory)
}

func pointerPlatformActionBuiltin(name string, action func(*object.Pointer) error) *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("%s expects 1 argument, got %d", name, len(args))
		}
		pointer, err := pointerArg(args[0], name)
		if err != nil {
			return err
		}
		if actionErr := action(pointer); actionErr != nil {
			return NewError("%s: %s", name, actionErr)
		}
		return NewBoolean(true)
	}}
}

func DynamicOpenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("dynamicOpen expects 1 argument, got %d", len(args))
		}
		path, err := stringArg(args[0], "dynamicOpen path")
		if err != nil {
			return err
		}
		library, openErr := openDynamicLibrary(path)
		if openErr != nil {
			return NewError("dynamicOpen: %s", openErr)
		}
		return library
	}}
}

func libraryArg(value object.Object, label string) (*object.DynamicLibrary, *object.Error) {
	library, ok := value.(*object.DynamicLibrary)
	if !ok {
		return nil, NewError("%s expects DynamicLibrary, got %s", label, value.Type())
	}
	return library, nil
}

func DynamicSymbolBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("dynamicSymbol expects 2 arguments, got %d", len(args))
		}
		library, libraryErr := libraryArg(args[0], "dynamicSymbol")
		if libraryErr != nil {
			return libraryErr
		}
		name, nameErr := stringArg(args[1], "dynamicSymbol name")
		if nameErr != nil {
			return nameErr
		}
		pointer, symbolErr := lookupDynamicSymbol(library, name)
		if symbolErr != nil {
			library.LastErr = symbolErr.Error()
			return NewError("dynamicSymbol: %s", symbolErr)
		}
		return pointer
	}}
}

func DynamicCallBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("dynamicCall expects symbol, return type and argument array")
		}
		symbol, symbolErr := pointerArg(args[0], "dynamicCall symbol")
		if symbolErr != nil {
			return symbolErr
		}
		returnName, returnErr := stringArg(args[1], "dynamicCall return type")
		if returnErr != nil {
			return returnErr
		}
		argumentArray, ok := args[2].(*object.Array)
		if !ok {
			return NewError("dynamicCall arguments must be an array")
		}
		if len(argumentArray.Elements) > 6 {
			return NewError("dynamicCall supports at most 6 arguments")
		}
		nativeArgs := make([]uintptr, len(argumentArray.Elements))
		for index, value := range argumentArray.Elements {
			switch current := value.(type) {
			case *object.Integer:
				nativeArgs[index] = uintptr(current.Value)
			case *object.FixedInteger:
				nativeArgs[index] = uintptr(current.UnsignedValue())
			case *object.Boolean:
				if current.Value {
					nativeArgs[index] = 1
				}
			case *object.Pointer:
				nativeArgs[index] = current.Address()
			default:
				return NewError("dynamicCall argument %d must be integer, bool or Pointer, got %s", index, value.Type())
			}
		}
		raw, err := callDynamicFunction(symbol.Address(), nativeArgs)
		if err != nil {
			return NewError("dynamicCall: %s", err)
		}
		normalized := strings.ToLower(strings.TrimSpace(returnName))
		if normalized == "void" || normalized == "null" {
			return &object.Null{}
		}
		t, ok := object.ParseSystemType(normalized)
		if !ok {
			return NewError("dynamicCall uses unsupported return type %q", returnName)
		}
		if t.Float {
			return NewError("dynamicCall does not support floating-point ABI returns")
		}
		if t.Bool {
			return NewBoolean(raw != 0)
		}
		if t.Ptr {
			return &object.Pointer{Raw: unsafe.Pointer(raw), RawType: t, Mutable: false}
		}
		if t.Kind != "" {
			return object.NewFixedIntegerRaw(t.Kind, uint64(raw))
		}
		return NewInteger(int64(raw))
	}}
}

func DynamicCloseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("dynamicClose expects 1 argument, got %d", len(args))
		}
		library, err := libraryArg(args[0], "dynamicClose")
		if err != nil {
			return err
		}
		if closeErr := closeDynamicLibrary(library); closeErr != nil {
			return NewError("dynamicClose: %s", closeErr)
		}
		return NewBoolean(true)
	}}
}

func DynamicIsOpenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("dynamicIsOpen expects 1 argument, got %d", len(args))
		}
		library, err := libraryArg(args[0], "dynamicIsOpen")
		if err != nil {
			return err
		}
		library.Mu.Lock()
		open := !library.Closed
		library.Mu.Unlock()
		return NewBoolean(open)
	}}
}

func DynamicErrorBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("dynamicError expects 1 argument, got %d", len(args))
		}
		library, err := libraryArg(args[0], "dynamicError")
		if err != nil {
			return err
		}
		return NewString(library.LastErr)
	}}
}

func SystemInfoBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return NewError("systemInfo expects 0 arguments, got %d", len(args))
		}
		var sample uint16 = 0x1
		endianness := "big"
		if *(*byte)(unsafe.Pointer(&sample)) == 1 {
			endianness = "little"
		}
		return dictFromObjects(map[string]object.Object{
			"os": NewString(runtime.GOOS), "arch": NewString(runtime.GOARCH), "cpu_count": NewInteger(int64(runtime.NumCPU())),
			"page_size": NewInteger(int64(os.Getpagesize())), "pointer_bits": NewInteger(int64(unsafe.Sizeof(uintptr(0)) * 8)),
			"endianness": NewString(endianness),
		})
	}}
}

func PageSizeBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return NewError("pageSize expects 0 arguments, got %d", len(args))
		}
		return NewInteger(int64(os.Getpagesize()))
	}}
}

func CPUCountBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return NewError("cpuCount expects 0 arguments, got %d", len(args))
		}
		return NewInteger(int64(runtime.NumCPU()))
	}}
}

func RawSyscallBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("rawSyscall expects 2 arguments, got %d", len(args))
		}
		if os.Getenv("ZUMBRA_ALLOW_RAW_SYSCALLS") != "1" {
			return NewError("rawSyscall is disabled; set ZUMBRA_ALLOW_RAW_SYSCALLS=1 to enable it")
		}
		number, numberErr := integerArg(args[0], "rawSyscall number")
		if numberErr != nil {
			return numberErr
		}
		array, ok := args[1].(*object.Array)
		if !ok || len(array.Elements) > 6 {
			return NewError("rawSyscall arguments must be an array with at most 6 integers")
		}
		values := make([]uintptr, 6)
		for index, value := range array.Elements {
			integer, err := integerArg(value, fmt.Sprintf("rawSyscall argument %d", index))
			if err != nil {
				return err
			}
			values[index] = uintptr(integer)
		}
		result, errno, callErr := platformRawSyscall(uintptr(number), values)
		return dictFromObjects(map[string]object.Object{
			"ok": NewBoolean(callErr == nil), "value": &object.FixedInteger{Kind: object.FixedU64, Raw: uint64(result)},
			"errno": NewInteger(int64(errno)), "error": NewString(errorString(callErr)),
		})
	}}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func ProfileNowBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return NewError("profileNowNs expects 0 arguments, got %d", len(args))
		}
		return &object.FixedInteger{Kind: object.FixedU64, Raw: uint64(time.Now().UnixNano())}
	}}
}

func ProfileElapsedBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("profileElapsedNs expects 1 argument, got %d", len(args))
		}
		start, err := integerArg(args[0], "profileElapsedNs start")
		if err != nil {
			if fixed, ok := args[0].(*object.FixedInteger); ok {
				start = int64(fixed.UnsignedValue())
			} else {
				return err
			}
		}
		return &object.FixedInteger{Kind: object.FixedU64, Raw: uint64(time.Now().UnixNano() - start)}
	}}
}

func registerZ17Builtins() {
	entries := []struct {
		name string
		fn   *object.Builtin
	}{
		{"alloc", AllocBuiltin()}, {"calloc", CallocBuiltin()}, {"nullPointer", NullPointerBuiltin()}, {"realloc", ReallocBuiltin()}, {"free", FreeBuiltin()},
		{"addressOf", AddressOfBuiltin()}, {"pointerFromAddress", PointerFromAddressBuiltin()}, {"dereference", DereferenceBuiltin()}, {"pointerRead", PointerReadBuiltin()}, {"pointerWrite", PointerWriteBuiltin()},
		{"pointerOffset", PointerOffsetBuiltin()}, {"pointerLength", PointerLengthBuiltin()}, {"pointerByteLength", PointerByteLengthBuiltin()}, {"pointerType", PointerTypeBuiltin()},
		{"pointerAddress", PointerAddressBuiltin()}, {"pointerEqual", PointerEqualBuiltin()}, {"pointerCompare", PointerCompareBuiltin()}, {"pointerIsAligned", PointerIsAlignedBuiltin()}, {"pointerIsNull", PointerIsNullBuiltin()}, {"pointerIsValid", PointerIsValidBuiltin()}, {"pointerOwned", PointerOwnedBuiltin()},
		{"pointerBorrowed", PointerBorrowedBuiltin()}, {"pointerMutable", PointerMutableBuiltin()}, {"borrowPointer", BorrowPointerBuiltin(false)}, {"borrowPointerMut", BorrowPointerBuiltin(true)},
		{"releaseBorrow", ReleaseBorrowBuiltin()}, {"movePointer", MovePointerBuiltin()}, {"pointerCopy", PointerCopyBuiltin()}, {"pointerFill", PointerFillBuiltin()},
		{"sizeOfType", SizeOfTypeBuiltin()}, {"alignOfType", AlignOfTypeBuiltin()}, {"byteSizeOf", ByteSizeOfBuiltin()}, {"nativeStructLayout", NativeStructLayoutBuiltin()},
		{"arenaCreate", ArenaCreateBuiltin()}, {"arenaAlloc", ArenaAllocBuiltin()}, {"arenaReset", ArenaResetBuiltin()}, {"arenaFree", ArenaFreeBuiltin()}, {"arenaStats", ArenaStatsBuiltin()},
		{"memoryStats", MemoryStatsBuiltin()}, {"memoryLeaks", MemoryLeaksBuiltin()}, {"memoryValidate", MemoryValidateBuiltin()}, {"memoryResetStats", MemoryResetStatsBuiltin()},
		{"mmapOpen", MmapOpenBuiltin()}, {"mmapPointer", MmapPointerBuiltin()}, {"mmapFlush", MmapFlushBuiltin()}, {"mmapClose", MmapCloseBuiltin()}, {"mmapSize", MmapSizeBuiltin()},
		{"sharedMemoryOpen", SharedMemoryOpenBuiltin()}, {"sharedMemoryPointer", SharedMemoryPointerBuiltin()}, {"sharedMemoryClose", SharedMemoryCloseBuiltin()}, {"sharedMemoryUnlink", SharedMemoryUnlinkBuiltin()},
		{"volatileRead", VolatileReadBuiltin()}, {"volatileWrite", VolatileWriteBuiltin()}, {"memoryFence", MemoryFenceBuiltin()},
		{"atomicPointerLoad", AtomicPointerLoadBuiltin()}, {"atomicPointerStore", AtomicPointerStoreBuiltin()}, {"atomicPointerAdd", AtomicPointerAddBuiltin()},
		{"atomicPointerSwap", AtomicPointerSwapBuiltin()}, {"atomicPointerCompareSwap", AtomicPointerCompareSwapBuiltin()},
		{"memoryProtect", MemoryProtectBuiltin()}, {"memoryLock", MemoryLockBuiltin()}, {"memoryUnlock", MemoryUnlockBuiltin()},
		{"dynamicOpen", DynamicOpenBuiltin()}, {"dynamicSymbol", DynamicSymbolBuiltin()}, {"dynamicCall", DynamicCallBuiltin()}, {"dynamicClose", DynamicCloseBuiltin()}, {"dynamicIsOpen", DynamicIsOpenBuiltin()}, {"dynamicError", DynamicErrorBuiltin()},
		{"systemInfo", SystemInfoBuiltin()}, {"pageSize", PageSizeBuiltin()}, {"cpuCount", CPUCountBuiltin()}, {"rawSyscall", RawSyscallBuiltin()},
		{"profileNowNs", ProfileNowBuiltin()}, {"profileElapsedNs", ProfileElapsedBuiltin()},
	}
	for _, entry := range entries {
		registerZ12Builtin(entry.name, entry.fn)
	}
}

func init() { registerZ17Builtins() }
