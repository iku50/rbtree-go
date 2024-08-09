// According to
// [1] Ma J. Lock-Free Insertions on Red-Black Trees[J]. Master’s thesis. The University of Manitoba, Canada October, 2003
// [2] Kim J. H., Cameron H., Graham P. Lock-free red-black trees using cas[J]. Concurrency and Computation: Practice and experience, 2006: 1-40.
package rbtree

import (
	"cmp"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrParentChildDoublRed = errors.New("parent child doubl red")
	ErrBlackHeightMisMatch = errors.New("black height mismatch")
)

type color int

const (
	red color = iota + 1
	black
)

type direction int

const (
	root direction = 0
	left direction = iota + 2
	right
)

type RBTreeNode[K cmp.Ordered, V any] struct {
	c color

	left   *RBTreeNode[K, V]
	right  *RBTreeNode[K, V]
	parent *RBTreeNode[K, V]

	key   K
	value V

	flag   atomic.Bool  // lock
	hpflag atomic.Int32 // readers
	marker atomic.Bool  // mark above node to avoid areas getting too close
	// l      localArea[K,V]     // a list to impl area lock
	// markers localArea[K,V] // markers this node holds
}

type LocalArea[K cmp.Ordered, V any] []*RBTreeNode[K, V]

func (n *RBTreeNode[K, V]) dir() direction {
	if n.parent == nil {
		return root
	}
	if n.parent.left != nil && n.parent.left == n {
		return left
	}
	return right
}

func (n *RBTreeNode[K, V]) uncle() *RBTreeNode[K, V] {
	if n.parent == nil || n.parent.parent == nil {
		return nil
	}
	if n.parent.dir() == left {
		return n.parent.parent.right
	}
	return n.parent.parent.left
}

func (n *RBTreeNode[K, V]) sibling() *RBTreeNode[K, V] {
	if n.parent == nil {
		return nil
	}
	if n.dir() == left {
		return n.parent.right
	}
	return n.parent.left
}

func (n *RBTreeNode[K, V]) isRed() bool {
	return n != nil && n.c == red
}

func (n *RBTreeNode[K, V]) isBlack() bool {
	return n == nil || n.c == black
}

func (n *RBTreeNode[K, V]) get(key K) (*V, bool) {
	if n == nil {
		return nil, true
	}
	if n.islock() {
		return nil, false
	}
	n.hpflag.Add(1)
	defer n.hpflag.Add(-1)
	switch cmp.Compare(key, n.key) {
	case 0:
		return &n.value, true
	case -1:
		return n.left.get(key)
	case 1:
		return n.right.get(key)
	default:
		return nil, true
	}
}

// rotate Left is like
//
//	  |                       |
//	  N                       S
//	 / \     l-rotate(N)     / \
//	L   S    ==========>    N   R
//	   / \                 / \
//	  M   R               L   M
func (t *RBTree[K, V]) rotateLeft(n *RBTreeNode[K, V]) {
	if n == nil || n.right == nil {
		return
	}
	n.cleanMarker(false)
	dir := n.dir()
	p := n.parent
	newn := n.right
	n.right = newn.left
	n.parent = newn
	if newn.left != nil {
		newn.left.parent = n
	}
	newn.parent = p
	newn.left = n
	switch dir {
	case root:
		t.root = newn
	case left:
		p.left = newn
	case right:
		p.right = newn
	}
}

// rotate right is like
//
//	    |                       |
//	    N                       L
//	   / \     r-rotate(N)     / \
//	  L   S    ==========>    M   N
//	 / \                         / \
//	M   R                       R   S
func (t *RBTree[K, V]) rotateRight(n *RBTreeNode[K, V]) {
	if n == nil || n.left == nil {
		return
	}
	n.cleanMarker(true)
	dir := n.dir()
	p := n.parent
	newn := n.left
	n.left = newn.right
	n.parent = newn
	if newn.right != nil {
		newn.right.parent = n
	}
	newn.parent = p
	newn.right = n
	switch dir {
	case root:
		t.root = newn
	case left:
		p.left = newn
	case right:
		p.right = newn
	}
}

