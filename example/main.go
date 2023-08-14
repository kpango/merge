package main

import (
	"encoding/json"
	"fmt"

	"github.com/kpango/merge"
)

type Details struct {
	Hobbies []string
	Friends map[string]string
	Active  bool
}

type ExtraInfo interface{}

type Person struct {
	Name    string
	Age     int
	Address *Details
	IsAlive bool
	Extra   ExtraInfo
}

func main() {
	person1 := &Person{Name: "Alice", Age: 30, Address: &Details{Active: true}, IsAlive: true}
	person2 := &Person{Name: "Bob", Address: &Details{Hobbies: []string{"Swimming"}, Friends: map[string]string{"Tom": "Neighbor"}}, Extra: "Extra Data"}

	merged, err := merge.Merge(person1, person2)
	if err != nil {
		fmt.Println(err)
	}
	body, err := json.Marshal(&merged)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("Merged Person: %s\n", string(body))
}
