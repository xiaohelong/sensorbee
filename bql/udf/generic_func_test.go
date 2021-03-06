package udf

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/sensorbee/sensorbee.v0/core"
	"gopkg.in/sensorbee/sensorbee.v0/data"
)

func TestGenericFunc(t *testing.T) {
	ctx := &core.Context{}
	Convey("Given a generic UDF generator", t, func() {
		normalCases := []struct {
			title string
			f     interface{}
		}{
			{
				title: "When passing a function receiving a context and two args and returning an error",
				f: func(ctx *core.Context, i int, f float32) (float32, error) {
					return float32(i) + f, nil
				},
			},
			{
				title: "When passing a function receiving two args and returning an error",
				f: func(i int, f float32) (float32, error) {
					return float32(i) + f, nil
				},
			},
			{
				title: "When passing a function receiving a context and two args and not returning an error",
				f: func(ctx *core.Context, i int, f float32) float32 {
					return float32(i) + f
				},
			},
			{
				title: "When passing a function receiving two args and not returning an error",
				f: func(i int, f float32) float32 {
					return float32(i) + f
				},
			},
		}

		for _, c := range normalCases {
			c := c
			Convey(c.title, func() {
				f, err := ConvertGeneric(c.f)
				So(err, ShouldBeNil)

				Convey("Then the udf should return a correct value", func() {
					v, err := f.Call(ctx, data.Int(1), data.Float(1.5))
					So(err, ShouldBeNil)
					res, err := data.ToFloat(v)
					So(err, ShouldBeNil)
					So(res, ShouldEqual, 2.5)
				})

				Convey("Then the udf's arity should be 2", func() {
					So(f.Accept(2), ShouldBeTrue)
					So(f.Accept(1), ShouldBeFalse)
					So(f.Accept(3), ShouldBeFalse)
				})
			})
		}

		aggCases := []struct {
			title string
			f     interface{}
		}{
			{
				title: "When passing an aggregation function receiving a context and two args and returning an error",
				f: func(ctx *core.Context, i []int, f float32) (int, error) {
					return len(i) + int(f), nil
				},
			},
			{
				title: "When passing an aggregation function receiving two args and returning an error",
				f: func(i []int, f float32) (int, error) {
					return len(i) + int(f), nil
				},
			},
			{
				title: "When passing an aggregation function receiving a context and two args and not returning an error",
				f: func(ctx *core.Context, i []int, f float32) int {
					return len(i) + int(f)
				},
			},
			{
				title: "When passing an aggregation function receiving two args and not returning an error",
				f: func(i []int, f float32) int {
					return len(i) + int(f)
				},
			},
		}

		for _, c := range aggCases {
			c := c
			Convey(c.title, func() {
				aggParams := []bool{true, false}
				f, err := ConvertGenericAggregate(c.f, aggParams)
				So(err, ShouldBeNil)

				Convey("Then the udf should return a correct value", func() {
					v, err := f.Call(ctx, data.Array([]data.Value{data.Int(1), data.Int(2)}), data.Float(1))
					So(err, ShouldBeNil)
					res, err := data.ToInt(v)
					So(err, ShouldBeNil)
					So(res, ShouldEqual, 3)
				})

				Convey("Then the udf's arity should be 2", func() {
					So(f.Accept(2), ShouldBeTrue)
					So(f.Accept(1), ShouldBeFalse)
					So(f.Accept(3), ShouldBeFalse)
				})

				Convey("Then the udf's IsAggregationParameter should return correct values", func() {
					So(f.IsAggregationParameter(0), ShouldBeTrue)
					So(f.IsAggregationParameter(1), ShouldBeFalse)
				})

				Convey("Then the udf's IsAggregationParameter with out of bounds index should be false", func() {
					So(f.IsAggregationParameter(2), ShouldBeFalse)
				})
			})
		}

		variadicCases := []struct {
			title string
			f     interface{}
		}{
			{
				title: "When passing a variadic function with a context and returning an error",
				f: func(ctx *core.Context, ss ...string) (string, error) {
					return strings.Join(ss, ""), nil
				},
			},
			{
				title: "When passing a variadic function without a context and returning an error",
				f: func(ss ...string) (string, error) {
					return strings.Join(ss, ""), nil
				},
			},
			{
				title: "When passing a variadic function with a context and not returing an error",
				f: func(ctx *core.Context, ss ...string) string {
					return strings.Join(ss, "")
				},
			},
			{
				title: "When passing a variadic function without a context and not returing an error",
				f: func(ss ...string) string {
					return strings.Join(ss, "")
				},
			},
		}

		for _, c := range variadicCases {
			c := c
			Convey(c.title, func() {
				f, err := ConvertGeneric(c.f)
				So(err, ShouldBeNil)

				Convey("And passing no arguments", func() {
					res, err := f.Call(ctx)

					Convey("Then it should succeed", func() {
						So(err, ShouldBeNil)
						s, err := data.AsString(res)
						So(err, ShouldBeNil)
						So(s, ShouldBeBlank)
					})
				})

				Convey("And passing one arguments", func() {
					res, err := f.Call(ctx, data.String("a"))

					Convey("Then it should succeed", func() {
						So(err, ShouldBeNil)
						s, err := data.AsString(res)
						So(err, ShouldBeNil)
						So(s, ShouldEqual, "a")
					})
				})

				Convey("And passing many arguments", func() {
					res, err := f.Call(ctx, data.String("a"), data.String("b"), data.String("c"), data.String("d"), data.String("e"))

					Convey("Then it should succeed", func() {
						So(err, ShouldBeNil)
						s, err := data.AsString(res)
						So(err, ShouldBeNil)
						So(s, ShouldEqual, "abcde")
					})
				})

				Convey("And passing a convertible value", func() {
					res, err := f.Call(ctx, data.String("a"), data.Int(1), data.String("c"))

					Convey("Then it should succeed", func() {
						So(err, ShouldBeNil)
						s, err := data.AsString(res)
						So(err, ShouldBeNil)
						So(s, ShouldEqual, "a1c")
					})
				})

				Convey("Then it should accept any arity", func() {
					So(f.Accept(0), ShouldBeTrue)
					So(f.Accept(1), ShouldBeTrue)
					So(f.Accept(123456789), ShouldBeTrue)
				})
			})
		}

		aggVariadicCases := []struct {
			title string
			f     interface{}
		}{
			{
				title: "When passing a variadic aggregation function with a context and returning an error",
				f: func(ctx *core.Context, i int, ss ...string) (string, error) {
					return strings.Join(ss, ""), nil
				},
			},
			{
				title: "When passing a variadic aggregation function without a context and returning an error",
				f: func(i int, ss ...string) (string, error) {
					return strings.Join(ss, ""), nil
				},
			},
			{
				title: "When passing a variadic aggregation function with a context and not returing an error",
				f: func(ctx *core.Context, i int, ss ...string) string {
					return strings.Join(ss, "")
				},
			},
			{
				title: "When passing a variadic aggregation function without a context and not returing an error",
				f: func(i int, ss ...string) string {
					return strings.Join(ss, "")
				},
			},
		}

		for _, c := range aggVariadicCases {
			c := c
			Convey(c.title, func() {
				f, err := ConvertGenericAggregate(c.f, []bool{false, true})
				So(err, ShouldBeNil)

				Convey("And passing with two arguments", func() {
					res, err := f.Call(ctx, data.Int(1), data.String("a"))

					Convey("Then it should succeed", func() {
						So(err, ShouldBeNil)
						s, err := data.AsString(res)
						So(err, ShouldBeNil)
						So(s, ShouldEqual, "a")
					})
				})

				Convey("Then the udf's IsAggregationParameter should return false", func() {
					So(f.IsAggregationParameter(0), ShouldBeFalse)
				})

				Convey("Then the udf's IsAggregationParameter for variadic parameter should be true", func() {
					So(f.IsAggregationParameter(1), ShouldBeTrue)
					So(f.IsAggregationParameter(2), ShouldBeTrue)
					So(f.IsAggregationParameter(10000000), ShouldBeTrue)
				})
			})
		}

		variadicExtraCases := []struct {
			title string
			f     interface{}
		}{
			{
				title: "When passing a variadic function having additional args with a context and returning values include error",
				f: func(ctx *core.Context, rep int, ss ...string) (string, error) {
					return strings.Repeat(strings.Join(ss, ""), rep), nil
				},
			},
			{
				title: "When passing a variadic function having additional args without a context and returning values include error",
				f: func(rep int, ss ...string) (string, error) {
					return strings.Repeat(strings.Join(ss, ""), rep), nil
				},
			},
			{
				title: "When passing a variadic function having additional args with a context and not returing an error",
				f: func(ctx *core.Context, rep int, ss ...string) string {
					return strings.Repeat(strings.Join(ss, ""), rep)
				},
			},
			{
				title: "When passing a variadic function having additional args without a context and not returing an error",
				f: func(rep int, ss ...string) string {
					return strings.Repeat(strings.Join(ss, ""), rep)
				},
			},
		}

		for _, c := range variadicExtraCases {
			c := c
			Convey(c.title, func() {
				f, err := ConvertGeneric(c.f)
				So(err, ShouldBeNil)

				Convey("And passing no arguments", func() {
					_, err := f.Call(ctx)

					Convey("Then it should fail", func() {
						So(err, ShouldNotBeNil)
					})
				})

				Convey("And only passing required arguments", func() {
					res, err := f.Call(ctx, data.Int(1))

					Convey("Then it should succeed", func() {
						So(err, ShouldBeNil)
						s, err := data.AsString(res)
						So(err, ShouldBeNil)
						So(s, ShouldEqual, "")
					})
				})

				Convey("And passing one argument", func() {
					res, err := f.Call(ctx, data.Int(5), data.String("a"))

					Convey("Then it should succeed", func() {
						So(err, ShouldBeNil)
						s, err := data.AsString(res)
						So(err, ShouldBeNil)
						So(s, ShouldEqual, "aaaaa")
					})
				})

				Convey("And passing many arguments", func() {
					res, err := f.Call(ctx, data.Int(2), data.String("a"), data.String("b"), data.String("c"), data.String("d"), data.String("e"))

					Convey("Then it should succeed", func() {
						So(err, ShouldBeNil)
						s, err := data.AsString(res)
						So(err, ShouldBeNil)
						So(s, ShouldEqual, "abcdeabcde")
					})
				})

				Convey("And passing a convertible value", func() {
					res, err := f.Call(ctx, data.Int(3), data.String("a"), data.Int(1), data.String("c"))

					Convey("Then it should succeed", func() {
						So(err, ShouldBeNil)
						s, err := data.AsString(res)
						So(err, ShouldBeNil)
						So(s, ShouldEqual, "a1ca1ca1c")
					})
				})

				Convey("Then it should accept any arity greater than 0", func() {
					So(f.Accept(0), ShouldBeFalse)
					So(f.Accept(1), ShouldBeTrue)
					So(f.Accept(123456789), ShouldBeTrue)
				})
			})
		}

		Convey("When creating a function returning an error", func() {
			f, err := ConvertGeneric(func() (int, error) {
				return 0, fmt.Errorf("test failure")
			})
			So(err, ShouldBeNil)

			Convey("Then calling it should fail", func() {
				_, err := f.Call(ctx)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "test failure")
			})
		})

		Convey("When creating a valid UDF with MustConvertGeneric", func() {
			Convey("Then it shouldn't panic", func() {
				So(func() {
					MustConvertGeneric(func() int { return 0 })
				}, ShouldNotPanic)
			})
		})

		Convey("When creating a invalid UDF with MustConvertGeneric", func() {
			Convey("Then it should panic", func() {
				So(func() {
					MustConvertGeneric(func() {})
				}, ShouldPanic)
			})
		})

		Convey("When creating a valid UDF with MustConvertGenericAggregate", func() {
			Convey("Then it shouldn't panic", func() {
				So(func() {
					MustConvertGenericAggregate(func([]int) int { return 0 }, []bool{true})
				}, ShouldNotPanic)
			})
		})

		Convey("When creating a invalid UDF with MustConvertGenericAggregate", func() {
			Convey("Then it should panic", func() {
				So(func() {
					MustConvertGenericAggregate(func() {}, []bool{})
				}, ShouldPanic)
			})
		})
	})
}

