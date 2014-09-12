package main

import (
	"fmt"
	"testing"
)

func BenchmarkHello(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ds := Datasource{Name: fmt.Sprintf("datasource %v", i)}
		addItemToState(ds)
	}
}

// All state is not reset

func TestSingleDsIntoState(t *testing.T) {
	tmp := len(State.Vals)
	ds1 := Datasource{Name: "some name"}
	addItemToState(ds1)
	if (len(State.Vals) - tmp) != 1 {
		t.Fatal("Not able to add datasource to internal state")
	}
}

func TestDuplicateEntriesIntoState(t *testing.T) {
	tmp := len(State.Vals)
	ds1 := Datasource{Name: "TestDuplicateEntriesIntoState same name"}
	ds2 := Datasource{Name: "TestDuplicateEntriesIntoState same name"}
	addItemToState(ds1)
	addItemToState(ds2)
	if (len(State.Vals) - tmp) > 1 {
		t.Fatal("Able to add more than one data source with the same name")
	}
}

func TestDontStoreMoreThan1k(t *testing.T) {
	ds := Datasource{Name: "tmp"}
	for i := 0; i < maxState+10; i++ {
		ds.Name = fmt.Sprintf("TestDontStoreMoreThan1k: %v", i)
		addItemToState(ds)
	}
	if (len(State.Vals) > maxState) {
		t.Fatal("Able to add more than maxState items into State.Vals")
	}
}
