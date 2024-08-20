package main

import (
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
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
		decoded, err := getTorrentDict(torrentFile)
		if err != nil {
			fmt.Println(err)
			return
		}

		// check if info key exists
		info, ok := decoded["info"]
		if !ok {
			fmt.Println("Info key not found")
			return
		}
		announce, ok := decoded["announce"]
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

		hashString := getInfoHash(info.(map[string]interface{}))
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

	} else if command == "peers" {
		if len(os.Args) < 3 {
			fmt.Println("Missing argument: peers <torrent file>")
			os.Exit(1)
		}
		torrentFile := os.Args[2]
		decoded, err := getTorrentDict(torrentFile)
		if err != nil {
			fmt.Println(err)
			return
		}
		tracker := decoded["announce"].(string)
		info := decoded["info"]
		infoHash := getInfoHash(info.(map[string]interface{}))

		// get length of the file
		length, ok := info.(map[string]interface{})["length"]
		if !ok {
			log.Panicln("Length not found")
		}
		peerId := "-MACBOOK-PRO-" + "123456789012"
		// trim to 20 characters
		peerId = peerId[:20]
		port := 6881
		uploaded := 0
		downloaded := 0
		left := length.(int)
		compact := 1

		//+ "?info_hash=" + infoHash + "&peer_id=" + peerId + "&port=" + strconv.Itoa(port) + "&uploaded=" + strconv.Itoa(uploaded) + "&downloaded=" + strconv.Itoa(downloaded) + "&left=" + strconv.Itoa(left) + "&compact=" + strconv.Itoa(compact)

		// make the url string using net/url
		// make the request
		requestString, err := url.ParseRequestURI(tracker)
		if err != nil {
			fmt.Println(err)
			return
		}
		// add the query parameters
		query := requestString.Query()
		query.Add("peer_id", peerId)
		query.Add("port", strconv.Itoa(port))
		query.Add("uploaded", strconv.Itoa(uploaded))
		query.Add("downloaded", strconv.Itoa(downloaded))
		query.Add("left", strconv.Itoa(left))
		query.Add("compact", strconv.Itoa(compact))
		requestString.RawQuery = query.Encode()

		// Convert the hex string to bytes
		hashBytes, err := hex.DecodeString(infoHash)
		if err != nil {
			fmt.Println("Invalid infoHash:", err)
			return
		}

		// Manually encode each byte of the hash to the desired URL format
		var encodedHash string
		for _, b := range hashBytes {
			encodedHash += fmt.Sprintf("%%%02x", b)
		}

		encodedQuery := requestString.String()
		finalQuery := encodedQuery + "&info_hash=" + encodedHash

		// make the request
		response, err := http.Get(finalQuery)
		if err != nil {
			fmt.Println(err)
			return
		}
		// read the response
		responseBytes, err := io.ReadAll(response.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		// decode the response
		decodedResponse, _, err := decodeBencode(responseBytes)
		if err != nil {
			fmt.Println(err)
			return
		}

		// peers are stored in the response as a list of dictionaries
		peers := []byte(decodedResponse.(map[string]interface{})["peers"].(string))
		//Each peer is represented using 6 bytes. The first 4 bytes are the peer's IP address and the last 2 bytes are the peer's port number
		numPeers := len(peers) / 6
		for i := range numPeers {
			// print the IP address
			ip := fmt.Sprintf("%s.%s.%s.%s",
				strconv.Itoa(int(peers[i*6])),
				strconv.Itoa(int(peers[i*6+1])),
				strconv.Itoa(int(peers[i*6+2])),
				strconv.Itoa(int(peers[i*6+3])),
			)
			//big-endian order parse
			port := binary.BigEndian.Uint16(peers[i*6+4 : i*6+4+2])
			fmt.Printf("%s:%d\n", ip, port)
		}
	} else if command == "handshake" {
		// sample.torrent <peer_ip>:<peer_port>
		if len(os.Args) < 4 {
			fmt.Println("Missing argument: handshake <torrent file> <peer>")
			os.Exit(1)
		}
		torrentFile := os.Args[2]
		peer := os.Args[3]

		decoded, err := getTorrentDict(torrentFile)
		if err != nil {
			fmt.Println(err)
			return
		}
		info := decoded["info"]
		infoHash := getInfoHash(info.(map[string]interface{}))
		peerId := "-MACBOOK-PRO-" + "123456789012"

		// create the handshake message
		protocol := make([]byte, 1+19+8+20+20)
		protocol[0] = 19
		copy(protocol[1:], "BitTorrent protocol")
		// copy the reserved bytes
		copy(protocol[20:], make([]byte, 8))
		// copy the info hash
		infoHashBytes, err := hex.DecodeString(infoHash)
		if err != nil {
			fmt.Println(err)
			return
		}
		copy(protocol[28:], infoHashBytes)
		// copy the peer id
		copy(protocol[48:], []byte(peerId))

		// create the connection
		conn, err := net.Dial("tcp", peer)
		if err != nil {
			fmt.Println(err)
			return
		}
		// send the handshake message
		_, err = conn.Write(protocol)
		if err != nil {
			fmt.Println(err)
			return
		}
		// read the response
		response := make([]byte, 100)
		_, err = conn.Read(response)
		if err != nil {
			fmt.Println(err)
			return
		}
		responsePeerId := response[48:68]
		responsePeerIdHex := hex.EncodeToString(responsePeerId)
		fmt.Println("Peer ID:", responsePeerIdHex)

	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}

func getTorrentDict(torrentFile string) (map[string]interface{}, error) {
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
		return nil, err
	}
	return decoded.(map[string]interface{}), nil
}

func getInfoHash(info map[string]interface{}) string {
	infoBencoded := bencodeDict(info)
	// calculate the SHA-1 hash
	hash := sha1.Sum([]byte(infoBencoded))
	// convert the hash to a string
	hashString := fmt.Sprintf("%x", hash)
	return hashString
}

func bencodeDict(dict interface{}) string {
	result := "d"
	// get the keys in sorted order
	keys := make([]string, len(dict.(map[string]interface{})))
	i := 0

	for key := range dict.(map[string]interface{}) {
		keys[i] = key
		i++
	}

	sort.Strings(keys)
	for k := range keys {
		key := keys[k]
		value := dict.(map[string]interface{})[keys[k]]
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
