package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"time"

	"go.uber.org/fx"
)

func FlipValue(v uint64, bl int) uint64 {
	mask := uint64((1 << uint(bl)) - 1)
	return mask & (mask ^ v)
}

func hello(logger *log.Logger) int {
	logger.Printf("this is hello test for fx\n")
	time.Sleep(2 * time.Second)
	logger.Printf("hello end\n")
	return 112
}

// will get error !!
func hello_again(logger *log.Logger) int {
	logger.Printf("hello again\n")
	return 222
}

type haa int

func hello_again_again(logger *log.Logger) haa {
	logger.Printf("hello again and again\n")
	return 356
}

type hello1 struct {
}

func getfromhello_1(x int, logger *log.Logger) hello1 {
	logger.Printf("hello1: %v\n", x)
	return hello1{}
}

type hello2 struct {
}

func getfromhello_2(x haa, logger *log.Logger) hello2 {
	logger.Printf("hello2: %v\n", x)
	return hello2{}
}

// Logger构造函数
func NewLogger() *log.Logger {
	logger := log.New(os.Stdout, "" /* prefix */, 0 /* flags */)
	logger.Print("Executing NewLogger.")
	return logger
}

type stopCh chan struct{}

var g_stop stopCh

func testinvoke(h1 hello1, h2 hello2, c stopCh) {
	_ = h1
	_ = h2
	// c <- struct{}{}
	fmt.Printf("stop chan: %v\n", c)
	g_stop = c
}

func as(in interface{}, as interface{}) interface{} {
	outType := reflect.TypeOf(as)

	if outType.Kind() != reflect.Ptr {
		panic("outType is not a pointer")
	}

	if reflect.TypeOf(in).Kind() != reflect.Func {
		ctype := reflect.FuncOf(nil, []reflect.Type{outType.Elem()}, false)

		return reflect.MakeFunc(ctype, func(args []reflect.Value) (results []reflect.Value) {
			out := reflect.New(outType.Elem())
			out.Elem().Set(reflect.ValueOf(in))

			fmt.Printf("as ~~ 1\n")
			return []reflect.Value{out.Elem()}
		}).Interface()
	}

	inType := reflect.TypeOf(in)

	ins := make([]reflect.Type, inType.NumIn())
	outs := make([]reflect.Type, inType.NumOut())

	for i := range ins {
		ins[i] = inType.In(i)
	}
	outs[0] = outType.Elem()
	for i := range outs[1:] {
		outs[i+1] = inType.Out(i + 1)
	}

	ctype := reflect.FuncOf(ins, outs, false)

	return reflect.MakeFunc(ctype, func(args []reflect.Value) (results []reflect.Value) {
		outs := reflect.ValueOf(in).Call(args)

		out := reflect.New(outType.Elem())
		if outs[0].Type().AssignableTo(outType.Elem()) {
			// Out: Iface = In: *Struct; Out: Iface = In: OtherIface
			out.Elem().Set(outs[0])
		} else {
			// Out: Iface = &(In: Struct)
			t := reflect.New(outs[0].Type())
			t.Elem().Set(outs[0])
			out.Elem().Set(t)
		}
		outs[0] = out.Elem()

		fmt.Printf("as ~~ 2\n")
		return outs
	}).Interface()
}

type id struct {
	val int
	sc  stopCh
}

func main() {
	val := uint32(63)
	a := ^val
	b := 130 & a
	fmt.Printf("%#08x %#08x %#08x\n", val, a, b)

	fmt.Printf("%#016x\n", FlipValue(2, 2))
	fmt.Printf("%#016x\n", FlipValue(3, 2))

	var fxval int
	shutdownChan := make(chan struct{}, 1)
	ctor := as(shutdownChan, new(stopCh))
	fmt.Printf("%v, %v\n", reflect.TypeOf(ctor).Name(), reflect.TypeOf(ctor).Kind())

	type type4 struct{ foo string }
	new4 := func() type4 { return type4{"foo"} }
	var extract_out struct {
		Val int
		// sc  stopCh
		T4 type4
	}
	// ret := ctor(shutdownChan)
	// fmt.Printf("%v, %v\n", reflect.TypeOf(ret).Name(), reflect.TypeOf(ret).Kind())

	app := fx.New(
		fx.Provide(hello),
		// fx.Provide(hello_again),
		fx.Provide(hello_again_again),
		fx.Provide(getfromhello_1),
		fx.Provide(getfromhello_2),
		fx.Provide(NewLogger),
		fx.Provide(new4),
		fx.Provide(ctor),

		fx.Populate(&fxval),
		fx.Extract(&extract_out),

		fx.Invoke(testinvoke),
		// fx.NopLogger,

	)

	if err := app.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
	defer app.Stop(context.Background())

	fmt.Printf("fxval: %v\n", fxval)
	fmt.Printf("extract_out: %v\n", extract_out)

	// for {
	//	fmt.Printf("fxval: %v\n", fxval)
	// 	time.Sleep(1 * time.Second)
	// }

	// shutdownChan <- struct{}{}

	go func() {
		time.Sleep(5 * time.Second)
		g_stop <- struct{}{}
	}()

	select {
	case <-shutdownChan:
		fmt.Printf("shutdown !!!!\n")
		// default:
		// 	fmt.Printf("????????\n")
	}
}