func TestGenericFuncReturnValue(t *testing.T) {
	ctx := &core.Context{}

	Convey("Given a generic UDF generator", t, func() {
		Convey("When a function has a slice return value", func() {
			f, err := ConvertGeneric(func(fs ...float64) []float64 {
				return fs
			})
			So(err, ShouldBeNil)

			Convey("Then it should return a slice", func() {
				v, err := f.Call(ctx, data.Float(1), data.Float(2))
				So(err, ShouldBeNil)
				So(v, ShouldResemble, data.Array{data.Float(1), data.Float(2)})
			})
		})

		// Other cases are tested in data.NewValue
	})
}

func TestGenericFuncInvalidCases(t *testing.T) {
	ctx := &core.Context{}
	toArgs := func(vs ...interface{}) data.Array {
		a, err := data.NewArray(vs)
		if err != nil {
			t.Fatal(err) // don't want to increase So's count unnecessarily
		}
		return a
	}

	Convey("Given a generic UDF generator", t, func() {
		genCases := []struct {
			title     string
			f         interface{}
			aggParams []bool
		}{
			{"with no return value", func() {}, nil},
			{"with an unsupported type", func(error) int { return 0 }, nil},
			{"with non-function type", 10, nil},
			{"with non-error second return type", func() (int, int) { return 0, 0 }, nil},
			{"with an unsupported type and non-error second return type", func(error) (int, int) { return 0, 0 }, nil},
			{"with an unsupported return type", func() *core.Context { return nil }, nil},
			{"with an unsupported interface return type", func() error { return nil }, nil},
			{"with an invalid aggParams 1", func(int) int { return 0 }, []bool{}},
			{"with an invalid aggParams 2", func(int) int { return 0 }, []bool{false, false}},
			{"with an invalid aggParams 3", func(*core.Context, int) int { return 0 }, []bool{false, false}},
			{"with an aggregate function having no argument", func() int { return 0 }, []bool{}},
			{"with an aggregate function which has non-slice aggregation parameter", func(int) int { return 0 }, []bool{true}},
			{"with an aggregate function which doesn't have an aggregation parameter", func(int) int { return 0 }, []bool{false}},
			{"with an aggregate function which has non-slice aggregation parameter with context", func(*core.Context, int) int { return 0 }, []bool{true}},
			{"with an aggregate function with wrong number of aggParams", func([]int) int { return 0 }, []bool{true, false}},
		}

		for _, c := range genCases {
			c := c
			Convey("When passing a function "+c.title, func() {
				var err error
				if c.aggParams == nil {
					_, err = ConvertGeneric(c.f)
				} else {
					_, err = ConvertGenericAggregate(c.f, c.aggParams)
				}

				Convey("Then it should fail", func() {
					So(err, ShouldNotBeNil)
				})
			})
		}

		callCases := []struct {
			title string
			f     interface{}
			args  data.Array
		}{
			{
				title: "When calling a function with too few arguments",
				f:     func(int, int) int { return 0 },
				args:  toArgs(1),
			},
			{
				title: "When calling a function too many arguments",
				f:     func(int, int) int { return 0 },
				args:  toArgs(1, 2, 3, 4),
			},
			{
				title: "When calling a function with inconvertible arguments",
				f:     func(data.Map) int { return 0 },
				args:  toArgs("hoge"),
			},
			{
				title: "When calling a variadic function with inconvertible regular arguments",
				f:     func(data.Array, ...data.Map) int { return 0 },
				args:  toArgs("owata", data.Map{}, data.Map{}),
			},
			{
				title: "When calling a variadic function with inconvertible variadic arguments",
				f:     func(data.Array, ...data.Map) int { return 0 },
				args:  toArgs(data.Array{}, data.Map{}, "damepo", data.Map{}),
			},
		}

		for _, c := range callCases {
			c := c
			Convey(c.title, func() {
				f, err := ConvertGeneric(c.f)
				So(err, ShouldBeNil)

				Convey("Then it should fail", func() {
					_, err := f.Call(ctx, c.args...)
					So(err, ShouldNotBeNil)
				})
			})
		}
	})
}

