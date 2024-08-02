# Red-Black Tree in Go

This repository provides an implementation of a generic Red-Black Tree in Go. A Red-Black Tree is a self-balancing binary search tree, which ensures that the tree remains approximately balanced during insertions and deletions, leading to efficient operations.

## Features

- Generic Implementation: The tree supports generic types for both keys and values, making it versatile for various use cases.
- Clean Code: The codebase follows best practices for readability and maintainability, ensuring that it is easy to understand and modify.
- Close to Original Algorithm: The implementation stays true to the original Red-Black Tree algorithm as described in academic literature, ensuring correctness and reliability.
- Comprehensive Testing: The project includes thorough testing for all operations, with test coverage exceeding 91%, providing confidence in the implementation's correctness and robustness.

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
    value, found := tree.Get(2)
    if found {
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

This implementation is inspired by traditional Red-Black Tree algorithms and adapted for Go's type system.
