package rbtree_test

import (
	"fmt"
	"math/rand/v2"
	"runtime/debug"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/iku50/rbtree-go"
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
	tree := rbtree.NewRBTree(4, 4)
	for i := range 1000 {
		tree.Insert(i, i)
		tree.Check()
		if err := tree.Check();err != nil{
			t.Error(err)
			t.FailNow()
		}
	}
	t.Log(tree.String())
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

func TestParaGetInsert(t *testing.T) {
	debug.SetGCPercent(-1)
	tree := rbtree.NewRBTree(0, "root")
	inserted := sync.Map{}
	numOperations := 1000000
	for i := 0; i < numOperations; i++ {
		key := rand.Int()
		value := fmt.Sprintf("%d", key)
		inserted.Store(key,value)
	}

	ch := make(chan int)
	go func ()  {
		inserted.Range(func(key, value any) bool {
			k, _ := key.(int)
			ch <- int(k)
			return true
		})
		close(ch)
	}()
	okch := make(chan int)
	value := "sdfsdfds"
	wg := sync.WaitGroup{}
	for i := 0;i < 100;i++{
		wg.Add(1)
		go func ()  {
			defer wg.Done()
			for v := range ch {
				tree.Insert(v, value)
				okch <- v
			}
		}()
	}

	for i :=0; i<100;i++{
		go func () {
			for k := range okch {
				v := tree.Get(k)
				if v == nil {
					t.Log("bad value")
					break
				}else{
					// t.Log(*v)
				}
			}
		}()
	}
	wg.Wait()
	close(okch)
	if err := tree.Check();err != nil{
		t.Log(tree.String())
		t.Error(err)
		t.FailNow()
	}
}


func TestParaInsertDel(t *testing.T) {
	tree := rbtree.NewRBTree(0, "root")
	inserted := sync.Map{}
	numOperations := 1000
	for i := 0; i < numOperations; i++ {
		key := rand.Int()
		value := fmt.Sprintf("%d", key)
		inserted.Store(key,value)
	}

	ch := make(chan int)
	go func ()  {
		inserted.Range(func(key, value any) bool {
			k, _ := key.(int)
			ch <- int(k)
			return true
		})
		close(ch)
	}()
	okch := make(chan int)
	wg := sync.WaitGroup{}
	wg.Add(100)
	for i := 0;i < 100;i++{
		go func ()  {
			defer wg.Done()
			for v := range ch {
				p,_ := inserted.Load(v)
				a, _ := p.(string)
				tree.Insert(v, string(a))
				okch <- v
			}
		}()
	}
	pg := sync.WaitGroup{}
	pg.Add(100)
	for i :=0; i<100;i++{
		go func () {
			defer pg.Done()
			for k := range okch {
				v := tree.Delete(k)
				if v == nil {
					t.Log("bad value")
				}else{
					t.Log("delete",*v)
				}
			}
		}()
	}
	wg.Wait()
	close(okch)
	pg.Wait()
	t.Log(tree.String())
	if err := tree.Check();err != nil{
		t.Error(err)
		t.FailNow()
	}
	t.Log(tree.String())
}


func TestRandomIDG(t *testing.T) {
	tree := rbtree.NewRBTree(0, "root")
	inserted := make(map[int]string)
	numOperations := 50000
	for i := 0; i < numOperations; i++ {
		key := rand.Int()
		value := fmt.Sprintf("%d", key)
		inserted[key] = value
		tree.Insert(key, value)
		err := tree.Check()
		// t.Log(tree.String())
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
			// if _, exists := inserted[key]; exists {
			// 	tree.Delete(key)
			// 	delete(inserted, key)
			// 	assert.Nil(t, tree.Get(key), "Value should be nil after deletion")
			// }
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

func BenchmarkInsertParallel(b *testing.B) {
    tree := rbtree.NewRBTree(rand.Int(), rand.Int())
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            tree.Insert(rand.Int(), rand.Int())
        }
    })
}

func BenchmarkGetParallel(b *testing.B) {
    tree := rbtree.NewRBTree(rand.Int(), rand.Int())
    for i := 0; i < b.N; i++ {
        tree.Insert(rand.Int(), rand.Int())
    }
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            tree.Get(rand.Int())
        }
    })
}


func BenchmarkDeleteParallel(b *testing.B) {
    tree := rbtree.NewRBTree(rand.Int(), rand.Int())
    m := make(map[int]int)
    for i := 0; i < b.N; i++ {
        k, v := rand.Int(), rand.Int()
        tree.Insert(k, v)
        m[k] = v
    }
    keys := make([]int, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            k := keys[rand.IntN(len(keys))]
            tree.Delete(k)
        }
    })
}