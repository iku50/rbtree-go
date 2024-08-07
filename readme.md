# Red-Black Tree in Go

This repository provides an implementation of a generic Red-Black Tree in Go. A Red-Black Tree is a self-balancing binary search tree, which ensures that the tree remains approximately balanced during insertions and deletions, leading to efficient operations.

## Features

- Generic Implementation: The tree supports generic types for both keys and values, making it versatile for various use cases.
- Clean Code: The codebase follows best practices for readability and maintainability, ensuring that it is easy to understand and modify.
- Close to Original Algorithm: The implementation stays true to the original Red-Black Tree algorithm as described in academic literature, ensuring correctness and reliability.
- Comprehensive Testing: The project includes thorough testing for all operations, with test coverage exceeding 91%, providing confidence in the implementation's correctness and robustness.
- Lock-Free and CAS for Concurrent Operations: The tree now includes lock-free and compare-and-swap (CAS) mechanisms to support concurrent insertions, deletions, and searches, enhancing performance in multi-threaded environments.

## Usage

Here's a quick example of how to use the Red-Black Tree:

```go
package main

import (
    "fmt"
    "github.com/iku50/rbtree"
)

func main() {
    // Create a new Red-Black Tree
    tree := rbtree.NewRBTree(0, "root")

    // Insert elements
    tree.Insert(1, "one")
    tree.Insert(2, "two")
    tree.Insert(3, "three")

    // Search for an element
    value := tree.Get(2)
    if value != nil {
        fmt.Println("Found:", value)
    } else {
        fmt.Println("Not found")
    }

    // Delete an element
    tree.Delete(2)

    // Check if the tree is valid
    if err := tree.Check(); err != nil {
        fmt.Println("Tree is invalid:", err)
    } else {
        fmt.Println("Tree is valid")
    }
}
```

## License

This project is licensed under the MIT License. See the LICENSE file for more details.

## Acknowledgments

This implementation is inspired by traditional Red-Black Tree algorithms and adapted for Go's type system. The concurrent operations are inspired by modern lock-free programming techniques to enhance performance and scalability.

## References

[1] Ma J. Lock-Free Insertions on Red-Black Trees[J]. Master’s thesis. The University of Manitoba, Canada October, 2003

[2] Kim J. H., Cameron H., Graham P. Lock-free red-black trees using cas[J]. Concurrency and Computation: Practice and experience, 2006: 1-40.

## Performance

```shell
goos: darwin
goarch: arm64
pkg: github.com/iku50/rbtree-go
=== RUN   BenchmarkInsert
BenchmarkInsert
BenchmarkInsert-8        1214280              1054 ns/op             148 B/op          5 allocs/op
=== RUN   BenchmarkGet
BenchmarkGet
BenchmarkGet-8           1224667               889.0 ns/op             0 B/op          0 allocs/op
=== RUN   BenchmarkDelete
BenchmarkDelete
BenchmarkDelete-8        1329610               964.7 ns/op            31 B/op          2 allocs/op
=== RUN   BenchmarkInsertParallel
BenchmarkInsertParallel
BenchmarkInsertParallel-8        2257762               565.7 ns/op           148 B/op          5 allocs/op
=== RUN   BenchmarkGetParallel
BenchmarkGetParallel
BenchmarkGetParallel-8           4433664               315.2 ns/op             0 B/op          0 allocs/op
=== RUN   BenchmarkDeleteParallel
BenchmarkDeleteParallel
BenchmarkDeleteParallel-8        3298515               415.6 ns/op             0 B/op          0 allocs/op
```
