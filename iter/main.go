package main

import (
	"fmt"
	"strconv"
	"strings"
)

func main() {
	r := strings.NewReader(`
1
3
5
6
10
`[1:])
	sum, err := Reduce(
		Select(
			Map(
				Lines(r),
				strconv.Atoi,
			),
			odd,
		),
		0,
		func(x, y int) (int, error) { return x + y, nil },
	)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	fmt.Println(sum)
}

func odd(x int) bool {
	return x%2 == 1
}
