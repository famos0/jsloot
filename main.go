package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type headers []string

// OUTPUT COLORS

var colorReset string = "\033[0m"
var colorRed string = "\033[31m"
var colorGreen string = "\033[32m"
var colorYellow string = "\033[33m"
var colorBlue string = "\033[34m"
var colorPurple string = "\033[35m"
var colorCyan string = "\033[36m"
var colorWhite string = "\033[37m"

// REGEX PATTERNS
//
// URL Regex from https://github.com/GerbenJavado/LinkFinder/blob/master/linkfinder.py

var url_regex1 string = "((?:[a-zA-Z]{1,10}://|//)[^\"'/]{1,}\\.[a-zA-Z]{2,}[^\"']{0,})"                                  // Basic URLS
var url_regex2 string = "((?:/|\\.\\./|\\./)[^\"'><,;| *()(%%$^/\\\\\\[\\]][^\"'><,;|()]{1,})"                            // Path that start with ../, / or ./
var url_regex3 string = "([a-zA-Z0-9_\\-/]{1,}/[a-zA-Z0-9_\\-/]{1,}\\.(?:[a-zA-Z]{1,4}|action)(?:[\\?|#][^\"|']{0,}|))"   // Relative endpoint with extension and ? or # parameters
var url_regex4 string = "([a-zA-Z0-9_\\-/]{1,}/[a-zA-Z0-9_\\-/]{3,}(?:[\\?|#][^\"|']{0,}|))"                              // Relative endpoints without extensions but with parameters
var url_regex5 string = "([a-zA-Z0-9_\\-]{1,}\\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml)(?:[\\?|#][^\"|']{0,}|))" // Filenames
var url_regex_full string = url_regex1 + "|" + url_regex2 + "|" + url_regex3 + "|" + url_regex4 + "|" + url_regex5
var aws_keys_regex string = "(AKIA|A3T|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{12,}"
var b64_regex string = "(eyJ|YTo|Tzo|PD[89]|aHR0cHM6L|aHR0cDo|rO0)[%a-zA-Z0-9+/]+={0,2}"
var ip_regex string = "(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])"
var quote_regex string = "(?:\"|')"

func isError(err error) bool {
	if err != nil {
		fmt.Println(err.Error())
	}
	return (err != nil)
}

func contentParser(content []byte, pattern string) [][]byte {
	re, err := regexp.Compile(pattern)
	if isError(err) {
		os.Exit(1)
	}
	return re.FindAll(content, -1)
}

func testMatch(regex string, target []byte) bool {
	re := regexp.MustCompile(regex)
	return re.Match(target)
}

func colorString(color string, output string) string {
	return "[" + color + output + colorReset + "] "
}

func showMatchType(target []byte) (output string) {
	if testMatch(url_regex1, target) {
		output = colorString(colorCyan, "URL")
	} else if testMatch(url_regex5, target) {
		output = colorString(colorCyan, "File")
	} else if testMatch(url_regex2, target) {
		output = colorString(colorCyan, "Path")
	} else if testMatch(url_regex3, target) {
		output = colorString(colorCyan, "Endpoint")
	} else if testMatch(url_regex4, target) {
		output = colorString(colorCyan, "Endpoint")
	} else if testMatch(aws_keys_regex, target) {
		output = colorString(colorYellow, "AWS-Key")
	} else if testMatch(b64_regex, target) {
		output = colorString(colorBlue, "Base64")
	} else if testMatch(ip_regex, target) {
		output = colorString(colorGreen, "IPv4")
	} else {
		output = colorString(colorPurple, "Custom")
	}
	return
}

func writeContentToFile(file *os.File, content string) {
	file.WriteString(content)
}

func printResults(filename string, result [][]byte, verbose bool, file *os.File) {

	for _, element := range result {
		var output string
		if verbose {
			output += showMatchType(element)
		}
		if filename != "" {
			output += filename + ":\t" + string(element)
		} else {
			output += string(element)
		}
		output += "\n"
		fmt.Print(output)
		if file != nil {
			file.WriteString(output)
		}
	}
}

func checkReponseContentType(response *http.Response, content_type string) bool {
	content_type_in_response := response.Header.Get("content-type")
	return strings.Contains(content_type_in_response, content_type)
}

func isResponseJavaScript(response *http.Response) bool {
	return checkReponseContentType(response, "javascript")
}

func isResponseHTML(response *http.Response) bool {
	return checkReponseContentType(response, "html")
}

func getSrcFromTags(tokenizer html.Token) (src string) {
	for _, attribute := range tokenizer.Attr {
		if attribute.Key == "src" {
			src = attribute.Val
		}
	}
	return
}

