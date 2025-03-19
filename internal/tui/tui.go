package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ctx  context.Context
	spnr spinner.Model

	// App Dimensions
	windowWidth             int
	windowHeight            int
	mainHeaderHeight        int
	mainFooterHeight        int
	viewportDvdrHeight      int
	verticalMarginHeight    int
	verticalLeftForMainView int

	menuTabs = []menuTab{tabMainPage, tabOrgs, tabUsers}
)

type menuTab string

const (
	tabMainPage menuTab = "M | Main Page"
	tabOrgs     menuTab = "O | Organizations"
	tabUsers    menuTab = "U | Users"
)

type RootModel struct {
	mainDir         string
	client          *migration.Client
	submodels       *submodels
	currentModel    tea.Model
	data            *MigrationData
	activeTab       menuTab
	viewport        viewport.Model
	ready           bool
	quitting        bool
	scrollOverride  bool
	scrollCountDown bool
}

type submodels struct {
	mainPage      tea.Model
	orgMigration  tea.Model
	userMigration tea.Model
}

type MigrationData struct {
	Output strings.Builder        `json:"output"`
	Orgs   []*orgMigrationDetails `json:"orgs"`
}

type orgMigrationDetails struct {
	ZendeskOrg   *zendesk.Organization `json:"zendesk_org"`
	PsaOrg       *psa.Company          `json:"psa_org"`
	OrgMigErrors []error               `json:"org_migration_errors"`

	Tag        *tagDetails `json:"zendesk_tag"`
	HasTickets bool        `json:"has_tickets"`

	ReadyUsers      bool                    `json:"ready_users"`
	UserMigSelected bool                    `json:"user_migration_selected"`
	UsersToMigrate  []*userMigrationDetails `json:"users_to_migrate"`
	UserMigErrors   []error                 `json:"user_migration_errors"`
	UserMigDone     bool                    `json:"user_migration_done"`
	// TODO: ticketsToMigrate []*ticketMigrationDetails
}

type userMigrationDetails struct {
	ZendeskUser *zendesk.User `json:"zendesk_user"`
	PsaContact  *psa.Contact  `json:"psa_contact"`
	PsaCompany  *psa.Company
	Migrated    bool `json:"migrated"`
}

type timeConversionDetails struct {
	startString string
	endString   string

	// fallback time strings, in case main entry is a blank string
	startFallback string
	endFallback   string
}

type switchModelMsg tea.Model

func NewModel(cx context.Context, client *migration.Client, mainDir string, importFile bool) (*RootModel, error) {
	ctx = cx

	spnr = spinner.New()
	spnr.Spinner = spinner.Ellipsis
	spnr.Style = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "248"})

	data := &MigrationData{}
	if importFile {
		var err error
		path := filepath.Join(mainDir, "migration_data.json")
		data, err = importJsonFile(path)
		if err != nil {
			return nil, fmt.Errorf("importing json file: %w", err)
		}
		slog.Info("imported file from JSON")
	}

	mm := newMainMenuModel(client, data)

	sm := &submodels{
		mainPage:      newMainMenuModel(client, data),
		orgMigration:  newOrgMigrationModel(client, data),
		userMigration: newUserMigrationModel(client, data),
	}

	return &RootModel{
		mainDir:      mainDir,
		client:       client,
		submodels:    sm,
		currentModel: mm,
		data:         data,
		activeTab:    tabMainPage,
	}, nil
}

func (m *RootModel) Init() tea.Cmd {
	return tea.Batch(
		spnr.Tick,
	)
}

func (m *RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		slog.Debug("got WindowSizeMsg")
		return m, m.calculateDimensions(msg.Width, msg.Height)

	case tea.KeyMsg:

		switch msg.String() {
		case "esc":
			m.quitting = true
			cmds = append(cmds, tea.Quit)
		case "c":
			cmds = append(cmds, m.copyToClipboard(m.data.Output.String()))
		case "j":
			cmds = append(cmds, m.writeDataToFile())
		case "m":
			if m.activeTab != tabMainPage {
				m.activeTab = tabMainPage
				cmds = append(cmds,
					switchModel(m.submodels.mainPage))
			}
		case "o":
			slog.Debug("chose org migration model", "totalOrgs", len(m.data.Orgs))
			// TODO: figure out why resize happens twice on double o press
			if m.activeTab != tabOrgs {
				m.activeTab = tabOrgs
				cmds = append(cmds,
					switchModel(m.submodels.orgMigration))
			}
		case " ":
			if m.currentModel == m.submodels.orgMigration && m.submodels.orgMigration.(*orgMigrationModel).status == awaitingStart {
				slog.Debug("org checker: user pressed space to start")
				return m, switchOrgMigStatus(gettingTags)
			}

		case "u":
			slog.Debug("chose user migration model", "totalOrgs", len(m.data.Orgs))
			if m.activeTab != tabUsers {
				m.activeTab = tabUsers
				cmds = append(cmds, switchModel(m.submodels.userMigration))

				if m.submodels.userMigration.(*userMigrationModel).status != pickingOrgs {
					cmds = append(cmds, switchUserMigStatus(pickingOrgs))
				}

				return m, tea.Sequence(cmds...)
			}
		}

	case tea.MouseMsg:
		// override automatic scrolling if user scrolls up or down so they can read output
		// if they scroll back to the bottom, resume automatic scrolling to the bottom (see
		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			m.scrollOverride = true
		}

	case switchModelMsg:
		slog.Debug("received new model via switchModelMsg", "model", msg)
		m.currentModel = msg
	}

	spnr, cmd = spnr.Update(msg)
	cmds = append(cmds, cmd)

	m.submodels.mainPage, cmd = m.submodels.mainPage.Update(msg)
	cmds = append(cmds, cmd)

	m.submodels.orgMigration, cmd = m.submodels.orgMigration.Update(msg)
	cmds = append(cmds, cmd)

	m.submodels.userMigration, cmd = m.submodels.userMigration.Update(msg)
	cmds = append(cmds, cmd)

	if m.ready {
		m.viewport.SetContent(m.data.Output.String())
		m.setAutoScrollBehavior()
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *RootModel) View() string {
	if m.quitting {
		return ""
	}

	if !m.ready {
		return runSpinner("Initializing")
	}

	mainView := lipgloss.NewStyle().
		Width(windowWidth).
		Height(verticalLeftForMainView).
		PaddingLeft(1).
		Render(m.currentModel.View())

	views := []string{menuBar(menuTabs, m.activeTab), mainView, viewportDivider(), m.viewport.View(), appFooter()}

	return lipgloss.JoinVertical(lipgloss.Top, views...)
}