func TestGenericFuncInconvertibleType(t *testing.T) {
	ctx := &core.Context{} // not used in this test

	udfs := []UDF{
		MustConvertGeneric(func(i int8) int8 {
			return i * 2
		}),
		MustConvertGeneric(func(i int16) int16 {
			return i * 2
		}),
		MustConvertGeneric(func(i int32) int32 {
			return i * 2
		}),
		MustConvertGeneric(func(i int64) int64 {
			return i * 2
		}),
		MustConvertGeneric(func(i uint8) uint8 {
			return i * 2
		}),
		MustConvertGeneric(func(i uint16) uint16 {
			return i * 2
		}),
		MustConvertGeneric(func(i uint32) uint32 {
			return i * 2
		}),
		MustConvertGeneric(func(i uint64) uint64 {
			return i * 2
		}),
		MustConvertGeneric(func(i float32) float32 {
			return i * 2
		}),
		MustConvertGeneric(func(i float64) float64 {
			return i * 2
		}),
		MustConvertGeneric(func(b []byte) []byte {
			return b
		}),
		MustConvertGeneric(func(t time.Time) time.Time {
			return t
		}),
	}
	funcTypes := []string{"int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64", "float32", "float64", "blob", "timestamp"}
	if len(udfs) != 12 {
		t.Fatal("len of udfs isn't 12 but", len(udfs))
	}

	type InputType struct {
		typeName string
		value    data.Value
	}

	inconvertible := [][]InputType{
		{ // int8
			{"int", data.Int(int(math.MaxInt8) + 1)},
			{"negative int", data.Int(int(math.MinInt8) - 1)},
			{"float", data.Float(float64(math.MaxInt8) + 1.0)},
			{"negative float", data.Float(float64(math.MinInt8) - 1.0)},
			{"time", data.Timestamp(time.Date(2015, time.May, 1, 14, 27, 0, 0, time.UTC))},
		},
		{ // int16
			{"int", data.Int(int32(math.MaxInt16) + 1)},
			{"negative int", data.Int(int32(math.MinInt16) - 1)},
			{"float", data.Float(float64(math.MaxInt16) + 1.0)},
			{"negative float", data.Float(float64(math.MinInt16) - 1.0)},
			{"time", data.Timestamp(time.Date(2015, time.May, 1, 14, 27, 0, 0, time.UTC))},
		},
		{ // int32
			{"int", data.Int(int64(math.MaxInt32) + 1)},
			{"negative int", data.Int(int64(math.MinInt32) - 1)},
			{"float", data.Float(float64(math.MaxInt32) + 1.0)},
			{"negative float", data.Float(float64(math.MinInt32) - 1.0)},
			// unix time of MaxInt32 = 2038-01-19 3:14:07
			{"time", data.Timestamp(time.Date(2038, time.January, 19, 3, 14, 8, 0, time.UTC))},
		},
		{ // int64
			{"float", data.Float(float64(math.MaxUint64))},
			{"negative float", data.Float(float64(math.MinInt64) * 2)},
		},
		{ // uint8
			{"int", data.Int(int(math.MaxUint8) + 1)},
			{"negative int", data.Int(-1)},
			{"float", data.Float(float64(math.MaxUint8) + 1.0)},
			{"negative float", data.Float(-1.0)},
			{"time", data.Timestamp(time.Date(2015, time.May, 1, 14, 27, 0, 0, time.UTC))},
		},
		{ // uint16
			{"int", data.Int(int32(math.MaxUint16) + 1)},
			{"negative int", data.Int(-1)},
			{"float", data.Float(float64(math.MaxUint16) + 1.0)},
			{"negative float", data.Float(-1.0)},
			{"time", data.Timestamp(time.Date(2015, time.May, 1, 14, 27, 0, 0, time.UTC))},
		},
		{ // uint32
			{"int", data.Int(int64(math.MaxUint32) + 1)},
			{"negative int", data.Int(-1)},
			{"float", data.Float(float64(math.MaxUint32) + 1.0)},
			{"negative float", data.Float(-1.0)},
			// unix time of MaxUint32 = 2106-02-07 06:28:15
			{"time", data.Timestamp(time.Date(2106, time.February, 7, 6, 28, 16, 0, time.UTC))},
		},
		{ // uint64
			{"negative int", data.Int(-1)},
			{"float", data.Float(float64(math.MaxUint64))},
			{"negative float", data.Float(-1.0)},
		},
		{ // float32 covered by common values
		},
		{ // float64 covered by common values
		},
		{ // blob
			{"int", data.Int(1)},
			{"float", data.Float(1.0)},
			{"time", data.Timestamp(time.Date(2015, time.May, 1, 14, 27, 0, 0, time.UTC))},
			{"map", data.Map{"key": data.Int(10)}},
		},
		{ // timestamp
			{"string", data.String("str")},
			{"blob", data.Blob([]byte("blob"))},
			{"array", data.Array([]data.Value{data.Int(10)})},
			{"map", data.Map{"key": data.Int(10)}},
		},
	}

	// common incovertible values for integer and float
	numInconvertibleValues := []InputType{
		{"string", data.String("str")},
		{"blob", data.Blob([]byte("blob"))},
		{"array", data.Array([]data.Value{data.Int(10)})},
		{"map", data.Map{"key": data.Int(10)}},
	}
	// append common values for int8, int16, int32, int64/ uint8, uint16, uint32, uint64/ float32, float64 (4 + 4 + 2 patterns)
	for i := 0; i < 10; i++ {
		inconvertible[i] = append(inconvertible[i], numInconvertibleValues...)
	}

	Convey("Given UDFs and inconvertible values", t, func() {
		for i, f := range udfs {
			f := f
			for _, inc := range inconvertible[i] {
				t := inc.typeName
				v := inc.value
				Convey(fmt.Sprintf("When passing inconvertible value of %v for %v", t, funcTypes[i]), func() {
					_, err := f.Call(ctx, v)

					Convey("Then it should fail", func() {
						So(err, ShouldNotBeNil)
					})
				})
			}
		}
	})
}

