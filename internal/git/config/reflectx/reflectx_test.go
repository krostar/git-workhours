package reflectx

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/krostar/test"
	"github.com/krostar/test/check"
)

func Test_WalkToPath(t *testing.T) {
	t.Run("provided dst must be a pointer to struct", func(t *testing.T) {
		var s string

		v, applyWO, err := WalkToPath(&s, "impossible")
		test.Require(t, err != nil && strings.Contains(err.Error(), "expected dst to be a struct, got *string"), err)
		test.Assert(t, applyWO == nil)
		test.Assert(t, !v.IsValid())
	})

	t.Run("provided dst must be non nil", func(t *testing.T) {
		var dst *struct{}

		v, applyWO, err := WalkToPath(dst, "impossible")
		test.Require(t, err != nil && strings.Contains(err.Error(), "dst must be a non nil pointer to a struct"), err)
		test.Assert(t, applyWO == nil)
		test.Assert(t, !v.IsValid())
	})

	t.Run("provided path must be non empty", func(t *testing.T) {
		var dst struct{}

		v, applyWO, err := WalkToPath(&dst, "")
		test.Require(t, err != nil && strings.Contains(err.Error(), "path cannot be empty"), err)
		test.Assert(t, applyWO == nil)
		test.Assert(t, !v.IsValid())
	})

	t.Run("non existing path must be distinct error", func(t *testing.T) {
		var dst struct{}

		v, applyWO, err := WalkToPath(&dst, "notexisting")
		test.Require(t, err != nil && errors.Is(err, ErrNotFound) && strings.Contains(err.Error(), "unable to find field"), err)
		test.Assert(t, applyWO == nil)
		test.Assert(t, !v.IsValid())
	})

	t.Run("any walking error means error, and dst in left as-is", func(t *testing.T) {
		var dst struct {
			Foo *struct {
				Bar int
			}
		}

		v, _, err := WalkToPath(&dst, "foo.bar.notpossible")
		test.Require(t, err != nil && strings.Contains(err.Error(), "unhandled type int to handle remaining path notpossible"), err)
		test.Assert(t, !v.IsValid())
		test.Assert(t, dst.Foo == nil, "even tho we need to instantiate Foo struct to go deeper, the original Foo should be left untouched")
	})

	t.Run("walking success return the right value, and dst is left as is", func(t *testing.T) {
		var dst struct {
			Foo *struct {
				Bar bool
			}
		}

		v, applyWO, err := WalkToPath(&dst, "foo.bar")
		test.Require(t, err == nil, err)
		test.Require(t, v.IsValid() && v.Kind() == reflect.Bool, "%+v", v)

		test.Assert(t, dst.Foo == nil, "even tho we have a valid value pointing to Bar, dst.Foo should be left untouched")
		v.SetBool(true)
		test.Assert(t, dst.Foo == nil, "even tho we have a valid value pointing to Bar, dst.Foo should be left untouched")

		test.Assert(func() (test.TestingT, bool, error) { err := applyWO(); return t, err == nil, err }())

		test.Assert(t, dst.Foo != nil && dst.Foo.Bar, "applyWO should have applied all the things resulting in a dst.Foo now being initialized")
	})

	t.Run("walking success and non-exported field along the way, without unsafe", func(t *testing.T) {
		var dst struct {
			foo *struct {
				Bar bool
			}
		}

		v, applyWO, err := WalkToPath(&dst, "foo.bar")
		test.Require(t, err == nil, err)
		test.Require(t, v.IsValid() && v.Kind() == reflect.Bool, "%+v", v)

		test.Assert(func() (test.TestingT, bool, error) {
			err := applyWO()
			return t, err != nil && strings.Contains(err.Error(), "dst can't be set without using unsafe, and unsafe is disabled"), err
		}())
		test.Assert(t, dst.foo == nil)
	})

	t.Run("walking success and non-exported field along the way, with unsafe", func(t *testing.T) {
		var dst struct {
			foo *struct {
				Bar bool
			}
		}

		v, applyWO, err := WalkToPath(&dst, "foo.bar", WithUnsafeAccess())
		test.Require(t, err == nil, err)
		test.Require(t, v.IsValid() && v.Kind() == reflect.Bool, "%+v", v)

		test.Assert(func() (test.TestingT, bool, error) { err := applyWO(); return t, err == nil, err }())
		test.Assert(t, dst.foo != nil)
	})
}

