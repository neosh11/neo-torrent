package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"unicode"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

// recursive function to decode bencoded string
func decodeBencode(bencodedString []byte) (value interface{}, EOL int, err error) {
	if unicode.IsDigit(rune(bencodedString[0])) {
		// STRING
		var firstColonIndex int

		for i := 0; i < len(bencodedString); i++ {
			if bencodedString[i] == ':' {
				firstColonIndex = i
				break
			}
		}
		length, err := strconv.Atoi(string(bencodedString[:firstColonIndex]))
		if err != nil {
			return nil, 0, err
		}
		EOL = firstColonIndex + 1 + length

		return string(bencodedString[firstColonIndex+1 : EOL]), EOL, nil
	} else if rune(bencodedString[0]) == 'i' {
		// INT
		var endIndex int
		for i := 0; i < len(bencodedString); i++ {
			if bencodedString[i] == 'e' {
				endIndex = i
				break
			}
		}
		v, err := strconv.Atoi(string(bencodedString[1:endIndex]))
		return v, endIndex + 1, err

	} else if rune(bencodedString[0]) == 'l' {
		// LIST

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
				return nil, 0, err
			}
			list = append(list, val)
			index += itemEOL
		}
	} else if rune(bencodedString[0]) == 'd' {
		// DICT

		index := 1
		var dict = make(map[string]interface{})
		for {
			if rune(bencodedString[index]) == 'e' {
				return dict, index + 1, nil
			}
			key, itemEOL, err := decodeBencode(bencodedString[index:])
			if err != nil {
				fmt.Errorf("Error decoding dict: %v", err)
				return nil, 0, err
			}
			index += itemEOL
			val, keyEOL, err := decodeBencode(bencodedString[index:])
			if err != nil {
				fmt.Errorf("Error decoding dict: %v", err)
				return nil, 0, err
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
		decoded, _, err := decodeBencode([]byte(bencodedValue))
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	} else if command == "info" {
		if len(os.Args) < 3 {
			fmt.Println("Missing argument: info <torrent file>")
			os.Exit(1)
		}
		torrentFile := os.Args[2]
		// read the torrent file
		file, err := os.Open(torrentFile)
		if err != nil {
			log.Fatal(err)
		}
		// close the file later
		defer func() {
			if err := file.Close(); err != nil {
				log.Fatal(err)
			}
		}()

		// read the entire file into a byte slice
		fileBytes, err := io.ReadAll(file)
		decoded, _, err := decodeBencode(fileBytes)
		if err != nil {
			fmt.Println(err)
			return
		}

		// check if info key exists
		info, ok := decoded.(map[string]interface{})["info"]
		if !ok {
			fmt.Println("Info key not found")
			return
		}
		announce, ok := decoded.(map[string]interface{})["announce"]
		if !ok {
			fmt.Println("Announce key not found")
			return
		}

		fmt.Println("Tracker URL:", announce)
		// get the length of the file
		length, ok := info.(map[string]interface{})["length"]
		if ok {
			fmt.Println("Length:", length)
		} else {
			log.Panicln("Length not found")
		}

		infoBencoded := bencodeDict(info.(map[string]interface{}))
		// calculate the SHA-1 hash
		hash := sha1.Sum([]byte(infoBencoded))
		// convert the hash to a string
		hashString := fmt.Sprintf("%x", hash)
		fmt.Println("Info Hash:", hashString)

		// get the piece length
		pieceLength, ok := info.(map[string]interface{})["piece length"]
		if !ok {
			log.Panicln("Piece Length not found")

		}
		fmt.Println("Piece Length:", pieceLength)
		// get the piece hashes
		pieces, ok := info.(map[string]interface{})["pieces"]
		if !ok {
			log.Panicln("Pieces not found")
		}
		// convert the pieces to a string
		piecesString := pieces.(string)
		// calculate the number of pieces
		numPieces := len(piecesString) / 20
		// print the hashes
		fmt.Println("Pieces Hashes:")
		for i := 0; i < numPieces; i++ {
			fmt.Printf("%x\n", piecesString[i*20:i*20+20])
		}
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}

func bencodeDict(dict interface{}) string {
	result := "d"
	for key, value := range dict.(map[string]interface{}) {
		switch value.(type) {
		case string:
			result += strconv.Itoa(len(key)) + ":" + key + bencodeString(value.(string))
		case int:
			result += strconv.Itoa(len(key)) + ":" + key + bencodeInt(value.(int))
		case []interface{}:
			result += strconv.Itoa(len(key)) + ":" + key + bencodeList(value.([]interface{}))
		case map[string]interface{}:
			result += strconv.Itoa(len(key)) + ":" + key + bencodeDict(value)
		}
	}
	result += "e"
	return result
}

func bencodeString(str string) string {
	return strconv.Itoa(len(str)) + ":" + str
}
func bencodeInt(i int) string {
	return "i" + strconv.Itoa(i) + "e"
}
func bencodeList(list []interface{}) string {
	result := "l"
	for _, item := range list {
		switch item.(type) {
		case string:
			result += bencodeString(item.(string))
		case int:
			result += bencodeInt(item.(int))
		case []interface{}:
			result += bencodeList(item.([]interface{}))
		case map[string]interface{}:
			result += bencodeDict(item)
		}
	}
	result += "e"
	return result
}
