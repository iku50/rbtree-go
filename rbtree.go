// According to
// [1] Ma J. Lock-Free Insertions on Red-Black Trees[J]. Master’s thesis. The University of Manitoba, Canada October, 2003
// [2] Kim J. H., Cameron H., Graham P. Lock-free red-black trees using cas[J]. Concurrency and Computation: Practice and experience, 2006: 1-40.
package rbtree

import (
	"cmp"
	"errors"
	"fmt"
	"strings"
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
	left direction = iota + 3
	right
)

type RBTreeNode[K cmp.Ordered, V any] struct {
	c      color

	left   *RBTreeNode[K, V]
	right  *RBTreeNode[K, V]
	parent *RBTreeNode[K, V]

	key    K
	value  V

	flag   atomic.Bool   // lock
	hpflag atomic.Int32 // readers
	marker atomic.Bool   // mark above node to avoid areas getting too close
	l      localArea[K,V]     // a list to impl area lock
}

type localArea[K cmp.Ordered, V any] struct {
	Next *localArea[K,V]
	Val  *RBTreeNode[K,V]
}

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

func (n *RBTreeNode[K, V]) getMarker() bool {
	d := n.parent
	for i := 0;i < 4&&d!=nil;i++{
		if d.islock(){
			return false
		}
		if !d.marker.CompareAndSwap(false,true){
			return false
		}
		d=d.parent
	}
	return true
}

func (n *RBTreeNode[K, V]) unlockMarker()  {
	d := n.parent
	for i := 0;i < 4&&d!=nil;i++{
		if d.islock(){
			return
		}
		d.marker.Swap(false)
		d=d.parent
	}
	return
}

type RBTree[K cmp.Ordered, V any] struct {
	root  *RBTreeNode[K, V]
	count int
}

func NewRBTree[K cmp.Ordered, V any](key K, value V) *RBTree[K, V] {
	return &RBTree[K, V]{
		count: 1,
		root: &RBTreeNode[K, V]{
			c:     red,
			key:   key,
			value: value,
		},
	}
}

func (t *RBTree[K, V]) maintainAfterInsert(n *RBTreeNode[K, V]) bool {
	if !n.lockInsert(){
		return false
	}
	defer n.unlockArea()
	if n.isBlack() || n.parent == nil || n.parent.c == black {
		return true
	}
	if n.parent.parent == nil {
		n.parent.c = black
		return true
	}
	if n.uncle().isRed() {
		n.parent.c = black
		n.parent.parent.c = red
		n.uncle().c = black
		n.unlockArea()
		return t.maintainAfterInsert(n.parent.parent)
	}
	if n.dir() != n.parent.dir() {
		p := n.parent
		if n.dir() == left {
			t.rotateRight(n.parent)
		} else {
			t.rotateLeft(n.parent)
		}
		n = p
	}
	if n.dir() == n.parent.dir() {
		if n.dir() == left {
			t.rotateRight(n.parent.parent)
		} else {
			t.rotateLeft(n.parent.parent)
		}
		n.parent.c = black
		if n.sibling() != nil {
			n.sibling().c = red
		}
	}
	return true
}

func (t *RBTree[K, V]) maintainAfterDelete(n *RBTreeNode[K, V]) bool {
	if n.parent == nil {
		return true
	}
	if !n.lockDelete(){
		return false
	}
	defer n.unlockArea()
	if !n.getMarker(){
		return false
	}
	defer n.unlockMarker()
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
		n.unlockArea()
		t.maintainAfterDelete(n.parent)
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
	ok := n.flag.CompareAndSwap(false, true)
	if !ok {
		return false
	}
	if n.hpflag.Load() > 1 {
		n.flag.Swap(false)
		return false
	}
	return true
}

func (n *RBTreeNode[K, V]) islock() bool {
	return n.flag.Load()
}

func (n *RBTreeNode[K, V]) unlock() bool {
	return n.flag.CompareAndSwap(true, false)
}

func (n *RBTreeNode[K, V]) unlockArea() {
	p := &n.l
	for p != nil && p.Val != nil {
		p.Val.flag.CompareAndSwap(true, false)
		p = p.Next
	}
	n.l = localArea[K,V]{}
}

