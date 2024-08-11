package main

import (
	"encoding/json"
	// Uncomment this line to pass the first stage
	// "encoding/json"
	"fmt"
	"os"
	"strconv"
	"unicode"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

func decodeBencode(bencodedString string) (interface{}, error) {
	if unicode.IsDigit(rune(bencodedString[0])) {
		var firstColonIndex int

		for i := 0; i < len(bencodedString); i++ {
			if bencodedString[i] == ':' {
				firstColonIndex = i
				break
			}
		}
		length, err := strconv.Atoi(bencodedString[:firstColonIndex])
		if err != nil {
			return "", err
		}
		return bencodedString[firstColonIndex+1 : firstColonIndex+1+length], nil
	} else if rune(bencodedString[0]) == 'i' {
		var endIndex int
		for i := 0; i < len(bencodedString); i++ {
			if bencodedString[i] == 'e' {
				endIndex = i
				break
			}
		}
		return strconv.Atoi(bencodedString[1:endIndex])
	} else {
		return "", fmt.Errorf("Only strings, ints are supported at the moment")
	}
}

func main() {
	command := os.Args[1]
	if command == "decode" {
		bencodedValue := os.Args[2]
		decoded, err := decodeBencode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
