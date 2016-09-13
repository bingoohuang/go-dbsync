package main

import (
	"crypto/cipher"
	"crypto/aes"
	"encoding/base64"
	"fmt"
	"io"
	"crypto/rand"
	"errors"
	"io/ioutil"
	"regexp"
	"bytes"
	"flag"
	"bufio"
	"os"
	"strings"
)

/*
对文本中形如${?:clear text}的文本进行加密，加密后，显示为${AES:密文}，反之解密
*/

func main() {
	mode := flag.String("mode", "encode", "mode(encode/decode)")
	key := flag.String("key", "", "key to encyption or decyption")
	file := flag.String("file", "", "input file")
	flag.Parse()
	if *file == "" {
		fmt.Printf("file argument is required!\n")
		Usage()
		os.Exit(-1)
	}

	keyStr := *key;
	if *key == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Please input key: ")
		keyStr, _ = reader.ReadString('\n')
		keyStr = strings.TrimSpace(keyStr)
	}

	keyStr = FixStrLength(keyStr, 16);

	txtBytes, _ := ioutil.ReadFile(*file)
	txt := string(txtBytes);

	if *mode == "encode" {
		regex, _ := regexp.Compile("\\$\\{\\?:(.*?)\\}")
		result := ReplaceAllGroupFunc(regex, txt, func(groups []string) string {
			cipher, _ := CBCEncrypt(keyStr, groups[1])
			return "${AES:" + cipher + "}"
		})

		fmt.Printf("%s\n", result)
	} else if *mode == "decode" {
		regex, _ := regexp.Compile("\\$\\{AES:(.*?)\\}")
		result := ReplaceAllGroupFunc(regex, txt, func(groups []string) string {
			clear, _ := CBCDecrypt(keyStr, groups[1])
			return clear
		})
		fmt.Printf("%s\n", result)
	} else {
		fmt.Printf("mode argument should be ecode or decode!\n")
		Usage();
		os.Exit(-1)
	}
}
func FixStrLength(s string, fixLen int) string {
	slen := len(s)
	if slen < fixLen {
		return s + strings.Repeat("0", fixLen - slen)
	}

	if (slen > fixLen) {
		return s[:fixLen]
	}

	return s
}

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
}

func ReplaceAllGroupFunc(re *regexp.Regexp, str string, repl func([]string) string) string {
	result := ""
	lastIndex := 0

	for _, v := range re.FindAllSubmatchIndex([]byte(str), -1) {
		groups := []string{}
		for i := 0; i < len(v); i += 2 {
			groups = append(groups, str[v[i]:v[i+1]])
		}

		result += str[lastIndex:v[0]] + repl(groups)
		lastIndex = v[1]
	}

	return result + str[lastIndex:]
}

func CBCEncrypt(strKey, strPlaintext string) (string, error) {
	key := []byte(strKey)
	plaintext := []byte(strPlaintext)

	// CBC mode works on blocks so plaintexts may need to be padded to the
	// next whole block. For an example of such padding, see
	// https://tools.ietf.org/html/rfc5246#section-6.2.3.2. Here we'll
	// assume that the plaintext is already of the correct length.
	//if len(plaintext) % aes.BlockSize != 0 {
	//	return "", errors.New("plaintext is not a multiple of the block size")
	//}
	plaintext = PKCS5Padding(plaintext, aes.BlockSize)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize + len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], plaintext)

	// It's important to remember that ciphertexts must be authenticated
	// (i.e. by using crypto/hmac) as well as being encrypted in order to
	// be secure.

	base64Text := base64.StdEncoding.EncodeToString(ciphertext)

	return base64Text, nil
}

func CBCDecrypt(strKey, strCiphertext string) (string, error) {
	key := []byte(strKey)
	ciphertext, _ := base64.StdEncoding.DecodeString(strCiphertext)

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	// CBC mode always works in whole blocks.
	if len(ciphertext) % aes.BlockSize != 0 {
		return "", errors.New("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)

	// CryptBlocks can work in-place if the two arguments are the same.
	mode.CryptBlocks(ciphertext, ciphertext)

	ciphertext = PKCS5UnPadding(ciphertext)

	// If the original plaintext lengths are not a multiple of the block
	// size, padding would have to be added when encrypting, which would be
	// removed at this point. For an example, see
	// https://tools.ietf.org/html/rfc5246#section-6.2.3.2. However, it's
	// critical to note that ciphertexts must be authenticated (i.e. by
	// using crypto/hmac) before being decrypted in order to avoid creating
	// a padding oracle.
	return string(ciphertext), nil
}

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	// 去掉最后一个字节 unpadding 次
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func ZeroPadding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{0}, padding)
	return append(ciphertext, padtext...)
}