func TestGenericIntAndFloatFunc(t *testing.T) {
	ctx := &core.Context{} // not used in this test

	udfs := []UDF{
		MustConvertGeneric(func(i int8) int8 {
			return i * 2
		}),
		MustConvertGeneric(func(i int16) int16 {
			return i * 2
		}),
		MustConvertGeneric(func(i int32) int32 {
			return i * 2
		}),
		MustConvertGeneric(func(i int64) int64 {
			return i * 2
		}),
		MustConvertGeneric(func(i uint8) uint8 {
			return i * 2
		}),
		MustConvertGeneric(func(i uint16) uint16 {
			return i * 2
		}),
		MustConvertGeneric(func(i uint32) uint32 {
			return i * 2
		}),
		MustConvertGeneric(func(i uint64) uint64 {
			return i * 2
		}),
		MustConvertGeneric(func(i float32) float32 {
			return i * 2
		}),
		MustConvertGeneric(func(i float64) float64 {
			return i * 2
		}),
	}

	funcTypes := []string{"int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64", "float32", "float64"}
	Convey("Given a function receiving integer", t, func() {
		for i, f := range udfs {
			f := f
			i := i
			Convey(fmt.Sprintf("When passing a valid value for %v", funcTypes[i]), func() {
				v, err := f.Call(ctx, data.String("1"))
				So(err, ShouldBeNil)

				Convey("Then it should be doubled", func() {
					i, err := data.ToInt(v)
					So(err, ShouldBeNil)
					So(i, ShouldEqual, 2)
				})
			})
		}
	})
}

