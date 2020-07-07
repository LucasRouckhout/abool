package abool

import (
	"math"
	"sync"
	"sync/atomic"
	"testing"
)

func TestBool(t *testing.T) {
	t.Parallel()

	v := NewBool(true)
	if !v.IsSet() {
		t.Fatal("NewValue(true) failed")
	}

	v = NewBool(false)
	if v.IsSet() {
		t.Fatal("NewValue(false) failed")
	}

	v = New()
	if v.IsSet() {
		t.Fatal("Empty value of AtomicBool should be false")
	}

	v.Set()
	if !v.IsSet() {
		t.Fatal("AtomicBool.Set() failed")
	}

	v.UnSet()
	if v.IsSet() {
		t.Fatal("AtomicBool.UnSet() failed")
	}

	v.SetTo(true)
	if !v.IsSet() {
		t.Fatal("AtomicBool.SetTo(true) failed")
	}

	v.SetTo(false)
	if v.IsSet() {
		t.Fatal("AtomicBool.SetTo(false) failed")
	}

	if set := v.SetToIf(true, false); set || v.IsSet() {
		t.Fatal("AtomicBool.SetTo(true, false) failed")
	}

	if set := v.SetToIf(false, true); !set || !v.IsSet() {
		t.Fatal("AtomicBool.SetTo(false, true) failed")
	}

	v = New()
	if v.IsSet() {
		t.Fatal("Empty value of AtomicBool should be false")
	}

	_ = v.Toggle()
	if !v.IsSet() {
		t.Fatal("AtomicBool.Toggle() to true failed")
	}

	prev := v.Toggle()
	if v.IsSet() == prev {
		t.Fatal("AtomicBool.Toggle() to false failed")
	}
}

func TestToogleMultipleTimes(t *testing.T) {
	t.Parallel()

	v := New()
	pre := !v.IsSet()
	for i := 0; i < 100; i++ {
		v.SetTo(false)
		for j := 0; j < i; j++ {
			pre = v.Toggle()
		}

		expected := i%2 != 0
		if v.IsSet() != expected {
			t.Fatalf("AtomicBool.Toogle() doesn't work after %d calls, expected: %v, got %v", i, expected, v.IsSet())
		}

		if pre == v.IsSet() {
			t.Fatalf("AtomicBool.Toogle() returned wrong value at the %dth calls, expected: %v, got %v", i, !v.IsSet(), pre)
		}
	}
}

func TestToogleAfterOverflow(t *testing.T) {
	t.Parallel()

	var value int32 = math.MaxInt32
	v := (*AtomicBool)(&value)

	valueBeforeToggle := *(*int32)(v)

	// test first toggle after overflow
	v.Toggle()
	expected := math.MaxInt32%2 == 0
	if v.IsSet() != expected {
		t.Fatalf("AtomicBool.Toogle() doesn't work after overflow, expected: %v, got %v", expected, v.IsSet())
	}

	// make sure overflow happened
	var valueAfterToggle int32 = *(*int32)(v)
	if valueAfterToggle >= valueBeforeToggle {
		t.Fatalf("Overflow does not happen as expected, before %d, after: %d", valueBeforeToggle, valueAfterToggle)
	}

	// test second toggle after overflow
	v.Toggle()
	expected = !expected
	if v.IsSet() != expected {
		t.Fatalf("AtomicBool.Toogle() doesn't work after the second call after overflow, expected: %v, got %v", expected, v.IsSet())
	}
}

func TestRace(t *testing.T) {
	t.Parallel()

	repeat := 10000
	var wg sync.WaitGroup
	wg.Add(repeat * 4)
	v := New()

	// Writer
	go func() {
		for i := 0; i < repeat; i++ {
			v.Set()
			wg.Done()
		}
	}()

	// Reader
	go func() {
		for i := 0; i < repeat; i++ {
			v.IsSet()
			wg.Done()
		}
	}()

	// Writer
	go func() {
		for i := 0; i < repeat; i++ {
			v.UnSet()
			wg.Done()
		}
	}()

	// Reader And Writer
	go func() {
		for i := 0; i < repeat; i++ {
			v.Toggle()
			wg.Done()
		}
	}()

	wg.Wait()
}

func ExampleAtomicBool() {
	cond := New()    // default to false
	cond.Set()       // set to true
	cond.IsSet()     // returns true
	cond.UnSet()     // set to false
	cond.SetTo(true) // set to whatever you want
	cond.Toggle()    // toggles the boolean value
}

// Benchmark Read

func BenchmarkMutexRead(b *testing.B) {
	var m sync.RWMutex
	var v bool
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RLock()
		_ = v
		m.RUnlock()
	}
}

func BenchmarkAtomicValueRead(b *testing.B) {
	var v atomic.Value
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Load() != nil
	}
}

func BenchmarkAtomicBoolRead(b *testing.B) {
	v := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.IsSet()
	}
}

// Benchmark Write

func BenchmarkMutexWrite(b *testing.B) {
	var m sync.RWMutex
	var v bool
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RLock()
		v = true
		m.RUnlock()
	}
	b.StopTimer()
	_ = v
}

func BenchmarkAtomicValueWrite(b *testing.B) {
	var v atomic.Value
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Store(true)
	}
}

func BenchmarkAtomicBoolWrite(b *testing.B) {
	v := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Set()
	}
}

// Benchmark CAS

func BenchmarkMutexCAS(b *testing.B) {
	var m sync.RWMutex
	var v bool
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Lock()
		if !v {
			v = true
		}
		m.Unlock()
	}
	b.StopTimer()
}

func BenchmarkAtomicBoolCAS(b *testing.B) {
	v := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.SetToIf(false, true)
	}
}

// Benchmark toggle boolean value

func BenchmarkMutexToggle(b *testing.B) {
	var m sync.RWMutex
	var v bool
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Lock()
		v = !v
		m.Unlock()
	}
	b.StopTimer()
}

func BenchmarkAtomicBoolToggle(b *testing.B) {
	v := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Toggle()
	}
}