func Test_SetToPath(t *testing.T) {
	t.Run("provided dst must be a pointer to struct", func(t *testing.T) {
		var s string

		err := SetToPath(&s, "impossible", "unset")
		test.Require(t, err != nil && strings.Contains(err.Error(), "expected dst to be a struct, got *string"), err)
	})

	t.Run("provided dst must be non nil", func(t *testing.T) {
		var dst *struct{}

		err := SetToPath(dst, "impossible", "unset")
		test.Require(t, err != nil && strings.Contains(err.Error(), "dst must be a non nil pointer to a struct"), err)
	})

	t.Run("provided path must be non empty", func(t *testing.T) {
		var dst struct{}

		err := SetToPath(&dst, "", "unset")
		test.Require(t, err != nil && strings.Contains(err.Error(), "path cannot be empty"), err)
	})

	t.Run("non existing path must be distinct error", func(t *testing.T) {
		var dst struct{}

		err := SetToPath(&dst, "notexisting", "unset")
		test.Require(t, err != nil && errors.Is(err, ErrNotFound) && strings.Contains(err.Error(), "unable to find field"), err)
	})

	t.Run("any walking error means error, and dst in left as-is", func(t *testing.T) {
		var dst struct {
			Foo *struct {
				Bar int
			}
		}

		err := SetToPath(&dst, "foo.bar.notpossible", "unset")
		test.Require(t, err != nil && strings.Contains(err.Error(), "unhandled type int to handle remaining path notpossible"), err)
		test.Assert(t, dst.Foo == nil, "even tho we need to instantiate Foo struct to go deeper, the original Foo should be left untouched")
	})

	t.Run("path found, field set", func(t *testing.T) {
		var dst struct {
			Foo *struct {
				Bar bool
			}
		}

		err := SetToPath(&dst, "foo.bar", "1")
		test.Require(t, err == nil, err)
		test.Assert(t, dst.Foo != nil && dst.Foo.Bar)
	})

	t.Run("path found but non-exported without unsafe", func(t *testing.T) {
		var dst struct {
			foo *struct {
				Bar bool
			}
		}

		err := SetToPath(&dst, "foo.bar", "1")
		test.Require(t, err != nil && strings.Contains(err.Error(), "dst can't be set without using unsafe, and unsafe is disabled"), err)
		test.Assert(t, dst.foo == nil)
	})

	t.Run("path found but non-exported with unsafe", func(t *testing.T) {
		var dst struct {
			foo *struct {
				Bar bool
			}
		}

		err := SetToPath(&dst, "foo.bar", "1", WithUnsafeAccess())
		test.Require(t, err == nil, err)
		test.Assert(t, dst.foo != nil && dst.foo.Bar)
	})

	t.Run("path found but value is impossible", func(t *testing.T) {
		var dst struct {
			Foo *struct {
				Bar bool
			}
		}

		err := SetToPath(&dst, "foo.bar", "42")
		test.Require(t, err != nil && strings.Contains(err.Error(), `unable to set field value to path "foo.bar": unable to parse bool value "42"`), err)
		test.Assert(t, dst.Foo == nil)
	})
}

