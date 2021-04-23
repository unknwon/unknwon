package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	log "unknwon.dev/clog/v2"
)

type project struct {
	Icon        string   `yaml:"icon"`
	Name        string   `yaml:"name"`
	Link        string   `yaml:"link"`
	Description string   `yaml:"desc"`
	Tags        []string `yaml:"tags"`
}

type profile struct {
	Projects []project `yaml:"projects"`
}

func main() {
	template := flag.String("template", "README_template.md", "Template path for the README.md")
	output := flag.String("output", "README.md", "Output path for the README.md")
	flag.Parse()

	defer log.Stop()
	err := log.NewConsole()
	if err != nil {
		panic(err)
	}

	profileBytes, err := os.ReadFile("profile.yml")
	if err != nil {
		log.Fatal("Failed to read profile.yml: %v", err)
	}

	var profile profile
	err = yaml.Unmarshal(profileBytes, &profile)
	if err != nil {
		log.Fatal("Failed to unmarshal profile: %v", err)
	}

	readmeBytes, err := os.ReadFile(*template)
	if err != nil {
		log.Fatal("Failed to read README template: %v", err)
	}

	projectsMarkdown := makeProjectMarkdown(profile.Projects)
	readmeBytes = bytes.ReplaceAll(readmeBytes, []byte("{{PROJECTS}}"), []byte(projectsMarkdown))

	err = os.WriteFile(*output, readmeBytes, 0644)
	if err != nil {
		log.Fatal("Failed to write README.md: %v", err)
	}

}

func makeProjectMarkdown(projects []project) string {
	var projectMarkdown string
	for _, project := range projects {
		name := project.Name
		if name == "" {
			name = path.Base(project.Link)
		}

		var tagMarkdown string
		tags := project.Tags
		if len(tags) != 0 {
			tagMarkdown += "/"
			for _, tag := range tags {
				tagMarkdown += fmt.Sprintf(" `%s`", tag)
			}
		}

		var starMarkdown string
		if strings.HasPrefix(project.Link, "https://github.com/") {
			log.Trace("Fetch %q star counts...", name)
			starCount, err := getRepoStarCount(project.Link)
			if err != nil {
				log.Error("Failed to repo's star count: %v", err)
			} else if starCount != 0 {
				starMarkdown = fmt.Sprintf("/ [★%d](%s/stargazers)", starCount, project.Link)
			}
		}

		// - 🔮 [Elaina](https://github.com/wuhan005/Elaina) - Docker-based remote code runner / [★1](https://github.com/wuhan005/Elaina/stargazers) `Docker`
		projectMarkdown += fmt.Sprintf("- %s [%s](%s) - %s %s %s\n",
			project.Icon, name, project.Link, project.Description,
			starMarkdown, tagMarkdown)
	}

	return projectMarkdown
}

func getRepoStarCount(link string) (int64, error) {
	link = strings.ReplaceAll(link, "https://github.com/", "https://api.github.com/repos/")

	resp, err := http.Get(link)
	if err != nil {
		return 0, errors.Wrap(err, "request GitHub API")
	}
	defer resp.Body.Close()

	type repoMeta struct {
		StargazersCount int64 `json:"stargazers_count"`
	}

	var meta repoMeta
	err = json.NewDecoder(resp.Body).Decode(&meta)
	if err != nil {
		return 0, errors.Wrap(err, "unmarshal")
	}
	return meta.StargazersCount, nil
}
