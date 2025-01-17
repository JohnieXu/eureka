//
// Eureka
// ======
// Eureka is a handy utility to encrypt files and folders. It follows several principles:
//
// - I want to encrypt and send a file to someone or myself.
// - Eureka should be easy to install and share.
// - PGP is too cumbersome to use, I want something simple (right-click > encrypt).
// - I already share two separate and secure channels with the recipient (mail + signal for example).
//
// Here how the code is organized:
//
// - eureka.go 		// the main code to encrypt files
// - folders.go 	// the code to compress folders
// - ui_windows.go 	// the code to add right-click encrypt/decrypt on windows
// - ui_macOS.go 	// the code to add right-click encrypt/decrypt on macOS
// - ui_linux.go 	// the code to add right-click encrypt/decrypt on linux

package main

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	// cross-platform clipboard
	"github.com/atotto/clipboard"
)

// open a link in your favorite browser
func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// prompt for the key with a GUI
func promptKey(noClipboard bool) (string, error) {
	var key string
	reader := bufio.NewReader(os.Stdin)
	if !noClipboard {
		fmt.Printf("Do you want to use your clipboard as the key? (Y/n): ")
		useClipboard, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		// clipboard option
		useClipboard = strings.TrimSpace(useClipboard)
		if strings.ToLower(useClipboard) != "n" {
			key, err := clipboard.ReadAll()
			if err != nil {
				return "", fmt.Errorf("error: couldn't read the key from clipboard: %s", err)
			}
			return key, nil
		} else {
			noClipboard = true
		}
	}

	if noClipboard {
		// terminal option
		fmt.Print("Enter 256-bit hexadecimal key: ")
		key, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("couldn't read the key: %s", err)
		}

		key = strings.TrimSpace(key)
		return key, nil
	}

	return key, nil
}

