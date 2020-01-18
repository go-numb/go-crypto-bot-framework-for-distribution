package db

import (
	"testing"
)

type TestStruct struct {
	Side  int
	Price float64
	Size  float64
}

func TestSet(t *testing.T) {
	client := New("../leveldb")
	defer client.Close()

	count := 100

	for i := 0; i < count; i++ {
		if err := client.Set("orders", &TestStruct{
			Side:  1,
			Price: 10000,
			Size:  0.01,
		}); err != nil {
			t.Error(err)
			continue
		}
	}
}