func getJsURLsFromHTML(response *http.Response) []string {

	var urls []string
	tokenizer := html.NewTokenizer(response.Body)
	site := response.Request.URL
	for {
		tt := tokenizer.Next()
		t := tokenizer.Token()
		err := tokenizer.Err()

		if err == io.EOF {
			break
		}

		switch tt {
		case html.ErrorToken:
			continue
		case html.StartTagToken:
			isAnchor := t.Data == "script"
			if !isAnchor {
				continue
			}

			u, err := url.QueryUnescape(getSrcFromTags(t))
			if isError(err) {
				continue
			}

			ur, err := url.Parse(u)
			if isError(err) {
				continue
			}

			u = site.ResolveReference(ur).String()

			if u == "" {
				continue
			}

			hasProto := strings.Index(u, "http") == 0
			if hasProto {
				fmt.Println(u)
				urls = append(urls, u)
			}
		}
	}
	return urls
}

func setHostHeaderIfExists(headers []string) (host string) {
	for _, header := range headers {
		h := strings.Split(header, ":")
		if len(h) != 2 {
			fmt.Printf("Error in headers declaration: %s\n", header)
		}
		if h[0] == "Host" {
			host = h[1]
		}
	}
	return
}

func createClient(proxy string, not_check_cert bool) *http.Client {

	var proxyClient func(*http.Request) (*url.URL, error)
	if proxy == "" {
		proxyClient = http.ProxyFromEnvironment
	} else {
		tmp, _ := url.Parse(proxy)
		proxyClient = http.ProxyURL(tmp)
	}

	transport := &http.Transport{
		Proxy:               proxyClient,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: not_check_cert},
	}

	client := &http.Client{
		Transport: transport,
	}

	return client
}

func createRequest(requestURL string, host string, cookies string, headers []string) *http.Request {

	req, err := http.NewRequest("GET", requestURL, nil)
	if isError(err) {
		os.Exit(1)
	}

	if cookies != "" {
		req.Header.Set("Cookie", cookies)
	}

	if host != "" {
		req.Host = host
	}

	for _, header := range headers {
		h := strings.Split(header, ":")
		req.Header.Set(h[0], h[1])
	}

	return req
}

func getResponseFromURL(url string, proxy string, not_check_cert bool, host string, cookies string, headers []string) *http.Response {
	client := createClient(proxy, not_check_cert)
	req := createRequest(url, host, cookies, headers)
	response, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return response
}

func getContentFromResponse(response *http.Response) []byte {
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return body
}

func getContentFromFile(file string) []byte {
	content, err := ioutil.ReadFile(file)
	if isError(err) {
		return nil
	}

	return content
}

func buildRegex(grep string, custom_expression string) (regex string) {
	if custom_expression != "" {
		regex += custom_expression
	}

	if strings.Contains(grep, "5") {
		return
	} else if grep == "" {
		grep = "1,2,3,4"
	}

	g := strings.Split(grep, ",")

	if len(g) > 0 {
		if custom_expression != "" {
			regex += "|"
		}

		regex += quote_regex + "("
		for index, element := range g {
			switch string(element) {
			case "1":
				regex += url_regex_full
			case "2":
				regex += aws_keys_regex
			case "3":
				regex += b64_regex
			case "4":
				regex += ip_regex
			default:
				fmt.Printf("No such grep case: %s\n", string(element))
				os.Exit(1)
			}
			if index+1 < len(g) {
				regex += "|"
			}
		}
		regex += ")" + quote_regex
	}

	return
}

func lootJS(content []byte, regex string, filename string, show_filename, verbose bool, output_file *os.File) {
	result := contentParser(content, regex)
	if !show_filename {
		filename = ""
	}
	printResults(filename, result, verbose, output_file)
}

func lootJSOnURL(url string, regex string, show_matching_location bool, verbose bool, proxy string, not_check_cert bool, host string, cookies string, headers []string, output_file *os.File) {
	var content []byte
	response := getResponseFromURL(url, proxy, not_check_cert, host, cookies, headers)
	if response != nil {
		if isResponseJavaScript(response) {
			content = getContentFromResponse(response)
			lootJS(content, regex, url, show_matching_location, verbose, output_file)
		} else if isResponseHTML(response) {
			urls := getJsURLsFromHTML(response)
			for _, url := range urls {
				response := getResponseFromURL(url, proxy, not_check_cert, host, cookies, headers)
				content = getContentFromResponse(response)
				lootJS(content, regex, url, show_matching_location, verbose, output_file)
			}
		} else {
			return
		}
	}
}

func lootJSOnFile(file_list []string, regex string, show_matching_location bool, verbose bool, output_file *os.File) {
	var content []byte
	for _, file := range file_list {
		content = getContentFromFile(file)
		lootJS(content, regex, file, show_matching_location, verbose, output_file)
	}
}

func listFilesInDirRec(dirname string) (files_in_dir []string) {
	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	for _, file := range files {
		if isDir(file.Name()) {
			for _, filerec := range listFilesInDirRec(file.Name()) {
				files_in_dir = append(files_in_dir, filerec)
			}
		} else {
			files_in_dir = append(files_in_dir, file.Name())
		}
	}
	return
}

func listFilesInDir(dirname string) (files_in_dir []string) {
	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	for _, file := range files {
		if !isDir(file.Name()) {
			files_in_dir = append(files_in_dir, file.Name())
		}
	}
	return
}

func isDir(dir string) bool {
	fi, err := os.Stat(dir)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return fi.IsDir()
}

