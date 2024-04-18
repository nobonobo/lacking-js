package app

import "syscall/js"

type wrappedError js.Value

func (w wrappedError) Error() string {
	return js.Value(w).Call("toString").String()
}

func (w wrappedError) JSValue() js.Value {
	return js.Value(w)
}

// Await equivalent for js await statement.
func Await(promise js.Value) (res js.Value, err error) {
	ch := make(chan bool)
	then := js.FuncOf(func(this js.Value, args []js.Value) any {
		res = args[0]
		close(ch)
		return js.Undefined()
	})
	defer then.Release()
	catch := js.FuncOf(func(this js.Value, args []js.Value) any {
		err = wrappedError(args[0])
		close(ch)
		return js.Undefined()
	})
	defer catch.Release()
	promise.Call("then", then).Call("catch", catch)
	<-ch
	return
}

// JS2Bytes convert from TypedArray for JS to byte slice for Go.
func JS2Bytes(dv js.Value) []byte {
	b := make([]byte, dv.Get("byteLength").Int())
	js.CopyBytesToGo(b, js.Global().Get("Uint8Array").New(dv.Get("buffer")))
	return b
}

// Bytes2JS convert from byte slice for Go to Uint8Array for JS.
func Bytes2JS(b []byte) js.Value {
	res := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(res, b)
	return res
}