func (n *RBTreeNode[K, V]) lockDelete() bool {
	n.l =  localArea[K,V]{}
	d := &n.l
	if ok := n.lock(); !ok {
		return false
	}
	d.Val = n
	d.Next = new(localArea[K,V])
	d = d.Next
	if n.parent != nil {
		if ok := n.parent.lock(); !ok {
			n.unlockArea()
			return false
		}
		d.Val = n.parent
		d.Next = new(localArea[K,V])
		d = d.Next

		if n.uncle() != nil {
			if ok := n.uncle().lock(); !ok {
				n.unlockArea()
				return false
			}
			d.Val = n.uncle()
			d.Next = new(localArea[K,V])
			d = d.Next
			if n.uncle().left != nil {
				if ok := n.uncle().left.lock(); !ok {
					n.unlockArea()
					return false
				}
				d.Val = n.uncle().left
				d.Next = new(localArea[K,V])
				d = d.Next
			}
			if n.uncle().right != nil {
				if ok := n.uncle().right.lock(); !ok {
					n.unlockArea()
					return false
				}
				d.Val = n.uncle().right
				d.Next = new(localArea[K,V])
				d = d.Next
			}
		}
	}
	return true
}

func (n *RBTreeNode[K, V]) lockInsert() bool {
	n.l = localArea[K,V]{}
	d := &n.l
	if ok := n.lock(); !ok {
		return false
	}
	d.Val = n
	d.Next = new(localArea[K,V])
	d = d.Next
	if n.parent != nil {
		if ok := n.parent.lock(); !ok {
			n.unlockArea()
			return false
		}
		d.Val = n.parent
		d.Next = new(localArea[K,V])
		d = d.Next
		if n.parent.parent != nil {
			if ok := n.parent.parent.lock(); !ok {
				n.unlockArea()
				return false
			}
			d.Val = n.parent.parent
			d.Next = new(localArea[K,V])
			d = d.Next
		}
		if n.uncle() != nil {
			if ok := n.uncle().lock(); !ok {
				n.unlockArea()
				return false
			}
			d.Val = n.uncle()
			d.Next = new(localArea[K,V])
			d = d.Next
		}
	}
	return true
}

func (t *RBTree[K, V]) insert(n *RBTreeNode[K, V], key K, value V) (isNew bool, succeed bool) {
	if ok := n.lock(); !ok {
		return false, false
	}
	defer n.unlock()
	if n.key == key {
		n.value = value
		return false, true
	}
	if n.key > key && n.left != nil {
		n.unlock()
		return t.insert(n.left, key, value)
	}
	if n.key < key && n.right != nil {
		n.unlock()
		return t.insert(n.right, key, value)
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
		n.unlock()
		if !t.maintainAfterInsert(insert){
			if n.key > key {
				n.left = nil
			}else{
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
	for new, ok = t.insert(t.root, key, value); !ok; new, ok = t.insert(t.root, key, value) {
		time.Sleep(100 * time.Nanosecond)
	}
	if new {
		t.count++
	}
}

func (n *RBTreeNode[K, V]) swap(d *RBTreeNode[K, V]) {
	n.key, d.key = d.key, n.key
	d.value, n.value = n.value, d.value
}

func (t *RBTree[K, V]) delete(n *RBTreeNode[K, V], key K) (*V, bool) {
	if n == nil {
		return nil, true
	}
	if ok := n.lock(); !ok {
		return nil, false
	}
	defer n.unlock()
	switch cmp.Compare(key, n.key) {
	case 0:
		{
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
					n.unlock()
					t.maintainAfterDelete(n)
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
		n.unlock()
		return t.delete(n.left, key)
	case 1:
		n.unlock()
		return t.delete(n.right, key)
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
	for b, ok = t.delete(t.root, key); !ok; b, ok = t.root.get(key) {
		time.Sleep(10 * time.Nanosecond)
	}
	return b
}

func (t *RBTree[K, V]) Get(key K) *V {
	var b *V
	var ok bool
	for b, ok = t.root.get(key); !ok; b, ok = t.root.get(key) {
		time.Sleep(10 * time.Nanosecond)
	}
	return b
}

func (t *RBTree[K, V]) check(n *RBTreeNode[K, V], bc int) (int, error) {
	if n == nil {
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
	parent := "nil"
	if n.parent != nil {
		parent = fmt.Sprintf("%v", n.parent.key)
	}
	return fmt.Sprintf("[key: %v, value: %v, color: %s, parent: %s, left: %s, right: %s]",
		n.key, n.value, n.c, parent, left, right)
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
