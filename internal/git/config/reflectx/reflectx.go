package reflectx

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

type sentinelError string

func (e sentinelError) Error() string { return string(e) }

// ErrNotFound indicates that a requested field was not found in a struct.
const ErrNotFound = sentinelError("not found")

// Option configures the behavior of path walking operations.
type Option func(*options)

// WithUnsafeAccess enables unsafe operations for setting unexported fields.
func WithUnsafeAccess() Option { return func(o *options) { o.allowUnsafeAccess = true } }

type options struct {
	allowUnsafeAccess bool
}

// WalkToPath navigates to a field at the given path in a struct and returns the field value and write operations.
func WalkToPath[T any](dst *T, path string, opts ...Option) (reflect.Value, func() error, error) {
	x, v, err := newPathWalker[T](dst, path, opts...)
	if err != nil {
		return reflect.Value{}, nil, fmt.Errorf("unable to create path walker: %w", err)
	}

	field, err := x.walk(v, strings.Split(path, "."))
	if err != nil {
		return reflect.Value{}, nil, fmt.Errorf("unable to walk to path %q: %w", path, err)
	}

	return field, x.applyWriteOperations, nil
}

// SetToPath sets a field at the given path in a struct to the specified string value.
func SetToPath[T any](dst *T, path, value string, opts ...Option) error {
	x, v, err := newPathWalker[T](dst, path, opts...)
	if err != nil {
		return fmt.Errorf("unable to create path walker: %w", err)
	}

	field, err := x.walk(v, strings.Split(path, "."))
	if err != nil {
		return fmt.Errorf("unable to walk to path %q: %w", path, err)
	}

	if err := x.setValue(field, value); err != nil {
		return fmt.Errorf("unable to set field value to path %q: %w", path, err)
	}

	if err := x.applyWriteOperations(); err != nil {
		return fmt.Errorf("unable to apply write operations to dst: %w", err)
	}

	return nil
}

func newPathWalker[T any](dst *T, path string, opts ...Option) (*pathWalker, reflect.Value, error) {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return nil, reflect.Value{}, errors.New("dst must be a non nil pointer to a struct")
	}

	v = v.Elem()

	if v.Kind() != reflect.Struct {
		return nil, reflect.Value{}, fmt.Errorf("expected dst to be a struct, got %T", dst)
	}

	if path == "" {
		return nil, reflect.Value{}, errors.New("path cannot be empty")
	}

	o := options{
		allowUnsafeAccess: false,
	}
	for _, opt := range opts {
		opt(&o)
	}

	return &pathWalker{allowUnsafe: o.allowUnsafeAccess}, v, nil
}

type pathWalker struct {
	allowUnsafe bool
	wo          []func() error
}

func (pw *pathWalker) applyWriteOperations() error {
	for i := range len(pw.wo) {
		if err := pw.wo[len(pw.wo)-1-i](); err != nil {
			return err
		}
	}

	return nil
}

func (pw *pathWalker) setValue(v reflect.Value, value string) error {
	value = strings.TrimSpace(value)

	switch v.Kind() {
	case reflect.String:
		v.SetString(value)
	case reflect.Bool:
		x, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("unable to parse bool value %q: %w", value, err)
		}

		pw.wo = append(pw.wo, func() error {
			v.SetBool(x)
			return nil
		})

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		x, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse int value %q: %w", value, err)
		}

		pw.wo = append(pw.wo, func() error {
			v.SetInt(x)
			return nil
		})

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		x, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse uint value %q: %w", value, err)
		}

		pw.wo = append(pw.wo, func() error {
			v.SetUint(x)
			return nil
		})

	case reflect.Float32, reflect.Float64:
		x, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("unable to parse float value %q: %w", value, err)
		}

		pw.wo = append(pw.wo, func() error {
			v.SetFloat(x)
			return nil
		})

	case reflect.Ptr:
		v = pw.ensurePointerInitialized(v)
		return pw.setValue(v.Elem(), value)
	case reflect.Slice:
		parts := strings.Split(value, ",")
		slice := reflect.MakeSlice(v.Type(), len(parts), len(parts))

		for i, part := range parts {
			part = strings.TrimSpace(part)
			elem := slice.Index(i)

			if err := pw.setValue(elem, part); err != nil {
				return fmt.Errorf("unable to set slice element %d: %w", i, err)
			}
		}

		pw.wo = append(pw.wo, func() error {
			v.Set(slice)
			return nil
		})

	default:
		return fmt.Errorf("unsupported field type: %v", v.Kind())
	}

	return nil
}

func (pw *pathWalker) walk(v reflect.Value, path []string) (reflect.Value, error) {
	if len(path) == 0 {
		return v, nil
	}

	switch v.Kind() {
	case reflect.Ptr:
		v = pw.ensurePointerInitialized(v)
		return pw.walk(v.Elem(), path)
	case reflect.Struct:
		return pw.walkInStruct(v, path)
	case reflect.Map:
		return pw.walkInMap(v, path)
	case reflect.Slice:
		return pw.walkInSlice(v, path)
	default:
		return reflect.Value{}, fmt.Errorf("unhandled type %s to handle remaining path %v", v.Kind().String(), strings.Join(path, "."))
	}
}

