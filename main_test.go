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
	if len(State.Vals) > maxState {
		t.Fatal("Able to add more than maxState items into State.Vals")
	}
}

func TestParsing(t *testing.T) {

	type testpair struct {
		incr int
		line string
	}

	// integer indicates with how much the count of internal state should go up
	// (could be 0, eg. not valid test input, or ds already captured)
	var testCases = []testpair{
		{1, "launchctl-carbon.stdout:24/08/2014 17:59:53 :: [creates] creating database file /opt/graphite/storage/whisper/mac-mini_local/collectd/df-Volumes-Recovery_HD/df_complex-free.wsp (archive=[(60, 525600), (600, 518400)] xff=None agg=None)"},
		{0, "astt"},
		{0, "launchctl-carbon.stdout:24/08/2014 17:59:53 :: [creates] creating database file / (archive=[(60, 525600), (600, 518400)] xff=None agg=None)"},
		{1, "13/09/2014 23:10:56 :: [creates] creating database file /opt/graphite/storage/whisper/local/random/diceroll.wsp (archive=[(60, 525600), (600, 518400)] xff=None agg=None)"},
	}

	State.Vals = nil // start fresh
	prev_count := len(State.Vals)

	for _, test := range testCases {
		parseLine(test.line)

		if len(State.Vals) != prev_count+test.incr {
			t.Fatal(fmt.Sprintf("Parsed line, should have seen %v new entries, saw %v. Line: %v", test.incr, len(State.Vals)-prev_count, test.line))
		}
		prev_count = len(State.Vals)
	}
}