func Test_pathWalker_applyWriteOperations(t *testing.T) {
	called := make(map[string]string)

	for name, tc := range map[string]struct {
		setup              func(t *testing.T) *pathWalker
		expectCalled       string
		expectErrorMessage string
	}{
		"no operations": {
			setup: func(*testing.T) *pathWalker { return new(pathWalker) },
		},
		"single operation success": {
			setup: func(t *testing.T) *pathWalker {
				return &pathWalker{wo: []func() error{
					func() error {
						called[t.Name()] += "1"
						return nil
					},
				}}
			},
			expectCalled: "1",
		},
		"operation fails": {
			setup: func(*testing.T) *pathWalker {
				return &pathWalker{wo: []func() error{func() error { return errors.New("test error") }}}
			},
			expectErrorMessage: "test error",
		},
		"multiple operations reverse order": {
			setup: func(t *testing.T) *pathWalker {
				return &pathWalker{wo: []func() error{
					func() error {
						called[t.Name()] += "1"
						return nil
					},
					func() error {
						called[t.Name()] += "2"
						return nil
					},
				}}
			},
			expectCalled: "21",
		},
	} {
		t.Run(name, func(t *testing.T) {
			pw := tc.setup(t)

			err := pw.applyWriteOperations()
			if tc.expectErrorMessage != "" {
				test.Require(t, err != nil && strings.Contains(err.Error(), tc.expectErrorMessage), err)
			} else {
				test.Require(t, err == nil, err)
				test.Assert(t, tc.expectCalled == called[t.Name()])
			}
		})
	}
}

func Test_pathWalker_setValue(t *testing.T) {
	for name, tc := range map[string]struct {
		value              reflect.Value
		input              string
		expectErrorMessage string
		expectedValue      reflect.Value
	}{
		"string": {
			value:         func() reflect.Value { var v string; return reflect.ValueOf(&v).Elem() }(),
			input:         "test",
			expectedValue: reflect.ValueOf("test"),
		},
		"bool true": {
			value:         func() reflect.Value { var v bool; return reflect.ValueOf(&v).Elem() }(),
			input:         "true",
			expectedValue: reflect.ValueOf(true),
		},
		"bool false": {
			value:         func() reflect.Value { var v bool; return reflect.ValueOf(&v).Elem() }(),
			input:         "false",
			expectedValue: reflect.ValueOf(false),
		},
		"bool invalid": {
			value:              func() reflect.Value { var v bool; return reflect.ValueOf(&v).Elem() }(),
			input:              "invalid",
			expectErrorMessage: "unable to parse bool value",
		},
		"int": {
			value:         func() reflect.Value { var v int; return reflect.ValueOf(&v).Elem() }(),
			input:         "42",
			expectedValue: reflect.ValueOf(42),
		},
		"int invalid": {
			value:              func() reflect.Value { var v int; return reflect.ValueOf(&v).Elem() }(),
			input:              "invalid",
			expectErrorMessage: "unable to parse int value",
		},
		"uint": {
			value:         func() reflect.Value { var v uint; return reflect.ValueOf(&v).Elem() }(),
			input:         "42",
			expectedValue: reflect.ValueOf(uint(42)),
		},
		"uint invalid": {
			value:              func() reflect.Value { var v uint; return reflect.ValueOf(&v).Elem() }(),
			input:              "invalid",
			expectErrorMessage: "unable to parse uint value",
		},
		"float64": {
			value:         func() reflect.Value { var v float64; return reflect.ValueOf(&v).Elem() }(),
			input:         "3.14",
			expectedValue: reflect.ValueOf(3.14),
		},
		"float64 invalid": {
			value:              func() reflect.Value { var v float64; return reflect.ValueOf(&v).Elem() }(),
			input:              "invalid",
			expectErrorMessage: "unable to parse float value",
		},
		"pointer to string": {
			value:         func() reflect.Value { var v *string; return reflect.ValueOf(&v).Elem() }(),
			input:         "test",
			expectedValue: func() reflect.Value { v := "test"; return reflect.ValueOf(&v) }(),
		},
		"slice of strings": {
			value:         func() reflect.Value { var v []string; return reflect.ValueOf(&v).Elem() }(),
			input:         "a,b,c",
			expectedValue: reflect.ValueOf([]string{"a", "b", "c"}),
		},
		"slice of invalid": {
			value:              func() reflect.Value { var v []int; return reflect.ValueOf(&v).Elem() }(),
			input:              "a,b,c",
			expectErrorMessage: `unable to set slice element 0: unable to parse int value "a"`,
		},
		"unsupported type": {
			value:              func() reflect.Value { var v complex64; return reflect.ValueOf(&v).Elem() }(),
			input:              "invalid",
			expectErrorMessage: "unsupported field type",
		},
	} {
		t.Run(name, func(t *testing.T) {
			pw := new(pathWalker)

			err := pw.setValue(tc.value, tc.input)
			if tc.expectErrorMessage != "" {
				test.Require(t, err != nil && strings.Contains(err.Error(), tc.expectErrorMessage), err)
			} else {
				test.Require(t, err == nil, err)
				test.Assert(func() (test.TestingT, bool, error) { err := pw.applyWriteOperations(); return t, err == nil, err }())
				test.Assert(check.Compare(t, tc.value.Interface(), tc.expectedValue.Interface()))
			}
		})
	}
}

