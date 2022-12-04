package main

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file.")
	}

	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		println(c.ClientIP())
		c.JSON(200, gin.H{
			"stuats": "not terribly broken (yet)",
		})
	})

	router.POST("/", webhookHandler)

	addr := os.Getenv("GIN_ADDR")
	if len(addr) == 0 {
		addr = "0.0.0.0"
	}
	port := os.Getenv("GIN_PORT")
	if len(port) == 0 {
		port = "8025"
	}
	router.Run(addr + ":" + port)
	log.Println("Started gin web server on: " + addr + ":" + port)
}

func webhookHandler(c *gin.Context) {
	// If hash checking is enabled, do so
	check, err := strconv.ParseBool(os.Getenv("CHECK_GITHUB_HASH"))
	if err != nil { // literally don't care
		check = false
	}

	if check {
		// Get the request body in bytes so we can recreate the sha256 hash
		body, _ := io.ReadAll(c.Request.Body)
		// Replace the request body so it can be used later and I don't have to do anything fancy
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		ghSig := c.Request.Header.Get("X-Hub-Signature-256")
		// If the hash header isn't there, reject
		if len(ghSig) == 0 {
			c.AbortWithStatus(400)
			return
		}

		// Check that the hashes match
		if rtn := hashItOut(body, []byte(ghSig)); !rtn {
			c.AbortWithStatus(401)
			return
		}
	}

	var newReq GithubRequest
	if err := c.BindJSON(&newReq); err != nil {
		return
	}

	splitRef := strings.Split(newReq.Ref, "/")
	branch := splitRef[len(splitRef)-1]
	proj := getProject(newReq.Repository.Name, branch)
	if (reflect.DeepEqual(proj, Project{})) {
		c.Status(204)
		return
	}

	if len(proj.ScriptName) != 0 {
		go execScript(proj.ScriptName)
	} else {
		fmt.Println("Project recognized, but no valid action to take found.")
	}

	c.Status(200) // not technically needed as 200 is default
}

func execScript(scriptName string) {
	cmd := exec.Command("/bin/sh", scriptName)
	stdout, _ := cmd.StdoutPipe()
	cmd.Start()

	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		m := scanner.Text()
		fmt.Println(m)
	}
}

func getProject(repoName string, currBranch string) Project {
	project_file_dir := os.Getenv("PRJ_FILE_DIR")
	if len(project_file_dir) == 0 {
		project_file_dir = "./"
	}
	project_file_name := os.Getenv("PRJ_FILE_NAME")
	if len(project_file_name) == 0 {
		project_file_name = "projects.json"
	}

	file, err := os.ReadFile(project_file_dir + project_file_name)
	if err != nil {
		fmt.Println("Error when opening file: ", err)
	}

	var proj ProjectParent
	err = json.Unmarshal(file, &proj)
	if err != nil {
		fmt.Println("Error during Unmarshal: ", err)
	}

	// Is there a better way to do this? Probably
	// Why does a built in function to find if a value exists in an array not exist?
	// Idk, I got a CS degree but that doesn't mean I'm a computer scientist
	for _, value := range proj.Projects {
		if value.RepoName == repoName { // check if repo names match
			for _, br := range value.AcceptedBranches { // check if branch exists in accepted branches
				if br == currBranch {
					return value // it does, return the curr project struct
				}
			}
		}
	}
	return Project{} // no match, return empty project struct
}

func hashItOut(payloadBody []byte, compHash []byte) bool {
	// Process outlined in the github webhook secret documentation
	// https://docs.github.com/en/developers/webhooks-and-events/webhooks/securing-your-webhooks

	// Create a HMAC hex digest w/ the secret
	secret := os.Getenv("GITHUB_SECRET")
	if len(secret) == 0 {
		log.Fatal("No github secret found with hash checking on.")
	}
	mac := hmac.New(sha256.New, []byte(secret))

	mac.Write(payloadBody)

	expectedMAC := mac.Sum(nil)

	// Re-create the hash with info we have
	val := "sha256=" + hex.EncodeToString(expectedMAC)

	// Compare the hashes, if match then 1 is returned, 0 otherwise
	return subtle.ConstantTimeCompare([]byte(val), compHash) == 1
}

type Project struct {
	RepoName         string   `json:"repoName"`
	AcceptedBranches []string `json:"acceptedBranches"`
	ScriptName       string   `json:"scriptName"`
	CommandList      []string `json:"commandList"`
	Desc             string   `json:"desc"`
}

type ProjectParent struct {
	Projects []Project `json:"projects"`
}

