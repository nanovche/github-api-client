package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
)

const usersURL string = "https://api.github.com/users/${username}"
const reposURL string = "https://api.github.com/users/${username}/repos"
const lanByRepoURL string = "https://api.github.com/repos/${username}/${repo-name}/languages"

type user struct {
	Login       string
	Name        string
	Id          int
	URL         string
	ReposURL    string
	Email       string
	PublicRepos int
	Created_at  string
	Updated_at  string
	Followers   int
	Repos       []repo
}

type repo struct {
	Id         int
	Name       string
	Created_at string
	Updated_at string
	Forks      int
	Languages  map[string]int
}

func main() {

	if len(os.Args) > 1 {

		usernames := readFile()

		var users []user
		for _, name := range usernames {


			var currentUser user

			URL := getUserURL(name)
			data := makeRequest(URL)
			decodeJSON(data, &currentUser)

			URL = getUserReposURL(name)
			data = makeRequest(URL)
			decodeJSON(data, &currentUser.Repos)

			for j, v := range currentUser.Repos {
				URL = getLangByRepoURL(name, v.Name)
				data := makeRequest(URL)
				decodeJSON(data, &currentUser.Repos[j].Languages)
			}
			users  = append(users, currentUser)
		}

		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 10, 8, 0, '\t', 0)
		defer w.Flush()
		if _, err := fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s", "Name", "Repos", "Languages", "Followers" , "Forks", "Activity" ); err != nil {
			log.Println(err)
		}
		if _, err := fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s","----", "-----", "---------", "---------", "-----", "--------"); err != nil {
			log.Println(err)
		}
		for _, u := range users {

			langMap := make(map[string]float32)
			keys := make([]string, 10)


			var valuesLangs []float32
			valuesAct := make([]float32, 1)
			usrAct := make(map[string]float32)
			numForks := 0
			for i, r := range u.Repos {
				m := r.Languages
				for k, v := range m {
					cv := float32(v)
					langMap[k] += cv
					keys = append(keys, k)
					valuesLangs = append(valuesLangs, cv)
				}
				numForks = u.Repos[i].Forks
				usrAct[r.Created_at[:4]] += 1
				usrAct[r.Updated_at[:4]] += 1
			}

			makePercents(valuesLangs, &langMap)
			langslc := sortLangsByValue(langMap)
			for _, v := range usrAct {
				valuesAct = append(valuesAct, v)
			}
			makePercents(valuesAct, &usrAct)
			usrslc := sortActivityByKey(usrAct)

			larger := max(len(langslc), len(usrslc))
			min := min(len(langslc), len(usrslc))

			empty := ""
			for i := 0; i < larger; i++ {

				if i == 0 {
					if _, err := fmt.Fprintf(w, "\n %s\t%d\t%-10s %.2f%%\t%d\t%d\t%-6s %.2f%%\t", u.Login, len(u.Repos), langslc[i].Key, langslc[i].Value,
						u.Followers, numForks,  usrslc[i].Key, usrslc[i].Value); err != nil {
						log.Println(err)
					}
				} else if i < min{
					if _, err := fmt.Fprintf(w, "\n %s\t%s\t%-10s %.2f%%\t%s\t%s\t%-6s %.2f%%\t", empty, empty, langslc[i].Key, langslc[i].Value,empty, empty,  usrslc[i].Key, usrslc[i].Value); err != nil {
						log.Println(err)
					}
				} else if len(langslc) < len(usrslc) {
					if _, err := fmt.Fprintf(w, "\n %s\t%s\t%-10s %s\t%s\t%s\t%-6s %.2f%%\t", empty, empty, empty, empty,empty, empty,  usrslc[i].Key, usrslc[i].Value); err != nil {
						log.Println(err)
					}
				} else if len(langslc) > len(usrslc) {
					if _, err := fmt.Fprintf(w, "\n %s\t%s\t%-10s %.2f%%\t%s\t%s\t%s %s\t", empty, empty, langslc[i].Key, langslc[i].Value,empty, empty,  empty, empty); err != nil {
						log.Println(err)
					}
				}
			}
			if _, err := fmt.Fprintf(w, "\n\n"); err != nil {
				log.Println(err)
			}
		}

	} else {
		fmt.Println("No file specified as command-line argument...")
	}
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

type kv struct {
	Key   string
	Value float32
}

func sortLangsByValue(values map[string]float32) []kv{
	ss := extractMapToSliceOfStructs(values)
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})
	return prepareOutput(ss)
}

func sortActivityByKey(values map[string]float32) []kv{
	ss := extractMapToSliceOfStructs(values)
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Key > ss[j].Key
	})
	return prepareOutput(ss)
}

func extractMapToSliceOfStructs(values map[string]float32) (ss []kv){
	for k, v := range values {
		ss = append(ss, kv{k, v})
	}
	return
}

func prepareOutput(ss []kv) []kv{
	fnslc := make([]kv, 6)
	if len(ss) > 5 {
		copy(fnslc, ss[:5])
		var sum float32
		for i := 5; i < len(ss); i++ {
			sum += ss[i].Value
		}
		fnslc[5] = kv{"others", sum}
		return fnslc
	} else {
		return ss
	}
}

func sum(vals []float32) (sum float32) {
	for i, _ := range vals {
		sum += vals[i]
	}
	return
}

func makePercents(vals []float32, mapping *map[string]float32) {
	sum := sum(vals)
	for k, v := range *mapping {
		(*mapping)[k] = v / sum * 100
	}
}


func decodeJSON(data []byte, v interface{}) {

	if err := json.Unmarshal(data, &v); err != nil {
		log.Fatalf("JSON unmarshaling failed: %s", err)
	}

}

func getUserURL(name string) string {
	return strings.Replace(usersURL, "${username}", name, 1)
}

func getUserReposURL(name string) string {
	return strings.Replace(reposURL, "${username}", name, 1)
}

func getLangByRepoURL(name, repo string) string {
	s1 := strings.Replace(lanByRepoURL, "${username}", name, 1)
	s2 := strings.Replace(s1, "${repo-name}", repo, 1)
	return s2
}


func makeRequest(URL string) []byte {

	if req, err := http.NewRequest("GET", URL, nil); err != nil {
		log.Println(err)
		return nil
	} else{

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("haha") // handle
			log.Fatal(err)
		}
		scanner := bufio.NewScanner(resp.Body)
		var data string
		for scanner.Scan() {
			data = scanner.Text()
		}

		if err := resp.Body.Close(); err != nil {
			fmt.Println(err)
		}

	return []byte(data)
	}
}
func readFile() (usernames []string) {

	file, err := os.Open(os.Args[1])

	if err != nil {
		if _, err := fmt.Fprintf(os.Stderr, "Error: %v", err); err != nil {
			fmt.Println(err)
		}
	}

	input := bufio.NewScanner(file)
	for index := 0; input.Scan(); index++ {
		usernames = append(usernames, input.Text())
	}

	if err := file.Close(); err != nil {
		fmt.Println(err)
	}

	return
}


