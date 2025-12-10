package app

import "syscall/js"

// Await equivalent for js await statement.
func Await(promise js.Value) (res js.Value, err error) {
	ch := make(chan struct{})
	var then js.Func
	then = js.FuncOf(func(this js.Value, args []js.Value) any {
		defer then.Release()
		res = args[0]
		close(ch)
		return nil
	})
	var catch js.Func
	catch = js.FuncOf(func(this js.Value, args []js.Value) any {
		defer catch.Release()
		err = js.Error{Value: args[0]}
		close(ch)
		return nil
	})
	promise.Call("then", then).Call("catch", catch)
	<-ch
	return
}

// JS2Bytes convert from TypedArray for JS to byte slice for Go.
func JS2Bytes(dv js.Value) []byte {
	b := make([]byte, dv.Get("byteLength").Int())
	buf := js.Global().Get("Uint8Array").New(dv.Get("buffer"))
	js.CopyBytesToGo(b, buf)
	dv.Set("buffer", js.Null())
	return b
}

// Bytes2JS convert from byte slice for Go to Uint8Array for JS.
func Bytes2JS(b []byte) js.Value {
	res := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(res, b)
	return res
}

/*
type Promise js.Value

func (g Promise) Then(cb func(value js.Value)) Promise {
	var jsFunc js.Func
	jsFunc = js.FuncOf(func(this js.Value, args []js.Value) any {
		defer jsFunc.Release()
		cb(args[0])
		return nil
	})
	js.Value(g).Call("then", jsFunc)
	return g
}

func (g Promise) Catch(cb func(err error)) Promise {
	var jsFunc js.Func
	jsFunc = js.FuncOf(func(this js.Value, args []js.Value) any {
		defer jsFunc.Release()
		cb(js.Error{
			Value: args[0],
		})
		return js.Undefined()
	})
	js.Value(g).Call("catch", jsFunc)
	return g
}
*/
