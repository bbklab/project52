package naivecrypto

import (
	"encoding/base64"
	"fmt"
	mrand "math/rand"
	"strconv"
	"strings"

	"github.com/bbklab/inf/pkg/utils"
)

var (
	luckyNumber = 9527 // I think 9527 is a lucky number
)

// Encode is exported
func Encode(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	dataEncoded := base64.StdEncoding.EncodeToString(data)               // encode data: base64 encoded original data
	randomText, _ := utils.GenPasswordNonSpecial(randomIntRange(10, 20)) // random text: seems like base64 encoded string [0-9a-zA-Z]
	luckyLength := fmt.Sprintf("%d", len(randomText)+luckyNumber)        // lucky length: random text real length + lucky number
	combinedText := randomText + dataEncoded + ";" + luckyLength         // final text: random text + encoded text + ":" + lucky length
	return obfuscate(combinedText)                                       // obfuscate the result
}

// Decode is exported
func Decode(encoded []byte) string {
	if len(encoded) == 0 {
		return ""
	}

	combinedText := deobfuscate(string(encoded))   // deobfuscate the text
	fields := strings.SplitN(combinedText, ";", 2) // split the text to find out the lucky length
	if len(fields) != 2 {
		return ""
	}

	randomAndEncodedText, luckyLength := fields[0], fields[1]

	luckyLengthN, _ := strconv.Atoi(luckyLength)
	if luckyLengthN == 0 {
		return ""
	}
	realRandomLength := luckyLengthN - luckyNumber // get the random text real length
	if len(randomAndEncodedText) < realRandomLength {
		return ""
	}

	encodedText := string(randomAndEncodedText[realRandomLength:]) // cut off the random text from: random text + encoded text

	data, _ := base64.StdEncoding.DecodeString(encodedText) // base64 decode the data
	return string(data)                                     // this is what we want
}

func obfuscate(s string) string {
	var obfuscated string
	for i := 0; i < len(s); i++ {
		obfuscated += string(int(s[i]) + 1)
	}
	return obfuscated
}

func deobfuscate(s string) string {
	var clear string
	for i := 0; i < len(s); i++ {
		clear += string(int(s[i]) - 1)
	}
	return clear
}

func randomIntRange(min, max int) int {
	return mrand.Intn(max-min) + min
}
