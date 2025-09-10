package main

import (
	"fmt"
	"strings"

	"github.com/w0rng/audit"
)

func main() {
	log := audit.New()

	log.Create(
		"entity:1",
		"Вася",
		"Создание сущности",
		map[string]audit.Value{
			"status": audit.PlainValue("new"),
			"money":  audit.HiddenValue(),
		},
	)

	log.Update(
		"entity:1",
		"Вася",
		"обновлен статус",
		map[string]audit.Value{
			"status": audit.PlainValue("approved"),
		},
	)

	log.Update(
		"entity:1",
		"Вася",
		"заявка отредактирована",
		map[string]audit.Value{
			"some_field": audit.PlainValue([]string{"a", "b"}),
		},
	)

	fmt.Println("\nLogs:")
	logs := log.Logs("entity:1")
	for _, c := range logs {
		fmt.Printf("%s\n", c.Description)
		for _, f := range c.Fields {
			fmt.Printf("\t%s: %v → %v\n", f.Field, f.From, f.To)
		}
	}

	fmt.Println("\n\nEvents (status):")
	events := log.Events("entity:1", "status")
	result := []string{}
	for _, c := range events {
		for _, k := range c.Payload {
			result = append(result, fmt.Sprintf("%s", k.Data))
		}
	}
	fmt.Println(strings.Join(result, " → "))
}