func (n *RBTreeNode[K, V]) cleanMarker(left bool) {
	n.marker.Swap(false)
	if n.parent != nil {
		n.parent.marker.Swap(false)
	}
	if left {
		n.left.marker.Swap(false)
	} else {
		n.right.marker.Swap(false)
	}
	return
}

func (n *RBTreeNode[K, V]) getMarker(m *LocalArea[K, V], isUp bool){
	if n.parent == nil {
		return
	}
	if isUp {
		if len(*m) == 0 ||  (*m)[len(*m)-1].parent == nil {
			return
		}
		for !(*m)[len(*m)-1].parent.flag.CompareAndSwap(false, true){
			time.Sleep(time.Nanosecond * 100)
		}
		for !(*m)[len(*m)-1].parent.marker.CompareAndSwap(false, true) {
			time.Sleep(time.Nanosecond * 100)
		}
		(*m)[len(*m)-1].parent.flag.CompareAndSwap(true,false)
		(*m) = append((*m), (*m)[len(*m)-1].parent)
		return
	}
	d := n.parent.parent
	for i := 0; i < 4 && d != nil; i++ {
		for !d.flag.CompareAndSwap(false, true){
			time.Sleep(time.Nanosecond * 100)
		}
		for !d.marker.CompareAndSwap(false, true) {
			time.Sleep(time.Nanosecond * 100)
		}
		d.flag.CompareAndSwap(true,false)
		(*m) = append((*m), d)
		d = d.parent
	}
}

func (n *RBTreeNode[K, V]) unlockMarker(m *LocalArea[K, V]) {
	for i := range *m {
		(*m)[i].marker.CompareAndSwap(true,false)
	}
	(*m) = (*m)[:0]
	return
}

type RBTree[K cmp.Ordered, V any] struct {
	root  *RBTreeNode[K, V]
	count int
	mu *sync.Mutex
}

func NewRBTree[K cmp.Ordered, V any](key K, value V) *RBTree[K, V] {
	return &RBTree[K, V]{
		count: 1,
		root: &RBTreeNode[K, V]{
			c:     red,
			key:   key,
			value: value,
		},
		mu: &sync.Mutex{},
	}
}

func (t *RBTree[K, V]) maintainAfterInsert(n *RBTreeNode[K, V], l *LocalArea[K, V]) bool {
	if n.isBlack() || n.parent == nil || n.parent.c == black {
		return true
	}
	if n.parent.parent == nil {
		n.parent.c = black
		return true
	}
	if n.uncle().isRed() {
		// get lock above all
		if !n.parent.parent.lockInsert(l) {
			return false
		}
		n.parent.c = black
		n.parent.parent.c = red
		n.uncle().c = black
		p := t.maintainAfterInsert(n.parent.parent, l)
		if !p {
			n.parent.c = red
			n.parent.parent.c = black
			n.uncle().c = red
		}
		return p
	}

	if n.dir() != n.parent.dir() {
		if n.dir() == left {
			t.rotateRight(n.parent)
			t.rotateLeft(n.parent)
			n.left.c = red
		} else {
			t.rotateLeft(n.parent)
			t.rotateRight(n.parent)
			n.right.c = red
		}
		n.c = black
		return true
	} else {
		if n.dir() == left {
			t.rotateRight(n.parent.parent)
		} else {
			t.rotateLeft(n.parent.parent)
		}
		n.parent.c = black
		n.sibling().c = red
	}
	return true
}

