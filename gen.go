package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"sort"
	"time"
)

const (
	contentDir     = "content"
	outputDir      = "posts"
	postDateLayout = "Jan 2, 2006"
)

type Post struct {
	Name, Title string // Always present.
	Date        string // Always present.
	RedirectTo  string `json:"redirect-to"` // "" in case of a regular post.
}

func (post *Post) inputFilename() string {
	return fmt.Sprintf("%s/%s.md", contentDir, post.Name)
}

func (post *Post) outputFilename() string {
	return fmt.Sprintf("%s/%s.html", outputDir, post.Name)
}

func (post *Post) parseDate() time.Time {
	date, err := time.Parse(postDateLayout, post.Date)
	if err != nil {
		log.Fatalf("Cannot parse date in '%s': %v.", post.Name, err)
	}

	return date
}

func main() {
	posts := readPostsMetadata()
	genPostPages(posts)
	genIndexHtml(posts)
}

func readPostsMetadata() []Post {
	postsData, err := os.ReadFile("metadata.json")
	if err != nil {
		log.Fatalf("Cannot read 'metadata.json': %v.", err)
	}

	var posts []Post
	if err := json.Unmarshal(postsData, &posts); err != nil {
		log.Fatal(err)
	}

	return posts
}

func genPostPages(posts []Post) {
	var outputFilenames []string

	for _, post := range posts {
		if post.RedirectTo != "" {
			genRedirectHtml(&post)
		} else {
			invokePandoc(&post)
			outputFilenames = append(outputFilenames, post.outputFilename())
		}
	}

	transformDOM(outputFilenames)
}

func genRedirectHtml(post *Post) {
	htmlFilename := fmt.Sprintf("%s/%s.html", outputDir, post.Name)
	file, err := os.Create(htmlFilename)
	defer file.Close()
	if err != nil {
		log.Fatalf("Cannot create '%s': %v.", htmlFilename, err)
	}

	w := bufio.NewWriter(file)
	defer w.Flush()

	t := template.Must(template.ParseFiles("templates/redirect.tmpl"))

	err = t.ExecuteTemplate(w, "redirect.tmpl", post)
	if err != nil {
		log.Fatal(err)
	}
}

func invokePandoc(post *Post) {
	cmd := exec.Command(
		"pandoc", post.inputFilename(),
		"--output", post.outputFilename(),
		"--metadata", fmt.Sprint("author=Hirrolot's Blog"),
		"--metadata", fmt.Sprintf("title=%s", post.Title),
		"--metadata", fmt.Sprintf("date=%s", post.Date),
		"--standalone",
		"-H", "header.html",
		"--table-of-contents",
		"--citeproc",
		"--css", "../style.css",
		"--include-after-body", "utterances.html",
		"--include-in-header", "post-header.html")

	output, err := cmd.CombinedOutput()
	fmt.Print(string(output))
	if err != nil {
		log.Fatal(err)
	}
}

func transformDOM(outputFilenames []string) {
	args := append([]string{"transform.js"}, outputFilenames...)
	cmd := exec.Command("node", args...)

	output, err := cmd.CombinedOutput()
	defer fmt.Print(string(output))
	if err != nil {
		log.Fatal(err)
	}
}

type IndexTemplate struct {
	OutputDir string
	Posts     []Post
	Contacts  []Contact
}

type Contact struct {
	Link, Img, Description string
}

func genIndexHtml(posts []Post) {
	file, err := os.Create("index.html")
	if err != nil {
		log.Fatalf("Cannot create 'index.html': %v.", err)
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	defer w.Flush()

	t := template.Must(template.ParseFiles("templates/index.tmpl", "header.html"))

	sort.Slice(posts, func(i, j int) bool {
		date_i := posts[i].parseDate()
		date_j := posts[j].parseDate()
		return date_i.After(date_j)
	})

	err = t.ExecuteTemplate(w, "index.tmpl",
		IndexTemplate{OutputDir: outputDir, Posts: posts, Contacts: readContacts()})
	if err != nil {
		log.Fatal(err)
	}
}

func readContacts() []Contact {
	contactsData, err := os.ReadFile("contacts.json")
	if err != nil {
		log.Fatalf("Cannot read 'contacts.json': %v.", err)
	}

	var contacts []Contact
	if err := json.Unmarshal(contactsData, &contacts); err != nil {
		log.Fatal(err)
	}

	return contacts
}
