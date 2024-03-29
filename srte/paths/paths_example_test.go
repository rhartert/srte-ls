package paths

import "fmt"

func ExampleNew() {
	p := New(0, 3, 2)

	fmt.Println(p)

	// Output:
	// 0 -> 3
}

func ExamplePathVar_Length() {
	p := New(0, 4, 4)

	fmt.Println(p.Length())
	p.Insert(1, 2)
	fmt.Println(p.Length())
	p.Insert(1, 1)
	fmt.Println(p.Length())

	// Output:
	// 2
	// 3
	// 4
}

func ExamplePathVar_Nodes() {
	p := New(0, 4, 5)
	p.Insert(1, 2)
	p.Insert(1, 1)

	fmt.Println(p.Nodes())

	// Output:
	// [0 1 2 4]
}

func ExamplePathVar_Node() {
	p := New(0, 4, 5)
	p.Insert(1, 2)
	p.Insert(1, 1)

	for i := 0; i < p.Length(); i++ {
		fmt.Println(p.Node(i))
	}
	// Output:
	// 0
	// 1
	// 2
	// 4
}

func ExamplePathVar_Insert() {
	p := New(0, 4, 4)

	fmt.Println(p.Insert(1, 2)) // valid
	fmt.Println(p.Insert(0, 1)) // invalid: cannot replace the source
	fmt.Println(p.Insert(1, 0)) // invalid: same consecutive
	fmt.Println(p.Insert(1, 2)) // invalid: same consecutive
	fmt.Println(p.Insert(1, 1)) // valid
	fmt.Println(p.Insert(3, 3)) // exceed length
	fmt.Println(p)

	// Output:
	// true
	// false
	// false
	// false
	// true
	// false
	// 0 -> 1 -> 2 -> 4
}

func ExamplePathVar_Clear() {
	p := New(0, 4, 4)
	p.Insert(1, 2)
	p.Insert(1, 1)

	fmt.Println(p)
	fmt.Println(p.Clear()) // valid
	fmt.Println(p)
	fmt.Println(p.Clear()) // invalid: cannot clear empty path

	// Output:
	// 0 -> 1 -> 2 -> 4
	// true
	// 0 -> 4
	// false
}

func ExamplePathVar_Remove() {
	// Build path 0 -> 1 -> 2 -> 3 -> 2 -> 4
	p := New(0, 4, 6)
	p.Insert(1, 2)
	p.Insert(1, 3)
	p.Insert(1, 2)
	p.Insert(1, 1)

	fmt.Println(p)
	fmt.Println(p.Remove(0)) // invalid: cannot remove source
	fmt.Println(p)
	fmt.Println(p.Remove(5)) // invalid: cannot remove destination
	fmt.Println(p)
	fmt.Println(p.Remove(3)) // valid
	fmt.Println(p)
	fmt.Println(p.Remove(2)) // valid
	fmt.Println(p)
	fmt.Println(p.Remove(1)) // valid
	fmt.Println(p)

	// Output:
	// 0 -> 1 -> 2 -> 3 -> 2 -> 4
	// false
	// 0 -> 1 -> 2 -> 3 -> 2 -> 4
	// false
	// 0 -> 1 -> 2 -> 3 -> 2 -> 4
	// true
	// 0 -> 1 -> 2 -> 4
	// true
	// 0 -> 1 -> 4
	// true
	// 0 -> 4
}

func ExamplePathVar_Update() {
	// Build path 0 -> 1 -> 2 -> 4
	p := New(0, 4, 4)
	p.Insert(1, 2)
	p.Insert(1, 1)

	fmt.Println(p)
	fmt.Println(p.Update(0, 5)) // invalid: cannot update source
	fmt.Println(p)
	fmt.Println(p.Update(3, 5)) // invalid: cannot update destination
	fmt.Println(p)
	fmt.Println(p.Update(1, 2)) // invalid: consecutive node
	fmt.Println(p)
	fmt.Println(p.Update(1, 0)) // invalid: consecutive node
	fmt.Println(p)
	fmt.Println(p.Update(1, 1)) // invalid: no change
	fmt.Println(p)
	fmt.Println(p.Update(1, 3)) // valid
	fmt.Println(p)
	fmt.Println(p.Update(2, 1)) // valid
	fmt.Println(p)
	fmt.Println(p.Update(1, 2)) // valid
	fmt.Println(p)

	// Output:
	// 0 -> 1 -> 2 -> 4
	// false
	// 0 -> 1 -> 2 -> 4
	// false
	// 0 -> 1 -> 2 -> 4
	// false
	// 0 -> 1 -> 2 -> 4
	// false
	// 0 -> 1 -> 2 -> 4
	// false
	// 0 -> 1 -> 2 -> 4
	// true
	// 0 -> 3 -> 2 -> 4
	// true
	// 0 -> 3 -> 1 -> 4
	// true
	// 0 -> 2 -> 1 -> 4
}