func (t *RBTree[K, V]) maintainAfterDelete(n *RBTreeNode[K, V], l, m *LocalArea[K, V]) bool {
	if n.parent == nil {
		return true
	}
	if n.sibling().isRed() {
		s := n.sibling()
		if n.dir() == left {
			t.rotateLeft(n.parent)
		} else {
			t.rotateRight(n.parent)
		}
		s.c = black
		n.parent.c = red
	}
	if n.sibling().left.isBlack() &&
		n.sibling().right.isBlack() &&
		n.parent.isRed() {
		n.sibling().c = red
		n.parent.c = black
		return true
	}
	if n.sibling().left.isBlack() &&
		n.sibling().right.isBlack() &&
		n.parent.c == black {
		n.sibling().c = red
		n.parent.getMarker(m,true)
		t.maintainAfterDelete(n.parent, l, m)
		return true
	}
	if n.dir() == left && n.sibling().left.isRed() && n.sibling().right.isBlack() ||
		n.dir() == right && n.sibling().right.isRed() && n.sibling().left.isBlack() {
		if n.dir() == left {
			t.rotateRight(n.sibling())
			n.sibling().right.c = red
		} else {
			t.rotateLeft(n.sibling())
			n.sibling().left.c = red
		}
		n.sibling().c = black
	}
	if n.dir() == left && n.sibling().right.isRed() || n.dir() == right && n.sibling().left.isRed() {
		if n.dir() == left {
			t.rotateLeft(n.parent)
		} else {
			t.rotateRight(n.parent)
		}
		n.parent.parent.c, n.parent.c = n.parent.c, n.parent.parent.c
		n.parent.sibling().c = black
	}
	return true
}

func (n *RBTreeNode[K, V]) lock() bool {
	if n == nil {
		return false
	}
	if ok := n.flag.CompareAndSwap(false, true); !ok {
		return false
	}
	if n.hpflag.Load() > 0 {
		n.flag.Swap(false)
		return false
	}
	return true
}

func (n *RBTreeNode[K, V]) islock() bool {
	return n.flag.Load()
}

func (n *RBTreeNode[K, V]) unlock() bool {
	if n != nil {
		return n.flag.CompareAndSwap(true, false)
	}
	return true
}

func (l *LocalArea[K, V]) unlockArea() {
	for i := range *l {
		(*l)[i].unlock()
	}
	(*l) = (*l)[:0]
}

func (n *RBTreeNode[K, V]) lockDelete(l *LocalArea[K, V]) bool {
	if ok := n.islock(); ok {
		return false
	}
	(*l) = append((*l), n)
	if n.parent != nil {
		if ok := n.parent.islock(); ok {
			l.unlockArea()
			return false
		}
		(*l) = append((*l), n.parent)
		if n.sibling() != nil {
			if ok := n.sibling().lock(); !ok {
				l.unlockArea()
				return false
			}
			(*l) = append((*l), n.sibling())
			if n.sibling().left != nil {
				if ok := n.sibling().left.lock(); !ok {
					l.unlockArea()
					return false
				}
				(*l) = append((*l), n.sibling().left)
			}
			if n.sibling().right != nil {
				if ok := n.sibling().right.lock(); !ok {
					l.unlockArea()
					return false
				}
				(*l) = append((*l), n.sibling().right)
			}
		}
	}
	return true
}

func (n *RBTreeNode[K, V]) upInsertLock(l *LocalArea[K, V]) bool {
	if !n.islock() || n.isMark() {
		l.unlockArea()
		return false
	}
	if n.parent != nil {
		if !n.parent.islock() || n.parent.isMark() {
			l.unlockArea()
			return false
		}
		if n.sibling() != nil {
			if !n.sibling().islock() || n.sibling().isMark(){
				l.unlockArea()
				return false
			}
		}
		if n.parent.parent != nil {
			if ok := n.parent.parent.lock(); !ok || n.parent.parent.isMark() {
				l.unlockArea()
				return false
			}
			(*l) = append((*l), n.parent.parent)
		}
		if n.uncle() != nil {
			if ok := n.uncle().lock(); !ok || n.uncle().isMark(){
				l.unlockArea()
				return false
			}
			(*l) = append((*l), n.uncle())
		}
	}
	return true
}

func (n *RBTreeNode[K, V]) isMark() bool {
	return n.marker.Load()
}