func main() {
	var err error

	// parse flags
	about := flag.Bool("about", false, "to get redirected to github.com/mimoo/eureka")
	noClipboard := flag.Bool("noclipboard", false, "no clipboard")
	fileNameAsKey := flag.Bool("fileNameAsKey", true, "use fileName as key, in encrypt: replace the out fileName with key, in decrypt: use the fileName as key")

	flag.Parse()

	// redirect to github.com/mimoo/eureka
	if *about {
		openBrowser("https://www.github.com/mimoo/eureka")
		return
	}

	if len(flag.Args()) == 0 {
		fmt.Println("===================ᕙ(⇀‸↼‶)ᕗ===================")
		fmt.Println(" Eureka is a tool to help you encrypt/decrypt a file")
		fmt.Println(" to encrypt:")
		fmt.Println("     eureka your-file")
		fmt.Println(" to decrypt:")
		fmt.Println("     eureka your-file.encrypted")
		fmt.Println("===================ᕙ(⇀‸↼‶)ᕗ===================")
		flag.Usage()
		return
	}

	encrypt, decrypt := new(bool), new(bool)
	inFile := &flag.Args()[0]
	ext := strings.ToLower(filepath.Ext(*inFile))
	fileName := filepath.Base(*inFile)
	fileName = fileName[:len(fileName)-len(ext)]
	fmt.Printf("eureka: fileName is %s\n", fileName)
	if ext != ".encrypted" {
		*encrypt = true
	} else {
		*decrypt = true
	}

	// nonce = 1111...
	nonce := bytes.Repeat([]byte{1}, 12)

	// key = ?
	var key []byte

	// generate random key if we're encrypting
	if *encrypt {
		key = make([]byte, 32)
		if _, err = io.ReadFull(rand.Reader, key); err != nil {
			fmt.Println("error: randomness cannot be generated on your system")
			flag.Usage()
			os.Exit(1)
		}
	}

	// get key if we are decrypting
	if *decrypt {
		// get key
		var keyHex string
		var err error

		if !*fileNameAsKey || len(fileName) != 64 {
			// use promptKey
			if !*fileNameAsKey {
				fmt.Printf("eureka: fileNameAsKey option is false, read key from Prompt\n")
			}
			if len(fileName) != 64 {
				fmt.Printf("eureka: fileName is not 64 hexadecimal string, read key from Prompt\n")
			}
			keyHex, err = promptKey(*noClipboard)
			if err != nil {
				fmt.Printf("eureka: %s\n", err)
				os.Exit(1)
			}
		} else {
			// use fileName as key
			keyHex = fileName
		}

		// decode and check key
		key, err = hex.DecodeString(keyHex)
		if err != nil || len(key) != 32 {
			fmt.Println("error: the key has to be a 256-bit hexadecimal string")
			os.Exit(1)
		}
	}

	// create AES-GCM instance
	cipherAES, err := aes.NewCipher(key)
	if err != nil {
		fmt.Println("Can't instantiate AES")
		os.Exit(1)
	}
	AESgcm, err := cipher.NewGCM(cipherAES)
	if err != nil {
		fmt.Println("Can't instantiate GCM")
		os.Exit(1)
	}

	// encrypt or decrypt
	var contentAfter []byte

	if *encrypt {
		// compress file or folder
		var buf bytes.Buffer
		if err := compress(*inFile, &buf); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		// encrypt compressed content
		contentAfter = AESgcm.Seal(nil, nonce, buf.Bytes(), nil)
		// write file to disk
		_, outFile := filepath.Split(*inFile)
		if *fileNameAsKey {
			outFile = hex.EncodeToString(key) + ".encrypted"
		} else {
			outFile = outFile + ".encrypted"
		}
		if err = ioutil.WriteFile(outFile, contentAfter, 0600); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		// place key in clipboard
		stringKey := fmt.Sprintf("%032x", key)
		// notification
		fmt.Printf("File encrypted at %s\n", outFile)
		if *fileNameAsKey {
			fmt.Printf("The key is the name of the file, you can decrypt the file with commond: eureka %s\n", outFile)
		} else {
			fmt.Println("Your recipient will need Eureka to decrypt the file: https://github.com/mimoo/eureka")
			fmt.Println("In a different secure channel, pass the following one-time key to your recipient.")
		}

		// clipboard option
		reader := bufio.NewReader(os.Stdin)
		if !*noClipboard {
			fmt.Printf("Do you want to copy the key to your clipboard? (Y/n): ")
			useClipboard, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("read clipboard input error: %s\nshow key here anyway:\n", err)
				fmt.Println(stringKey)
				return
			}

			useClipboard = strings.TrimSpace(useClipboard)
			if strings.ToLower(useClipboard) != "n" { // use clipboard
				if err := clipboard.WriteAll(stringKey); err != nil {
					fmt.Printf("write clipboard error: %s\nshow key here anyway:\n", err)
					fmt.Println(stringKey)
				} else {
					fmt.Println("key copied to your clipboard")
				}
			} else { // print to terminal and pause
				fmt.Println(stringKey)
			}
		} else {
			fmt.Println(stringKey)
		}

		return
	}

	if *decrypt {
		// open file
		content, err := ioutil.ReadFile(*inFile)
		if err != nil {
			fmt.Println("error: cannot open input file")
			flag.Usage()
			os.Exit(1)
		}
		// decrypt
		contentAfter, err = AESgcm.Open(nil, nonce, content, nil)
		if err != nil {
			fmt.Println("error: cannot decrypt. The key is not correct or someone tried to modify your file.")
			os.Exit(1)
		}
		// create a decrypted folder
		if _, err := os.Stat("./decrypted"); err != nil {
			if err := os.MkdirAll("./decrypted", 0755); err != nil {
				fmt.Println("error: cannot create folder 'decrypted'")
				os.Exit(1)
			}
		} else {
			fmt.Println("info: the folder 'decrypted' already exists. Decrypting the file could overwrite files.")
			return
		}
		// decompress it
		buf := bytes.NewReader(contentAfter)
		if err := decompress(buf, "./decrypted"); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		// notification
		fmt.Println("File decrypted at decrypted/\nCheers.")

		return
	}
}