type GithubRequest struct {
	Ref        string `json:"ref,omitempty"`
	Before     string `json:"before,omitempty"`
	After      string `json:"after,omitempty"`
	Repository struct {
		ID       int    `json:"id,omitempty"`
		NodeID   string `json:"node_id,omitempty"`
		Name     string `json:"name,omitempty"`
		FullName string `json:"full_name,omitempty"`
		Private  bool   `json:"private,omitempty"`
		Owner    struct {
			Name              string `json:"name,omitempty"`
			Email             string `json:"email,omitempty"`
			Login             string `json:"login,omitempty"`
			ID                int    `json:"id,omitempty"`
			NodeID            string `json:"node_id,omitempty"`
			AvatarURL         string `json:"avatar_url,omitempty"`
			GravatarID        string `json:"gravatar_id,omitempty"`
			URL               string `json:"url,omitempty"`
			HTMLURL           string `json:"html_url,omitempty"`
			FollowersURL      string `json:"followers_url,omitempty"`
			FollowingURL      string `json:"following_url,omitempty"`
			GistsURL          string `json:"gists_url,omitempty"`
			StarredURL        string `json:"starred_url,omitempty"`
			SubscriptionsURL  string `json:"subscriptions_url,omitempty"`
			OrganizationsURL  string `json:"organizations_url,omitempty"`
			ReposURL          string `json:"repos_url,omitempty"`
			EventsURL         string `json:"events_url,omitempty"`
			ReceivedEventsURL string `json:"received_events_url,omitempty"`
			Type              string `json:"type,omitempty"`
			SiteAdmin         bool   `json:"site_admin,omitempty"`
		} `json:"owner,omitempty"`
		HTMLURL                  string        `json:"html_url,omitempty"`
		Description              string        `json:"description,omitempty"`
		Fork                     bool          `json:"fork,omitempty"`
		URL                      string        `json:"url,omitempty"`
		ForksURL                 string        `json:"forks_url,omitempty"`
		KeysURL                  string        `json:"keys_url,omitempty"`
		CollaboratorsURL         string        `json:"collaborators_url,omitempty"`
		TeamsURL                 string        `json:"teams_url,omitempty"`
		HooksURL                 string        `json:"hooks_url,omitempty"`
		IssueEventsURL           string        `json:"issue_events_url,omitempty"`
		EventsURL                string        `json:"events_url,omitempty"`
		AssigneesURL             string        `json:"assignees_url,omitempty"`
		BranchesURL              string        `json:"branches_url,omitempty"`
		TagsURL                  string        `json:"tags_url,omitempty"`
		BlobsURL                 string        `json:"blobs_url,omitempty"`
		GitTagsURL               string        `json:"git_tags_url,omitempty"`
		GitRefsURL               string        `json:"git_refs_url,omitempty"`
		TreesURL                 string        `json:"trees_url,omitempty"`
		StatusesURL              string        `json:"statuses_url,omitempty"`
		LanguagesURL             string        `json:"languages_url,omitempty"`
		StargazersURL            string        `json:"stargazers_url,omitempty"`
		ContributorsURL          string        `json:"contributors_url,omitempty"`
		SubscribersURL           string        `json:"subscribers_url,omitempty"`
		SubscriptionURL          string        `json:"subscription_url,omitempty"`
		CommitsURL               string        `json:"commits_url,omitempty"`
		GitCommitsURL            string        `json:"git_commits_url,omitempty"`
		CommentsURL              string        `json:"comments_url,omitempty"`
		IssueCommentURL          string        `json:"issue_comment_url,omitempty"`
		ContentsURL              string        `json:"contents_url,omitempty"`
		CompareURL               string        `json:"compare_url,omitempty"`
		MergesURL                string        `json:"merges_url,omitempty"`
		ArchiveURL               string        `json:"archive_url,omitempty"`
		DownloadsURL             string        `json:"downloads_url,omitempty"`
		IssuesURL                string        `json:"issues_url,omitempty"`
		PullsURL                 string        `json:"pulls_url,omitempty"`
		MilestonesURL            string        `json:"milestones_url,omitempty"`
		NotificationsURL         string        `json:"notifications_url,omitempty"`
		LabelsURL                string        `json:"labels_url,omitempty"`
		ReleasesURL              string        `json:"releases_url,omitempty"`
		DeploymentsURL           string        `json:"deployments_url,omitempty"`
		CreatedAt                int           `json:"created_at,omitempty"`
		UpdatedAt                time.Time     `json:"updated_at,omitempty"`
		PushedAt                 int           `json:"pushed_at,omitempty"`
		GitURL                   string        `json:"git_url,omitempty"`
		SSHURL                   string        `json:"ssh_url,omitempty"`
		CloneURL                 string        `json:"clone_url,omitempty"`
		SvnURL                   string        `json:"svn_url,omitempty"`
		Homepage                 interface{}   `json:"homepage,omitempty"`
		Size                     int           `json:"size,omitempty"`
		StargazersCount          int           `json:"stargazers_count,omitempty"`
		WatchersCount            int           `json:"watchers_count,omitempty"`
		Language                 interface{}   `json:"language,omitempty"`
		HasIssues                bool          `json:"has_issues,omitempty"`
		HasProjects              bool          `json:"has_projects,omitempty"`
		HasDownloads             bool          `json:"has_downloads,omitempty"`
		HasWiki                  bool          `json:"has_wiki,omitempty"`
		HasPages                 bool          `json:"has_pages,omitempty"`
		HasDiscussions           bool          `json:"has_discussions,omitempty"`
		ForksCount               int           `json:"forks_count,omitempty"`
		MirrorURL                interface{}   `json:"mirror_url,omitempty"`
		Archived                 bool          `json:"archived,omitempty"`
		Disabled                 bool          `json:"disabled,omitempty"`
		OpenIssuesCount          int           `json:"open_issues_count,omitempty"`
		License                  interface{}   `json:"license,omitempty"`
		AllowForking             bool          `json:"allow_forking,omitempty"`
		IsTemplate               bool          `json:"is_template,omitempty"`
		WebCommitSignoffRequired bool          `json:"web_commit_signoff_required,omitempty"`
		Topics                   []interface{} `json:"topics,omitempty"`
		Visibility               string        `json:"visibility,omitempty"`
		Forks                    int           `json:"forks,omitempty"`
		OpenIssues               int           `json:"open_issues,omitempty"`
		Watchers                 int           `json:"watchers,omitempty"`
		DefaultBranch            string        `json:"default_branch,omitempty"`
		Stargazers               int           `json:"stargazers,omitempty"`
		MasterBranch             string        `json:"master_branch,omitempty"`
	} `json:"repository,omitempty"`
	Pusher struct {
		Name  string `json:"name,omitempty"`
		Email string `json:"email,omitempty"`
	} `json:"pusher,omitempty"`
	Sender struct {
		Login             string `json:"login,omitempty"`
		ID                int    `json:"id,omitempty"`
		NodeID            string `json:"node_id,omitempty"`
		AvatarURL         string `json:"avatar_url,omitempty"`
		GravatarID        string `json:"gravatar_id,omitempty"`
		URL               string `json:"url,omitempty"`
		HTMLURL           string `json:"html_url,omitempty"`
		FollowersURL      string `json:"followers_url,omitempty"`
		FollowingURL      string `json:"following_url,omitempty"`
		GistsURL          string `json:"gists_url,omitempty"`
		StarredURL        string `json:"starred_url,omitempty"`
		SubscriptionsURL  string `json:"subscriptions_url,omitempty"`
		OrganizationsURL  string `json:"organizations_url,omitempty"`
		ReposURL          string `json:"repos_url,omitempty"`
		EventsURL         string `json:"events_url,omitempty"`
		ReceivedEventsURL string `json:"received_events_url,omitempty"`
		Type              string `json:"type,omitempty"`
		SiteAdmin         bool   `json:"site_admin,omitempty"`
	} `json:"sender,omitempty"`
	Created bool        `json:"created,omitempty"`
	Deleted bool        `json:"deleted,omitempty"`
	Forced  bool        `json:"forced,omitempty"`
	BaseRef interface{} `json:"base_ref,omitempty"`
	Compare string      `json:"compare,omitempty"`
	Commits []struct {
		ID        string `json:"id,omitempty"`
		TreeID    string `json:"tree_id,omitempty"`
		Distinct  bool   `json:"distinct,omitempty"`
		Message   string `json:"message,omitempty"`
		Timestamp string `json:"timestamp,omitempty"`
		URL       string `json:"url,omitempty"`
		Author    struct {
			Name     string `json:"name,omitempty"`
			Email    string `json:"email,omitempty"`
			Username string `json:"username,omitempty"`
		} `json:"author,omitempty"`
		Committer struct {
			Name     string `json:"name,omitempty"`
			Email    string `json:"email,omitempty"`
			Username string `json:"username,omitempty"`
		} `json:"committer,omitempty"`
		Added    []string      `json:"added,omitempty"`
		Removed  []interface{} `json:"removed,omitempty"`
		Modified []interface{} `json:"modified,omitempty"`
	} `json:"commits,omitempty"`
	HeadCommit struct {
		ID        string `json:"id,omitempty"`
		TreeID    string `json:"tree_id,omitempty"`
		Distinct  bool   `json:"distinct,omitempty"`
		Message   string `json:"message,omitempty"`
		Timestamp string `json:"timestamp,omitempty"`
		URL       string `json:"url,omitempty"`
		Author    struct {
			Name     string `json:"name,omitempty"`
			Email    string `json:"email,omitempty"`
			Username string `json:"username,omitempty"`
		} `json:"author,omitempty"`
		Committer struct {
			Name     string `json:"name,omitempty"`
			Email    string `json:"email,omitempty"`
			Username string `json:"username,omitempty"`
		} `json:"committer,omitempty"`
		Added    []string      `json:"added,omitempty"`
		Removed  []interface{} `json:"removed,omitempty"`
		Modified []interface{} `json:"modified,omitempty"`
	} `json:"head_commit,omitempty"`
}