func (pw *pathWalker) walkInStruct(v reflect.Value, path []string) (reflect.Value, error) {
	if len(path) == 0 {
		return reflect.Value{}, errors.New("path cannot be empty")
	}

	if !v.IsValid() || v.Kind() != reflect.Struct {
		return reflect.Value{}, errors.New("provided value should be valid and of type struct")
	}

	field, err := pw.getStructFieldByName(v, strings.TrimSpace(path[0]))
	if err != nil {
		return reflect.Value{}, fmt.Errorf("unable to find field: %w", err)
	}

	return pw.walk(field, path[1:])
}

// v.FieldByName() would panic if any field retrieved is nil
// so we need to find the indexes, and deal with each nil possibilities
func (pw *pathWalker) getStructFieldByName(v reflect.Value, fieldName string) (reflect.Value, error) {
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return reflect.Value{}, errors.New("provided value should be valid and of type struct")
	}

	sfield, found := v.Type().FieldByNameFunc(func(name string) bool {
		return strings.EqualFold(name, fieldName)
	})
	if !found {
		return reflect.Value{}, fmt.Errorf("struct field %w", ErrNotFound)
	}

	for i, idx := range sfield.Index {
		v = v.Field(idx)

		if v.Kind() == reflect.Ptr {
			v = pw.ensurePointerInitialized(v).Elem()
		}

		if i < len(sfield.Index)-1 && v.Kind() != reflect.Struct {
			return reflect.Value{}, fmt.Errorf("field %d is expected to be a struct, got %T", idx, v)
		}
	}

	return v, nil
}

func (pw *pathWalker) walkInMap(v reflect.Value, path []string) (reflect.Value, error) {
	if len(path) == 0 {
		return reflect.Value{}, errors.New("path cannot be empty")
	}

	if !v.IsValid() || v.Kind() != reflect.Map {
		return reflect.Value{}, errors.New("provided value should be valid and of type map")
	}

	if v.IsNil() {
		oldV := v
		newV := reflect.MakeMap(v.Type())

		pw.wo = append(pw.wo, func() error {
			return pw.reflectSetValue(oldV, newV)
		})

		v = newV
	}

	mapKeyType := v.Type().Key()
	if mapKeyType.Kind() != reflect.String {
		return reflect.Value{}, fmt.Errorf("only string keys are supported for maps, got: %T", mapKeyType)
	}

	mapKeyValue := reflect.ValueOf(strings.TrimSpace(path[0]))
	mapValueValue := reflect.New(v.Type().Elem()).Elem()

	if existingMapValueValue := v.MapIndex(mapKeyValue); existingMapValueValue.IsValid() {
		// we need to recreate the existing value because map values are not addressable
		if err := pw.reflectSetValue(mapValueValue, existingMapValueValue); err != nil {
			return reflect.Value{}, fmt.Errorf("boom: %w", err)
		}
	}

	pv, err := pw.walk(mapValueValue, path[1:])
	if err == nil {
		pw.wo = append(pw.wo, func() error {
			v.SetMapIndex(mapKeyValue, mapValueValue)
			return nil
		})
	}

	return pv, err
}

func (pw *pathWalker) walkInSlice(v reflect.Value, path []string) (reflect.Value, error) {
	if len(path) == 0 {
		return reflect.Value{}, errors.New("path cannot be empty")
	}

	if !v.IsValid() || v.Kind() != reflect.Slice {
		return reflect.Value{}, errors.New("provided value should be valid and of type slice")
	}

	rawIndex := strings.TrimSpace(path[0])

	index, err := strconv.Atoi(rawIndex)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("unable to parse slice index %s: %w", rawIndex, err)
	}

	if v.Len() <= index {
		oldV := v
		newV := reflect.MakeSlice(v.Type(), index+1, index+1)
		reflect.Copy(newV, v)

		pw.wo = append(pw.wo, func() error {
			return pw.reflectSetValue(oldV, newV)
		})

		v = newV
	}

	return pw.walk(v.Index(index), path[1:])
}

func (pw *pathWalker) ensurePointerInitialized(v reflect.Value) reflect.Value {
	if !v.IsNil() {
		return v
	}

	oldV := v
	newV := reflect.New(v.Type().Elem())

	pw.wo = append(pw.wo, func() error {
		return pw.reflectSetValue(oldV, newV)
	})

	return newV
}

func (pw *pathWalker) reflectSetValue(dst, src reflect.Value) error {
	if !dst.IsValid() || !src.IsValid() {
		return fmt.Errorf("invalid value: dst.IsValid=%v src.IsValid=%v", dst.IsValid(), src.IsValid())
	}

	if dst.Type() != src.Type() {
		return fmt.Errorf("type mismatch: %v != %v", dst.Type(), src.Type())
	}

	if dst.CanSet() {
		dst.Set(src)
		return nil
	}

	if !dst.CanAddr() {
		return fmt.Errorf("dst is not addressable; type=%v", dst.Type())
	}

	if !pw.allowUnsafe {
		return fmt.Errorf("dst can't be set without using unsafe, and unsafe is disabled; dst=%v", dst.Type())
	}

	dst = reflect.NewAt(dst.Type(), unsafe.Pointer(dst.UnsafeAddr())).Elem()
	dst.Set(src)

	return nil
}