func (m *RootModel) writeDataToFile() tea.Cmd {
	return func() tea.Msg {
		f := filepath.Join(m.mainDir, "migration_data.json")

		jsonString, err := json.MarshalIndent(m.data, "", "  ")
		if err != nil {
			slog.Error("marshaling migration data to file", "error", err)
			m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("couldn't write data to file due to json marshal error: %w", err)))
			return nil
		}

		if err := os.WriteFile(f, jsonString, os.ModePerm); err != nil {
			slog.Error("writing migration data to file", "error", err)
			m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("couldn't write data to file: %w", err)))
			return nil
		}
		m.data.writeToOutput(goodGreenOutput("SUCCESS", "Saved data to file - run with -f flag to import next time"))
		return nil
	}
}

func importJsonFile(path string) (*MigrationData, error) {
	data := &MigrationData{}
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading migration data file: %w", err)
	}

	if err := json.Unmarshal(file, data); err != nil {
		return nil, fmt.Errorf("unmarshaling migration data file: %w", err)
	}

	return data, nil
}

func switchModel(sm tea.Model) tea.Cmd {
	return func() tea.Msg {
		return switchModelMsg(sm)
	}
}

func runSpinner(text string) string {
	return fmt.Sprintf("%s%s", text, spnr.View())
}

func convertDateStringsToTimeTime(details *timeConversionDetails) (time.Time, time.Time, error) {
	var startDate, endDate time.Time
	var err error

	start := details.startString
	if start == "" {
		start = details.startFallback
	}

	end := details.endString
	if end == "" {
		end = details.endFallback
	}

	if start != "" {
		startDate, err = migration.ConvertStringToTime(start)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("converting start date string to time.Time: %w", err)
		}
	}

	if end != "" {
		endDate, err = migration.ConvertStringToTime(end)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("converting end date string to time.Time: %w", err)
		}
	}

	return startDate, endDate, nil
}

func (m *RootModel) copyToClipboard(s string) tea.Cmd {
	return func() tea.Msg {
		if err := clipboard.WriteAll(s); err != nil {
			slog.Error("copying results to clipboard", "error", err)
			m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("couldn't copy results to clipboard: %w", err)))
			return nil
		}
		slog.Debug("copied result to clipboard")
		m.data.writeToOutput(goodGreenOutput("SUCCESS", "copied results to clipboard"))
		return nil
	}
}

func (m *RootModel) calculateDimensions(w, h int) tea.Cmd {
	dummyMenuTabs := []menuTab{tabMainPage}
	return func() tea.Msg {
		windowWidth = w
		windowHeight = h
		mainHeaderHeight = lipgloss.Height(menuBar(dummyMenuTabs, dummyMenuTabs[0]))
		mainFooterHeight = lipgloss.Height(appFooter())
		viewportDvdrHeight = lipgloss.Height(viewportDivider())
		verticalMarginHeight = mainHeaderHeight + mainFooterHeight + viewportDvdrHeight
		viewportHeight := (windowHeight - verticalMarginHeight) * 1 / 2
		verticalLeftForMainView = windowHeight - verticalMarginHeight - viewportHeight
		slog.Debug("got calculateDimensionsMsg")

		if !m.ready {
			m.viewport = viewport.New(windowWidth, viewportHeight)
		} else {
			m.viewport.Width = windowWidth
			m.viewport.Height = viewportHeight
		}

		m.viewport.SetContent(m.data.Output.String())
		m.setAutoScrollBehavior()
		slog.Debug("setting ready to true")
		m.ready = true

		return nil
	}
}

func (d *MigrationData) writeToOutput(s string) {
	d.Output.WriteString(s)
}

func (m *RootModel) setAutoScrollBehavior() {
	if m.viewport.AtBottom() {
		m.scrollOverride = false
	}

	if !m.scrollOverride {
		m.viewport.GotoBottom()
	}
}
