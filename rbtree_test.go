package rbtree_test

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/iku50/rbtree"
)

func TestInsertCase1(t *testing.T) {
	tree := rbtree.NewRBTree(1, 1)
	t.Log(tree.String())
}

func TestInsertCase2(t *testing.T) {
	tree := rbtree.NewRBTree(1, 1)
	tree.Insert(0, 2)
	tree.Insert(2, 2)
	t.Log(tree.String())

}

func TestInsertCase3(t *testing.T) {
	tree := rbtree.NewRBTree(1, 1)
	tree.Insert(0, 0)
	tree.Insert(2, 2)
	tree.Insert(4, 4)
	// tree.Insert(1, 1)
	t.Log(tree.String())
}

func TestInsertCase4(t *testing.T) {
	tree := rbtree.NewRBTree(2, 2)
	tree.Insert(1, 1)
	tree.Insert(3, 3)
	tree.Insert(0, 0)
	t.Log(tree.String())
}

func TestInsertCase56(t *testing.T) {
	tree := rbtree.NewRBTree(3, 3)
	tree.Insert(2, 2)
	tree.Insert(1, 1)
	t.Log(tree.String())
}

func TestGet(t *testing.T) {
	tree := rbtree.NewRBTree(1, 1)
	tree.Insert(2, 2)
	tree.Insert(3, 3)
	tree.Insert(0, 0)
	tree.Insert(4, 4)
	tree.Insert(9, 9)
	tree.Insert(7, 7)
	tree.Insert(8, 8)
	assert.Equal(t, 7, *tree.Get(7))
}

func TestDeleteCase0(t *testing.T) {
	tree := rbtree.NewRBTree(1, 1)
	tree.Delete(1)
	assert.Equal(t, (*int)(nil), tree.Get(1))
}

func TestDeleteCase1(t *testing.T) {
	tree := rbtree.NewRBTree(1, 1)
	tree.Insert(0, 0)
	tree.Insert(2, 2)
	tree.Delete(1)
	assert.Equal(t, (*int)(nil), tree.Get(1))
	t.Log(tree.String())
}

func TestDeleteCase2(t *testing.T) {
	tree := rbtree.NewRBTree(1, 1)
	tree.Insert(0, 0)
	tree.Insert(2, 2)
	tree.Delete(0)
	assert.Equal(t, (*int)(nil), tree.Get(0))
	t.Log(tree.String())
}

func TestDeleteCase3(t *testing.T) {
	tree := rbtree.NewRBTree(1, 1)
	tree.Insert(0, 0)
	tree.Insert(2, 2)
	tree.Insert(4, 4)
	tree.Delete(2)
	assert.Equal(t, (*int)(nil), tree.Get(2))
	t.Log(tree.String())
}

func TestRandomIDG(t *testing.T) {
	tree := rbtree.NewRBTree(0, "root")
	inserted := make(map[int]string)
	numOperations := 1000
	for i := 0; i < numOperations; i++ {
		key := rand.Int()
		value := fmt.Sprintf("%d", key)
		inserted[key] = value
		tree.Insert(key, value)
		err := tree.Check()
		t.Log(tree.String())
		if !assert.Nil(t, err, "Check Failed") {
			t.Log(tree.String())
			t.FailNow()
		}
	}
	// t.Log(tree.String())
	for key := range inserted {
		operation := rand.IntN(3)
		switch operation {
		case 0:
			k := rand.Int()
			v := fmt.Sprintf("%d", key)
			inserted[k] = v
			tree.Insert(k, v)
			tree.Insert(k, v)
			inserted[k] = v

			assert.Equal(t, v, *tree.Get(k), "Value should be correctly inserted and retrieved")
		case 1:
			if _, exists := inserted[key]; exists {
				tree.Delete(key)
				delete(inserted, key)
				assert.Nil(t, tree.Get(key), "Value should be nil after deletion")
			}
		case 2:
			expectedValue, exists := inserted[key]
			actualValue := tree.Get(key)

			if exists {
				assert.Equal(t, expectedValue, *actualValue, "Value should be correctly retrieved")
			} else {
				assert.Nil(t, actualValue, "Value should be nil for non-existent key")
			}
		}
	}

	for key, value := range inserted {
		assert.Equal(t, value, *tree.Get(key), "Final check: Value should match for key")
	}
}

func BenchmarkInsert(b *testing.B) {
	tree := rbtree.NewRBTree(rand.Int(), rand.Int())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.Insert(rand.Int(), rand.Int())
	}
}

func BenchmarkGet(b *testing.B) {
	tree := rbtree.NewRBTree(rand.Int(), rand.Int())
	for i := 0; i < b.N; i++ {
		tree.Insert(rand.Int(), rand.Int())
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.Get(rand.Int())
	}
}

func BenchmarkDelete(b *testing.B) {
	tree := rbtree.NewRBTree(rand.Int(), rand.Int())
	m := make(map[int]int)
	for i := 0; i < b.N; i++ {
		k, v := rand.Int(), rand.Int()
		tree.Insert(k, v)
		m[k] = v
	}
	b.ResetTimer()
	for k := range m {
		tree.Delete(k)
	}
}