func TestGenericBoolFunc(t *testing.T) {
	ctx := &core.Context{} // not used in this test

	Convey("Given a function receiving bool", t, func() {
		f, err := ConvertGeneric(func(b bool) bool {
			return !b
		})
		So(err, ShouldBeNil)

		Convey("When passing a valid value", func() {
			v, err := f.Call(ctx, data.Int(1))
			So(err, ShouldBeNil)

			Convey("Then it should be false", func() {
				b, err := data.ToBool(v)
				So(err, ShouldBeNil)
				So(b, ShouldBeFalse)
			})
		})
	})
}

func TestGenericBlobFunc(t *testing.T) {
	ctx := &core.Context{} // not used in this test

	Convey("Given a function receiving blob", t, func() {
		f, err := ConvertGeneric(func(b []byte) []byte {
			return bytes.ToLower(b)
		})
		So(err, ShouldBeNil)

		Convey("When passing a valid value", func() {
			v, err := f.Call(ctx, data.String(`QUJD`)) // = "ABC"
			So(err, ShouldBeNil)

			Convey("Then it should be lowered bytes", func() {
				b, err := data.ToBlob(v)
				So(err, ShouldBeNil)
				So(b, ShouldResemble, []byte("abc"))
			})
		})
	})
}

func TestGenericTimeFunc(t *testing.T) {
	ctx := &core.Context{} // not used in this test

	Convey("Given a function receiving time", t, func() {
		f, err := ConvertGeneric(func(t time.Time) time.Time {
			return t
		})
		So(err, ShouldBeNil)

		Convey("When passing a valid value", func() {
			v, err := f.Call(ctx, data.Int(0))
			So(err, ShouldBeNil)

			Convey("Then it should be time", func() {
				t, err := data.ToTimestamp(v)
				So(err, ShouldBeNil)
				So(t, ShouldResemble, time.Unix(0, 0))
			})
		})
	})
}

func TestGenericArrayFunc(t *testing.T) {
	ctx := core.NewContext(nil)

	Convey("Given a function receiving an array", t, func() {
		f, err := ConvertGeneric(func(a data.Array) data.Array {
			return a
		})
		So(err, ShouldBeNil)

		Convey("When passing an empty array", func() {
			v, err := f.Call(ctx, data.Array{})
			So(err, ShouldBeNil)
			So(reflect.ValueOf(v).IsNil(), ShouldBeFalse) // To detect an old bug

			Convey("Then, it should return an empty array", func() {
				So(v.Type(), ShouldNotEqual, data.TypeNull)
				a, err := data.AsArray(v)
				So(err, ShouldBeNil)
				So(a, ShouldBeEmpty)
			})
		})

		Convey("When passing a non-empty array", func() {
			v, err := f.Call(ctx, data.Array{data.Int(1)})
			So(err, ShouldBeNil)

			Convey("Then, it should return nil", func() {
				So(v.Type(), ShouldNotEqual, data.TypeNull)
				a, err := data.AsArray(v)
				So(err, ShouldBeNil)
				So(len(a), ShouldEqual, 1)
				So(a[0], ShouldEqual, 1)
			})
		})
	})
}
