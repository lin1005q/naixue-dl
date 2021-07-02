package main

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var url string
var fileName string
var header = http.Header{
	"Referer":    []string{"https://e.naixuejiaoyu.com/"},
	"User-Agent": []string{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36"},
}
var client = &http.Client{}

func getBaseUrl(url string) (baseUrl string) {
	ss := strings.Split(url, "/")
	s := ss[len(ss)-1]
	return strings.Replace(url, s, "", -1)
}

func m3u8(url string) (flag bool, err error) {
	m3u8FileBytes := getRequestBody(url)
	m3u8FileString := string(m3u8FileBytes)

	if !strings.Contains(m3u8FileString, "#EXTM3U") {
		return false, errors.New("这不是一个m3u8的视频链接！")
	}
	if !strings.Contains(m3u8FileString, "EXT-X-KEY") {
		return false, errors.New("没有加密")
	}

	regExtXKey, _ := regexp.Compile(`#EXT-X-KEY:(.*)\n`)
	match := regExtXKey.FindAllString(m3u8FileString, 1)
	if len(match) != 1 {
		return false, errors.New("获取EXT-X-KEY失败")
	}

	key, _ := regexp.Compile(`URI="(.*)"`)
	findKey := key.FindStringSubmatch(match[0])

	keyContent := getRequestBody(findKey[1])

	//得到每一个ts视频链接
	//https://my.oschina.net/lemos/blog/1217828
	tsUrlListKey, _ := regexp.Compile("EXTINF:(.*),\n(.*)\n#")
	tsUrlOriginList := tsUrlListKey.FindAllStringSubmatch(m3u8FileString, -1)
	baseUrl := getBaseUrl(url)
	f := createFile(fileName)
	var length = len(tsUrlOriginList)
	for index := range tsUrlOriginList {
		s := tsUrlOriginList[index][2]
		encryptedBody := getRequestBody(baseUrl + s)
		decryptedBody := AesDecryptCBC(encryptedBody, keyContent)
		writeFile(f, decryptedBody)
		var i = index + 1
		var equalSignNumber = i * 100 / length / 2
		h := strings.Repeat("=", equalSignNumber) + strings.Repeat(" ", 50-equalSignNumber)
		fmt.Printf("\r%d%%[%s] %d/%d", i*100/length, h, i, length)
	}
	closeFile(f)
	return true, nil
}

func checkFileIsExist(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return true
}

func createFile(fileName string) (ff *os.File) {
	var f *os.File
	if checkFileIsExist(fileName) { //如果文件存在
		f, _ = os.OpenFile(fileName, os.O_APPEND, 0666) //打开文件
		fmt.Println("文件存在")
	} else {
		f, _ = os.Create(fileName) //创建文件
		fmt.Println("文件不存在")
	}
	return f
}

func writeFile(f *os.File, content []byte) {
	_, err1 := f.Write(content) //写入文件(字符串)
	if err1 != nil {
		panic(err1)
	}
	//fmt.Printf("写入 %d 个字节n\n", n)
}

func closeFile(f *os.File) {
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {

		}
	}(f)
}

func AesDecryptCBC(encrypted []byte, key []byte) (decrypted []byte) {
	block, _ := aes.NewCipher(key)                              // 分组秘钥
	blockSize := block.BlockSize()                              // 获取秘钥块的长度
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize]) // 加密模式
	decrypted = make([]byte, len(encrypted))                    // 创建数组
	blockMode.CryptBlocks(decrypted, encrypted)                 // 解密
	decrypted = pkcs5UnPadding(decrypted)                       // 去除补全码
	return decrypted
}
func pkcs5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func getRequestBody(url string) (bytes []byte) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header = header
	resp, _ := client.Do(req)
	body, _ := ioutil.ReadAll(resp.Body)
	return body
}

func init() {
	flag.StringVar(&url, "url", "", "The ts file url.")
	flag.StringVar(&fileName, "fileName", "", "The ts file name.")
}

// url := "https://1252524126.vod2.myqcloud.com/9764a7a5vodtransgzp1252524126/d2fd68315285890819227516851/drm/v.f146750.m3u8"
func main() {
	flag.Parse()
	if len(url) == 0 {
		fmt.Println("errorMsg is: url is nil")
		os.Exit(1)
	}
	if len(fileName) == 0 {
		ss := strings.Split(url, "/")
		m3u8FileName := ss[len(ss)-1]
		fileName = strings.Replace(m3u8FileName, ".m3u8", ".ts", -1)
		fmt.Println("fileName is use default ", fileName)
	}
	if checkFileIsExist(fileName) { //如果文件存在
		fmt.Println("文件存在")
		os.Exit(1)
	}
	fmt.Println("视频下载开始...")
	_, err := m3u8(url)
	if err != nil {
		fmt.Println("视频下载失败！", err)
	} else {
		fmt.Println("\n视频下载完成！")
	}
}