func Test_pathWalker_walk(t *testing.T) {
	for name, tc := range map[string]struct {
		value              reflect.Value
		path               []string
		expectValue        func(t *testing.T, applyWO func() error, o, v reflect.Value) (test.TestingT, bool)
		expectErrorMessage string
	}{
		"empty path": {
			value: reflect.ValueOf(&struct{}{}).Elem(),
			path:  []string{},
			expectValue: func(t *testing.T, _ func() error, o, v reflect.Value) (test.TestingT, bool) {
				return t, o.Addr().Pointer() == v.Addr().Pointer()
			},
		},
		"struct": {
			value: reflect.ValueOf(&struct{ Field int }{}).Elem(),
			path:  []string{"field"},
			expectValue: func(t *testing.T, _ func() error, o, v reflect.Value) (test.TestingT, bool) {
				v.SetInt(42)
				return t, o.FieldByName("Field").Int() == int64(42)
			},
		},
		"map": {
			value: func() reflect.Value { v := make(map[string]int); return reflect.ValueOf(&v).Elem() }(),
			path:  []string{"key"},
			expectValue: func(t *testing.T, applyWO func() error, o, v reflect.Value) (test.TestingT, bool) {
				v.SetInt(42)
				return t, applyWO() == nil && o.MapIndex(reflect.ValueOf("key")).Int() == int64(42)
			},
		},
		"slice": {
			value: func() reflect.Value { v := make([]int, 3); return reflect.ValueOf(&v).Elem() }(),
			path:  []string{"2"},
			expectValue: func(t *testing.T, _ func() error, o, v reflect.Value) (test.TestingT, bool) {
				v.SetInt(42)
				return t, o.Index(2).Int() == 42
			},
		},
		"nil pointer": {
			value: func() reflect.Value { var v *struct{ Field int }; return reflect.ValueOf(&v).Elem() }(),
			path:  []string{"field"},
			expectValue: func(t *testing.T, applyWO func() error, o, v reflect.Value) (test.TestingT, bool) {
				a := o.IsNil()
				b := v.Type().String() == "int"

				v.SetInt(42)
				c := o.IsNil()
				err := applyWO()

				d := !o.IsNil()
				e := o.Elem().FieldByName("Field").Addr().Pointer() == v.Addr().Pointer()
				f := o.Elem().FieldByName("Field").Int() == 42

				return t, a && b && c && d && e && f && err == nil
			},
		},
		"non-nil pointer": {
			value: func() reflect.Value { v := &struct{ Field int }{Field: 21}; return reflect.ValueOf(&v).Elem() }(),
			path:  []string{"field"},
			expectValue: func(t *testing.T, _ func() error, o, v reflect.Value) (test.TestingT, bool) {
				a := v.Type().String() == "int"
				b := o.Elem().FieldByName("Field").Int() == 21
				v.SetInt(42)
				c := o.Elem().FieldByName("Field").Int() == 42

				return t, a && b && c
			},
		},
		"unhandled type": {
			value:              func() reflect.Value { var v int; return reflect.ValueOf(&v).Elem() }(),
			path:               []string{"invalid"},
			expectErrorMessage: "unhandled type int to handle remaining path",
		},
	} {
		t.Run(name, func(t *testing.T) {
			pw := new(pathWalker)

			v, err := pw.walk(tc.value, tc.path)
			if tc.expectErrorMessage != "" {
				test.Require(t, err != nil && strings.Contains(err.Error(), tc.expectErrorMessage), err)
			} else {
				test.Require(t, err == nil, err)
				test.Assert(tc.expectValue(t, pw.applyWriteOperations, tc.value, v))
			}
		})
	}
}

