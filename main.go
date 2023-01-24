package main

import (
	"flag"
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const divisor = 3

type columnType int

const ( //indices to determine which list is focused
	repoColumn columnType = iota
	branchColumn
	commitColumn
)

/*  STYLING*/
var (
	columnStyle = lipgloss.NewStyle().
			Padding(1, 2)
	focusedStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			BorderTop(false).
			BorderBottom(false)
)

func main() {
	var dir string
	flag.StringVar(&dir, "d", "", "Search repos of the given directory")
	commitCountPtr := flag.Int("cc", 5, "Search commits of each branch")

	flag.Parse()

	if dir == "" {
		fmt.Println("Please input option d to search repos")
		os.Exit(1)
	}

	file, err := os.Open(dir)
	CheckIfError(err)

	fileInfos, err := file.Readdir(-1)
	CheckIfError(err)

	if len(fileInfos) == 0 {
		fmt.Printf("There's no local repoMap under %v\n", dir)
		os.Exit(1)
	}

	repoMap := map[string]*git.Repository{}
	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			localRepo := fmt.Sprintf("%s\\%s", dir, fileInfo.Name())
			r, err := git.PlainOpen(localRepo)
			if err != nil {
				continue
			}
			repoMap[fileInfo.Name()] = r
		}
	}

	m := New(repoMap, *commitCountPtr)
	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		fmt.Printf("Error: %v", err)
	}
}

type Model struct {
	loaded      bool
	emptyColumn list.Model
	focused     columnType
	repoMap     map[string]*git.Repository
	columns     []list.Model
	commitCount int
}

