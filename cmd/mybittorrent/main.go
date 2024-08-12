package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"unicode"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

// recursive function to decode bencoded string

func decodeBencode(bencodedString string) (value interface{}, EOL int, err error) {
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
			return "", 0, err
		}
		EOL = firstColonIndex + 1 + length

		return bencodedString[firstColonIndex+1 : EOL], EOL, nil
	} else if rune(bencodedString[0]) == 'i' {
		var endIndex int
		for i := 0; i < len(bencodedString); i++ {
			if bencodedString[i] == 'e' {
				endIndex = i
				break
			}
		}
		v, err := strconv.Atoi(bencodedString[1:endIndex])
		return v, endIndex + 1, err

	} else if rune(bencodedString[0]) == 'l' {
		log.Printf("List", bencodedString)
		// this is a list -> keep processing until we find a closing e
		index := 1
		var list = make([]interface{}, 0)
		for {
			if rune(bencodedString[index]) == 'e' {
				return list, index + 1, nil
			}
			val, itemEOL, err := decodeBencode(bencodedString[index:])
			if err != nil {
				fmt.Errorf("Error decoding list: %v", err)
				return "", 0, err
			}
			list = append(list, val)
			index += itemEOL
		}
	} else if rune(bencodedString[0]) == 'd' {
		log.Printf("Dict", bencodedString)
		index := 1
		var dict = make(map[string]interface{})
		for {
			if rune(bencodedString[index]) == 'e' {
				return dict, index + 1, nil
			}
			key, itemEOL, err := decodeBencode(bencodedString[index:])
			if err != nil {
				fmt.Errorf("Error decoding dict: %v", err)
				return "", 0, err
			}
			index += itemEOL
			val, keyEOL, err := decodeBencode(bencodedString[index:])
			if err != nil {
				fmt.Errorf("Error decoding dict: %v", err)
				return "", 0, err
			}
			index += keyEOL
			dict[key.(string)] = val
		}
	}
	return "", 0, fmt.Errorf("Unsupported")
}

func main() {
	command := os.Args[1]
	if command == "decode" {
		bencodedValue := os.Args[2]
		decoded, _, err := decodeBencode(bencodedValue)
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