func Test_pathWalker_walkInStruct(t *testing.T) {
	for name, tc := range map[string]struct {
		value              reflect.Value
		path               []string
		expectValue        func(t *testing.T, applyWO func() error, o, v reflect.Value) (test.TestingT, bool)
		expectErrorMessage string
		expectError        error
	}{
		"empty path": {
			value:              reflect.ValueOf(struct{}{}),
			path:               []string{},
			expectErrorMessage: "path cannot be empty",
		},
		"invalid value": {
			value:              reflect.Value{},
			path:               []string{"field"},
			expectErrorMessage: "provided value should be valid and of type struct",
		},
		"not struct": {
			value:              func() reflect.Value { var v int; return reflect.ValueOf(&v).Elem() }(),
			path:               []string{"field"},
			expectErrorMessage: "provided value should be valid and of type struct",
		},
		"field found": {
			value: reflect.ValueOf(&struct{ Field int }{}).Elem(),
			path:  []string{"field"},
			expectValue: func(t *testing.T, _ func() error, o, v reflect.Value) (test.TestingT, bool) {
				v.SetInt(42)
				return t, o.FieldByName("Field").Int() == int64(42)
			},
		},
		"nested": {
			value: reflect.ValueOf(&struct{ Nested struct{ Field int } }{}).Elem(),
			path:  []string{"nested", "field"},
			expectValue: func(t *testing.T, _ func() error, o, v reflect.Value) (test.TestingT, bool) {
				v.SetInt(42)
				return t, o.FieldByName("Nested").FieldByName("Field").Int() == int64(42)
			},
		},
		"field not found": {
			value:              reflect.ValueOf(struct{}{}),
			path:               []string{"notfound"},
			expectErrorMessage: "unable to find field",
			expectError:        ErrNotFound,
		},
	} {
		t.Run(name, func(t *testing.T) {
			pw := new(pathWalker)

			v, err := pw.walkInStruct(tc.value, tc.path)
			if tc.expectErrorMessage != "" {
				test.Require(t, err != nil && strings.Contains(err.Error(), tc.expectErrorMessage), err)

				if tc.expectError != nil {
					test.Assert(t, errors.Is(err, tc.expectError), err)
				}
			} else {
				test.Require(t, err == nil, err)
				test.Assert(tc.expectValue(t, pw.applyWriteOperations, tc.value, v))
			}
		})
	}
}

