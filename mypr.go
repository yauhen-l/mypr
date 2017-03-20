package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/tmc/keyring"
	"github.com/yauhen-l/stash"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/yaml.v2"
)

var debug bool

func trace(msg string, args ...interface{}) {
	if !debug {
		return
	}
	fmt.Printf(msg+"\n", args...)
}

var cfg struct {
	URL        string `yaml:"url"`
	User       string `yaml:"user"`
	Password   string `yaml:"password"`
	UseKeyring bool   `yaml:"useKeyring"`
}

func askPassword(username string) string {
	fmt.Printf("Enter Password(%s): ", username)
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal(err)
	}
	return string(bytePassword)
}

func credentials(username, password string) (string, string) {
	reader := bufio.NewReader(os.Stdin)

	if len(username) == 0 {
		fmt.Print("Enter Username: ")
		username, _ = reader.ReadString('\n')
	}

	if len(password) == 0 {
		var err error

		if cfg.UseKeyring {
			password, err = keyring.Get(cfg.URL, username)
			if err != nil {
				fmt.Printf("Failed to get paswword from keyring due: %v\n", err)

				password = askPassword(username)

				fmt.Printf("\nDo you want to save password in kering(y/n)?")
				answer, _ := reader.ReadString('\n')
				if strings.HasPrefix(answer, "y") {
					err = keyring.Set(cfg.URL, username, password)
					if err != nil {
						fmt.Printf("Failed to save password into keyring due: %v\n", err)
					} else {
						fmt.Println("Password was saved.")
					}
				}
			}
		} else {
			password = askPassword(username)
		}
	}

	return strings.TrimSpace(username), strings.TrimSpace(password)
}

func main() {
	debugPtr := flag.Bool("d", false, "debug output")
	flag.Parse()

	debug = *debugPtr

	home := os.Getenv("HOME")
	cfgPath := home + "/.config/mypr.yaml"

	trace("reading config file: %s", cfgPath)

	data, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		log.Fatalf("failed to read file %q due: %v", cfgPath, err)
	}

	trace("config: \n%s", string(data))

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatalf("bad config: %v", err)
	}

	trace("parsed config: %+v", cfg)

	baseURL, err := url.Parse(cfg.URL)
	if err != nil {
		log.Fatalf("bad URL %q due: %v", cfg.URL, err)
	}

	cfg.User, cfg.Password = credentials(cfg.User, cfg.Password)

	c := stash.NewClient(cfg.User, cfg.Password, baseURL)
	repos, err := c.GetRecentRepositories()
	if err != nil {
		log.Fatal(err)
	}

	commentsCh := make(chan Comment)

	go func() {
		var wg sync.WaitGroup
		for _, r := range repos {
			trace("discover repo: %s/%s", r.Project.Key, r.Slug)
			wg.Add(1)
			go discoverComments(c, r, &wg, commentsCh)
		}
		wg.Wait()
		close(commentsCh)
	}()

	info := Info{
		Comments: make(map[string]map[string]PR),
	}

	for c := range commentsCh {
		task, ok := info.Comments[c.TaskID]
		if !ok {
			task = make(map[string]PR)
		}

		repo, ok := task[c.Repository]
		if !ok {
			repo = PR{
				PullRequest: c.PullRequest,
				Comments:    []stash.Comment{c.Comment},
			}
		} else {
			repo.Comments = append(repo.Comments, c.Comment)
		}

		task[c.Repository] = repo
		info.Comments[c.TaskID] = task
	}

	info.Print()
}

func discoverComments(c stash.Stash, r stash.Repository, wg *sync.WaitGroup, commentsCh chan Comment) {
	defer wg.Done()

	prs, err := c.GetPullRequests(r.Project.Key, r.Slug, "OPEN")
	if err != nil {
		log.Println(err)
		return
	}

	for _, pr := range prs {
		trace("found PR: %s/%s/%s", r.Project.Key, r.Slug, pr.FromRef.DisplayID)
		if pr.Author.User.Slug != cfg.User {
			continue
		}

		commentsCh <- Comment{
			TaskID:      pr.FromRef.DisplayID,
			Repository:  r.Slug,
			PullRequest: pr,
			Comment:     stash.Comment{},
		}

		files, err := c.GetPullRequestChanges(r.Project.Key, r.Slug, pr.ID)
		if err != nil {
			log.Println(err)
			continue
		}

		for _, f := range files {
			wg.Add(1)
			go getComments(c, r, pr, f, wg, commentsCh)
		}
	}
}

func getComments(c stash.Stash, r stash.Repository, pr stash.PullRequest, path string, wg *sync.WaitGroup, commentsCh chan Comment) {
	defer wg.Done()

	comments, err := c.GetComments(r.Project.Key, r.Slug, strconv.Itoa(pr.ID), path)
	if err != nil {
		log.Println(err)
	}
	for _, comment := range comments {
		commentsCh <- Comment{
			TaskID:      pr.FromRef.DisplayID,
			Repository:  r.Slug,
			PullRequest: pr,
			Comment:     comment,
		}
	}
}

func statusColor(text, status string) string {
	color := ""

	switch status {
	case "APPROVED":
		color = "\033[0;32m"
	case "UNAPPROVED":
		color = "\033[0;36m"
	case "NEEDS_WORK":
		color = "\033[0;33m"
	}

	return color + text + "\033[0m"

}

func printComments(indent string, comments []stash.Comment, review map[string]string) {
	for _, c := range comments {
		if c.Text != "" {
			fmt.Printf("%s %s: %s\n", indent, statusColor(c.Author.Name, review[c.Author.Name]), c.Text)
		}
		printComments(indent+"  ", c.Comments, review)
	}
}

type Info struct {
	// JIRA Task ID -> repository -> comments
	Comments map[string]map[string]PR
}

func (i Info) Print() {
	for taskID, repos := range i.Comments {
		fmt.Println(taskID)
		for repo, pr := range repos {
			fmt.Println("\t" + repo)
			pr.Print()
		}
	}
}

type PR struct {
	stash.PullRequest
	Comments []stash.Comment
}

func (pr PR) Print() {
	fmt.Println("\t" + pr.Links.Self[0].Href)

	review := make(map[string]string)
	overall := make(map[string]int)
	for _, r := range pr.Reviewers {
		review[r.User.Name] = r.Status
		overall[r.Status] += 1
	}
	for status, count := range overall {
		fmt.Printf("\t%s: %d\n", statusColor(status, status), count)
	}
	printComments("\t ", pr.Comments, review)
	fmt.Println("")
}

type Comment struct {
	TaskID     string
	Repository string
	stash.PullRequest
	stash.Comment
}
