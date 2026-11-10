package libraryDereferer

import (
	libraryErrors "github.com/pperminov/library/errors"
	librarySync "github.com/pperminov/library/sync"
)

const defaultStackSize = 100

func GetDefaultDereferer() (func(func()) error, func()) {
	return GetDereferer(defaultStackSize, false)
}

func GetDereferer(stackSize int, FIFO bool) (func(func()) error, func()) {
	derefererStack := make(chan func(), stackSize)
	add, done, wait := librarySync.GetWait()
	lock, unlock := librarySync.GetMutex()
	closeOnce := librarySync.GetOnce()

	push := func(f func()) error {
		if f == nil {
			return libraryErrors.NewError("empty function")
		}
		select {
		case derefererStack <- f:
		default:
			return libraryErrors.NewError("dereferer stack is full, function rejected")
		}
		return nil
	}

	drain := func() {
		lock()
		defer unlock()

		closeOnce(func() {
			close(derefererStack)
		})

		if FIFO {
			for f := range derefererStack {
				f()
			}
		} else {
			for f := range derefererStack {
				add()
				go func(f func()) {
					defer done()
					f()
				}(f)
			}
			wait()
		}
	}

	return push, drain
}

func WrapErrorFromFunctionForDereferer(f func() error) func() {
	return func() {
		libraryErrors.Errorer(f())
	}
}