func Test_pathWalker_getStructFieldByName(t *testing.T) {
	for name, tc := range map[string]struct {
		value     reflect.Value
		fieldName string

		expectValue        func(t *testing.T, applyWO func() error, o, v reflect.Value) (test.TestingT, bool)
		expectErrorMessage string
		expectError        error
	}{
		"invalid value": {
			value:              reflect.Value{},
			fieldName:          "field",
			expectErrorMessage: "provided value should be valid and of type struct",
		},
		"not struct": {
			value:              func() reflect.Value { var v int; return reflect.ValueOf(&v).Elem() }(),
			fieldName:          "field",
			expectErrorMessage: "provided value should be valid and of type struct",
		},
		"field exists": {
			value:     reflect.ValueOf(&struct{ Field int }{}).Elem(),
			fieldName: "field",
			expectValue: func(t *testing.T, _ func() error, o, v reflect.Value) (test.TestingT, bool) {
				v.SetInt(42)
				return t, o.FieldByName("Field").Int() == int64(42)
			},
		},
		"field exists below nil exported embedded struct ptr": {
			value: func() reflect.Value {
				type EmbeddedType struct{ Embedded int }
				s := struct{ *EmbeddedType }{}
				return reflect.ValueOf(&s).Elem()
			}(),
			fieldName: "embedded",
			expectValue: func(t *testing.T, applyWO func() error, o, v reflect.Value) (test.TestingT, bool) {
				v.SetInt(42)
				err := applyWO()
				return t, err == nil && o.FieldByName("Embedded").Int() == 42
			},
		},
		"field exists below nil non exported embedded struct ptr without unsafe": {
			value: func() reflect.Value {
				type embeddedType struct{ Embedded int }
				s := struct{ *embeddedType }{}
				return reflect.ValueOf(&s).Elem()
			}(),
			fieldName: "embedded",
			expectValue: func(t *testing.T, applyWO func() error, _, v reflect.Value) (test.TestingT, bool) {
				v.SetInt(42)
				err := applyWO()
				return t, err != nil && strings.Contains(err.Error(), "dst can't be set without using unsafe, and unsafe is disabled")
			},
		},
		"field case insensitive": {
			value:     reflect.ValueOf(&struct{ Field int }{}).Elem(),
			fieldName: "FIELD",
			expectValue: func(t *testing.T, _ func() error, o, v reflect.Value) (test.TestingT, bool) {
				v.SetInt(42)
				return t, o.FieldByName("Field").Int() == 42
			},
		},
		"field not found": {
			value:              reflect.ValueOf(struct{}{}),
			fieldName:          "notfound",
			expectErrorMessage: "struct field not found",
		},
	} {
		t.Run(name, func(t *testing.T) {
			pw := new(pathWalker)

			v, err := pw.getStructFieldByName(tc.value, tc.fieldName)
			if tc.expectErrorMessage != "" {
				test.Require(t, err != nil && strings.Contains(err.Error(), tc.expectErrorMessage), err)
			} else {
				test.Require(t, err == nil, err)
				test.Assert(tc.expectValue(t, pw.applyWriteOperations, tc.value, v))
			}
		})
	}
}

func Test_pathWalker_walkInMap(t *testing.T) {
	for name, tc := range map[string]struct {
		value              reflect.Value
		path               []string
		expectValue        func(t *testing.T, applyWO func() error, o, v reflect.Value) (test.TestingT, bool)
		expectErrorMessage string
	}{
		"empty path": {
			value:              func() reflect.Value { v := make(map[string]int); return reflect.ValueOf(&v).Elem() }(),
			path:               []string{},
			expectErrorMessage: "path cannot be empty",
		},
		"invalid value": {
			value:              reflect.Value{},
			path:               []string{"key"},
			expectErrorMessage: "provided value should be valid and of type map",
		},
		"not map": {
			value:              func() reflect.Value { var v int; return reflect.ValueOf(&v).Elem() }(),
			path:               []string{"key"},
			expectErrorMessage: "provided value should be valid and of type map",
		},
		"non-string key": {
			value:              func() reflect.Value { v := make(map[int]string); return reflect.ValueOf(&v).Elem() }(),
			path:               []string{"key"},
			expectErrorMessage: "only string keys are supported for maps",
		},
		"nil map": {
			value: func() reflect.Value { var v map[string]int; return reflect.ValueOf(&v).Elem() }(),
			path:  []string{"key"},
			expectValue: func(t *testing.T, applyWO func() error, o, v reflect.Value) (test.TestingT, bool) {
				a := o.IsNil()
				v.SetInt(42)
				b := o.IsNil()
				err := applyWO()
				c := err == nil && !o.IsNil() && o.MapIndex(reflect.ValueOf("key")).Int() == 42
				return t, a && b && c
			},
		},
		"existing key": {
			value: func() reflect.Value { v := map[string]int{"key": 21}; return reflect.ValueOf(&v).Elem() }(),
			path:  []string{"key"},
			expectValue: func(t *testing.T, applyWO func() error, o, v reflect.Value) (test.TestingT, bool) {
				a := v.Int() == 21
				v.SetInt(42)
				err := applyWO()
				b := err == nil && o.MapIndex(reflect.ValueOf("key")).Int() == 42
				return t, a && b
			},
		},
		"new key": {
			value: func() reflect.Value { v := map[string]int{"key": 21}; return reflect.ValueOf(&v).Elem() }(),
			path:  []string{"newkey"},
			expectValue: func(t *testing.T, applyWO func() error, o, v reflect.Value) (test.TestingT, bool) {
				a := o.MapIndex(reflect.ValueOf("key")).Int() == 21
				v.SetInt(42)
				err := applyWO()
				b := err == nil && o.MapIndex(reflect.ValueOf("newkey")).Int() == 42
				return t, a && b
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			pw := new(pathWalker)

			v, err := pw.walkInMap(tc.value, tc.path)
			if tc.expectErrorMessage != "" {
				test.Require(t, err != nil && strings.Contains(err.Error(), tc.expectErrorMessage), err)
			} else {
				test.Require(t, err == nil, err)
				test.Assert(tc.expectValue(t, pw.applyWriteOperations, tc.value, v))
			}
		})
	}
}