func (n *RBTreeNode[K, V]) lockInsert(l *LocalArea[K, V]) bool {
	if !n.islock() || n.isMark() {
		return false
	}
	(*l) = append((*l), n)
	if n.parent != nil {
		if ok := n.parent.lock(); !ok || n.parent.isMark(){
			l.unlockArea()
			return false
		}
		(*l) = append((*l), n.parent)
		if n.sibling() != nil {
			if ok := n.sibling().lock(); !ok || n.sibling().isMark() {
				l.unlockArea()
				return false
			}
			(*l) = append((*l), n.sibling())
		}
		if n.parent.parent != nil {
			if ok := n.parent.parent.lock(); !ok || n.parent.parent.isMark() {
				l.unlockArea()
				return false
			}
			(*l) = append((*l), n.parent.parent)
		}
		if n.uncle() != nil {
			if ok := n.uncle().lock(); !ok || n.uncle().isMark() {
				l.unlockArea()
				return false
			}
			(*l) = append((*l), n.uncle())
		}
	}
	return true
}

func (t *RBTree[K, V]) insert(n *RBTreeNode[K, V], key K, value V, l *LocalArea[K, V]) (isNew bool, succeed bool) {
	if ok := n.lock(); !ok {
		return false, false
	}
	defer n.unlock()
	if n.parent != nil {
		n.parent.unlock()
	}
	if n.key == key {
		n.value = value
		return false, true
	}
	if n.key > key && n.left != nil {
		return t.insert(n.left, key, value, l)
	}
	if n.key < key && n.right != nil {
		return t.insert(n.right, key, value, l)
	}
	insert := &RBTreeNode[K, V]{
		c:      red,
		key:    key,
		value:  value,
		parent: n,
	}
	if n.key > key {
		n.left = insert
	} else {
		n.right = insert
	}
	if n.isRed() {
		insert.lock()
		defer insert.unlock()
		n.unlock()
		if insert.lockInsert(l) {
			defer l.unlockArea()
			if !t.maintainAfterInsert(insert, l) {
				if n.key > key {
					n.left = nil
				} else {
					n.right = nil
				}
				return true, false
			}
		} else {
			if n.key > key {
				n.left = nil
			} else {
				n.right = nil
			}
			return true, false
		}
	}
	return true, true
}

func (t *RBTree[K, V]) Insert(key K, value V) {
	// case 1
	if t.root == nil {
		t.root = &RBTreeNode[K, V]{
			c:     red,
			key:   key,
			value: value,
		}
		return
	}
	var new bool
	var ok bool
	l := make(LocalArea[K, V], 0)
	for new, ok = t.insert(t.root, key, value, &l); !ok; new, ok = t.insert(t.root, key, value, &l) {
		time.Sleep(100 * time.Microsecond)
		l = l[:0]
	}
	if new {
		t.count++
	}
}

func (n *RBTreeNode[K, V]) swap(d *RBTreeNode[K, V]) {
	n.key, d.key = d.key, n.key
	d.value, n.value = n.value, d.value
}

func (t *RBTree[K, V]) delete(n *RBTreeNode[K, V], key K, l, m *LocalArea[K, V]) (*V, bool) {
	if n == nil {
		return nil, true
	}
	if ok := n.lock(); !ok {
		return nil, false
	}
	defer n.unlock()
	if n.parent != nil {
		if n.parent.parent != nil {
			n.parent.parent.unlock()
		}
	}
	switch cmp.Compare(key, n.key) {
	case 0:
		{
			if !n.lockDelete(l){
				return nil,false
			}
			defer l.unlockArea()
			n.getMarker(m,false)
			defer n.unlockMarker(m)
			v := n.value
			// case 1
			if n.left != nil && n.right != nil {
				// step 1: find successor s
				s := n.right
				p := n
				for s.left != nil {
					p = s
					s = p.left
				}
				// step 2: swap data
				n.swap(s)
				n = s
				// step 3: fall into case 2,3
			}
			// case 2: if is leaf node
			if n.left == nil && n.right == nil {
				if n.c == black {
					t.maintainAfterDelete(n, l, m)
				}
				if n.dir() == left {
					n.parent.left = nil
				} else {
					n.parent.right = nil
				}
				// case 3: only have one non-nil child
			} else {
				var rep *RBTreeNode[K, V]
				if n.left == nil {
					rep = n.right
				} else {
					rep = n.left
				}
				p := n.parent
				switch n.dir() {
				case root:
					t.root = rep
				case left:
					p.left = rep
				case right:
					p.right = rep
				}
				rep.parent = p
				rep.c = black
			}
			t.count--
			return &v, true
		}
	case -1:
		return t.delete(n.left, key, l, m)
	case 1:
		return t.delete(n.right, key, l, m)
	}
	return nil, true
}