func showHelper() {
	helper := []string{
		"JSLoot by zblurx",
		"",
		"Looting URLs, IPv4 addresses, base64 encoded stuff and aws-keys from JavaScript",
		"",
		"-- WHERE TO LOOT ? -- ",
		" -u, --url <url>\t\tLoot from on the URL",
		" -f, --file <path>\t\tLoot from a local file",
		" -d, --directory <path>\t\tLoot from a directory",
		" -r, --recurse <path>\t\tTo combine with -d option. Loot recursively",
		" -s, --stdin\t\t\tLoot from URLs given by Stdin",
		"",
		"-- HOW TO LOOT ? -- ",
		" -H, --header <header>\t\tSpecify header. Can be used multiple times",
		" -c, --cookies <cookies>\tSpecify cookies",
		" -x, --proxy <proxy>\t\tSpecify proxy",
		" -k, --insecure\t\t\tAllow insecure server connections when using SSL",
		"",
		"-- WHAT TO LOOT ? -- ",
		" -e, --regexp <PATTERNS>\tLoot with a custom pattern",
		" -g, --grep-pattern <1,2,...>\tWhen specified, custom the looting patterns :",
		"\t\t\t\t- 1 : Looting URLs",
		"\t\t\t\t- 2 : Looting AWS Keys",
		"\t\t\t\t- 3 : Looting Base64 artifacts",
		"\t\t\t\t- 4 : Looting IPv4 addresses",
		"\t\t\t\t- 5 : Looting nothing on the default patterns",
		"\t\t\t\t      Can be used in complement with -e",
		"",
		"-- SHOW THE LOOT -- ",
		" -o, --output-file <path>\tWrite down result to file",
		" -w, --with-filename\t\tShow filename/URL of loot location",
		" -v, --verbose\t\t\tPrint with detailed output and colors",
	}

	fmt.Println(strings.Join(helper, "\n"))
}

func init() {
	flag.Usage = func() {
		showHelper()
	}
}

func (i *headers) String() string {
	var rep string
	for _, e := range *i {
		rep += e
	}
	return rep
}

func (i *headers) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {

	var url string
	flag.StringVar(&url, "url", "", "")
	flag.StringVar(&url, "u", "", "")

	var file string
	flag.StringVar(&file, "file", "", "")
	flag.StringVar(&file, "f", "", "")

	var dir string
	flag.StringVar(&dir, "directory", "", "")
	flag.StringVar(&dir, "d", "", "")

	var recurse bool
	flag.BoolVar(&recurse, "recurse", false, "")
	flag.BoolVar(&recurse, "r", false, "")

	var stdin bool
	flag.BoolVar(&stdin, "stdin", false, "")
	flag.BoolVar(&stdin, "s", false, "")

	var grep string
	flag.StringVar(&grep, "grep-pattern", "", "")
	flag.StringVar(&grep, "g", "", "")

	var custom_regex string
	flag.StringVar(&custom_regex, "regexp", "", "")
	flag.StringVar(&custom_regex, "e", "", "")

	var headers headers
	flag.Var(&headers, "header", "")
	flag.Var(&headers, "H", "")

	var proxy string
	flag.StringVar(&proxy, "proxy", "", "")
	flag.StringVar(&proxy, "x", "", "")

	var cookies string
	flag.StringVar(&cookies, "cookies", "", "")
	flag.StringVar(&cookies, "c", "", "")

	var not_check_cert bool
	flag.BoolVar(&not_check_cert, "insecure", false, "")
	flag.BoolVar(&not_check_cert, "k", false, "")

	var show_matching_location bool
	flag.BoolVar(&show_matching_location, "with-filename", false, "")
	flag.BoolVar(&show_matching_location, "w", false, "")

	var output string
	flag.StringVar(&output, "output-file", "", "")
	flag.StringVar(&output, "o", "", "")

	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "")
	flag.BoolVar(&verbose, "v", false, "")

	flag.Parse()

	regex := buildRegex(grep, custom_regex)

	host := setHostHeaderIfExists(headers)

	if regex != "" {

		var output_file *os.File
		if output != "" {
			var err error
			output_file, err = os.Create(output)
			if isError(err) {
				os.Exit(1)
			}

			defer output_file.Close()
		}

		if url != "" {
			lootJSOnURL(url, regex, show_matching_location, verbose, proxy, not_check_cert, host, cookies, headers, output_file)

		} else if file != "" {
			lootJSOnFile([]string{file}, regex, show_matching_location, verbose, output_file)
		} else if dir != "" {
			if !isDir(dir) {
				return
			}
			var file_list []string
			if recurse {
				file_list = listFilesInDirRec(dir)
			} else {
				file_list = listFilesInDir(dir)
			}

			lootJSOnFile(file_list, regex, show_matching_location, verbose, output_file)
		} else if stdin {
			sc := bufio.NewScanner(os.Stdin)
			for sc.Scan() {
				url = sc.Text()
				lootJSOnURL(url, regex, show_matching_location, verbose, proxy, not_check_cert, host, cookies, headers, output_file)
			}
		} else {
			showHelper()
			return
		}
	}
}