func Test_pathWalker_walkInSlice(t *testing.T) {
	for name, tc := range map[string]struct {
		value              reflect.Value
		path               []string
		expectValue        func(t *testing.T, applyWO func() error, o, v reflect.Value) (test.TestingT, bool)
		expectErrorMessage string
	}{
		"empty path": {
			value:              func() reflect.Value { v := make([]int, 5); return reflect.ValueOf(&v).Elem() }(),
			path:               []string{},
			expectErrorMessage: "path cannot be empty",
		},
		"invalid value": {
			value:              reflect.Value{},
			path:               []string{"0"},
			expectErrorMessage: "provided value should be valid and of type slice",
		},
		"not slice": {
			value:              func() reflect.Value { var v int; return reflect.ValueOf(&v).Elem() }(),
			path:               []string{"0"},
			expectErrorMessage: "provided value should be valid and of type slice",
		},
		"invalid index": {
			value:              func() reflect.Value { v := make([]int, 5); return reflect.ValueOf(&v).Elem() }(),
			path:               []string{"invalid"},
			expectErrorMessage: "unable to parse slice index invalid",
		},
		"valid index": {
			value: func() reflect.Value { v := make([]int, 5); return reflect.ValueOf(&v).Elem() }(),
			path:  []string{"2"},
			expectValue: func(t *testing.T, _ func() error, o, v reflect.Value) (test.TestingT, bool) {
				v.SetInt(42)
				return t, o.Index(2).Int() == 42
			},
		},
		"index out of bounds": {
			value: func() reflect.Value { v := make([]int, 5); return reflect.ValueOf(&v).Elem() }(),
			path:  []string{"10"},
			expectValue: func(t *testing.T, applyWO func() error, o, v reflect.Value) (test.TestingT, bool) {
				a := o.Len() == 5
				v.SetInt(42)
				err := applyWO()
				b := err == nil && o.Len() == 11
				c := b && o.Index(10).Int() == 42
				return t, a && b && c
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			pw := new(pathWalker)

			v, err := pw.walkInSlice(tc.value, tc.path)
			if tc.expectErrorMessage != "" {
				test.Require(t, err != nil && strings.Contains(err.Error(), tc.expectErrorMessage), err)
			} else {
				test.Require(t, err == nil, err)
				test.Assert(tc.expectValue(t, pw.applyWriteOperations, tc.value, v))
			}
		})
	}
}