func (t *RBTree[K, V]) Delete(key K) *V {
	// case 0
	if t.count == 1 && t.root.key == key {
		v := t.root.value
		t.root = nil
		t.count--
		return &v
	}
	var b *V
	var ok bool
	l := make(LocalArea[K, V], 0)
	m := make(LocalArea[K, V], 0)
		for b, ok = t.delete(t.root, key, &l, &m); !ok; b, ok = t.delete(t.root, key, &l, &m) {
			time.Sleep((50+time.Duration(rand.Int64N(40)))* time.Microsecond)
			l = l[:0]
			m = m[:0]
		}
	return b
}

func (t *RBTree[K, V]) Get(key K) *V {
	var b *V
	var ok bool
	for b, ok = t.root.get(key); !ok; b, ok = t.root.get(key) {
		time.Sleep(10 * time.Microsecond)
	}
	return b
}

func (t *RBTree[K, V]) check(n *RBTreeNode[K, V], bc int) (int, error) {
	if n == nil || n.flag.Load() {
		return bc, nil
	}
	if n.isRed() {
		if n.left.isRed() || n.right.isRed() {
			return 0, ErrParentChildDoublRed
		}
	}
	if n.isBlack() {
		bc++
	}
	lc, le := t.check(n.left, bc)
	if le != nil {
		return 0, le
	}
	rc, re := t.check(n.right, bc)
	if re != nil {
		return 0, re
	}
	if lc != rc {
		return 0, ErrBlackHeightMisMatch
	}
	return lc, nil
}

func (t *RBTree[K, V]) Check() error {
	if t.root == nil {
		return nil
	}
	_, err := t.check(t.root, 0)
	return err
}

func (c color) String() string {
	switch c {
	case red:
		return "red"
	case black:
		return "black"
	default:
		return "unknown"
	}
}

func (n *RBTreeNode[K, V]) String() string {
	if n == nil {
		return "nil"
	}
	left := "nil"
	if n.left != nil {
		left = fmt.Sprintf("%v", n.left.key)
	}
	right := "nil"
	if n.right != nil {
		right = fmt.Sprintf("%v", n.right.key)
	}
	// parent := "nil"
	// if n.parent != nil {
	// 	parent = fmt.Sprintf("%v", n.parent.key)
	// }
	return fmt.Sprintf("[key: %v, color: %s,left: %s, right: %s, flag: %v, mark: %v]",
		n.key, n.c, left, right, n.flag.Load(), n.marker.Load())
}

func (t *RBTree[K, V]) String() string {
	if t.root == nil {
		return "nil"
	}
	var sb strings.Builder
	t.buildString(t.root, "", &sb)
	return sb.String()
}

func (t *RBTree[K, V]) buildString(n *RBTreeNode[K, V], prefix string, sb *strings.Builder) {
	if n == nil {
		return
	}
	sb.WriteString(fmt.Sprintf("%s%s\n", prefix, n))
	if n.left != nil || n.right != nil {
		t.buildString(n.left, prefix+"L-> ", sb)
		t.buildString(n.right, prefix+"R-> ", sb)
	}
}
