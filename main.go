package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"os"
	"strings"
)

const divisor = 4

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
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

func main() {
	dir := "C:\\daehan\\development\\projects"
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

	m := New(repoMap)
	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		fmt.Printf("Error: %v", err)
	}
}

type Model struct {
	loaded  bool
	focused columnType
	repoMap map[string]*git.Repository
	columns []list.Model
}

func New(repoMap map[string]*git.Repository) *Model {
	return &Model{repoMap: repoMap}
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

func (m *Model) ChangeColumn() {
	switch m.focused {
	case repoColumn:
		repoItem := m.columns[repoColumn].SelectedItem()
		m.columns[branchColumn] = repoItem.(Repo).branches
	}
}

type Repo struct {
	columnType
	title    string
	branches list.Model
}

func (r Repo) FilterValue() string {
	return r.title
}

func (r Repo) Title() string {
	return r.title
}

func (r Repo) Description() string {
	return fmt.Sprintf("%d branchColumn(es)", len(r.branches.Items()))
}

type Branch struct {
	name       string
	lastCommit string
	commits    list.Model
}

func (b Branch) FilterValue() string {
	return b.name
}

func (b Branch) Title() string {
	return b.name
}

func (b Branch) Description() string {
	return "branchColumn description"
}

type Commit struct {
	hash string
	msg  string
	time string
}

func (c Commit) FilterValue() string {
	return c.msg
}

func (c Commit) Title() string {
	return c.msg
}

func (c Commit) Description() string {
	return "commitColumn description"
}

func (m *Model) initLists(width, height int) {
	defaultList := NewListModel(width, height)
	m.columns = []list.Model{defaultList, defaultList, defaultList}

	m.columns[repoColumn].Title = "Repository"
	var repoItems []list.Item
	for dirName, repo := range m.repoMap {
		var branchItems []list.Item
		b, err := repo.Branches()
		CheckIfError(err)
		err = b.ForEach(func(br *plumbing.Reference) error {
			// Todo: add 10 commits
			branchPrefix := "refs/heads/"
			name := strings.TrimPrefix(br.Name().String(), branchPrefix)
			branchItems = append(branchItems, Branch{
				name: name,
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
	m.columns[repoColumn].Select(0)
	repo := m.columns[repoColumn].SelectedItem().(Repo)
	m.columns[branchColumn] = repo.branches
}

func NewListModel(width, height int) list.Model {
	defaultList := list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)
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
			m.ChangeColumn()
		case "down":
			m.ChangeColumn()
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
		// Todo: Add commits
		//branchItem := repoColumn.branchView.SelectedItem()

		switch m.focused {
		case repoColumn:
			return lipgloss.JoinHorizontal(
				lipgloss.Left,
				focusedStyle.Render(repoView),
				columnStyle.Render(branchView),
			)
		case branchColumn:
			return lipgloss.JoinHorizontal(
				lipgloss.Left,
				columnStyle.Render(repoView),
				focusedStyle.Render(branchView),
			)
		default:
			return lipgloss.JoinHorizontal(
				lipgloss.Left,
				focusedStyle.Render(repoView),
				columnStyle.Render(branchView),
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
