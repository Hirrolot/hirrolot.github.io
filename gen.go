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
	Name       string `json:"name"`                  // Always present.
	Title      string `json:"title"`                 // Always present.
	Date       string `json:"date"`                  // Always present.
	RedirectTo string `json:"redirect-to,omitempty"` // "" in case of a regular post.
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
	if len(os.Args) == 2 && os.Args[1] == "new" {
		newPost()
		return
	}

	posts := readPostsMetadata()
	genPostPages(posts)
	genIndexHtml(posts)
}

func readPostsMetadata() []Post {
	postsData, err := os.ReadFile("metadata/posts.json")
	if err != nil {
		log.Fatalf("Cannot read 'metadata/posts.json': %v.", err)
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
		"--mathjax",
		"-H", "header.html",
		"--table-of-contents",
		"--citeproc",
		"--css", "../style.css",
		"--include-after-body", "giscus.html",
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
	contactsData, err := os.ReadFile("metadata/contacts.json")
	if err != nil {
		log.Fatalf("Cannot read 'metadata/contacts.json': %v.", err)
	}

	var contacts []Contact
	if err := json.Unmarshal(contactsData, &contacts); err != nil {
		log.Fatal(err)
	}

	return contacts
}

func newPost() {
	scanner := bufio.NewScanner((os.Stdin))

	var post Post

	fmt.Print("Title: ")
	if hasMore := scanner.Scan(); !hasMore {
		log.Fatal("Cannot read more lines.")
	}
	post.Title = scanner.Text()

	fmt.Print("Name: ")
	if hasMore := scanner.Scan(); !hasMore {
		log.Fatal("Cannot read more lines.")
	}
	post.Name = scanner.Text()

	fmt.Print("Is it a redirect? (y/n): ")
	if hasMore := scanner.Scan(); !hasMore {
		log.Fatal("Cannot read more lines.")
	}
	isRedirect := scanner.Text()

	switch isRedirect {
	case "y":
		fmt.Print("Redirect to: ")
		if hasMore := scanner.Scan(); !hasMore {
			log.Fatal("Cannot read more lines.")
		}
		post.RedirectTo = scanner.Text()

	case "n":
		fmt.Println("Ok, no redirecting.")

	default:
		log.Fatal("Invalid answer; aborting.")
	}

	post.Date = time.Now().Format(postDateLayout)

	posts := readPostsMetadata()
	posts = append([]Post{post}, posts...)
	postsData, err := json.MarshalIndent(posts, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile("metadata/posts.json", postsData, 0644); err != nil {
		log.Fatal(err)
	}

	if isRedirect == "n" {
		postContent := "<div class=\"introduction\">\n\n\n\n</div>\n"
		if err := os.WriteFile(post.inputFilename(), []byte(postContent), 0644); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("Done.")
}