func Test_pathWalker_ensurePointerInitialized(t *testing.T) {
	for name, tc := range map[string]struct {
		setup       func() reflect.Value
		expectValue func(t *testing.T, applyWO func() error, o, v reflect.Value) (test.TestingT, bool)
	}{
		"non-nil pointer": {
			setup: func() reflect.Value {
				v := &struct{ Field int }{Field: 42}
				return reflect.ValueOf(&v).Elem()
			},
			expectValue: func(t *testing.T, _ func() error, o, v reflect.Value) (test.TestingT, bool) {
				a := v.IsValid() && !v.IsNil()
				b := o.Addr().Pointer() == v.Addr().Pointer()
				return t, a && b
			},
		},
		"nil pointer": {
			setup: func() reflect.Value {
				var v *struct{ Field int }
				return reflect.ValueOf(&v).Elem()
			},
			expectValue: func(t *testing.T, applyWO func() error, o, v reflect.Value) (test.TestingT, bool) {
				a := o.IsNil()
				b := v.IsValid() && !v.IsNil()
				err := applyWO()
				c := !o.IsNil()
				d := o.Pointer() == v.Pointer()
				return t, a && b && err == nil && c && d
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			pw := &pathWalker{}
			original := tc.setup()
			result := pw.ensurePointerInitialized(original)
			test.Assert(tc.expectValue(t, pw.applyWriteOperations, original, result))
		})
	}
}

func Test_pathWalker_reflectSetValue(t *testing.T) {
	for name, tc := range map[string]struct {
		setup              func() (reflect.Value, reflect.Value)
		unsafe             bool
		expectValue        func(t *testing.T, dst, src reflect.Value) (test.TestingT, bool)
		expectErrorMessage string
	}{
		"invalid dst": {
			setup: func() (reflect.Value, reflect.Value) {
				return reflect.Value{}, reflect.ValueOf(42)
			},
			expectErrorMessage: "invalid value: dst.IsValid=false",
		},
		"invalid src": {
			setup: func() (reflect.Value, reflect.Value) {
				var v int
				return reflect.ValueOf(&v).Elem(), reflect.Value{}
			},
			expectErrorMessage: "invalid value: dst.IsValid=true src.IsValid=false",
		},
		"type mismatch": {
			setup: func() (reflect.Value, reflect.Value) {
				var dst int
				src := "string"
				return reflect.ValueOf(&dst).Elem(), reflect.ValueOf(src)
			},
			expectErrorMessage: "type mismatch",
		},
		"can set": {
			setup: func() (reflect.Value, reflect.Value) {
				var dst int
				src := 42
				return reflect.ValueOf(&dst).Elem(), reflect.ValueOf(src)
			},
			expectValue: func(t *testing.T, dst, _ reflect.Value) (test.TestingT, bool) {
				return t, dst.Int() == 42
			},
		},
		"cannot set without unsafe": {
			setup: func() (reflect.Value, reflect.Value) {
				s := struct{ field int }{}
				src := 42
				field := reflect.ValueOf(&s).Elem().FieldByName("field")
				return field, reflect.ValueOf(src)
			},
			expectErrorMessage: "dst can't be set without using unsafe, and unsafe is disabled",
		},
		"can set with unsafe": {
			setup: func() (reflect.Value, reflect.Value) {
				s := struct{ field int }{}
				src := 42
				return reflect.ValueOf(&s).Elem().FieldByName("field"), reflect.ValueOf(src)
			},
			unsafe: true,
			expectValue: func(t *testing.T, dst, _ reflect.Value) (test.TestingT, bool) {
				return t, dst.Int() == 42
			},
		},
		"not addressable": {
			setup: func() (reflect.Value, reflect.Value) {
				s := struct{ Field int }{}
				src := 42
				field := reflect.ValueOf(s).FieldByName("Field")
				return field, reflect.ValueOf(src)
			},
			expectErrorMessage: "dst is not addressable",
		},
	} {
		t.Run(name, func(t *testing.T) {
			pw := &pathWalker{allowUnsafe: tc.unsafe}
			dst, src := tc.setup()

			err := pw.reflectSetValue(dst, src)
			if tc.expectErrorMessage != "" {
				test.Require(t, err != nil && strings.Contains(err.Error(), tc.expectErrorMessage), err)
			} else {
				test.Require(t, err == nil, err)
				test.Assert(tc.expectValue(t, dst, src))
			}
		})
	}
}
