package utils

type Item struct {
	name string
	age  int
}

func foo(item *Item) {
	rand := Item{age: 3, name: "ed"}
	print(rand.age)
	print(item.name)
}

func Bar() {
	//var item Item
	foo(nil)
}