func New(repoMap map[string]*git.Repository, commitCount int) *Model {
	return &Model{repoMap: repoMap, commitCount: commitCount}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Next() {
	if m.focused == commitColumn {
		m.focused = repoColumn
	} else {
		m.focused++
	}
}

func (m *Model) Prev() {
	if m.focused == repoColumn {
		m.focused = commitColumn
	} else {
		m.focused--
	}
}

func (m *Model) Up() {
	switch m.focused {
	case repoColumn:
		m.ChangeBranches(-1)
		m.ChangeCommits()
	case branchColumn:
		m.ChangeCommits()
	}
}

func (m *Model) Down() {
	switch m.focused {
	case repoColumn:
		m.ChangeBranches(1)
		m.ChangeCommits()
	case branchColumn:
		m.ChangeCommits()
	}
}

func (m *Model) ChangeBranches(delta int) {
	repoColumn := m.columns[repoColumn]
	repoCount := len(repoColumn.Items())
	nextIdx := repoColumn.Cursor() + delta
	if 0 <= nextIdx && nextIdx < repoCount {
		repoItem := repoColumn.Items()[nextIdx]
		m.columns[branchColumn] = repoItem.(Repo).branches
	}
}

func (m *Model) ChangeCommits() {
	if len(m.columns[branchColumn].Items()) > 0 {
		branchCursor := m.columns[branchColumn].Cursor()
		branch := m.columns[branchColumn].Items()[branchCursor].(Branch)
		m.columns[commitColumn] = branch.commits
	} else {
		m.columns[commitColumn] = m.emptyColumn
	}
}

type Repo struct {
	columnType
	title        string
	lastSelected int
	branches     list.Model
}

func (r Repo) FilterValue() string {
	return r.title
}

func (r Repo) Title() string {
	return r.title
}

func (r Repo) Description() string {
	return fmt.Sprintf("%d branch(es)", len(r.branches.Items()))
}

type Branch struct {
	name         string
	lastCommit   string
	lastSelected int
	commits      list.Model
}

func (b Branch) FilterValue() string {
	return b.name
}

func (b Branch) Title() string {
	return b.name
}

func (b Branch) Description() string {
	latestCommit := b.commits.Items()[0].(Commit)
	return fmt.Sprintf("latest commit at %v", latestCommit.when)
}

type Commit struct {
	hash string
	msg  string
	when time.Time
}

func (c Commit) FilterValue() string {
	return c.msg
}

func (c Commit) Title() string {
	return c.msg
}

func (c Commit) Description() string {
	return fmt.Sprintf("%s", c.msg)
}

func (m *Model) initLists(width, height int) {
	defaultList := NewListModel(width, height)
	m.emptyColumn = defaultList
	m.columns = []list.Model{defaultList, defaultList, defaultList}

	m.columns[repoColumn].Title = "Repository"
	var repoItems []list.Item
	for dirName, repo := range m.repoMap {
		var branchItems []list.Item
		branch, err := repo.Branches()
		CheckIfError(err)
		err = branch.ForEach(func(br *plumbing.Reference) error {
			branchPrefix := "refs/heads/"
			name := strings.TrimPrefix(br.Name().String(), branchPrefix)

			b := plumbing.NewBranchReferenceName(name)
			CheckIfError(err)
			ref, err := repo.Reference(b, true)

			log, err := repo.Log(&git.LogOptions{
				From:  ref.Hash(),
				Order: git.LogOrderCommitterTime,
			})
			var commitItems []list.Item
			i := 0
			err = log.ForEach(func(c *object.Commit) error {
				if i < 5 {
					commitItems = append(commitItems, Commit{
						hash: c.Hash.String(),
						msg:  c.Message,
						when: c.Committer.When,
					})
					i++
					return nil
				}
				return nil
			})
			CheckIfError(err)
			commits := NewListModel(width, height)
			commits.Title = "Commit"
			commits.SetItems(commitItems)
			branchItems = append(branchItems, Branch{
				name:    name,
				commits: commits,
			})
			return nil
		})
		CheckIfError(err)
		branches := NewListModel(width, height)
		branches.Title = "Branch"
		branches.SetItems(branchItems)
		repoItems = append(repoItems, Repo{title: dirName, branches: branches})
	}

	m.columns[repoColumn].SetItems(repoItems)
	repoItem := m.columns[repoColumn].Items()[0]
	repo := repoItem.(Repo)
	m.columns[branchColumn] = repo.branches
	if len(repo.branches.Items()) > 0 {
		branchItem := m.columns[branchColumn].Items()[0]
		branch := branchItem.(Branch)
		m.columns[commitColumn] = branch.commits
	}
}

func NewListModel(width, height int) list.Model {
	defaultList := list.New([]list.Item{}, list.NewDefaultDelegate(), width/divisor, height)
	defaultList.SetShowHelp(false)
	return defaultList
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.loaded {
			m.initLists(msg.Width, msg.Height)
			m.loaded = true
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "left", "h":
			m.Prev()
		case "up":
			m.Up()
		case "down":
			m.Down()
		case "right", "l":
			m.Next()
		}
	}
	var cmd tea.Cmd
	m.columns[m.focused], cmd = m.columns[m.focused].Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.loaded {
		repoView := m.columns[repoColumn].View()
		branchView := m.columns[branchColumn].View()
		commitView := m.columns[commitColumn].View()

		switch m.focused {
		case repoColumn:
			return lipgloss.JoinHorizontal(
				lipgloss.Left,
				focusedStyle.Render(repoView),
				columnStyle.Render(branchView),
				columnStyle.Render(commitView),
			)
		case branchColumn:
			return lipgloss.JoinHorizontal(
				lipgloss.Left,
				columnStyle.Render(repoView),
				focusedStyle.Render(branchView),
				columnStyle.Render(commitView),
			)
		case commitColumn:
			return lipgloss.JoinHorizontal(
				lipgloss.Left,
				columnStyle.Render(repoView),
				columnStyle.Render(branchView),
				focusedStyle.Render(commitView),
			)
		default:
			return lipgloss.JoinHorizontal(
				lipgloss.Left,
				focusedStyle.Render(repoView),
				columnStyle.Render(branchView),
				columnStyle.Render(commitView),
			)
		}
	} else {
		return "loading..."
	}
}

func CheckIfError(err error) {
	if err == nil {
		return
	}

	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", fmt.Sprintf("error: %s", err))
	os.Exit(1)
}
