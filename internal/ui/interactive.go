package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var stdinReader = bufio.NewReader(os.Stdin)

func Prompt(label, fallback string) (string, error) {
	fmt.Printf("%s", label)
	if fallback != "" {
		fmt.Printf(" [%s]", fallback)
	}
	fmt.Print(": ")
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(line)
	if v == "" {
		return fallback, nil
	}
	return v, nil
}

func SelectMany(max int) ([]int, error) {
	fmt.Print("Seleziona commit (es. 1,3,5): ")
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}
	parts := strings.Split(line, ",")
	seen := map[int]bool{}
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("selezione non valida: %q", p)
		}
		if n < 1 || n > max {
			return nil, fmt.Errorf("indice fuori range: %d", n)
		}
		if seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out, nil
}
